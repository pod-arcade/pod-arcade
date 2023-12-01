package api

import (
	"io"
)

type GamepadRumbleHandler func(GamepadRumble)

type Gamepad interface {
	// GetName returns the name of the gamepad
	GetName() string

	// SetGamepadRumbleHandler sets the handler for gamepad rumble
	SetGamepadRumbleHandler(GamepadRumbleHandler)
	// SetGamepadInputState sets the input state of the gamepad
	SetGamepadInputState(GamepadInput) error

	// OpenGamepad opens the gamepad for use
	OpenGamepad() error

	io.Closer // The gamepad does IO, and should be closable
}
