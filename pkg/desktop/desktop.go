package desktop

import (
	"context"
	"time"

	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/api"
	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/input"
	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/session"
	PAWebRTC "github.com/JohnCMcDonough/pod-arcade/pkg/desktop/webrtc"
	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	"github.com/JohnCMcDonough/pod-arcade/pkg/metrics"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog"
)

type Desktop struct {
	ClientAPI api.ClientAPI
	sessions  map[string]*session.Session
	mixer     *PAWebRTC.Mixer
	inputHub  *input.InputHub

	l   zerolog.Logger
	ctx context.Context
}

func NewDesktop(ctx context.Context, api api.ClientAPI, mixer *PAWebRTC.Mixer, inputHub *input.InputHub) *Desktop {
	d := Desktop{
		ClientAPI: api,
		ctx:       ctx,
		mixer:     mixer,
		inputHub:  inputHub,
		sessions:  map[string]*session.Session{},
		l: logger.CreateLogger(map[string]string{
			"Component": "Desktop",
		}),
	}
	d.ClientAPI.OnOffer(d.onOffer)
	d.ClientAPI.OnIceCandidate(d.onIceCandidate)
	d.captureMetrics()
	return &d
}

func (d *Desktop) captureMetrics() {
	go func() {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-d.ctx.Done():
					return
				case <-ticker.C:
					for sessionId, session := range d.sessions {
						if session.PeerConnection.ConnectionState() != webrtc.PeerConnectionStateConnected {
							continue // everything breaks if we try to get metrics for a disconnected client.
						}
						stats := session.PeerConnection.GetStats()
						metrics.ProcessWebRTCStats(d.ctx, sessionId, stats)
					}
				}
			}
		}()
	}()
}

func (d *Desktop) onOffer(sessionId string, sdp webrtc.SessionDescription) {
	sl := d.l.With().Str("SessionID", sessionId).Logger()
	// lookup session by ID, create if not exists
	if d.sessions[sessionId] == nil {
		s, err := session.NewSession(d.ctx, d.ClientAPI, d.mixer, d.inputHub, sessionId)
		if err != nil {
			sl.Error().Err(err).Msg("Failed to create a new session")
			return
		}
		sl.Info().Err(err).Msg("Created new session")
		d.sessions[sessionId] = s
	}
	// pass offer to the session
	s := d.sessions[sessionId]
	if err := s.OnOffer(sdp); err != nil {
		sl.Error().Err(err).Str("SessionID", sessionId).Msg("Failed to process offer")
	} else {
		sl.Debug().Err(err).Msg("Processed offer")
	}
}

func (d *Desktop) onIceCandidate(sessionId string, candidate webrtc.ICECandidateInit) {
	sl := d.l.With().Str("SessionID", sessionId).Logger()
	// lookup session by ID, create if not exists
	session := d.sessions[sessionId]

	if session == nil {
		sl.Warn().Msg("Received ICE Candidate for session that doesn't exist")
		return
	}
	session.OnRemoteICECandidate(candidate)
}
