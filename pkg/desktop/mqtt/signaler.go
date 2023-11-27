package mqtt

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/pkg/desktop"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/pod-arcade/pod-arcade/pkg/metrics"
	"github.com/rs/zerolog"
)

type MQTTConfig struct {
	Host       string
	DesktopID  string
	DesktopPSK string
}

var _ api.Signaler = (*MQTTSignaler)(nil)

type MQTTSignaler struct {
	Client mqtt.Client
	cfg    MQTTConfig

	desktop api.Desktop

	newSessionHandler api.NewSessionHandler

	sessions map[api.SessionID]api.Session

	ctx context.Context
	l   zerolog.Logger
}

func NewMQTTSignaler(cfg MQTTConfig) *MQTTSignaler {
	client := &MQTTSignaler{
		cfg: cfg,
		l: log.NewLogger("mqtt-client", map[string]string{
			"DesktopID": cfg.DesktopID,
			"MQTTHost":  cfg.Host,
		}),
		sessions: map[api.SessionID]api.Session{},
	}

	return client
}

func (c *MQTTSignaler) GetName() string {
	return "mqtt-client"
}

func (c *MQTTSignaler) SetNewSessionHandler(h api.NewSessionHandler) error {
	c.newSessionHandler = h
	return nil
}

// Used to open the Signaler, and use this webrtc api to create new sessions
func (c *MQTTSignaler) Run(ctx context.Context, desktop api.Desktop) error {
	c.desktop = desktop
	c.ctx = ctx

	opts := mqtt.NewClientOptions()

	opts.AddBroker(c.cfg.Host)

	if c.cfg.DesktopPSK != "" {
		opts.
			SetUsername("desktop:" + c.cfg.DesktopID).
			SetPassword(c.cfg.DesktopPSK)
	}

	opts.SetAutoReconnect(true) // For reconnecting to the same server
	opts.SetConnectRetry(true)  // For reconnecting if the server doesn't remember the session

	opts.OnConnect = c.onConnect
	opts.OnConnectAttempt = func(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
		c.l.Debug().Msg("Attempting to connect to MQTT Broker " + broker.String())
		return tlsCfg
	}
	opts.OnConnectionLost = func(_ mqtt.Client, err error) {
		c.l.Error().Msgf("Lost connection to MQTT — %v", err)
	}
	opts.OnReconnecting = func(_ mqtt.Client, _ *mqtt.ClientOptions) {
		c.l.Warn().Msgf("Reconnecting...")
	}

	opts.WillEnabled = true
	opts.SetWill(c.getTopicPrefix()+"/status", "offline", 0, true)

	// periodically publish online messages while we're still running
	go func() {
		ticker := time.NewTicker(time.Second * 60)
		for {
			select {
			case <-ticker.C:
				c.publishOnlineMessage()
			case <-c.ctx.Done():
				return
			}
		}
	}()

	// Create a new client from the provided options, and connect to the MQTT Broker
	c.Client = mqtt.NewClient(opts)
	c.l.Info().Msg("Starting MQTT Client")
	// Calling connect should trigger the onConnect when the client is connected.
	// This is where we setup our subscriptions and handle session events
	c.Client.Connect()

	metrics.StartAdvancedMQTTMetricsPublisher(ctx, c.cfg.DesktopID, &c.Client, time.Second*5)

	// Wait for the done context
	<-c.ctx.Done()

	// If we're shutting down, disconnect the client.
	c.Client.Disconnect(1000)
	return nil
}

func (c *MQTTSignaler) onConnect(client mqtt.Client) {
	c.l.Debug().Msg("Connected over MQTT")
	// Setup subscription for offers
	client.Subscribe(c.getTopicPrefix()+"/sessions/+/webrtc-offer", 0, func(client mqtt.Client, m mqtt.Message) {
		components := strings.Split(m.Topic(), "/")
		sessionId := components[3]
		sdp := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  string(m.Payload()),
		}
		c.onOffer(api.SessionID(sessionId), sdp)
	})
	c.l.Debug().Msg("Subscribed to webrtc-offer")

	// Listen for Remote ICE Candidates
	client.Subscribe(c.getTopicPrefix()+"/sessions/+/offer-ice-candidate", 0, func(client mqtt.Client, m mqtt.Message) {
		components := strings.Split(m.Topic(), "/")
		sessionId := components[3]
		candidate := webrtc.ICECandidateInit{}
		err := json.Unmarshal(m.Payload(), &candidate)
		if err != nil {
			c.l.Error().Msgf("Payload is not an ICECandidateInit — %v", string(m.Payload()))
			return
		}
		if c.sessions[api.SessionID(sessionId)] != nil {
			c.sessions[api.SessionID(sessionId)].GetPeerConnection().AddICECandidate(candidate)
		} else {
			c.l.Error().Msgf("Received ICE Candidate for unknown session %v", sessionId)
		}
	})
	c.l.Debug().Msg("Subscribed to offer-ice-candidate")

	// Detect bugged status
	client.Subscribe(c.getTopicPrefix()+"/status", 0, func(client mqtt.Client, m mqtt.Message) {
		if string(m.Payload()) == "offline" {
			// If we're online to see this, reset us to online.
			c.l.Debug().Msg("Saw offline published from us, but we're online to see that message. Resetting status back to online.")
			c.publishOnlineMessage()
		}
	})

	c.Client.Publish(c.getTopicPrefix()+"/status", 0, true, "online")
	c.l.Debug().Msg("Published online status")
}

func (c *MQTTSignaler) getTopicPrefix() string {
	return fmt.Sprintf("desktops/%v", c.cfg.DesktopID)
}

func (c *MQTTSignaler) publishOnlineMessage() {
	c.Client.Publish(c.getTopicPrefix()+"/status", 0, true, "online")
}

func (c *MQTTSignaler) onOffer(sessionId api.SessionID, sdp webrtc.SessionDescription) {
	c.l.Debug().Msgf("Received offer for session %v", sessionId)

	var session api.Session
	var pc *webrtc.PeerConnection

	if c.sessions[sessionId] == nil {
		c.l.Debug().Msgf("Creating new session %v", sessionId)
		session = desktop.NewSession(c.ctx, sessionId)
		c.sessions[sessionId] = session
		_, err := session.CreatePeerConnection(c.desktop.GetWebRTCAPI())
		if err != nil {
			c.l.Error().Err(err).Msg("Failed to create peer connection")
			return
		}
		if c.newSessionHandler != nil {
			c.newSessionHandler(session)
		}
		pc = session.GetPeerConnection()
		pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
			if state == webrtc.PeerConnectionStateDisconnected || state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateFailed {
				c.l.Debug().Msgf("Peer Connection for session %v has disconnected", sessionId)
				delete(c.sessions, sessionId)
			}
		})
		pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
			c.l.Debug().Msgf("Got ICE Candidate for session %v", sessionId)
			if candidate == nil {
				return
			}
			c.publishICECandidate(string(sessionId), candidate.ToJSON())
		})
	} else {
		pc = session.GetPeerConnection()
	}
	err := pc.SetRemoteDescription(sdp)
	if err != nil {
		c.l.Error().Err(err).Msg("Failed to set local description")
		return
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		c.l.Error().Err(err).Msg("Failed to create answer")
		return
	}
	pc.SetLocalDescription(answer)
	c.publishAnswer(string(sessionId), answer)
}

func (c *MQTTSignaler) publishAnswer(sessionId string, answerSdp webrtc.SessionDescription) {
	c.Client.Publish(c.getTopicPrefix()+"/sessions/"+sessionId+"/webrtc-answer", 0, false, answerSdp.SDP)
}

func (c *MQTTSignaler) publishICECandidate(sessionId string, candidate webrtc.ICECandidateInit) {
	candidateBytes, err := json.Marshal(candidate)
	if err != nil {
		panic(err) // This really can't happen...
	}
	c.Client.Publish(c.getTopicPrefix()+"/sessions/"+sessionId+"/answer-ice-candidate", 0, false, candidateBytes)
}