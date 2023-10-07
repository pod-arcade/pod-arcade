package input

import (
	"context"
	"fmt"

	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/gamepad"
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

	done chan interface{}
	ctx  zerolog.Context
	l    zerolog.Logger
}

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

	context.AfterFunc(ctx, hub.close)

	return hub, nil
}

func (h *InputHub) HandleInput(input []byte) error {
	if len(input) == 0 {
		return fmt.Errorf("Payload length too small — 0 bytes", input[0])
	}
	switch InputType(input[0]) {
	case INPUT_KEYBOARD:
	case INPUT_MOUSE:
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
		return fmt.Errorf("Unknown input type %x", input[0])
	}
	return nil
}

func (h *InputHub) close() {
	h.l.Info().Msg("Shutting down Input Hub...")
	go func() {
		h.l.Debug().Msg("Waiting on Gamepad Hub...")
		<-h.gamepadHub.Done()
		h.l.Debug().Msg("Gamepad Hub Closed")
		h.l.Info().Msg("Input Hub Closed")
		for {
			h.done <- nil
		}
	}()
}

func (h *InputHub) Done() <-chan interface{} {
	return h.done
}
