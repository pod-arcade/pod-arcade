package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/api"
	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog"
)

type MQTTConfig struct {
	Host      string
	DesktopID string
}

var _ api.ClientAPI = (*MQTTClient)(nil)

type MQTTClient struct {
	Client  mqtt.Client
	cfg     MQTTConfig
	onOffer func(sessionId string, offerSdp webrtc.SessionDescription)

	ctx context.Context
	l   zerolog.Logger
}

func NewMQTTClient(ctx context.Context, cfg MQTTConfig) *MQTTClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(cfg.Host)
	opts.WillEnabled = true
	opts.AutoReconnect = true

	client := &MQTTClient{
		cfg: cfg,
		l: logger.CreateLogger(map[string]string{
			"Component": "MQTTClient",
			"DesktopID": cfg.DesktopID,
			"MQTTHost":  cfg.Host,
		}),
		ctx: ctx,
	}

	opts.SetWill(client.getTopicPrefix()+"/status", "offline", 0, true)

	client.Client = mqtt.NewClient(opts)
	client.l.Info().Msg("Starting MQTT Client")
	connToken := client.Client.Connect()
	connToken.Wait()
	if err := connToken.Error(); err != nil {
		client.l.Error().Err(err).Msg("Failed to connect to MQTT Broker")
	} else {
		client.setupMQTT()
		client.l.Info().Msg("Connected!")
	}

	return client
}

func (c *MQTTClient) getTopicPrefix() string {
	return fmt.Sprintf("desktops/%v", c.cfg.DesktopID)
}

func (c *MQTTClient) setupMQTT() {
	c.Client.Subscribe(c.getTopicPrefix()+"/sessions/+/webrtc-offer", 0, func(client mqtt.Client, m mqtt.Message) {
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
	}).Wait()
	c.l.Debug().Msg("Subscribed to webrtc-offer")

	c.Client.Publish(c.getTopicPrefix()+"/status", 0, true, "online")
	c.l.Debug().Msg("Published online status")

}

func (c *MQTTClient) OnOffer(onOffer func(sessionId string, offerSdp webrtc.SessionDescription)) {
	c.onOffer = onOffer
}
func (c *MQTTClient) SendAnswer(sessionId string, answerSdp webrtc.SessionDescription) {
	c.Client.Publish(c.getTopicPrefix()+"/sessions/"+sessionId+"/webrtc-answer", 0, false, answerSdp.SDP)
}
