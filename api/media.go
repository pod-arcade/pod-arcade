package api

import (
	"context"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type AudioSource interface {
	GetAudioCodecParameters() webrtc.RTPCodecParameters
	StreamAudio(ctx context.Context, pktChan chan<- *rtp.Packet) error
	MediaSource
}
type VideoSource interface {
	GetVideoCodecParameters() webrtc.RTPCodecParameters
	StreamVideo(ctx context.Context, pktChan chan<- *rtp.Packet) error
	MediaSource
}

type MediaSource interface {
	GetName() string
}
