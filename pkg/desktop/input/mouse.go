package input

type Mouse interface {
	MoveMouse(dx, dy float64) error
	MoveMouseWheel(dx, dy float64) error

	SetMouseButtonRight(bool) error
	SetMouseButtonLeft(bool) error
	SetMouseButtonMiddle(bool) error
}
