package cmd_capture

import (
	"context"
	"net"
	"sync"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/pod-arcade/pod-arcade/pkg/util"
	"github.com/rs/zerolog"
)

type CommandConfiguratorRTP interface {
	GetName() string
	GetProgramRunnerUDP(addr net.UDPAddr) (*util.ProgramRunner, error)
	GetVideoCodecParameters() *webrtc.RTPCodecParameters
	GetAudioCodecParameters() *webrtc.RTPCodecParameters
}

var _ api.VideoSource = (*CommandCaptureRTP)(nil)
var _ api.AudioSource = (*CommandCaptureRTP)(nil)

type CommandCaptureRTP struct {
	configurator CommandConfiguratorRTP

	l  zerolog.Logger
	wg sync.WaitGroup
}

func NewCommandCaptureRTP(c CommandConfiguratorRTP) *CommandCaptureRTP {
	cap := &CommandCaptureRTP{
		configurator: c,
		l:            log.NewLogger(c.GetName(), nil),
	}

	return cap
}

func (c *CommandCaptureRTP) GetName() string {
	return c.configurator.GetName()
}

// This function launches a UDP server that runs asynchronously. To close it, cancel the context
func (c *CommandCaptureRTP) startUDPListener(ctx context.Context) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 0})
	if err != nil {
		return nil, err
	}
	conn.SetReadBuffer(1024 * 1024 * 50)

	c.l = c.l.With().Str("Listener", conn.LocalAddr().String()).Logger()

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		<-ctx.Done()
		c.l.Info().Msg("Shutting down UDP Listener")
		if err := conn.Close(); err != nil {
			c.l.Error().Err(err).Msg("Failed shutting down UDP server")
		}
		c.l.Info().Msg("Shut down UDP Listener")
	}()

	return conn, nil
}

// asynchronously runs a handler that reads UDP packets, and converts them into RTP packets, publishing it to a channel
func (c *CommandCaptureRTP) handleUDPPackets(ctx context.Context, udpConn *net.UDPConn, pktChan chan<- *rtp.Packet) {

	readChan := make(chan []byte, 1000) // Buffered channel to store incoming data before we can process it

	// for i := 0; i < 4; i++ {
	// Goroutine for reading UDP packets
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.l.Info().Msg("Starting UDP Reader")

		for {
			data := make([]byte, 24000)
			n, _, err := udpConn.ReadFrom(data)
			if err != nil {
				c.l.Error().Err(err).Msg("Failed to read from video stream")
				close(readChan) // Close the channel to signal the other goroutine to stop
				return
			}

			if n == len(data) {
				c.l.Warn().Msg("Read full buffer, data was likely truncated")
				continue
			}
			select {
			case readChan <- data[:n]: // Send the data to the channel
			default:
				c.l.Warn().Msgf("Dropping UDP Packet of size %v", n)
			}
		}
	}()
	// }

	c.wg.Add(1)
	// Goroutine for parsing RTP packets
	go func() {
		defer c.wg.Done()
		c.l.Info().Msg("Starting RTP Packet Parser")

		for data := range readChan { // Continuously read from the channel
			pkt := rtp.Packet{}
			err := pkt.Unmarshal(data)

			if err != nil {
				c.l.Warn().Err(err).Msg("RTP packet failed to unmarshal")
			} else {
				select {
				case pktChan <- &pkt:
				default:
					c.l.Warn().Msgf("Dropping RTP Packet of size %v", len(pkt.Payload))
				}
			}
		}
	}()
}

func (c *CommandCaptureRTP) Stream(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	c.l.Info().Msg("Starting Stream")
	// no matter what, wait for everything to shut down.
	defer c.wg.Wait()

	// Create new context so we can cancel the UDP server in the event of an error
	udpCtx, stopUDP := context.WithCancel(ctx)
	defer stopUDP()

	// Start the UDP listener. This will shut down when the context closes
	udpConn, err := c.startUDPListener(udpCtx)
	if err != nil {
		return err
	}

	// spawn goroutine that reads UDP packets and pipes them to a channel
	// This will run until context cancel
	c.handleUDPPackets(udpCtx, udpConn, pktChan)

	program, err := c.configurator.GetProgramRunnerUDP(*udpConn.LocalAddr().(*net.UDPAddr))
	c.l.Info().Msgf("Starting Program — %v", program.String())
	if err != nil {
		return err
	}

	// Run program until cancelled
	if err := program.Run(c.GetName(), ctx); err != nil {
		c.l.Error().Err(err).Msg("Program exited with error")
		return err
	} else {
		c.l.Error().Err(err).Msg("Program exited without error")
	}

	return nil
}

func (c *CommandCaptureRTP) StreamVideo(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	return c.Stream(ctx, pktChan)
}
func (c *CommandCaptureRTP) StreamAudio(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	return c.Stream(ctx, pktChan)
}

func (c *CommandCaptureRTP) GetVideoCodecParameters() webrtc.RTPCodecParameters {
	return *c.configurator.GetVideoCodecParameters()
}

func (c *CommandCaptureRTP) GetAudioCodecParameters() webrtc.RTPCodecParameters {
	return *c.configurator.GetAudioCodecParameters()
}
