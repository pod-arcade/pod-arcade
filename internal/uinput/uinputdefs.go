package uinput

import "syscall"

// types needed from uinput.h
const (
	uinputMaxNameSize = 80
	uiDevCreate       = 0x5501
	uiDevDestroy      = 0x5502
	uiDevSetup        = 0x405c5503
	// this is for 64 length buffer to store name
	// for another length generate using : (len << 16) | 0x8000552C
	uiGetSysname = 0x8041552c
	uiSetEvBit   = 0x40045564
	uiSetKeyBit  = 0x40045565

	uiSetRelBit = 0x40045566
	uiSetAbsBit = 0x40045567
	busUsb      = 0x03
)

// input event codes as specified in input-event-codes.h
const (
	evSyn     = 0x00
	evKey     = 0x01
	evRel     = 0x02
	evAbs     = 0x03
	relX      = 0x0
	relY      = 0x1
	relHWheel = 0x6
	relWheel  = 0x8
	relDial   = 0x7

	absX     = 0x00
	absY     = 0x01
	absZ     = 0x02
	absRX    = 0x03
	absRY    = 0x04
	absRZ    = 0x05
	absHat0X = 0x10
	absHat0Y = 0x11

	synReport        = 0
	evMouseBtnLeft   = 0x110
	evMouseBtnRight  = 0x111
	evMouseBtnMiddle = 0x112
	evBtnTouch       = 0x14a
)

const (
	btnStateReleased = 0
	btnStatePressed  = 1
	absSize          = 64
)

type inputID struct {
	Bustype uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

// translated to go from uinput.h
type uinputUserDev struct {
	Name       [uinputMaxNameSize]byte
	ID         inputID
	EffectsMax uint32
	Absmax     [absSize]int32
	Absmin     [absSize]int32
	Absfuzz    [absSize]int32
	Absflat    [absSize]int32
}

// translated to go from input.h
type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}
