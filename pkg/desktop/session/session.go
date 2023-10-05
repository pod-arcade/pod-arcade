package session

import (
	"context"
	"fmt"
	"strings"

	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/api"
	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/input"
	PAWebRTC "github.com/JohnCMcDonough/pod-arcade/pkg/desktop/webrtc"
	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	"github.com/JohnCMcDonough/pod-arcade/pkg/metrics"
	"github.com/pion/webrtc/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

var INPUT_PROTOCOL string = "pod-arcade-input-v1"
var TRUE bool = true
var DATACHANNEL_ID uint16 = 0

type Session struct {
	SessionID        string
	API              api.ClientAPI
	PeerConnection   *webrtc.PeerConnection
	InputDataChannel *webrtc.DataChannel
	InputHub         *input.InputHub
	Mixer            *PAWebRTC.Mixer

	l   zerolog.Logger
	ctx context.Context
}

func NewSession(ctx context.Context, api api.ClientAPI, mixer *PAWebRTC.Mixer, inputHub *input.InputHub, sessionId string) (*Session, error) {
	session := &Session{
		SessionID: sessionId,
		API:       api,
		Mixer:     mixer,
		InputHub:  inputHub,
		l: logger.CreateLogger(map[string]string{
			"Component": "Session",
			"SessionID": sessionId,
		}),
		ctx: ctx,
	}
	session.l.Debug().Msgf("Using ICE Servers — %v", cfg.ICEServers)
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: cfg.ICEServers,
			},
		},
	})
	if err != nil {
		return nil, err
	}
	session.PeerConnection = pc

	dc, err := pc.CreateDataChannel("input", &webrtc.DataChannelInit{Protocol: &INPUT_PROTOCOL, Negotiated: &TRUE, Ordered: &TRUE, ID: &DATACHANNEL_ID})
	if err != nil {
		return nil, err
	}
	session.InputDataChannel = dc
	dc.OnMessage(session.onInputMessage)

	for _, track := range session.Mixer.CreateTracks() {
		session.l.Debug().Msgf("Adding track %v", track)
		sender, err := pc.AddTrack(track)
		if err != nil {
			session.l.Error().Err(err).Msg("Failed to add track to peer connection")
		} else {
			session.disposeRTPSender(sender)
		}
	}

	pc.OnICECandidate(session.onIceCandidate)

	return session, nil
}

func (s *Session) onInputMessage(msg webrtc.DataChannelMessage) {
	s.l.Trace().MsgFunc(func() string {
		str := "Handling input from data channel —"
		for _, d := range msg.Data {
			str += (strings.Trim(fmt.Sprintf(" %08b", d), "[]"))
		}
		return str
	})
	metrics.GlobalMetricCache.GetCounter("input_datachannel_messages", prometheus.Labels{
		"session_id": s.SessionID,
	}).Inc()

	if err := s.InputHub.HandleInput(msg.Data); err != nil {
		s.l.Error().Err(err).Msg("Failed to process input")
	}
}

func (s *Session) onIceCandidate(c *webrtc.ICECandidate) {
	if c != nil {
		s.l.Debug().Msgf("Got ICE Candidate — %v", c.String())
	} else {
		s.l.Debug().Msg("Finished gathering ice candidates")
	}
	if c == nil {
		localDesc := s.PeerConnection.LocalDescription()
		s.l.Debug().Msgf("Sending Answer %v", *localDesc)
		s.API.SendAnswer(s.SessionID, *localDesc)
	}
}

func (s *Session) OnOffer(sdp webrtc.SessionDescription) error {
	s.l.Debug().Msg("Received offer")
	if err := s.PeerConnection.SetRemoteDescription(sdp); err != nil {
		return err
	}
	answer, err := s.PeerConnection.CreateAnswer(nil)
	if err != nil {
		return err
	}
	answer = *s.Mixer.AddSDPExtensions(&answer)

	s.PeerConnection.SetLocalDescription(answer)

	return nil
}

func (s *Session) disposeRTPSender(sender *webrtc.RTPSender) {
	go func() {
		bytes := make([]byte, 1200)
		for {
			_, _, err := sender.Read(bytes)
			if err != nil {
				return
			}
		}
	}()
}
