package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	"github.com/JohnCMcDonough/pod-arcade/pkg/metrics"
	"github.com/JohnCMcDonough/pod-arcade/pkg/server/handlers"
	"github.com/JohnCMcDonough/pod-arcade/pkg/server/hooks"
	palisteners "github.com/JohnCMcDonough/pod-arcade/pkg/server/listeners"
	"github.com/caarlos0/env/v9"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/pion/webrtc/v4"
)

var ServerConfig struct {
	ICEServersJSON string `env:"ICE_SERVERS" envDefault:"[]"`
	ICEServers     []webrtc.ICEServer
}

func init() {
	env.Parse(&ServerConfig)
	err := json.Unmarshal([]byte(ServerConfig.ICEServersJSON), &ServerConfig.ICEServers)
	if err != nil {
		log.Fatalf("Failed to decode ICE Servers, should be json array. %v", err)
	}
}

func publishICEServers(server *mqtt.Server) {
	iceServerJSON, err := json.Marshal(ServerConfig.ICEServers)
	if err != nil {
		log.Fatalf("Failed to encode ICE Servers. %v", err)
	}

	err = server.Publish("server/ice-servers", iceServerJSON, true, 0)
	if err != nil {
		log.Fatalf("Failed to send ICE Servers. %v", err)
	}
}

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
	server := mqtt.New(&mqtt.Options{
		InlineClient: true,
	})

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
	spaHandler, err := handlers.NewSinglePageAppHandler(httpStaticContent, "/index.html", "build")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", spaHandler)
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

	publishICEServers(server)

	<-done
}
