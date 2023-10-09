package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/caarlos0/env/v9"
	"github.com/pod-arcade/pod-arcade/pkg/desktop"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/input"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/media/audio"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/media/video"
	mqtt "github.com/pod-arcade/pod-arcade/pkg/desktop/mqtt"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/webrtc"
	"github.com/pod-arcade/pod-arcade/pkg/metrics"
)

var cfg DesktopConfig

type DesktopConfig struct {
	MQTTHost   string `env:"MQTT_HOST,expand" envDefault:"ws://localhost:8080/mqtt"`
	DesktopPSK string `env:"DESKTOP_PSK,expand" envDefault:""`

	DesktopID               string `env:"DESKTOP_ID,expand" envDefault:""`
	H264Quality             int    `env:"H264_QUALITY" envDefault:"30"`
	H264Profile             string `env:"H264_PROFILE,expand" envDefault:"constrained_baseline"`
	DisableHardwareEncoding bool   `env:"DISABLE_HW_ACCEL,expand" envDefault:"false"`

	AudioDropRate float32 `env:"SIM_AUDIO_DROP_RATE,expand" envDefault:"0"`
	VideoDropRate float32 `env:"SIM_VIDEO_DROP_RATE,expand" envDefault:"0"`
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
		Host:       cfg.MQTTHost,
		DesktopID:  cfg.DesktopID,
		DesktopPSK: cfg.DesktopPSK,
	})

	audioSource := audio.NewPulseAudioCapture()
	videoSource := video.NewScreenCapture(cfg.H264Quality, !cfg.DisableHardwareEncoding, cfg.H264Profile)

	metrics.CaptureMetricsForMediaSource(audioSource)
	metrics.CaptureMetricsForMediaSource(videoSource)

	mixer, err := webrtc.NewWebRTCMixer(ctx, audioSource, videoSource)
	if err != nil {
		panic(err)
	}
	mixer.AudioDropRate = cfg.AudioDropRate
	mixer.VideoDropRate = cfg.VideoDropRate

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
