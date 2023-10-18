package input

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/gamepad"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/udev"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/wayland"
	"github.com/pod-arcade/pod-arcade/pkg/logger"
	"github.com/rs/zerolog"
)

type InputType int

const (
	INPUT_KEYBOARD    InputType = 0x01
	INPUT_MOUSE       InputType = 0x02
	INPUT_TOUCHSCREEN InputType = 0x03
	INPUT_GAMEPAD     InputType = 0x04
)

type InputHub struct {
	udev          *udev.UDev
	gamepadHub    *gamepad.GamepadHub
	keyboard      Keyboard
	mouse         Mouse
	waylandClient *wayland.WaylandInputClient

	done chan interface{}
	ctx  zerolog.Context
	l    zerolog.Logger
}

// func (h *InputHub) WriteUDevRules() error {
// 	if stat, err := os.Stat("/etc/host-udev/rules"); err != nil || !stat.IsDir() {
// 		h.l.Warn().Msg("Unable to write udev rules to host. /etc/host-udev/rules is not mounted. You may not have proper device isolation, and your host may be able to see gamepads and keyboards created inside pod-arcade.")
// 		return nil
// 	}
// 	file, err := os.OpenFile("/etc/host-udev/rules/50-pod-arcade.rules", os.O_CREATE|os.O_RDWR, 0o777)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	return nil
// }

func NewInputHub(ctx context.Context) (*InputHub, error) {
	hub := &InputHub{
		done: make(chan interface{}, 1),
		l: logger.CreateLogger(map[string]string{
			"Component": "InputHub",
		}),
	}

	if udev, err := udev.CreateUDev(); err != nil {
		return nil, err
	} else {
		hub.udev = udev
	}

	hub.gamepadHub = gamepad.NewGamepadHub(ctx, hub.udev)
	hub.l.Info().Msg("Created Virtual Gamepad Hub")

	hub.waylandClient = wayland.NewWaylandInputClient(ctx)
	if err := hub.waylandClient.Open(); err != nil {
		return nil, err
	}
	hub.mouse = hub.waylandClient
	// hub.keyboard = hub.waylandClient

	context.AfterFunc(ctx, hub.close)

	return hub, nil
}

func (h *InputHub) HandleInput(input []byte) error {
	if len(input) == 0 {
		return fmt.Errorf("Payload length too small — 0 bytes")
	}
	// h.l.Debug().MsgFunc(func() string {
	// 	sb := strings.Builder{}
	// 	sb.WriteString("Received Input — ")
	// 	for _, b := range input {
	// 		sb.WriteString(fmt.Sprintf("%08b", b))
	// 	}
	// 	return sb.String()
	// })
	switch InputType(input[0]) {
	case INPUT_KEYBOARD:
		if len(input) != 4 {
			return fmt.Errorf("invalid payload length for keyboard input. Received %v bytes, wanted 4 bytes", len(input))
		}
		if h.keyboard == nil {
			return fmt.Errorf("client send input for a keyboard, but no keyboard is connected")
		}
		keyDown := input[1] != 0
		keyCode := binary.LittleEndian.Uint16(input[2:4])
		h.keyboard.SetKeyboardKey(int(keyCode), keyDown)
	case INPUT_MOUSE:
		if len(input) != 18 {
			return fmt.Errorf("invalid payload length for mouse input. Received %v bytes, wanted 18 bytes", len(input))
		}
		if h.mouse == nil {
			return fmt.Errorf("client send input for a mouse, but no mouse is connected")
		}
		leftDown := input[1]&(1<<0) != 0
		rightDown := input[1]&(1<<1) != 0
		middleDown := input[1]&(1<<2) != 0
		mouseX := math.Float32frombits(binary.LittleEndian.Uint32(input[2:6]))
		mouseY := math.Float32frombits(binary.LittleEndian.Uint32(input[6:10]))
		wheelX := math.Float32frombits(binary.LittleEndian.Uint32(input[10:14]))
		wheelY := math.Float32frombits(binary.LittleEndian.Uint32(input[14:18]))
		h.mouse.SetMouseButtonLeft(leftDown)
		h.mouse.SetMouseButtonRight(rightDown)
		h.mouse.SetMouseButtonMiddle(middleDown)
		h.mouse.MoveMouse(float64(mouseX), float64(mouseY))
		h.mouse.MoveMouseWheel(float64(wheelX), float64(wheelY))
		// h.l.Debug().Msgf("Mouse — L=%v R=%v M=%v MM=(%v,%v) WM(%v,%v)", leftDown, rightDown, middleDown, mouseX, mouseY, wheelX, wheelY)
	case INPUT_TOUCHSCREEN:
		return fmt.Errorf("client send input for a touchscreen, but no touchscreen is connected")
	case INPUT_GAMEPAD:
		if len(input) != 28 {
			return fmt.Errorf("invalid payload length for gamepad input. Received %v bytes, wanted 28 bytes", len(input))
		}
		gamepadId := input[1]
		bitfield := gamepad.GamepadBitfield{}
		if err := bitfield.UnmarshalBinary(input[2:]); err != nil {
			return err
		}
		return h.gamepadHub.SendInput(int(gamepadId), bitfield)
	default:
		return fmt.Errorf("unknown input type %x", input[0])
	}
	return nil
}

func (h *InputHub) close() {
	h.l.Info().Msg("Shutting down Input Hub...")
	go func() {
		h.l.Debug().Msg("Waiting on Gamepad Hub...")
		<-h.gamepadHub.Done()
		if h.waylandClient != nil {
			h.l.Debug().Msg("Waiting on Wayland Client...")
			h.waylandClient.Close()
		}
		h.l.Info().Msg("Input Hub Closed")
		for {
			h.done <- nil
		}
	}()
}

func (h *InputHub) Done() <-chan interface{} {
	return h.done
}
