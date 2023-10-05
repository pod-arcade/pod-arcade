package session

import "github.com/caarlos0/env/v9"

type sessionConfig struct {
	// ICEServers []string `env:"ICE_SERVERS" envSeparator:"," envDefault:"stun:stun.l.google.com:19302"`
	ICEServers []string `env:"ICE_SERVERS" envSeparator:"," envDefault:""`
}

var cfg sessionConfig

func init() {
	env.Parse(&cfg)
}
