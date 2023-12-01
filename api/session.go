package api

import "github.com/pion/webrtc/v4"

type SessionID string

type Session interface {
	// GetName returns the name of the session
	GetName() string

	// GetID returns the ID of the session
	GetID() SessionID

	// CreatePeerConnection creates a peer connection
	CreatePeerConnection(*webrtc.API, *webrtc.Configuration) (*webrtc.PeerConnection, error)

	// GetPeerConnection returns the peer connection
	GetPeerConnection() *webrtc.PeerConnection
}
