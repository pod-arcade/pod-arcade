package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/caarlos0/env"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/internal/udev"
	"github.com/pod-arcade/pod-arcade/pkg/desktop"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/cmd_capture"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/mqtt"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/pulseaudio"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/uinput"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/wayland"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/wf_recorder"
)

var DesktopConfig struct {
	MQTT_HOST   string `env:"MQTT_HOST" envDefault:"tcp://localhost:1883"`
	DESKTOP_ID  string `env:"DESKTOP_ID"`
	DESKTOP_PSK string `env:"DESKTOP_PSK"`

	VIDEO_QUALITY    int    `env:"VIDEO_QUALITY" envDefault:"30"`
	DISABLE_HW_ACCEL bool   `env:"DISABLE_HW_ACCEL" envDefault:"true"`
	VIDEO_PROFILE    string `env:"VIDEO_PROFILE" envDefault:"constrained_baseline"`
}

func main() {
	env.Parse(&DesktopConfig)
	if DesktopConfig.DESKTOP_ID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		DesktopConfig.DESKTOP_ID = hostname
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	// Open udev
	// This is used by our game controllers to register themselves in applications
	udev := udev.NewUDev(ctx)
	if err := udev.Open(); err != nil {
		panic(err)
	}
	defer udev.Close()

	// Create Desktop
	// Register all of the inputs, video sources, audio sources, and signalers.
	d := desktop.
		NewDesktop().
		WithVideoSource(
			cmd_capture.NewCommandCaptureH264(
				wf_recorder.NewScreenCapture(DesktopConfig.VIDEO_QUALITY, DesktopConfig.DISABLE_HW_ACCEL, DesktopConfig.VIDEO_PROFILE),
			)).
		WithAudioSource(cmd_capture.NewCommandCaptureOgg(pulseaudio.NewGSTPulseAudioCapture())).
		WithSignaler(mqtt.NewMQTTSignaler(mqtt.MQTTConfig{
			Host:       DesktopConfig.MQTT_HOST,
			DesktopID:  DesktopConfig.DESKTOP_ID,
			DesktopPSK: DesktopConfig.DESKTOP_PSK,
		})).
		WithGamepad(uinput.CreateVirtualGamepad(udev, 0, 0x045E, 0x02D1)).
		WithGamepad(uinput.CreateVirtualGamepad(udev, 1, 0x045E, 0x02D1)).
		WithGamepad(uinput.CreateVirtualGamepad(udev, 2, 0x045E, 0x02D1)).
		WithGamepad(uinput.CreateVirtualGamepad(udev, 3, 0x045E, 0x02D1)).
		WithMouse(wayland.NewWaylandInputClient(ctx))

	// Register a webrtc API. Includes all of the codecs, interceptors, etc.
	webrtcAPI, err := desktop.GetWebRTCAPI(d)
	if err != nil {
		panic(err)
	}

	// Get the webrtc API with registered NACKs and Interceptors.
	d.WithWebRTCAPI(webrtcAPI, &webrtc.Configuration{})

	// Run the desktop
	d.Run(ctx)
}
