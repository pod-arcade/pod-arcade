package hooks

import (
	"bytes"
	"regexp"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type ClearRetainedHook struct {
	server *mqtt.Server

	mqtt.HookBase
}

func NewClearRetainedHook(server *mqtt.Server) *ClearRetainedHook {
	return &ClearRetainedHook{
		server: server,
	}
}

func (h *ClearRetainedHook) ID() string {
	return "ClearRetainedHook"
}

func (h *ClearRetainedHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnWillSent,
	}, []byte{b})
}

func (h *ClearRetainedHook) OnWillSent(cl *mqtt.Client, pk packets.Packet) {
	topic := pk.TopicName
	h.Log.Info("Sending last will message — " + topic)
	if !regexp.MustCompile(`desktops/([^\/]+)/sessions/([^\/]+)/status`).MatchString(topic) {
		h.Log.Info("Ignoring last will message that isn't from a session going offline — " + topic)
		return
	}
	if string(pk.Payload) != "offline" {
		return
	}
	h.Log.Info("Payload is offline — " + topic)

	for id, pk := range h.server.Topics.Retained.GetAll() {
		if pk.TopicName == topic {
			h.Log.Info("Found and deleting retained topic — " + topic)
			h.server.Topics.Retained.Delete(id)
			return
		}
	}
}
