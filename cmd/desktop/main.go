package main

import (
	"context"
	"encoding/json"
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
	"github.com/pod-arcade/pod-arcade/pkg/log"
)

var DesktopConfig struct {
	MQTT_HOST   string `env:"MQTT_HOST" envDefault:"tcp://localhost:1883"`
	DESKTOP_ID  string `env:"DESKTOP_ID"`
	DESKTOP_PSK string `env:"DESKTOP_PSK"`

	VIDEO_QUALITY    int    `env:"VIDEO_QUALITY" envDefault:"30"`
	DISABLE_HW_ACCEL bool   `env:"DISABLE_HW_ACCEL" envDefault:"true"`
	VIDEO_PROFILE    string `env:"VIDEO_PROFILE" envDefault:"constrained_baseline"`

	WEBRTC_PORT int      `env:"WEBRTC_PORT" envDefault:"0"`
	WEBRTC_IPS  []string `env:"WEBRTC_IPS"`

	ICEServers     []webrtc.ICEServer `json:"-"`
	ICEServersJSON string             `env:"ICE_SERVERS" envDefault:"" json:"-"`

	CLOUD_AUTH_KEY string `env:"CLOUD_AUTH_KEY" envDefault:""`
	CLOUD_URL      string `env:"CLOUD_URL" envDefault:"https://play.podarcade.com"`
}

var logger = log.NewLogger("desktop", map[string]string{})

func configureICE() error {
	if DesktopConfig.ICEServersJSON != "" {
		err := json.Unmarshal([]byte(DesktopConfig.ICEServersJSON), &DesktopConfig.ICEServers)
		if err != nil {
			logger.Fatal().Msgf("Failed to decode ICE Servers, should be json array. %v", err)
		}
	}
	return nil
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
	err := configureICE()
	if err != nil {
		panic(err)
	}
	if DesktopConfig.DESKTOP_ID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		DesktopConfig.DESKTOP_ID = hostname
	}
	logger.Debug().Msgf("Starting Desktop with ID: %v", DesktopConfig.DESKTOP_ID)
	logger.Debug().Msgf("\tMQTT_HOST: %v", DesktopConfig.MQTT_HOST)
	logger.Debug().Msgf("\tVIDEO_QUALITY: %v", DesktopConfig.VIDEO_QUALITY)
	logger.Debug().Msgf("\tVIDEO_PROFILE: %v", DesktopConfig.VIDEO_PROFILE)
	logger.Debug().Msgf("\tHARDWARE_ACCELERATION: %v", !DesktopConfig.DISABLE_HW_ACCEL)
	logger.Debug().Msgf("\tWEBRTC_PORT: %v (0 means auto discover them)", DesktopConfig.WEBRTC_PORT)
	logger.Debug().Msgf("\tWEBRTC_IPS: %v", DesktopConfig.WEBRTC_IPS)

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

	wc := wayland.NewWaylandInputClient(ctx)

	d := desktop.
		NewDesktop().
		WithVideoSource(
			cmd_capture.NewCommandCaptureH264(
				wf_recorder.NewScreenCapture(DesktopConfig.VIDEO_QUALITY, !DesktopConfig.DISABLE_HW_ACCEL, DesktopConfig.VIDEO_PROFILE),
			)).
		WithAudioSource(cmd_capture.NewCommandCaptureOgg(pulseaudio.NewGSTPulseAudioCapture())).
		WithSignaler(mqtt.NewMQTTSignaler(getMQTTConfigurator())).
		WithGamepad(uinput.CreateVirtualGamepad(udev, 0, 0x045E, 0x02D1)).
		WithGamepad(uinput.CreateVirtualGamepad(udev, 1, 0x045E, 0x02D1)).
		WithGamepad(uinput.CreateVirtualGamepad(udev, 2, 0x045E, 0x02D1)).
		WithGamepad(uinput.CreateVirtualGamepad(udev, 3, 0x045E, 0x02D1)).
		WithMouse(wc).
		WithKeyboard(wc)

	// Register a webrtc API. Includes all of the codecs, interceptors, etc.
	webrtcAPI, err := desktop.GetWebRTCAPI(d, &desktop.WebRTCAPIConfig{
		SinglePort:  DesktopConfig.WEBRTC_PORT,
		ExternalIPs: DesktopConfig.WEBRTC_IPS,
	})
	if err != nil {
		panic(err)
	}

	// Get the webrtc API with registered NACKs and Interceptors.
	d.WithWebRTCAPI(webrtcAPI, &webrtc.Configuration{
		ICEServers: DesktopConfig.ICEServers,
	})

	// Run the desktop
	d.Run(ctx)
}
