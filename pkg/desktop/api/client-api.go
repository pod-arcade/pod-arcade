package api

import "github.com/pion/webrtc/v4"

type ClientAPI interface {
	OnOffer(func(sessionId string, offerSdp webrtc.SessionDescription))
	SendAnswer(sessionId string, answerSdp webrtc.SessionDescription)

	// can contain more APIS around metrics gathering, etc
}
