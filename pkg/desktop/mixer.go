package desktop

import (
	"context"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rs/zerolog"
)

var _ api.Mixer = (*Mixer)(nil)

type Mixer struct {
	video map[api.VideoSource]*webrtc.TrackLocalStaticRTP
	audio map[api.AudioSource]*webrtc.TrackLocalStaticRTP

	l zerolog.Logger
}

func NewMixer() *Mixer {
	return &Mixer{
		video: map[api.VideoSource]*webrtc.TrackLocalStaticRTP{},
		audio: map[api.AudioSource]*webrtc.TrackLocalStaticRTP{},
		l:     log.NewLogger("Mixer", nil),
	}
}

func (m *Mixer) AddVideoSource(v api.VideoSource) error {
	track, err := webrtc.NewTrackLocalStaticRTP(v.GetVideoCodecParameters().RTPCodecCapability, "video", "pion-video")
	if err != nil {
		return err
	}
	m.video[v] = track
	return nil
}
func (m *Mixer) AddAudioSource(a api.AudioSource) error {
	track, err := webrtc.NewTrackLocalStaticRTP(a.GetAudioCodecParameters().RTPCodecCapability, "audio", "pion-audio")
	if err != nil {
		return err
	}
	m.audio[a] = track
	return nil
}

func (m *Mixer) GetAudioSources() []api.AudioSource {
	srcs := []api.AudioSource{}
	for src, _ := range m.audio {
		srcs = append(srcs, src)
	}
	return srcs
}

func (m *Mixer) GetVideoSources() []api.VideoSource {
	srcs := []api.VideoSource{}
	for src, _ := range m.video {
		srcs = append(srcs, src)
	}
	return srcs
}

func (m *Mixer) GetAudioTracks() []*webrtc.TrackLocalStaticRTP {
	tracks := []*webrtc.TrackLocalStaticRTP{}
	for _, track := range m.audio {
		tracks = append(tracks, track)
	}
	return tracks
}
func (m *Mixer) GetVideoTracks() []*webrtc.TrackLocalStaticRTP {
	tracks := []*webrtc.TrackLocalStaticRTP{}
	for _, track := range m.video {
		tracks = append(tracks, track)
	}
	return tracks
}

func (m *Mixer) stream(ctx context.Context, pkts chan *rtp.Packet, track *webrtc.TrackLocalStaticRTP) {
	for {
		select {
		case pkt, open := <-pkts:
			if !open {
				return
			}
			m.l.Trace().Msgf("Got packet from %s", track.ID())
			if err := track.WriteRTP(pkt); err != nil {
				// This shouldn't be TOO big of a deal. It just means that one of the
				// Clients had an error writing an RTP packet.
				m.l.Trace().Err(err).Msg("Failed to write RTP packet")
				continue
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *Mixer) Stream(ctx context.Context) error {
	wg := sync.WaitGroup{}

	// Start all the video tracks
	for src, track := range m.video {
		pkts := make(chan *rtp.Packet, 5000)

		wg.Add(2)
		go func(pkts chan *rtp.Packet) {
			defer wg.Done()
			m.l.Trace().Msgf("Starting to Capture RTP Video Packets %s", track.ID())
			err := src.StreamVideo(ctx, pkts)
			if err != nil {
				m.l.Error().Err(err).Msg("Failed to stream video")
				close(pkts)
			}
			m.l.Trace().Msgf("Done streaming %s", track.ID())
		}(pkts)
		go func(pkts chan *rtp.Packet, track *webrtc.TrackLocalStaticRTP) {
			defer wg.Done()
			m.l.Trace().Msgf("Starting to stream RTP Video Packets %s", track.ID())
			m.stream(ctx, pkts, track)
			m.l.Trace().Msgf("Done streaming %s", track.ID())
		}(pkts, track)
	}

	// Start all of the audio tracks
	for src, track := range m.audio {
		pkts := make(chan *rtp.Packet, 5000)

		wg.Add(2)
		go func(pkts chan *rtp.Packet) {
			defer wg.Done()
			m.l.Trace().Msgf("Starting to Capture RTP Audio Packets %s", track.ID())
			err := src.StreamAudio(ctx, pkts)
			if err != nil {
				m.l.Error().Err(err).Msg("Failed to stream audio")
				close(pkts)
			}
			m.l.Trace().Msgf("Done streaming %s", track.ID())
		}(pkts)
		go func(pkts chan *rtp.Packet, track *webrtc.TrackLocalStaticRTP) {
			defer wg.Done()
			m.l.Trace().Msgf("Starting to stream RTP Audio Packets %s", track.ID())
			m.stream(ctx, pkts, track)
			m.l.Trace().Msgf("Done streaming %s", track.ID())
		}(pkts, track)
	}

	wg.Wait()
	return nil
}
