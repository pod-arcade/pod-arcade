package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/pkg/desktop"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/mqtt"
)

func main() {
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
		WithSignaler(mqtt.NewMQTTSignaler(mqtt.MQTTConfig{
			Host:       os.Getenv("MQTT_HOST"),
			DesktopID:  os.Getenv("DESKTOP_ID"),
			DesktopPSK: os.Getenv("DESKTOP_PSK"),
		}))

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
