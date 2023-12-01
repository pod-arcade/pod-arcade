package api

import (
	"context"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type AudioSource interface {
	// GetAudioCodecParameters returns the audio codec parameters
	GetAudioCodecParameters() webrtc.RTPCodecParameters

	// StreamAudio streams audio to the channel. This is a blocking call.
	// to stop streaming, cancel the context.
	StreamAudio(ctx context.Context, pktChan chan<- *rtp.Packet) error

	MediaSource
}
type VideoSource interface {
	// GetVideoCodecParameters returns the video codec parameters
	GetVideoCodecParameters() webrtc.RTPCodecParameters
	// StreamVideo streams video to the channel. This is a blocking call.
	// to stop streaming, cancel the context.
	StreamVideo(ctx context.Context, pktChan chan<- *rtp.Packet) error

	MediaSource
}

type MediaSource interface {
	// GetName returns the name of the media source
	GetName() string
}
