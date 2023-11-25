package api

import "github.com/pion/webrtc/v4"

type SessionID string

type Session interface {
	GetName() string
	GetID() SessionID

	CreatePeerConnection(*webrtc.API, *webrtc.Configuration) (*webrtc.PeerConnection, error)
	GetPeerConnection() *webrtc.PeerConnection
}
