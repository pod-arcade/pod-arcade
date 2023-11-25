package api

import (
	"context"

	"github.com/pion/webrtc/v4"
)

type Mixer interface {
	AddVideoSource(VideoSource) error
	AddAudioSource(AudioSource) error

	GetAudioSources() []AudioSource
	GetVideoSources() []VideoSource

	GetAudioTracks() []*webrtc.TrackLocalStaticRTP
	GetVideoTracks() []*webrtc.TrackLocalStaticRTP

	Stream(ctx context.Context) error
}
