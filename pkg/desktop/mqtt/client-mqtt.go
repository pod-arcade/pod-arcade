package api

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/api"
	"github.com/pod-arcade/pod-arcade/pkg/logger"
	"github.com/rs/zerolog"
)

type MQTTConfig struct {
	Host       string
	DesktopID  string
	DesktopPSK string
}

var _ api.ClientAPI = (*MQTTClient)(nil)

type MQTTClient struct {
	Client         mqtt.Client
	cfg            MQTTConfig
	onOffer        func(sessionId string, offerSdp webrtc.SessionDescription)
	onIceCandidate func(sessionId string, candidate webrtc.ICECandidateInit)

	ctx context.Context
	l   zerolog.Logger
}

func NewMQTTClient(ctx context.Context, cfg MQTTConfig) *MQTTClient {
	client := &MQTTClient{
		cfg: cfg,
		l: logger.CreateLogger(map[string]string{
			"Component": "MQTTClient",
			"DesktopID": cfg.DesktopID,
			"MQTTHost":  cfg.Host,
		}),
		ctx: ctx,
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Host)

	if cfg.DesktopPSK != "" {
		opts.
			SetUsername("desktop:" + cfg.DesktopID).
			SetPassword(cfg.DesktopPSK)
	}

	opts.SetAutoReconnect(true) // For reconnecting to the same server
	opts.SetConnectRetry(true)  // For reconnecting if the server doesn't remember the session

	opts.OnConnect = client.OnConnect
	opts.OnConnectAttempt = client.OnConnectionAttempt
	opts.OnConnectionLost = client.OnConnectionLost
	opts.OnReconnecting = client.OnReconnecting

	opts.WillEnabled = true
	opts.SetWill(client.getTopicPrefix()+"/status", "offline", 0, true)

	client.Client = mqtt.NewClient(opts)
	client.l.Info().Msg("Starting MQTT Client")
	client.Client.Connect()
	return client
}

func (c *MQTTClient) getTopicPrefix() string {
	return fmt.Sprintf("desktops/%v", c.cfg.DesktopID)
}

func (c *MQTTClient) OnConnect(client mqtt.Client) {
	c.l.Debug().Msg("Connected over MQTT")
	// Setup subscription for offers
	client.Subscribe(c.getTopicPrefix()+"/sessions/+/webrtc-offer", 0, func(client mqtt.Client, m mqtt.Message) {
		components := strings.Split(m.Topic(), "/")
		sessionId := components[3]
		sdp := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  string(m.Payload()),
		}
		if c.onOffer != nil {
			c.onOffer(sessionId, sdp)
		} else {
			c.l.Warn().Msg("Received offer, but no onOffer handler has been supplied")
		}
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

		if c.onIceCandidate != nil {
			c.onIceCandidate(sessionId, candidate)
		} else {
			c.l.Warn().Msg("Received ICE Candidate, but no onIceCandidate handler has been supplied")
		}
	})
	c.l.Debug().Msg("Subscribed to offer-ice-candidate")

	// Detect bugged status
	client.Subscribe(c.getTopicPrefix()+"/status", 0, func(client mqtt.Client, m mqtt.Message) {
		if string(m.Payload()) == "offline" {
			// If we're online to see this, reset us to online.
			c.l.Debug().Msg("Saw offline published from us, but we're online to see that message. Resetting status back to online.")
			c.Client.Publish(c.getTopicPrefix()+"/status", 0, true, "online")
		}
	})

	c.Client.Publish(c.getTopicPrefix()+"/status", 0, true, "online")
	c.l.Debug().Msg("Published online status")
}

func (c *MQTTClient) OnConnectionAttempt(broker *url.URL, tlsCfg *tls.Config) *tls.Config {
	c.l.Debug().Msg("Attempting to connect to MQTT Broker " + broker.String())
	return tlsCfg
}

func (c *MQTTClient) OnConnectionLost(client mqtt.Client, err error) {
	c.l.Error().Msgf("Lost connection to MQTT — %v", err)
}

func (c *MQTTClient) OnReconnecting(client mqtt.Client, opts *mqtt.ClientOptions) {
	c.l.Warn().Msgf("Reconnecting...")
}

func (c *MQTTClient) OnOffer(onOffer func(sessionId string, offerSdp webrtc.SessionDescription)) {
	c.onOffer = onOffer
}

func (c *MQTTClient) OnIceCandidate(onIceCandidate func(sessionId string, candidate webrtc.ICECandidateInit)) {
	c.onIceCandidate = onIceCandidate
}

func (c *MQTTClient) SendAnswer(sessionId string, answerSdp webrtc.SessionDescription) {
	c.Client.Publish(c.getTopicPrefix()+"/sessions/"+sessionId+"/webrtc-answer", 0, false, answerSdp.SDP)
}

func (c *MQTTClient) SendICECandidate(sessionId string, candidate webrtc.ICECandidateInit) {
	candidateBytes, err := json.Marshal(candidate) // This really shouldn't error...
	if err != nil {
		panic(err)
	}
	c.Client.Publish(c.getTopicPrefix()+"/sessions/"+sessionId+"/answer-ice-candidate", 0, false, candidateBytes)
}
