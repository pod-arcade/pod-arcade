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
	"github.com/pion/webrtc/v4/pkg/media/h264reader"
	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/pod-arcade/pod-arcade/pkg/util"
	"github.com/rs/zerolog"
)

type CommandConfiguratorH264 interface {
	GetName() string
	GetProgramRunnerH264(path *os.File) (*util.ProgramRunner, error)
	GetVideoCodecParameters() *webrtc.RTPCodecParameters
	GetAudioCodecParameters() *webrtc.RTPCodecParameters
}

var _ api.VideoSource = (*CommandCaptureH264)(nil)
var _ api.AudioSource = (*CommandCaptureH264)(nil)

type CommandCaptureH264 struct {
	configurator CommandConfiguratorH264

	l  zerolog.Logger
	wg sync.WaitGroup
}

func NewCommandCaptureH264(c CommandConfiguratorH264) *CommandCaptureH264 {
	cap := &CommandCaptureH264{
		configurator: c,
		l:            log.NewLogger(c.GetName(), nil),
	}

	return cap
}

func (c *CommandCaptureH264) GetName() string {
	return c.configurator.GetName()
}

func (c *CommandCaptureH264) handleFifoCreate(ctx context.Context) (*os.File, error) {
	uuid := uuid.NewString()
	path := path.Join(os.TempDir(), "pipe-"+uuid+"-"+c.GetName()+".h264")
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

// asynchronously runs a handler that reads UDP packets, and converts them into RTP packets, publishing it to a channel
func (c *CommandCaptureH264) handleH264Stream(ctx context.Context, stream io.ReadCloser, pktChan chan<- *rtp.Packet) error {
	reader, err := h264reader.NewReader(stream)
	payloader := &codecs.H264Payloader{}
	pktizer := rtp.NewPacketizer(
		1200,
		0, // handled when writing
		0, // handled when writing
		payloader,
		rtp.NewRandomSequencer(),
		c.GetVideoCodecParameters().ClockRate,
	)

	if err != nil {
		c.l.Error().Err(err).Msg("Failed to create h264 reader")
		return err
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				nal, err := reader.NextNAL()
				if err != nil {
					c.l.Error().Err(err).Msg("Failed to read NAL")
					return
				}
				if nal == nil {
					c.l.Debug().Msg("NAL is nil, no more NALs available for reading")
					return
				}
				pkts := pktizer.Packetize(nal.Data, 1)
				for _, p := range pkts {
					select {
					case pktChan <- p:
					default:
						c.l.Warn().Msgf("Dropping RTP Packet of size %v", len(p.Payload))
					}
				}
			}
		}
	}()

	return nil
}

func (c *CommandCaptureH264) Stream(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	c.l.Info().Msg("Starting Stream")

	// Create new context so we can cancel the UDP server in the event of an error
	fileCtx, stopFile := context.WithCancel(ctx)
	defer stopFile()

	// Start the UDP listener. This will shut down when the context closes
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
	c.handleH264Stream(fileCtx, file, pktChan)

	c.l.Debug().Msg("Getting Program Runner")
	program, err := c.configurator.GetProgramRunnerH264(file)
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

func (c *CommandCaptureH264) StreamVideo(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	return c.Stream(ctx, pktChan)
}
func (c *CommandCaptureH264) StreamAudio(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	return c.Stream(ctx, pktChan)
}

func (c *CommandCaptureH264) GetVideoCodecParameters() webrtc.RTPCodecParameters {
	return *c.configurator.GetVideoCodecParameters()
}

func (c *CommandCaptureH264) GetAudioCodecParameters() webrtc.RTPCodecParameters {
	return *c.configurator.GetAudioCodecParameters()
}
