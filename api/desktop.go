package api

import (
	"context"

	"github.com/pion/webrtc/v4"
)

type Desktop interface {
	// WithSignaler adds a signaler to the desktop
	WithSignaler(Signaler) Desktop
	// WithGamepad adds a gamepad to the desktop
	WithGamepad(Gamepad) Desktop
	// WithKeyboard adds a keyboard to the desktop
	WithKeyboard(Keyboard) Desktop
	// WithMouse adds a mouse to the desktop
	WithMouse(Mouse) Desktop
	// WithVideoSource adds a video source to the desktop
	WithVideoSource(VideoSource) Desktop
	// WithAudioSource adds an audio source to the desktop
	WithAudioSource(AudioSource) Desktop
	// WithWebRTCAPI adds a webrtc api to the desktop
	WithWebRTCAPI(*webrtc.API, *webrtc.Configuration) Desktop

	// GetSignalers returns the signalers
	GetSignalers() []Signaler
	// GetGamepads returns the gamepads
	GetGamepads() []Gamepad
	// GetAudioSources returns the audio sources
	GetAudioSources() []AudioSource
	// GetVideoSources returns the video sources
	GetVideoSources() []VideoSource
	// GetKeyboard returns the keyboard
	GetKeyboard() Keyboard
	// GetMouse returns the mouse
	GetMouse() Mouse
	// GetWebRTCAPI returns the webrtc api
	GetWebRTCAPI() (*webrtc.API, *webrtc.Configuration)

	// Run starts the desktop. This is a blocking call.
	// to stop the desktop, cancel the context.
	Run(ctx context.Context) error
}
