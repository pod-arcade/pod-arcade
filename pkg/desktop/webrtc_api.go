package desktop

import (
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/webrtc_interceptors/nack"
)

func GetWebRTCAPI(d api.Desktop) (*webrtc.API, error) {

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
	return webrtc.NewAPI(webrtc.WithMediaEngine(mediaEngine), webrtc.WithInterceptorRegistry(registry)), nil
}
