package hooks

import (
	"bytes"
	"context"
	"strings"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
)

type DesktopPSKHook struct {
	PreSharedKey string

	ctx context.Context

	mqtt.HookBase
}

func NewDesktopPSKHook(ctx context.Context, psk string) *DesktopPSKHook {
	h := &DesktopPSKHook{
		PreSharedKey: psk,
		ctx:          ctx,
	}

	return h
}

func (h *DesktopPSKHook) ID() string {
	return "DesktopPSKHook"
}

func (h *DesktopPSKHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnectAuthenticate,
		mqtt.OnACLCheck,
	}, []byte{b})
}

func (h *DesktopPSKHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	user := string(pk.Connect.Username)
	pass := string(pk.Connect.Password)

	if !strings.HasPrefix(user, "desktop:") {
		h.Log.Debug("User %v does not match prefix `user:`", user)
		return false
	}

	user = strings.TrimPrefix(user, "desktop:")

	if pass != h.PreSharedKey {
		h.Log.Warn("Desktop failed to authenticated. Incorrect PSK provided", "desktop", user)
	}

	h.Log.Info("Desktop authenticated", "desktop", user)

	return true
}

func (h *DesktopPSKHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {
	return true // do some checking later maybe
}
