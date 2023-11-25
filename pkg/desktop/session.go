package desktop

import (
	"context"

	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/api"
)

var _ api.Session = (*DesktopSession)(nil)

type DesktopSession struct {
	id             api.SessionID
	peerConnection *webrtc.PeerConnection

	ctx context.Context
}

func NewSession(ctx context.Context, sessionId api.SessionID) *DesktopSession {
	session := &DesktopSession{
		id:  sessionId,
		ctx: ctx,
	}
	return session
}

func (s *DesktopSession) GetName() string {
	return "desktop-session"
}

func (s *DesktopSession) GetID() api.SessionID {
	return s.id
}

func (s *DesktopSession) CreatePeerConnection(api *webrtc.API, conf *webrtc.Configuration) (*webrtc.PeerConnection, error) {
	pc, err := api.NewPeerConnection(*conf)
	if err != nil {
		return nil, err
	}
	s.peerConnection = pc
	return pc, nil
}

func (s *DesktopSession) GetPeerConnection() *webrtc.PeerConnection {
	return s.peerConnection
}
