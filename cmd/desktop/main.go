package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop"
	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/input"
	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/media/audio"
	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/media/video"
	mqtt "github.com/JohnCMcDonough/pod-arcade/pkg/desktop/mqtt"
	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/webrtc"
	"github.com/JohnCMcDonough/pod-arcade/pkg/metrics"
	"github.com/caarlos0/env/v9"
)

var cfg DesktopConfig

type DesktopConfig struct {
	MQTTHost  string `env:"MQTT_HOST,expand" envDefault:"ws://localhost:8080/mqtt"`
	DesktopID string `env:"DESKTOP_ID,expand" envDefault:""`
}

func init() {
	env.Parse(&cfg)
	if cfg.DesktopID == "" {
		if hostname, err := os.Hostname(); err != nil {
			panic(err)
		} else {
			cfg.DesktopID = hostname
		}
	}
}

func main() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt, os.Kill)

	api := mqtt.NewMQTTClient(ctx, mqtt.MQTTConfig{
		Host:      cfg.MQTTHost,
		DesktopID: cfg.DesktopID,
	})

	audioSource := audio.NewPulseAudioCapture()
	videoSource := video.NewScreenCapture(30, true)

	metrics.CaptureMetricsForMediaSource(audioSource)
	metrics.CaptureMetricsForMediaSource(videoSource)

	mixer, err := webrtc.NewWebRTCMixer(ctx, audioSource, videoSource)
	if err != nil {
		panic(err)
	}

	inputHub, err := input.NewInputHub(ctx)
	if err != nil {
		panic(err)
	}

	_ = desktop.NewDesktop(ctx, api, mixer, inputHub)

	metrics.StartMQTTMetricsPublisher(ctx, cfg.DesktopID, &api.Client, 5*time.Second)

	// begin streaming
	mixer.Stream()

	<-inputHub.Done()

	<-ctx.Done()
}
