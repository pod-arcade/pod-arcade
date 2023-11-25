package api

import (
	"io"
)

type GamepadRumbleHandler func(GamepadRumble)

type Gamepad interface {
	GetName() string

	SetGamepadRumbleHandler(GamepadRumbleHandler)
	SetGamepadInputState(GamepadInput) error

	OpenGamepad() error
	io.Closer // This does IO, and should be closable
}
