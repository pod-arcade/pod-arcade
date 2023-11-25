package api

import "io"

type Mouse interface {
	GetName() string

	MoveMouse(dx, dy float64) error
	MoveMouseWheel(dx, dy float64) error

	SetMouseButtonRight(bool) error
	SetMouseButtonLeft(bool) error
	SetMouseButtonMiddle(bool) error

	Open() error
	io.Closer
}
