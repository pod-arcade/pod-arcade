package hooks

import (
	"bytes"

	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"github.com/rs/zerolog"
)

type LogHook struct {
	l zerolog.Logger

	mqtt.HookBase
}

func NewHookLogger(logger zerolog.Logger) *LogHook {
	return &LogHook{
		l: logger,
	}
}

func (h *LogHook) ID() string {
	return "LogHook"
}

func (h *LogHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnStarted,
		mqtt.OnStopped,
		mqtt.OnConnect,
		mqtt.OnSessionEstablish,
		mqtt.OnSessionEstablished,
		mqtt.OnDisconnect,
		mqtt.OnAuthPacket,
		mqtt.OnPacketRead,
		mqtt.OnPacketEncode,
		mqtt.OnPacketSent,
		mqtt.OnPacketProcessed,
		mqtt.OnSubscribe,
		mqtt.OnSubscribed,
		mqtt.OnSelectSubscribers,
		mqtt.OnUnsubscribe,
		mqtt.OnUnsubscribed,
		mqtt.OnPublish,
		mqtt.OnPublished,
		mqtt.OnPublishDropped,
		mqtt.OnRetainMessage,
		mqtt.OnRetainPublished,
		mqtt.OnQosPublish,
		mqtt.OnQosComplete,
		mqtt.OnQosDropped,
		mqtt.OnPacketIDExhausted,
		mqtt.OnWill,
		mqtt.OnWillSent,
		mqtt.OnClientExpired,
		mqtt.OnRetainedExpired,
	}, []byte{b})
}

func (h *LogHook) OnStarted() {

}

func (h *LogHook) OnStopped() {

}

func (h *LogHook) OnConnect(cl *mqtt.Client, pk packets.Packet) error {
	h.l.Info().Msgf("Client %v connected from %v", cl.ID, cl.Net.Remote)
	return nil
}

func (h *LogHook) OnSessionEstablish(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnSessionEstablished(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnDisconnect(cl *mqtt.Client, err error, expire bool) {

}

func (h *LogHook) OnAuthPacket(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	return pk, nil
}

func (h *LogHook) OnPacketRead(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	return pk, nil
}

func (h *LogHook) OnPacketEncode(cl *mqtt.Client, pk packets.Packet) packets.Packet {
	return pk
}

func (h *LogHook) OnPacketSent(cl *mqtt.Client, pk packets.Packet, b []byte) {

}

func (h *LogHook) OnPacketProcessed(cl *mqtt.Client, pk packets.Packet, err error) {

}

func (h *LogHook) OnSubscribe(cl *mqtt.Client, pk packets.Packet) packets.Packet {
	return pk
}

func (h *LogHook) OnSubscribed(cl *mqtt.Client, pk packets.Packet, reasonCodes []byte) {

}

func (h *LogHook) OnSelectSubscribers(subs *mqtt.Subscribers, pk packets.Packet) *mqtt.Subscribers {
	return subs
}

func (h *LogHook) OnUnsubscribe(cl *mqtt.Client, pk packets.Packet) packets.Packet {
	return pk
}

func (h *LogHook) OnUnsubscribed(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnPublish(cl *mqtt.Client, pk packets.Packet) (packets.Packet, error) {
	return pk, nil
}

func (h *LogHook) OnPublished(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnPublishDropped(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnRetainMessage(cl *mqtt.Client, pk packets.Packet, r int64) {

}

func (h *LogHook) OnRetainPublished(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnQosPublish(cl *mqtt.Client, pk packets.Packet, sent int64, resends int) {

}

func (h *LogHook) OnQosComplete(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnQosDropped(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnPacketIDExhausted(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnWill(cl *mqtt.Client, will mqtt.Will) (mqtt.Will, error) {
	return will, nil
}

func (h *LogHook) OnWillSent(cl *mqtt.Client, pk packets.Packet) {

}

func (h *LogHook) OnClientExpired(cl *mqtt.Client) {

}

func (h *LogHook) OnRetainedExpired(filter string) {

}
