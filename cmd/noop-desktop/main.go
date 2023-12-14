package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/caarlos0/env"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/pkg/desktop"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/cmd_capture"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/mqtt"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/sample_recorder"
)

var DesktopConfig struct {
	MQTT_HOST   string `env:"MQTT_HOST" envDefault:"tcp://localhost:1883"`
	DESKTOP_ID  string `env:"DESKTOP_ID"`
	DESKTOP_PSK string `env:"DESKTOP_PSK"`

	WEBRTC_PORT int      `env:"WEBRTC_PORT" envDefault:"0"`
	WEBRTC_IPS  []string `env:"WEBRTC_IPS"`

	CLOUD_AUTH_KEY string `env:"CLOUD_AUTH_KEY" envDefault:""`
	CLOUD_URL      string `env:"CLOUD_URL" envDefault:"https://cloud.podarcade.com"`
}

func getMQTTConfigurator() mqtt.MQTTConfigurator {
	if DesktopConfig.CLOUD_AUTH_KEY == "" {
		return mqtt.NewLocalMQTTConfigurator(DesktopConfig.MQTT_HOST, DesktopConfig.DESKTOP_PSK, DesktopConfig.DESKTOP_ID)
	} else {
		return mqtt.NewCloudMQTTConfigurator(DesktopConfig.CLOUD_URL, DesktopConfig.CLOUD_AUTH_KEY)
	}
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
	// udev := udev.NewUDev(ctx)
	// if err := udev.Open(); err != nil {
	// 	panic(err)
	// }
	// defer udev.Close()

	// Create Desktop
	// Register all of the inputs, video sources, audio sources, and signalers.
	d := desktop.
		NewDesktop().
		WithSignaler(mqtt.NewMQTTSignaler(getMQTTConfigurator())).
		WithVideoSource(cmd_capture.NewCommandCaptureH264(&sample_recorder.SampleRecorder{}))

	// Register a webrtc API. Includes all of the codecs, interceptors, etc.
	webrtcAPI, err := desktop.GetWebRTCAPI(d, &desktop.WebRTCAPIConfig{
		SinglePort:  DesktopConfig.WEBRTC_PORT,
		ExternalIPs: DesktopConfig.WEBRTC_IPS,
	})
	if err != nil {
		panic(err)
	}

	// Get the webrtc API with registered NACKs and Interceptors.
	d.WithWebRTCAPI(webrtcAPI, &webrtc.Configuration{})

	// Run the desktop
	d.Run(ctx)
}
