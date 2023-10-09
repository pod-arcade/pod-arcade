package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/caarlos0/env/v9"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/pkg/metrics"
	"github.com/pod-arcade/pod-arcade/pkg/server/handlers"
	"github.com/pod-arcade/pod-arcade/pkg/server/hooks"
	palisteners "github.com/pod-arcade/pod-arcade/pkg/server/listeners"
)

var ServerConfig struct {
	ICEServers   []webrtc.ICEServer `json:"ice_servers"`
	OIDCServer   string             `env:"OIDC_SERVER" envDefault:"" json:"oidc_server"`
	OIDCClientId string             `env:"OIDC_CLIENT_ID" envDefault:"" json:"oidc_client_id"`

	// Not returned back from config endpoint
	DesktopPSK     string `env:"DESKTOP_PSK" envDefault:"" json:"-"`
	ICEServersJSON string `env:"ICE_SERVERS" envDefault:"[]" json:"-"`
	RequireAuth    bool   `env:"AUTH_REQUIRED" envDefault:"true" json:"-"`
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
		Logger:       slog.Default(),
	})

	if !ServerConfig.RequireAuth {
		// Allow all connections.
		_ = server.AddHook(new(auth.AllowHook), nil)
	}

	if ServerConfig.OIDCServer != "" {
		// If we have an OIDCServer setup, allow user authentication
		hook, err := hooks.NewOauthHook(context.Background(), ServerConfig.OIDCServer, ServerConfig.OIDCClientId)
		if err != nil {
			panic(err)
		}
		_ = server.AddHook(hook, nil)
	}

	if ServerConfig.DesktopPSK != "" {
		// If we have an OIDCServer setup, allow user authentication
		hook, err := hooks.NewDesktopPSKHook(context.Background(), ServerConfig.DesktopPSK)
		if err != nil {
			panic(err)
		}
		_ = server.AddHook(hook, nil)
	}

	// Always set up the clear retained hook
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

	server.Log.Debug("Serving FileSystem")

	mux.Handle("/metrics", metrics.Handle())
	server.Log.Debug("Serving Metrics")

	mux.HandleFunc("/config.json", func(w http.ResponseWriter, r *http.Request) {
		response, err := json.Marshal(ServerConfig)
		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte("Failed to marshal JSON config"))
			return
		}
		w.WriteHeader(200)
		w.Write(response)
	})
	server.Log.Debug("Serving Metrics")

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
