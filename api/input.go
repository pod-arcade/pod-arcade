package api

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/pod-arcade/pod-arcade/pkg/util"
)

type InputType int

const (
	InputTypeKeyboard      InputType = 1
	InputTypeMouse         InputType = 2
	InputTypeTouchscreen   InputType = 3
	InputTypeGamepad       InputType = 4
	InputTypeGamepadRumble InputType = 5
)

// GamepadInput describes the state of a gamepad's inputs.
type GamepadInput struct {
	PadID byte
	// North represents the top face button.
	// Xbox: Y, PlayStation: △, Switch: X
	North bool

	// South represents the bottom face button.
	// Xbox: A, PlayStation: x, Switch: B
	South bool

	// West represents the left face button.
	// Xbox: X, PlayStation: ◻, Switch: Y
	West bool

	// East represents the right face button.
	// Xbox: B, PlayStation: ○, Switch: A
	East bool

	// L1 represents the left bumper button.
	L1 bool

	// R1 represents the right bumper button.
	R1 bool

	// L2 represents the left trigger as a button.
	L2 bool

	// R2 represents the right trigger as a button.
	R2 bool

	// LZ represents clicking the left thumbstick.
	LZ bool

	// RZ represents clicking the right thumbstick.
	RZ bool

	// Select represents the 'Select' button.
	// Xbox: Back, PlayStation: Share, Switch: -
	Select bool

	// Start represents the 'Start' button.
	// Xbox: Start, PlayStation: Options, Switch: +
	Start bool

	// DPadUp represents pressing up on the D-Pad.
	DPadUp bool

	// DPadDown represents pressing down on the D-Pad.
	DPadDown bool

	// DPadLeft represents pressing left on the D-Pad.
	DPadLeft bool

	// DPadRight represents pressing right on the D-Pad.
	DPadRight bool

	// Home represents the 'Home' button.
	// Xbox: Xbox button, PlayStation: PS button, Switch: Home button
	Home bool

	// Capture represents the screenshot button on the Switch.
	Capture bool

	// AxisLeftX represents the horizontal axis of the left thumbstick.
	// Range: -1 to 1
	AxisLeftX float32

	// AxisLeftY represents the vertical axis of the left thumbstick.
	// Range: -1 to 1
	AxisLeftY float32

	// AxisRightX represents the horizontal axis of the right thumbstick.
	// Range: -1 to 1
	AxisRightX float32

	// AxisRightY represents the vertical axis of the right thumbstick.
	// Range: -1 to 1
	AxisRightY float32

	// AxisLeftTrigger represents the intensity of the left trigger.
	// Range: 0 to 1
	AxisLeftTrigger float32

	// AxisRightTrigger represents the intensity of the right trigger.
	// Range: 0 to 1
	AxisRightTrigger float32
}

func (i *GamepadInput) ToBytes() []byte {
	output := make([]byte, 28)
	output[0] = byte(InputTypeGamepad)
	output[1] = byte(i.PadID)
	data := output[2:]

	data[0] = util.PackBits(
		i.North,
		i.South,
		i.West,
		i.East,
		i.L1,
		i.R1,
		i.LZ,
		i.RZ,
	)

	data[1] = util.PackBits(
		i.Select,
		i.Start,
		i.DPadUp,
		i.DPadDown,
		i.DPadLeft,
		i.DPadRight,
		i.Home,
		false,
	)

	binary.LittleEndian.PutUint32(data[2:6], math.Float32bits(i.AxisLeftX))
	binary.LittleEndian.PutUint32(data[6:10], math.Float32bits(i.AxisLeftY))
	binary.LittleEndian.PutUint32(data[10:14], math.Float32bits(i.AxisRightX))
	binary.LittleEndian.PutUint32(data[14:18], math.Float32bits(i.AxisRightY))
	binary.LittleEndian.PutUint32(data[18:22], math.Float32bits(i.AxisLeftTrigger))
	binary.LittleEndian.PutUint32(data[22:26], math.Float32bits(i.AxisRightTrigger))

	return output
}

func (i *GamepadInput) FromBytes(input []byte) error {
	if input[0] != byte(InputTypeGamepad) || len(input) < 2 {
		return errors.New("data is not a gamepad input")
	}

	i.PadID = input[1]
	data := input[2:]

	if len(data) != 26 {
		return fmt.Errorf("invalid payload size %d should be 26 bytes", len(data))
	}

	i.North,
		i.South,
		i.West,
		i.East,
		i.L1,
		i.R1,
		i.LZ,
		i.RZ = util.UnpackBits(data[0])

	i.Select,
		i.Start,
		i.DPadUp,
		i.DPadDown,
		i.DPadLeft,
		i.DPadRight,
		i.Home,
		_ = util.UnpackBits(data[1])

	// extract left thumbstick in little-endian format
	i.AxisLeftX = math.Float32frombits(binary.LittleEndian.Uint32(data[2:6]))
	i.AxisLeftY = math.Float32frombits(binary.LittleEndian.Uint32(data[6:10]))

	// extract right thumbstick in little-endian format
	i.AxisRightX = math.Float32frombits(binary.LittleEndian.Uint32(data[10:14]))
	i.AxisRightY = math.Float32frombits(binary.LittleEndian.Uint32(data[14:18]))

	// extract left and right trigger in little-endian format
	i.AxisLeftTrigger = math.Float32frombits(binary.LittleEndian.Uint32(data[18:22]))
	i.AxisRightTrigger = math.Float32frombits(binary.LittleEndian.Uint32(data[22:26]))

	return nil
}

// GamepadRumble describes the rumble settings for a gamepad.
type GamepadRumble struct {
	PadID byte
	// LeftRumble represents the intensity of the left rumble motor.
	// Range: 0.0 to 1.0
	LeftRumble float32

	// RightRumble represents the intensity of the right rumble motor.
	// Range: 0.0 to 1.0
	RightRumble float32
}

func (i *GamepadRumble) ToBytes() []byte {
	output := make([]byte, 9)
	output[0] = byte(InputTypeGamepadRumble)
	output[1] = byte(i.PadID)
	data := output[2:]

	binary.LittleEndian.PutUint32(data[0:4], math.Float32bits(i.LeftRumble))
	binary.LittleEndian.PutUint32(data[4:8], math.Float32bits(i.RightRumble))

	return output
}

func (i *GamepadRumble) FromBytes(input []byte) error {
	if input[0] != byte(InputTypeGamepadRumble) || len(input) < 2 {
		return errors.New("data is not a gamepad rumble input")
	}

	i.PadID = input[1]
	data := input[2:]

	if len(data) != 26 {
		return fmt.Errorf("invalid payload size %d should be 9 bytes", len(data))
	}

	// extract left and right trigger in little-endian format
	i.LeftRumble = math.Float32frombits(binary.LittleEndian.Uint32(data[0:4]))
	i.RightRumble = math.Float32frombits(binary.LittleEndian.Uint32(data[4:8]))

	return nil
}