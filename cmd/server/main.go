package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
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
	OIDCServer   string `env:"OIDC_SERVER" envDefault:"" json:"oidc_server,omitempty"`
	OIDCClientId string `env:"OIDC_CLIENT_ID" envDefault:"" json:"oidc_client_id,omitempty"`

	AuthMethod string `json:"auth_method"`

	// Not returned back from config endpoint
	DesktopPSK     string             `env:"DESKTOP_PSK" envDefault:"" json:"-"`
	ClientPSK      string             `env:"CLIENT_PSK" envDefault:"" json:"-"`
	ICEServers     []webrtc.ICEServer `json:"-"`
	ICEServersJSON string             `env:"ICE_SERVERS" envDefault:"[]" json:"-"`
	RequireAuth    bool               `env:"AUTH_REQUIRED" envDefault:"true" json:"-"`
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
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	// Create the new MQTT Server.
	server := mqtt.New(&mqtt.Options{
		InlineClient: true,
		Logger:       slog.Default(),
	})

	if !ServerConfig.RequireAuth {
		// Allow all connections.
		_ = server.AddHook(new(auth.AllowHook), nil)
		ServerConfig.AuthMethod = "none"
	} else {

		if ServerConfig.OIDCServer != "" {
			// If we have an OIDCServer setup, allow user authentication
			hook := hooks.NewOIDCHook(ctx, ServerConfig.OIDCServer, ServerConfig.OIDCClientId)
			err := server.AddHook(hook, nil)
			if err != nil {
				panic(err)
			}

			ServerConfig.AuthMethod = "oidc"
		} else if ServerConfig.ClientPSK != "" {
			// Fallback to offering ClientPSK
			hook := hooks.NewClientPSKHook(ctx, ServerConfig.ClientPSK)
			err := server.AddHook(hook, nil)
			if err != nil {
				panic(err)
			}

			ServerConfig.AuthMethod = "psk"
		} else {
			server.Log.Warn("No user authentication method was provided (PSK or OIDC)")
		}

		if ServerConfig.DesktopPSK != "" {
			// If we have an OIDCServer setup, allow user authentication
			hook := hooks.NewDesktopPSKHook(ctx, ServerConfig.DesktopPSK)
			err := server.AddHook(hook, nil)
			if err != nil {
				panic(err)
			}
		} else {
			server.Log.Warn("No Desktop PSK was configured, and auth is required. Desktops may not be able to connect.")
		}
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

	<-ctx.Done()
}
