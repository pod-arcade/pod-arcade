package gamepad

import (
	"context"
	"fmt"

	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/input/udev"
	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	"github.com/rs/zerolog"
)

const MAX_GAMEPADS = 4

type GamepadHub struct {
	gamepads []*VirtualGamepad
	udev     *udev.UDev

	done chan interface{}
	ctx  context.Context
	l    zerolog.Logger
}

func NewGamepadHub(ctx context.Context, udev *udev.UDev) *GamepadHub {
	hub := &GamepadHub{
		done: make(chan interface{}, 1),
		udev: udev,
		ctx:  ctx,
		l: logger.CreateLogger(map[string]string{
			"Component": "GamepadHub",
		}),
	}

	context.AfterFunc(ctx, hub.close)

	hub.initializeGamepads()

	return hub
}

func (h *GamepadHub) SendInput(padNum int, input GamepadBitfield) error {
	if padNum < 0 || padNum > len(h.gamepads) {
		return fmt.Errorf("gamepad %v is outside the range 0 - %v", padNum, len(h.gamepads))
	}
	h.gamepads[padNum].SendInput(input)
	return nil
}

func (h *GamepadHub) close() {
	h.l.Debug().Msg("Closing down gamepad hub...")
	for i, pad := range h.gamepads {
		h.l.Debug().Msg("Closing pad " + string(i))
		pad.Close()
	}

	// write that we're closed forever
	go func() {
		for {
			h.done <- nil
		}
	}()
	h.l.Debug().Msg("Gamepad Hub Closed")
}

func (h *GamepadHub) Done() <-chan interface{} {
	return h.done
}

func (h *GamepadHub) initializeGamepads() {
	for i := 0; i < MAX_GAMEPADS; i++ {
		if gamepad, err := CreateVirtualGamepad(h.udev, i, 0x045E, 0x02D1); err != nil {
			h.l.Error().Err(err).Msg("Failed to create virtual gamepad")
		} else {
			h.gamepads = append(h.gamepads, gamepad)
		}
	}
}
