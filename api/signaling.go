package api

import (
	"context"

	"github.com/pion/webrtc/v4"
)

// OfferHandler is a function that handles an offer SDP
type OfferHandler func(sessionID SessionID, offerSdp webrtc.SessionDescription)

// ICECandidateHandler is a function that handles when an ICE Candidate is received
type ICECandidateHandler func(sessionID SessionID, offerSdp webrtc.SessionDescription)

// NewSessionHandler is a function that handles when a new session is created.
type NewSessionHandler func(Session) error

type Signaler interface {
	// GetName returns the name of the signaler
	GetName() string

	// Used to open the Signaler, and use this webrtc api to create new sessions.
	// This is a blocking call. To stop the signaler, cancel the context.
	Run(context.Context, Desktop) error

	// SetNewSessionHandler sets the new session handler.
	// When the signaler receives a request from a client to create a new session,
	// it will call this handler.
	SetNewSessionHandler(NewSessionHandler) error
}
