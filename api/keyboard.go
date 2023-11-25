package api

import "io"

type Keyboard interface {
	GetName() string

	Open() error
	io.Closer
}
