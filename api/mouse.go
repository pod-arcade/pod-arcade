package api

import "io"

type Mouse interface {
	// GetName returns the name of the mouse
	GetName() string

	// MoveMouse moves the mouse by the given amount. The amount is in pixels.
	MoveMouse(dx, dy float64) error

	// MoveMouseWheel moves the mouse wheel by the given amount. The amount is in lines scrolled.
	MoveMouseWheel(dx, dy float64) error

	// SetMouseButtonRight sets the state of the right mouse button
	SetMouseButtonRight(bool) error

	// SetMouseButtonLeft sets the state of the left mouse button
	SetMouseButtonLeft(bool) error

	// SetMouseButtonMiddle sets the state of the middle mouse button
	SetMouseButtonMiddle(bool) error

	// Open opens the mouse for use
	Open() error

	io.Closer // The mouse does IO, and should be closable
}
