//go:build linux
// +build linux

package wf_recorder

import (
	"fmt"
	"net"
	"os"
	"syscall"

	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/cmd_capture"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/pod-arcade/pod-arcade/pkg/util"
	"github.com/rs/zerolog"
)

var _ cmd_capture.CommandConfiguratorRTP = (*WaylandScreenCapture)(nil)

const PACKET_SIZE = 1200
const MAX_WF_RECORDER_RESTARTS = 10

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
}

func NewScreenCapture(quality int, hwAccel bool, profile string) *WaylandScreenCapture {
	cap := &WaylandScreenCapture{
		Quality:              quality,
		Profile:              profile,
		HardwareAcceleration: hwAccel,
		l: log.NewLogger("WFRecorder", map[string]string{
			"Quality": fmt.Sprint(quality),
		}),
	}

	return cap
}

func (c *WaylandScreenCapture) GetName() string {
	return "Wayland Screen Capture"
}

func (c *WaylandScreenCapture) GetProgramRunnerUDP(addr net.UDPAddr) (*util.ProgramRunner, error) {
	udpAddr := fmt.Sprintf("rtp://127.0.0.1:%v?pkt_size=%vbuffer_size=%v", addr.Port, PACKET_SIZE, 4194304)

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
			"slices":         "2",
		}
	} else {
		if c.Profile == "constrained_baseline" {
			c.Profile = "baseline"
		}
		// No hardware acceleration
		args = []string{
			"-c", "libx264",
			"-D",
			"-r", "60",
			"-m", "rtp",
			"-f", udpAddr,
			"-x", "yuv420p",
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

	runner := &util.ProgramRunner{}
	runner.Program = "wf-recorder"
	runner.Args = args

	// Linux-specific: set Pdeathsig to ensure child termination
	runner.SysProcAttr = syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	return runner, nil
}

func (c *WaylandScreenCapture) GetProgramRunnerH264(file *os.File) (*util.ProgramRunner, error) {
	var properties map[string]string
	var args []string

	if c.HardwareAcceleration {
		// Hardware Acceleration
		args = []string{
			"-c", "h264_vaapi", // also look into h264_nvenc
			"-D",
			"-r", "60",
			"-m", "h264",
			"-f", file.Name(),
		}

		properties = map[string]string{
			"preset":         "ultrafast",
			"tune":           "zerolatency",
			"profile":        c.Profile,
			"async_depth":    "1",
			"global_quality": fmt.Sprint(c.Quality),
			"gop_size":       "30",
			"open_gop":       "0",
			"slice-max-size": "1200",
			"slices":         "1",
			"forced-idr":     "1",
		}
	} else {
		if c.Profile == "constrained_baseline" {
			c.Profile = "baseline"
		}
		// No hardware acceleration
		args = []string{
			"-c", "libx264",
			"-D",
			"-r", "60",
			"-m", "h264",
			"-f", file.Name(),
			"-x", "yuv420p",
		}

		properties = map[string]string{
			"preset":         "ultrafast",
			"tune":           "zerolatency",
			"profile":        c.Profile,
			"async_depth":    "1",
			"global_quality": fmt.Sprint(c.Quality),
			"gop_size":       "30",
			"open_gop":       "0",
			"slice-max-size": "1200",
			"slices":         "1",
			"forced-idr":     "1",
		}
	}

	for k, v := range properties {
		args = append(args, "-p", fmt.Sprintf("%v=%v", k, v))
	}

	runner := &util.ProgramRunner{}
	runner.Program = "wf-recorder"
	runner.Args = args

	// Linux-specific: set Pdeathsig to ensure child termination
	runner.SysProcAttr = syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	return runner, nil
}

func (c *WaylandScreenCapture) GetVideoCodecParameters() *webrtc.RTPCodecParameters {
	return &webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, PayloadType: 102}
}

func (c *WaylandScreenCapture) GetAudioCodecParameters() *webrtc.RTPCodecParameters {
	return nil
}
