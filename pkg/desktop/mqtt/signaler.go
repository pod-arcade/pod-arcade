package mqtt

import (
	"context"
	"crypto/tls"
	"encoding/json"
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

type MQTTConfigurator interface {
	GetConfiguration(ctx context.Context) *MQTTConfig
}

type MQTTConfig struct {
	Host        string
	Password    string
	Username    string
	TopicPrefix string
}

var _ api.Signaler = (*MQTTSignaler)(nil)

type MQTTSignaler struct {
	Client       mqtt.Client
	configurator MQTTConfigurator

	desktop api.Desktop

	newSessionHandler api.NewSessionHandler

	sessions map[api.SessionID]api.Session

	extraIceServers []webrtc.ICEServer

	ctx context.Context
	l   zerolog.Logger
}

func NewMQTTSignaler(configurator MQTTConfigurator) *MQTTSignaler {
	client := &MQTTSignaler{
		configurator:    configurator,
		sessions:        map[api.SessionID]api.Session{},
		extraIceServers: []webrtc.ICEServer{},
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
	cfg := c.configurator.GetConfiguration(c.ctx)
	c.l = log.NewLogger("mqtt-client", map[string]string{
		"Username": cfg.Username,
		"MQTTHost": cfg.Host,
	})

	opts.AddBroker(cfg.Host)

	if cfg.Password != "" {
		opts.
			SetUsername(cfg.Username).
			SetPassword(cfg.Password)
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
	opts.OnReconnecting = func(cl mqtt.Client, opts *mqtt.ClientOptions) {
		c.l.Warn().Msgf("Reconnecting...")
		// reset the configuration on a reconnect attempt
		cfg := c.configurator.GetConfiguration(c.ctx)
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
		opts.Servers = []*url.URL{}
		opts.AddBroker(cfg.Host)
	}

	opts.WillEnabled = true
	opts.SetWill(c.getTopicPrefix()+"status", "offline", 0, true)

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

	metrics.StartAdvancedMQTTMetricsPublisher(ctx, c.getTopicPrefix(), &c.Client, time.Second*5)

	// Wait for the done context
	<-c.ctx.Done()
	c.publishOfflineMessage() // last will doesn't fire on graceful disconnect

	// If we're shutting down, disconnect the client.
	c.Client.Disconnect(1000)
	return nil
}

func (c *MQTTSignaler) onConnect(client mqtt.Client) {
	c.l.Debug().Msg("Connected over MQTT")
	// Setup subscription for offers
	client.Subscribe(c.getTopicPrefix()+"sessions/+/webrtc-offer", 0, func(client mqtt.Client, m mqtt.Message) {
		components := strings.Split(strings.Replace(m.Topic(), c.getTopicPrefix(), "", 1), "/")
		sessionId := components[1]
		sdp := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  string(m.Payload()),
		}
		c.onOffer(api.SessionID(sessionId), sdp)
	})
	c.l.Debug().Msg("Subscribed to webrtc-offer")

	// Listen for Remote ICE Candidates
	client.Subscribe(c.getTopicPrefix()+"sessions/+/offer-ice-candidate", 0, func(client mqtt.Client, m mqtt.Message) {
		components := strings.Split(strings.Replace(m.Topic(), c.getTopicPrefix(), "", 1), "/")
		sessionId := components[1]
		candidate := webrtc.ICECandidateInit{}
		err := json.Unmarshal(m.Payload(), &candidate)
		if err != nil {
			c.l.Error().Msgf("Payload is not an ICECandidateInit — %v", string(m.Payload()))
			return
		}
		if c.sessions[api.SessionID(sessionId)] != nil {
			c.l.Debug().Msgf("Remote ICE Candidate session=%v — %v", sessionId, candidate.Candidate)
			c.sessions[api.SessionID(sessionId)].GetPeerConnection().AddICECandidate(candidate)
		} else {
			c.l.Error().Msgf("Received ICE Candidate for unknown session %v", sessionId)
		}
	})
	c.l.Debug().Msg("Subscribed to offer-ice-candidate")

	// Detect bugged status
	client.Subscribe(c.getTopicPrefix()+"status", 0, func(client mqtt.Client, m mqtt.Message) {
		if string(m.Payload()) == "offline" {
			// If we're online to see this, reset us to online.
			c.l.Debug().Msg("Saw offline published from us, but we're online to see that message. Resetting status back to online.")
			c.publishOnlineMessage()
		}
	})

	// Listen for ICE Servers from the server
	client.Subscribe("server/ice-servers", 0, func(client mqtt.Client, m mqtt.Message) {
		iceServers := []webrtc.ICEServer{}
		err := json.Unmarshal(m.Payload(), &iceServers)
		if err != nil {
			c.l.Error().Msgf("Failed to decode ICE Servers. %v", err)
			return
		}
		c.l.Debug().Msgf("Received ICE Servers from server. %v", iceServers)
		c.extraIceServers = iceServers
	})
	c.l.Debug().Msg("Subscribed to offer-ice-candidate")

	c.Client.Publish(c.getTopicPrefix()+"status", 0, true, "online")
	c.l.Debug().Msg("Published online status")
}

func (c *MQTTSignaler) getTopicPrefix() string {
	cfg := c.configurator.GetConfiguration(c.ctx)
	return cfg.TopicPrefix
}

func (c *MQTTSignaler) publishOnlineMessage() {
	c.Client.Publish(c.getTopicPrefix()+"status", 0, true, "online")
}

func (c *MQTTSignaler) publishOfflineMessage() {
	c.Client.Publish(c.getTopicPrefix()+"status", 0, true, "offline")
}

func (c *MQTTSignaler) getWebRTCConfig() (*webrtc.API, *webrtc.Configuration) {
	webrtcAPI, webRTCAPIConfig := c.desktop.GetWebRTCAPI()
	// Let's not modify the original config
	// Copy the config
	var copyConfig webrtc.Configuration
	if webRTCAPIConfig != nil {
		copyConfig = *webRTCAPIConfig
		// Replace the pointer to the ICE Servers with a new slice
		copyConfig.ICEServers = []webrtc.ICEServer{}
		// Copy the old ICE Servers into the new slice
		copyConfig.ICEServers = append(copyConfig.ICEServers, webRTCAPIConfig.ICEServers...)
	} else {
		// If we don't have a config, create a new one
		copyConfig = webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{},
		}
	}

	// Append our extra ICE Servers if we have any
	if len(c.extraIceServers) > 0 {
		c.l.Debug().Msgf("Appending extra ICE Servers to config. %v + %v", webRTCAPIConfig.ICEServers, c.extraIceServers)
		copyConfig.ICEServers = append(copyConfig.ICEServers, c.extraIceServers...)
	}

	// Return the original API, and a copy of our modified config
	return webrtcAPI, &copyConfig
}

func (c *MQTTSignaler) onOffer(sessionId api.SessionID, sdp webrtc.SessionDescription) {
	c.l.Debug().Msgf("Received offer for session %v", sessionId)

	var session api.Session = c.sessions[sessionId]
	var pc *webrtc.PeerConnection

	if session == nil {
		c.l.Debug().Msgf("Creating new session %v", sessionId)
		session = desktop.NewSession(c.ctx, sessionId)
		c.sessions[sessionId] = session
		_, err := session.CreatePeerConnection(c.getWebRTCConfig())
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
			if candidate == nil {
				c.l.Debug().Msgf("Finished Gathering ICE Candidates for session %v", sessionId)
				return
			}
			c.l.Debug().Msgf("Local  ICE Candidate session=%v — %v", sessionId, candidate.ToJSON().Candidate)
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
	c.l.Debug().Msgf("Publishing answer for session %v", sessionId)
	c.Client.Publish(c.getTopicPrefix()+"sessions/"+sessionId+"/webrtc-answer", 1, false, answerSdp.SDP)
}

func (c *MQTTSignaler) publishICECandidate(sessionId string, candidate webrtc.ICECandidateInit) {
	candidateBytes, err := json.Marshal(candidate)
	if err != nil {
		panic(err) // This really can't happen...
	}
	c.l.Debug().Msgf("Publishing ice candidate for session %v", sessionId)
	c.Client.Publish(c.getTopicPrefix()+"sessions/"+sessionId+"/answer-ice-candidate", 0, false, candidateBytes)
}
