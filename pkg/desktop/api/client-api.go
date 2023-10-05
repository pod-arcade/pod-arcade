package api

import "github.com/pion/webrtc/v4"

type ClientAPI interface {
	OnOffer(func(sessionId string, offerSdp webrtc.SessionDescription))
	OnIceCandidate(func(sessionId string, candidate webrtc.ICECandidateInit))
	SendAnswer(sessionId string, answerSdp webrtc.SessionDescription)
	SendICECandidate(sessionId string, candidate webrtc.ICECandidateInit)

	// can contain more APIS around metrics gathering, etc
}
