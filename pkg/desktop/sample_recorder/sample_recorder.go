package sample_recorder

import (
	"os"

	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/cmd_capture"
	"github.com/pod-arcade/pod-arcade/pkg/util"
)

type SampleRecorder struct {
}

var _ cmd_capture.CommandConfiguratorH264 = (*SampleRecorder)(nil)

func (r *SampleRecorder) GetProgramRunnerH264(path *os.File) (*util.ProgramRunner, error) {
	pr := &util.ProgramRunner{}

	pr.Program = "gst-launch-1.0"
	pr.Args = []string{
		"videotestsrc",
		"!",
		"video/x-raw,width=1280,height=720,framerate=30/1",
		"!",
		"x264enc",
		"!",
		"filesink",
		"location=" + path.Name(),
	}

	return pr, nil
}

func (r *SampleRecorder) GetAudioCodecParameters() *webrtc.RTPCodecParameters {
	return nil
}

func (r *SampleRecorder) GetName() string {
	return "sample-recorder"
}

func (r *SampleRecorder) GetVideoCodecParameters() *webrtc.RTPCodecParameters {
	// Clock Rate should probably be set to something
	return &webrtc.RTPCodecParameters{RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 0}, PayloadType: 102}
}
