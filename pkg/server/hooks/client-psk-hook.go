package hooks

import (
	"bytes"
	"context"
	"strings"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type ClientPSKHook struct {
	PreSharedKey string

	ctx context.Context

	mqtt.HookBase
}

func NewClientPSKHook(ctx context.Context, psk string) *ClientPSKHook {
	h := &ClientPSKHook{
		PreSharedKey: psk,
		ctx:          ctx,
	}

	return h
}

func (h *ClientPSKHook) ID() string {
	return "ClientPSKHook"
}

func (h *ClientPSKHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnectAuthenticate,
		mqtt.OnACLCheck,
	}, []byte{b})
}

func (h *ClientPSKHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	user := string(pk.Connect.Username)
	pass := string(pk.Connect.Password)

	if !strings.HasPrefix(user, "user:") {
		h.Log.Debug("User %v does not match prefix `user:`", user)
		return false
	}

	user = strings.TrimPrefix(user, "user:")

	if pass != h.PreSharedKey {
		h.Log.Warn("User failed to authenticated. Incorrect PSK provided", "User", user)
	}

	h.Log.Info("User authenticated", "User", user)

	return true
}

func (h *ClientPSKHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	return true // do some checking later maybe
}
