//go:build linux
// +build linux

package video

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/media"
	"github.com/pod-arcade/pod-arcade/pkg/logger"
	"github.com/pod-arcade/pod-arcade/pkg/metrics"
	"github.com/pod-arcade/pod-arcade/pkg/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

var _ media.RTPMediaSource = (*WaylandScreenCapture)(nil)

const PACKET_SIZE = 1200
const MAX_WF_RECORDER_RESTARTS = 10

var queueDepthMetric = metrics.GlobalMetricCache.GetGauge("video_channel_depth", prometheus.Labels{"media_source": "Wayland Screen Capture", "media_type": "video"})

type WaylandScreenCapture struct {
	// The quality of the stream. This can be anything between 0 and 51
	// 0 is lossless, 51 is barely a single pixel
	// Default is 23 in FFMPEG
	// Sane Range is 17-28
	// 17-18 is visually lossless
	Quality              int
	HardwareAcceleration bool
	Profile              string
	l                    zerolog.Logger
	wg                   sync.WaitGroup
	media.RTPMediaSourceBase
}

func NewScreenCapture(quality int, hwAccel bool, profile string) *WaylandScreenCapture {
	cap := &WaylandScreenCapture{
		Quality:              quality,
		Profile:              profile,
		HardwareAcceleration: hwAccel,
		l: logger.CreateLogger(map[string]string{
			"Component": "ScreenCapture",
			"Quality":   fmt.Sprint(quality),
		}),
	}

	return cap
}

func (c *WaylandScreenCapture) GetName() string {
	return "Wayland Screen Capture"
}

func (c *WaylandScreenCapture) GetType() media.RTPMediaSourceType {
	return media.TYPE_VIDEO
}

// This function synchronously launches WF-Recorder and returns back when it exits
func (c *WaylandScreenCapture) spawnWFRecorder(ctx context.Context, udpConn *net.UDPConn) error {
	udpAddr := fmt.Sprintf("rtp://127.0.0.1:%v?pkt_size=%v", udpConn.LocalAddr().(*net.UDPAddr).Port, PACKET_SIZE)

	var properties map[string]string
	var args []string

	if c.HardwareAcceleration {
		// Hardware Acceleration
		args = []string{
			"-c", "h264_vaapi", // also look into h264_nvenc
			"-D",
			"-r", "60",
			"-m", "rtp",
			"-f", udpAddr,
		}

		properties = map[string]string{
			"preset":         "ultrafast",
			"tune":           "zerolatency",
			"profile":        c.Profile,
			"async_depth":    "1",
			"global_quality": fmt.Sprint(c.Quality),
			"gop_size":       "5",
			"open_gop":       "0",
		}
	} else {
		// No hardware acceleration
		args = []string{
			"-c", "libx264",
			"-D",
			"-r", "60",
			"-m", "rtp",
			"-f", udpAddr,
		}

		properties = map[string]string{
			"preset":         "ultrafast",
			"tune":           "zerolatency",
			"profile":        c.Profile,
			"async_depth":    "1",
			"global_quality": fmt.Sprint(c.Quality),
			"gop_size":       "5",
			"open_gop":       "0",
		}
	}

	for k, v := range properties {
		args = append(args, "-p", fmt.Sprintf("%v=%v", k, v))
	}

	videoCmd := exec.CommandContext(ctx, "wf-recorder", args...)

	// Linux-specific: set Pdeathsig to ensure child termination
	videoCmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	videoCmd.WaitDelay = time.Second * 5

	videoCmd.Stderr = util.NewProcessLogWrapper(c.l, zerolog.InfoLevel)
	videoCmd.Stdout = util.NewProcessLogWrapper(c.l, zerolog.ErrorLevel)

	c.l.Info().Msgf("Launching WF-Recorder with args %v", strings.Join(args, " "))

	if err := videoCmd.Start(); err != nil {
		return err
	}

	return videoCmd.Wait()
}

// This function launches a UDP server that runs asynchronously. To close it, cancel the context
func (c *WaylandScreenCapture) startUDPListener(ctx context.Context) (*net.UDPConn, error) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 0})
	if err != nil {
		return nil, err
	}

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

// This function synchronously runs WF-Recorder in a loop. If it crashes more than MAX_WF_RECORDER_RESTARTS number of times
// We exit the function with an error. If the context is cancelled, it returns without an error
func (c *WaylandScreenCapture) runWFRecorder(ctx context.Context, udpConn *net.UDPConn) error {
	for restartCount := 0; restartCount < MAX_WF_RECORDER_RESTARTS; restartCount++ {
		select {
		case <-ctx.Done():
			c.l.Info().Msg("Shut down WF-Recorder")
			return nil
		default:
			// todo some kind of exponential backoff on an error
			time.Sleep(time.Second * time.Duration(restartCount))
			if err := c.spawnWFRecorder(ctx, udpConn); err != nil {
				c.l.Error().Err(err).Msg("WF-Recorder crashed due to an error...")
			}
			c.l.Info().Msg("Shutting down WF-Recorder")
		}
	}
	return fmt.Errorf("reached maximum number of restarts for wf-recorder — %v", MAX_WF_RECORDER_RESTARTS)
}

// asynchronously runs a handler that reads UDP packets, and converts them into RTP packets, publishing it to a channel
func (c *WaylandScreenCapture) handleUDPPackets(ctx context.Context, udpConn *net.UDPConn, pktChan chan<- *rtp.Packet) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			data := make([]byte, 1600)

			n, _, err := udpConn.ReadFrom(data)

			if err != nil {
				c.l.Error().Err(err).Msg("Failed to read from video stream")
				return
			}

			pkt := rtp.Packet{}
			err = pkt.Unmarshal(data[:n])

			if err != nil {
				c.l.Debug().Err(err).Msg("RTP packet failed to unmarshal")
			} else {
				select {
				case pktChan <- &pkt:
					queueDepthMetric.Set(float64(len(pktChan)))
				default:
					c.l.Warn().Msgf("Dropping RTP Packet of size %v", len(pkt.Payload))
					c.DropRTPPacket(&pkt)
				}
			}
		}
	}()
}

func (c *WaylandScreenCapture) Stream(ctx context.Context, pktChan chan<- *rtp.Packet) error {
	defer c.wg.Wait()
	// Create new context so we can cancel the UDP server in the event of an error
	udpCtx, stopUDP := context.WithCancel(ctx)
	defer stopUDP()

	// Start UDP Listener
	udpConn, err := c.startUDPListener(udpCtx)
	if err != nil {
		return err
	}

	// spawn goroutine that reads UDP packets and pipes them to a channel
	c.handleUDPPackets(udpCtx, udpConn, pktChan)

	// Run WF-Recorder until cancelled
	if err := c.runWFRecorder(ctx, udpConn); err != nil {
		return err
	}

	return nil
}

func (c *WaylandScreenCapture) GetCodecParameters() webrtc.RTPCodecParameters {
	return webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, PayloadType: 102}
}
