package desktop

import (
	"context"
	"sync"

	"github.com/pion/webrtc/v4"
	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/pod-arcade/pod-arcade/pkg/util"
	"github.com/rs/zerolog"
)

var _ api.Desktop = (*Desktop)(nil)

type Desktop struct {
	signalers []api.Signaler
	gamepads  []api.Gamepad
	keyboard  api.Keyboard
	mouse     api.Mouse

	mixer         *Mixer
	webrtcAPI     *webrtc.API
	webrtcAPIConf *webrtc.Configuration

	inputChannels map[api.SessionID]*webrtc.DataChannel

	rwm sync.RWMutex
	l   zerolog.Logger
}

func NewDesktop() api.Desktop {
	return &Desktop{
		l:             log.NewLogger("Desktop", nil),
		mixer:         NewMixer(),
		inputChannels: map[api.SessionID]*webrtc.DataChannel{},
	}
}

func (d *Desktop) WithSignaler(s api.Signaler) api.Desktop {
	d.l.Info().Msgf("Adding signaler %s", s.GetName())
	d.signalers = append(d.signalers, s)
	s.SetNewSessionHandler(d.HandleSession)
	return d
}
func (d *Desktop) WithGamepad(g api.Gamepad) api.Desktop {
	d.l.Info().Msgf("Adding gamepad %s", g.GetName())
	d.gamepads = append(d.gamepads, g)
	return d
}
func (d *Desktop) WithKeyboard(k api.Keyboard) api.Desktop {
	d.l.Info().Msgf("Adding keyboard %s", k.GetName())
	d.keyboard = k
	return d
}
func (d *Desktop) WithMouse(m api.Mouse) api.Desktop {
	d.l.Info().Msgf("Adding mouse %s", m.GetName())
	d.mouse = m
	return d
}
func (d *Desktop) WithVideoSource(v api.VideoSource) api.Desktop {
	d.l.Info().Msgf("Adding video source %s", v.GetName())
	d.mixer.AddVideoSource(v)
	return d
}
func (d *Desktop) WithAudioSource(a api.AudioSource) api.Desktop {
	d.l.Info().Msgf("Adding audio source %s", a.GetName())
	d.mixer.AddAudioSource(a)
	return d
}
func (d *Desktop) WithWebRTCAPI(api *webrtc.API, conf *webrtc.Configuration) api.Desktop {
	d.webrtcAPI = api
	d.webrtcAPIConf = conf
	return d
}

func (d *Desktop) GetSignalers() []api.Signaler {
	return d.signalers
}
func (d *Desktop) GetGamepads() []api.Gamepad {
	return d.gamepads
}
func (d *Desktop) GetAudioSources() []api.AudioSource {
	return d.mixer.GetAudioSources()
}
func (d *Desktop) GetVideoSources() []api.VideoSource {
	return d.mixer.GetVideoSources()
}
func (d *Desktop) GetKeyboard() api.Keyboard {
	return d.keyboard
}
func (d *Desktop) GetMouse() api.Mouse {
	return d.mouse
}
func (d *Desktop) GetWebRTCAPI() (api *webrtc.API, conf *webrtc.Configuration) {
	return d.webrtcAPI, d.webrtcAPIConf
}

func (d *Desktop) HandleGamepadRumble(rumble api.GamepadRumble) {
	d.rwm.RLock()
	defer d.rwm.RUnlock()

	d.l.Trace().Msgf("Handling gamepad rumble %v", rumble)
	data := rumble.ToBytes()
	for _, c := range d.inputChannels {
		if c != nil {
			c.Send(data)
		}
	}
}

func (d *Desktop) HandleInputMessage(data []byte) {
	d.l.Trace().Msgf("Handling input message %v", data)

	switch api.InputType(data[0]) {
	case api.InputTypeKeyboard:
		input := api.KeyboardInput{}
		err := input.FromBytes(data)
		if err != nil {
			d.l.Warn().Err(err).Msg("Failed to parse keyboard input")
			return
		}
		d.l.Debug().Msgf("Handling keyboard input %v", input)
		if err := d.keyboard.SetKeyboardKey(input); err != nil {
			d.l.Warn().Err(err).Msg("Failed to set keyboard key")
		}
	case api.InputTypeMouse:
		input := api.MouseInput{}
		err := input.FromBytes(data)
		if err != nil {
			d.l.Warn().Err(err).Msg("Failed to parse mouse input")
			return
		}
		d.l.Debug().Msgf("Handling mouse input %v", input)
		d.mouse.SetMouseButtonLeft(input.ButtonLeft)
		d.mouse.SetMouseButtonRight(input.ButtonRight)
		d.mouse.SetMouseButtonMiddle(input.ButtonMiddle)
		d.mouse.MoveMouse(float64(input.MouseX), float64(input.MouseY))
		d.mouse.MoveMouseWheel(float64(input.WheelX), float64(input.WheelY))
	case api.InputTypeTouchscreen:
	case api.InputTypeGamepad:
		input := api.GamepadInput{}
		err := input.FromBytes(data)
		if err != nil {
			d.l.Warn().Err(err).Msg("Failed to parse gamepad input")
			return
		}
		if int(input.PadID) >= len(d.gamepads) {
			d.l.Warn().Msgf("Received gamepad input for gamepad %v, but we only have %v gamepads", input.PadID, len(d.gamepads))
			return
		}
		if err := d.gamepads[input.PadID].SetGamepadInputState(input); err != nil {
			d.l.Warn().Err(err).Msgf("Failed to set gamepad input state for gamepad %v", input.PadID)
		}
	default:
		d.l.Warn().Msgf("Unknown input type %v", data[0])
	}
}

func (d *Desktop) HandleSession(s api.Session) error {
	d.rwm.Lock()
	defer d.rwm.Unlock()
	pc := s.GetPeerConnection()

	// Register Video with peer connection
	for _, v := range d.mixer.GetVideoTracks() {
		sender, err := pc.AddTrack(v)
		if err != nil {
			return err
		}
		d.runPacketDisposer(sender)
	}

	// Register Audio with peer connection
	for _, v := range d.mixer.GetAudioTracks() {
		sender, err := pc.AddTrack(v)
		if err != nil {
			return err
		}
		d.runPacketDisposer(sender)
	}

	// Create Input Channel
	input, err := pc.CreateDataChannel("input", &webrtc.DataChannelInit{
		ID:         util.TypeToPointer[uint16](0),
		Ordered:    util.TypeToPointer(true),
		Protocol:   util.TypeToPointer("pod-arcade-input-v1"),
		Negotiated: util.TypeToPointer(true),
	})
	if err != nil {
		return err
	}
	d.inputChannels[s.GetID()] = input

	// Handle Input Messages
	input.OnMessage(func(msg webrtc.DataChannelMessage) {
		d.HandleInputMessage(msg.Data)
	})

	// Handle Peer Connection disconnect
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		// If we're disconnected
		if state == webrtc.PeerConnectionStateDisconnected ||
			state == webrtc.PeerConnectionStateFailed ||
			state == webrtc.PeerConnectionStateClosed {
			d.inputChannels[s.GetID()] = nil
		}
	})

	return nil
}

func (d *Desktop) Run(ctx context.Context) error {
	d.l.Debug().Msg("Starting Desktop...")
	if d.webrtcAPI == nil {
		d.webrtcAPI = webrtc.NewAPI()
	}

	// Start Gamepads
	for _, g := range d.gamepads {
		d.l.Debug().Msgf("Opening Gamepad — %v...", g.GetName())
		err := g.OpenGamepad()
		if err != nil {
			return err
		}
		defer g.Close()
	}

	// Start Keyboard
	if d.keyboard != nil {
		d.l.Debug().Msgf("Opening Keyboard — %v...", d.keyboard.GetName())
		err := d.keyboard.Open()
		if err != nil {
			return err
		}
		defer d.keyboard.Close()
	}

	// Start Mouse
	if d.mouse != nil {
		d.l.Debug().Msgf("Opening Mouse — %v...", d.mouse.GetName())
		err := d.mouse.Open()
		if err != nil {
			return err
		}
		defer d.mouse.Close()
	}

	// Register Signalers
	wg := sync.WaitGroup{}
	for _, s := range d.signalers {
		d.l.Debug().Msgf("Registering signaler — %v...", s.GetName())
		// Start the signaler with the context
		s.SetNewSessionHandler(d.HandleSession)
		wg.Add(1)
		go func(s api.Signaler) {
			defer wg.Done()
			s.Run(ctx, d)
		}(s)
	}

	d.l.Debug().Msg("Starting Mixer...")
	err := d.mixer.Stream(ctx)

	// Wait for all of our signalers to shut down
	// no matter whether we had an error streaming or not.
	// we need them to clean up first
	d.l.Debug().Msg("Desktop running!")
	wg.Wait()
	d.l.Debug().Msg("Desktop shutting down...")

	return err
}

func (d *Desktop) runPacketDisposer(s *webrtc.RTPSender) {
	go func() {
		buf := make([]byte, 1500)
		for {
			_, _, err := s.Read(buf)
			if err != nil {
				d.l.Debug().Err(err).Msg("Stopping RTP Sender")
				return
			}
		}
	}()
}
