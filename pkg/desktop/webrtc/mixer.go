package webrtc

import (
	"context"
	"sync"

	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/media"
	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/rs/zerolog"
)

type Mixer struct {
	VideoSource media.RTPMediaSource
	AudioSource media.RTPMediaSource

	videoTrack *webrtc.TrackLocalStaticRTP
	audioTrack *webrtc.TrackLocalStaticRTP

	videoChan chan *rtp.Packet
	audioChan chan *rtp.Packet

	wg  sync.WaitGroup
	ctx context.Context
	l   zerolog.Logger
}

func NewWebRTCMixer(ctx context.Context, audioSource media.RTPMediaSource, videoSource media.RTPMediaSource) (*Mixer, error) {
	audioTrack, err := webrtc.NewTrackLocalStaticRTP(audioSource.GetCodecParameters().RTPCodecCapability, "audio", "pion-audio")
	if err != nil {
		return nil, err
	}

	videoTrack, err := webrtc.NewTrackLocalStaticRTP(videoSource.GetCodecParameters().RTPCodecCapability, "video", "pion-video")
	if err != nil {
		return nil, err
	}

	mix := &Mixer{
		audioTrack: audioTrack,
		videoTrack: videoTrack,

		AudioSource: audioSource,
		VideoSource: videoSource,

		audioChan: make(chan *rtp.Packet, 3),
		videoChan: make(chan *rtp.Packet, 100), // give video a little more leeway. This is still only a max of 120KB

		ctx: ctx,
		l: logger.CreateLogger(map[string]string{
			"Component": "WebRTCMixer",
		}),
	}

	return mix, nil
}

func (m *Mixer) Stream() {
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case pkt := <-m.audioChan:
				m.l.Trace().Msg("Sending Audio Packet")
				m.audioTrack.WriteRTP(pkt)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-m.ctx.Done():
				return
			case pkt := <-m.videoChan:
				m.l.Trace().Msg("Sending Video Packet")
				m.videoTrack.WriteRTP(pkt)
			}
		}
	}()

	m.wg.Add(2)
	go func() {
		defer m.wg.Done()
		err := m.AudioSource.Stream(m.ctx, m.audioChan)
		if err != nil {
			m.l.Error().Err(err).Msg("Failed to start audio stream")
		}
		m.l.Info().Msg("Shut down Audio Source")
	}()

	go func() {
		defer m.wg.Done()
		err := m.VideoSource.Stream(m.ctx, m.videoChan)
		if err != nil {
			m.l.Error().Err(err).Msg("Failed to start video stream")
		}
		m.l.Info().Msg("Shut down Video Source")
	}()

	m.wg.Wait()
}

func (m *Mixer) CreateTracks() []webrtc.TrackLocal {
	return []webrtc.TrackLocal{
		m.audioTrack,
		m.videoTrack,
	}
}

func (m *Mixer) AddSDPExtensions(sdp *webrtc.SessionDescription) *webrtc.SessionDescription {

	return sdp
}
