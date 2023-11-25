package api

import (
	"context"

	"github.com/pion/webrtc/v4"
)

type Desktop interface {
	WithSignaler(Signaler) Desktop
	WithGamepad(Gamepad) Desktop
	WithKeyboard(Keyboard) Desktop
	WithMouse(Mouse) Desktop
	WithVideoSource(VideoSource) Desktop
	WithAudioSource(AudioSource) Desktop
	WithWebRTCAPI(*webrtc.API, *webrtc.Configuration) Desktop

	GetSignalers() []Signaler
	GetGamepads() []Gamepad
	GetAudioSources() []AudioSource
	GetVideoSources() []VideoSource
	GetKeyboard() Keyboard
	GetMouse() Mouse
	GetWebRTCAPI() (*webrtc.API, *webrtc.Configuration)

	Run(ctx context.Context) error
}
