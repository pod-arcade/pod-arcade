package api

import "io"

type Keyboard interface {
	// GetName returns the name of the keyboard
	GetName() string

	// Open opens the keyboard for use
	Open() error

	io.Closer // The keyboard does IO, and should be closable
}
