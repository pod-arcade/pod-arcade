package audio

import (
	"context"

	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/media"
	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	"github.com/JohnCMcDonough/pod-arcade/pkg/metrics"
	"github.com/jfreymuth/pulse"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/sdp/v3"
	"github.com/pion/webrtc/v4"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/zaf/g711"
)

var _ media.RTPMediaSource = (*PulseAudioCapture)(nil)

const AUDIO_BUFFER_SIZE = 200
const AUDIO_SAMPLE_RATE = 8000
const RTP_OUTBOUND_MTU = 8000

var queueDepthMetric = metrics.GlobalMetricCache.GetGauge("audio_channel_depth", prometheus.Labels{"media_source": "Pulse Audio Capture", "media_type": "audio"})

type PulseAudioCapture struct {
	sampleChan       chan []byte
	audioFrame       []byte
	audioFrameOffset int
	packetizer       rtp.Packetizer

	l zerolog.Logger
	media.RTPMediaSourceBase
}

func NewPulseAudioCapture() *PulseAudioCapture {
	packetizer := rtp.NewPacketizer(RTP_OUTBOUND_MTU, 0, 0, &codecs.G711Payloader{},
		rtp.NewRandomSequencer(), AUDIO_SAMPLE_RATE)

	return &PulseAudioCapture{
		audioFrame:       make([]byte, AUDIO_BUFFER_SIZE),
		sampleChan:       make(chan []byte),
		audioFrameOffset: 0,
		packetizer:       packetizer,
		l: logger.CreateLogger(map[string]string{
			"Component": "PulseAudioCapture",
		}),
	}
}

func (c *PulseAudioCapture) GetName() string {
	return "Pulse Audio Capture"
}

func (c *PulseAudioCapture) GetType() media.RTPMediaSourceType {
	return media.TYPE_AUDIO
}

func (b *PulseAudioCapture) AddSDPExtensions(sdp *sdp.SessionDescription) *sdp.SessionDescription {

	return sdp
}

func (c *PulseAudioCapture) EncodeSamples(data []int16) (int, error) {
	// c.l.Trace().Msgf("Got Audio Recorder Frame — %v", len(data))

	for _, d := range data {
		sample := g711.EncodeUlawFrame(d)
		c.audioFrame[c.audioFrameOffset] = sample
		c.audioFrameOffset++

		if c.audioFrameOffset == AUDIO_BUFFER_SIZE {
			c.sampleChan <- c.audioFrame
			c.audioFrame = make([]byte, AUDIO_BUFFER_SIZE)
			c.audioFrameOffset = 0
		}
	}

	return len(data), nil
}

func (c *PulseAudioCapture) Stream(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	client, err := pulse.NewClient()
	if err != nil {
		return errors.Wrap(err, "Failed to create Pulse Client")
	}
	defer client.Close()

	c.l.Info().Msg("Created Pulse Client")

	stream, err := client.NewRecord(pulse.Int16Writer(c.EncodeSamples), pulse.RecordSampleRate(AUDIO_SAMPLE_RATE))
	if err != nil {
		return errors.Wrap(err, "Failed to create Pulse Recorder")
	}
	defer stream.Close()
	c.l.Info().Msg("Created Pulse Recorder")

	stream.Start()
	c.l.Debug().Msg("Started Pulse Recorder")
	for {
		select {
		case <-ctx.Done():
			c.l.Info().Msg("Shutting down Pulse Audio")
			stream.Stop()
			return nil
		case frame := <-c.sampleChan:
			if len(c.sampleChan) > 2 {
				c.l.Warn().Msgf("Audio Backpressure is getting high — %v recorder frames", len(c.sampleChan))
			}

			// The TrackLocalStatic does some weird math to calculate the number of samples
			// that I don't think we need to do since our sample rate matches our clock rate.
			rtpPackets := c.packetizer.Packetize(frame, uint32(len(frame)))
			// c.l.Trace().Msgf("Repacked frame into %v packets", len(rtpPackets))

			for _, pkt := range rtpPackets {
				select {
				case pktChan <- pkt:
					// c.l.Trace().Msgf("Sent frame with size %v", len(pkt.Payload))
					queueDepthMetric.Set(float64(len(pktChan)))
				default:
					c.l.Warn().Msgf("Ignoring frame of size %v", len(pkt.Payload))
					c.DropRTPPacket(pkt)
				}
			}
		}
	}
}

func (c *PulseAudioCapture) GetCodecCapability() webrtc.RTPCodecCapability {
	return webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypePCMU}
}
