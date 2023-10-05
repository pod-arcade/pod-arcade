package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	"github.com/JohnCMcDonough/pod-arcade/pkg/metrics"
	"github.com/JohnCMcDonough/pod-arcade/pkg/server/hooks"
	palisteners "github.com/JohnCMcDonough/pod-arcade/pkg/server/listeners"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
)

//go:embed build
var httpStaticContent embed.FS

func main() {

	l := logger.CreateLogger(map[string]string{
		"Component": "Server",
	})

	// Create signals channel to run server until interrupted
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()

	// Create the new MQTT Server.
	server := mqtt.New(nil)

	// Allow all connections.
	_ = server.AddHook(new(auth.AllowHook), nil)
	_ = server.AddHook(hooks.NewClearRetainedHook(server), nil)

	// Create a TCP listener on a standard port.
	tcp := listeners.NewTCP("tcp1", ":1883", nil)
	err := server.AddListener(tcp)
	if err != nil {
		log.Fatal(err)
	}

	// Create a WebRTC listener on a standard port.
	ws := palisteners.NewWebsocket("ws1")
	err = server.AddListener(ws)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/mqtt", ws.Handler)
	// load static content embedded in the app
	contentStatic, _ := fs.Sub(httpStaticContent, "build")
	mux.Handle("/", http.FileServer(http.FS(contentStatic)))
	l.Debug().Msg("Serving FileSystem")

	mux.Handle("/metrics", metrics.Handle())
	l.Debug().Msg("Serving Metrics")

	go func() {
		err := server.Serve()
		if err != nil {
			log.Fatal(err)
		}
	}()

	go func() {
		if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
			panic(err)
		}
	}()

	<-done
}
