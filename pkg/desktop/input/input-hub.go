package input

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/gamepad"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/keyboard"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/mouse"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/udev"
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
	udev       *udev.UDev
	gamepadHub *gamepad.GamepadHub
	keyboard   *keyboard.VirtualKeyboard
	mouse      *mouse.VirtualMouse

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

	kb, err := keyboard.NewVirtualKeyboard(ctx, hub.udev)
	if err != nil {
		hub.l.Warn().Err(err).Msg("Unable to create virtual keyboard")
	} else {
		hub.keyboard = kb
		hub.l.Info().Msg("Created Virtual Keyboard")
	}
	mouse, err := mouse.NewVirtualMouse(ctx, hub.udev)
	if err != nil {
		hub.l.Warn().Err(err).Msg("Unable to create virtual mouse")
	} else {
		hub.mouse = mouse
		hub.l.Info().Msg("Created Virtual Mouse")
	}

	context.AfterFunc(ctx, hub.close)

	return hub, nil
}

func (h *InputHub) HandleInput(input []byte) error {
	if len(input) == 0 {
		return fmt.Errorf("Payload length too small — 0 bytes", input[0])
	}
	switch InputType(input[0]) {
	case INPUT_KEYBOARD:
		if len(input) != 4 {
			return fmt.Errorf("invalid payload length for keyboard input. Received %v bytes, wanted 4 bytes", len(input))
		}
		if h.keyboard == nil {
			return nil
		}
		keyDown := input[1] != 0
		keyCode := binary.LittleEndian.Uint16(input[2:4])
		h.keyboard.KeyEvent(keyDown, int(keyCode))
	case INPUT_MOUSE:
		if h.mouse == nil {
			return nil
		}
		if len(input) != 18 {
			return fmt.Errorf("invalid payload length for mouse input. Received %v bytes, wanted 18 bytes", len(input))
		}
		leftDown := input[1]&(1<<0) != 0
		rightDown := input[1]&(1<<1) != 0
		middleDown := input[1]&(1<<2) != 0
		mouseX := math.Float32frombits(binary.LittleEndian.Uint32(input[2:6]))
		mouseY := math.Float32frombits(binary.LittleEndian.Uint32(input[6:10]))
		wheelX := math.Float32frombits(binary.LittleEndian.Uint32(input[10:14]))
		wheelY := math.Float32frombits(binary.LittleEndian.Uint32(input[14:18]))
		h.mouse.LeftClick(leftDown)
		h.mouse.RightClick(rightDown)
		h.mouse.MiddleClick(middleDown)
		h.mouse.MoveCursor(mouseX, mouseY)
		h.mouse.MoveWheel(wheelX, wheelY)

	case INPUT_TOUCHSCREEN:
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
		if h.keyboard != nil {
			h.l.Debug().Msg("Waiting on Keyboard...")
			<-h.keyboard.Done()
		}
		if h.mouse != nil {
			h.l.Debug().Msg("Waiting on Mouse...")
			<-h.mouse.Done()
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
