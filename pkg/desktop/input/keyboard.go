package input

type Keyboard interface {
	SetKeyboardKey(vk int, state bool) error
}
