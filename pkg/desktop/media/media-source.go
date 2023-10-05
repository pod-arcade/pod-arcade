package media

import (
	"context"

	"github.com/pion/rtp"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
)

type RTPMediaSourceType int

const (
	TYPE_VIDEO RTPMediaSourceType = iota
	TYPE_AUDIO
)

type RTPMediaSource interface {
	GetName() string
	GetType() RTPMediaSourceType
	GetCodecParameters() webrtc.RTPCodecParameters
	OnDroppedRTPPacket(func(*rtp.Packet))
	AddSDPExtensions(*sdp.SessionDescription) *sdp.SessionDescription

	Stream(context.Context, chan<- *rtp.Packet) error
}

type RTPMediaSourceBase struct {
	DroppedPacketSubscribers []func(*rtp.Packet)
}

func (b *RTPMediaSourceBase) DropRTPPacket(p *rtp.Packet) {
	for _, s := range b.DroppedPacketSubscribers {
		s(p)
	}
}

func (b *RTPMediaSourceBase) OnDroppedRTPPacket(f func(*rtp.Packet)) {
	b.DroppedPacketSubscribers = append(b.DroppedPacketSubscribers, f)
}

func (b *RTPMediaSourceBase) AddSDPExtensions(sdp *sdp.SessionDescription) *sdp.SessionDescription {
	return sdp
}
