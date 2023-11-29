package desktop

import (
	"github.com/pion/ice/v3"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/webrtc_interceptors/nack"
)

type WebRTCAPIConfig struct {
	SinglePort  int
	ExternalIPs []string
}

// If the GetWebRTCAPI's second parameter is not set to 0, it will use a single port for all of the
// WebRTC connections.
func GetWebRTCAPI(d api.Desktop, c *WebRTCAPIConfig) (*webrtc.API, error) {
	settingEngine := webrtc.SettingEngine{}

	if c != nil {
		if c.SinglePort != 0 {
			mux, err := ice.NewMultiUDPMuxFromPort(c.SinglePort)
			if err != nil {
				return nil, err
			}
			settingEngine.SetICEUDPMux(mux)
		}
		if c.ExternalIPs != nil {
			settingEngine.SetNAT1To1IPs(c.ExternalIPs, webrtc.ICECandidateTypeSrflx)
		}
	}

	// Setup Media Engine
	mediaEngine := &webrtc.MediaEngine{}

	for _, s := range d.GetAudioSources() {
		if err := mediaEngine.RegisterCodec(s.GetAudioCodecParameters(), webrtc.RTPCodecTypeAudio); err != nil {
			return nil, err
		}
	}

	for _, s := range d.GetVideoSources() {
		if err := mediaEngine.RegisterCodec(s.GetVideoCodecParameters(), webrtc.RTPCodecTypeVideo); err != nil {
			return nil, err
		}
	}

	// Register NACK Interceptor
	registry := &interceptor.Registry{}
	responderFac, err := nack.NewResponderInterceptor(nack.ResponderSize(32768)) //at most, 36MB of data.
	if err != nil {
		return nil, err
	}

	registry.Add(responderFac)

	// don't advertise supporting PLI in the parameter, since we can't actually trigger an IDR frame.
	mediaEngine.RegisterFeedback(webrtc.RTCPFeedback{Type: "nack", Parameter: ""}, webrtc.RTPCodecTypeVideo)

	// Create WebRTC API
	api := webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine), webrtc.WithInterceptorRegistry(registry), webrtc.WithSettingEngine(settingEngine))

	return api, nil
}
