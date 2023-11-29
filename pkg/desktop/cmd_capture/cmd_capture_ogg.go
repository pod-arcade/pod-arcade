package cmd_capture

import (
	"context"
	"io"
	"os"
	"path"
	"sync"
	"syscall"

	"github.com/google/uuid"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media/oggreader"
	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/pod-arcade/pod-arcade/pkg/util"
	"github.com/rs/zerolog"
)

type CommandConfiguratorOgg interface {
	GetName() string
	GetProgramRunnerOgg(path *os.File) (*util.ProgramRunner, error)
	GetVideoCodecParameters() *webrtc.RTPCodecParameters
	GetAudioCodecParameters() *webrtc.RTPCodecParameters
}

var _ api.VideoSource = (*CommandCaptureOgg)(nil)
var _ api.AudioSource = (*CommandCaptureOgg)(nil)

type CommandCaptureOgg struct {
	configurator CommandConfiguratorOgg

	l  zerolog.Logger
	wg sync.WaitGroup
}

func NewCommandCaptureOgg(c CommandConfiguratorOgg) *CommandCaptureOgg {
	cap := &CommandCaptureOgg{
		configurator: c,
		l:            log.NewLogger(c.GetName(), nil),
	}

	return cap
}

func (c *CommandCaptureOgg) GetName() string {
	return c.configurator.GetName()
}

func (c *CommandCaptureOgg) handleFifoCreate(ctx context.Context) (*os.File, error) {
	uuid := uuid.NewString()
	path := path.Join(os.TempDir(), "pipe-"+uuid+"-"+c.GetName()+".ogg")
	c.l.Debug().Msgf("Creating FIFO at %v", path)
	err := syscall.Mkfifo(path, 0o777)
	if err != nil {
		c.l.Err(err).Msgf("Failed to create FIFO at %v", path)
		return nil, err
	}
	c.l.Debug().Msgf("Opening FIFO at %v", path)
	file, err := os.OpenFile(path, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		c.l.Err(err).Msgf("Failed to open FIFO at %v", path)
		return nil, err
	}

	return file, nil
}

// asynchronously runs a handler that reads Ogg frames from the stream, and converts them into RTP packets, publishing it to a channel
func (c *CommandCaptureOgg) handleOggStream(ctx context.Context, stream io.ReadCloser, pktChan chan<- *rtp.Packet) error {
	payloader := &codecs.OpusPayloader{}
	pktizer := rtp.NewPacketizer(
		1200,
		0, // handled when writing
		0, // handled when writing
		payloader,
		rtp.NewRandomSequencer(),
		c.GetAudioCodecParameters().ClockRate,
	)

	go func() {
		reader, _, err := oggreader.NewWith(stream)
		if err != nil {
			c.l.Error().Err(err).Msg("Failed to create ogg reader")
			return
		}
		var lastGranule uint64
		for {
			pageData, pageHeader, err := reader.ParseNextPage()
			if err != nil {
				c.l.Error().Err(err).Msg("Failed to parse next page")
				return
			}
			sampleCount := float64(pageHeader.GranulePosition - lastGranule)
			lastGranule = pageHeader.GranulePosition
			pkts := pktizer.Packetize(pageData, uint32(sampleCount))
			for _, p := range pkts {
				select {
				case pktChan <- p:
					// c.l.Debug().
					// 	Float64("samples", sampleCount).
					// 	Int("packets", len(pkts)).
					// 	Uint64("granule", lastGranule).
					// 	Msgf("Sent RTP Packets — %v", pageHeader)
				default:
					c.l.Warn().Msgf("Dropping RTP Packet of size %v", len(p.Payload))
				}
			}
		}
	}()
	return nil
}

func (c *CommandCaptureOgg) Stream(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	c.l.Info().Msg("Starting Stream")

	// Create new context so we can cancel the stream in the event of an error
	fileCtx, stopFile := context.WithCancel(ctx)
	defer stopFile()

	// Open the FIFO stream. This will shut down when the context closes
	c.l.Debug().Msg("Creating FIFO")
	file, err := c.handleFifoCreate(fileCtx)
	if err != nil {
		return err
	}
	defer file.Close()
	defer os.Remove(file.Name())

	c.l.Debug().Msg("Starting Reader")
	// spawn goroutine that reads NAL frames and pipes them to a channel
	// This will run until context cancel
	c.handleOggStream(fileCtx, file, pktChan)

	c.l.Debug().Msg("Getting Program Runner")
	program, err := c.configurator.GetProgramRunnerOgg(file)
	c.l.Info().Msgf("Starting Program — %v", program.String())
	if err != nil {
		return err
	}

	// Run program until cancelled
	c.l.Debug().Msg("Running")
	if err := program.Run(c.GetName(), ctx); err != nil {
		c.l.Error().Err(err).Msg("Program exited with error")
		return err
	} else {
		c.l.Error().Err(err).Msg("Program exited without error")
	}

	return nil
}

func (c *CommandCaptureOgg) StreamVideo(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	return c.Stream(ctx, pktChan)
}
func (c *CommandCaptureOgg) StreamAudio(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	return c.Stream(ctx, pktChan)
}

func (c *CommandCaptureOgg) GetVideoCodecParameters() webrtc.RTPCodecParameters {
	return *c.configurator.GetVideoCodecParameters()
}

func (c *CommandCaptureOgg) GetAudioCodecParameters() webrtc.RTPCodecParameters {
	return *c.configurator.GetAudioCodecParameters()
}
