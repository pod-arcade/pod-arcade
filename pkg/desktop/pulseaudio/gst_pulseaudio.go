package pulseaudio

import (
	"os"
	"syscall"

	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/cmd_capture"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/pod-arcade/pod-arcade/pkg/util"
	"github.com/rs/zerolog"
)

var _ cmd_capture.CommandConfiguratorOgg = (*GSTPulseAudioCapture)(nil)

type GSTPulseAudioCapture struct {
	l zerolog.Logger
}

func NewGSTPulseAudioCapture() *GSTPulseAudioCapture {
	cap := &GSTPulseAudioCapture{
		l: log.NewLogger("GSTPulseAudioCapture", nil),
	}

	return cap
}

func (c *GSTPulseAudioCapture) GetName() string {
	return "GST-PulseAudioCapture"
}

func (c *GSTPulseAudioCapture) GetProgramRunnerOgg(path *os.File) (*util.ProgramRunner, error) {
	runner := &util.ProgramRunner{}
	runner.Program = "gst-launch-1.0"
	runner.Args = []string{
		"pulsesrc",
		"!",
		"audioconvert",
		"!",
		"audioresample",
		"!",
		"audio/x-raw,rate=48000,channels=2,format=S16LE",
		"!",
		"opusenc",
		"frame-size=2",
		"max-payload-size=1200",
		"bitrate=48000",
		"!",
		"oggmux",
		// "max-delay=1",
		"max-page-delay=1",
		// "max-tolerance=20",
		"!",
		"filesink",
		"buffer-mode=unbuffered",
		"location=" + path.Name(),
	}

	// Linux-specific: set Pdeathsig to ensure child termination
	runner.SysProcAttr = syscall.SysProcAttr{
		Pdeathsig: syscall.SIGKILL,
	}

	return runner, nil
}
func (c *GSTPulseAudioCapture) GetVideoCodecParameters() *webrtc.RTPCodecParameters {
	return nil
}
func (c *GSTPulseAudioCapture) GetAudioCodecParameters() *webrtc.RTPCodecParameters {
	return &webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000}, PayloadType: 111}
}
