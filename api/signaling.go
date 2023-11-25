package api

import (
	"context"

	"github.com/pion/webrtc/v4"
)

type OfferHandler func(sessionID SessionID, offerSdp webrtc.SessionDescription)
type ICECandidateHandler func(sessionID SessionID, offerSdp webrtc.SessionDescription)
type NewSessionHandler func(Session) error

type Signaler interface {
	GetName() string

	// Used to open the Signaler, and use this webrtc api to create new sessions
	Run(context.Context, Desktop) error

	SetNewSessionHandler(NewSessionHandler) error
}
