package api

import (
	"context"

	"github.com/pion/webrtc/v4"
)

// Mixer is a mixer of audio and video sources.
// You can add any number of audio and video sources to the mixer, and get out
// video and audio tracks that can be added to a webrtc peer connection.
type Mixer interface {
	AddVideoSource(VideoSource) error
	AddAudioSource(AudioSource) error

	GetAudioSources() []AudioSource
	GetVideoSources() []VideoSource

	GetAudioTracks() []*webrtc.TrackLocalStaticRTP
	GetVideoTracks() []*webrtc.TrackLocalStaticRTP

	Stream(ctx context.Context) error
}
