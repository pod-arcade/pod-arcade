// Generated by go-wayland-scanner
// https://github.com/rajveermalviya/go-wayland/cmd/go-wayland-scanner
// XML file : ./wlr-virtual-keyboard-unstable-v1.xml
//
// virtual_keyboard_unstable_v1 Protocol Copyright:
//
// Copyright © 2008-2011  Kristian Høgsberg
// Copyright © 2010-2013  Intel Corporation
// Copyright © 2012-2013  Collabora, Ltd.
// Copyright © 2018       Purism SPC
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice (including the next
// paragraph) shall be included in all copies or substantial portions of the
// Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package wlr_virtual_keyboard

import (
	"github.com/rajveermalviya/go-wayland/wayland/client"
	"golang.org/x/sys/unix"
)

// ZwpVirtualKeyboardV1 : virtual keyboard
//
// The virtual keyboard provides an application with requests which emulate
// the behaviour of a physical keyboard.
//
// This interface can be used by clients on its own to provide raw input
// events, or it can accompany the input method protocol.
type ZwpVirtualKeyboardV1 struct {
	client.BaseProxy
}

// NewZwpVirtualKeyboardV1 : virtual keyboard
//
// The virtual keyboard provides an application with requests which emulate
// the behaviour of a physical keyboard.
//
// This interface can be used by clients on its own to provide raw input
// events, or it can accompany the input method protocol.
func NewZwpVirtualKeyboardV1(ctx *client.Context) *ZwpVirtualKeyboardV1 {
	zwpVirtualKeyboardV1 := &ZwpVirtualKeyboardV1{}
	ctx.Register(zwpVirtualKeyboardV1)
	return zwpVirtualKeyboardV1
}

// Keymap : keyboard mapping
//
// Provide a file descriptor to the compositor which can be
// memory-mapped to provide a keyboard mapping description.
//
// Format carries a value from the keymap_format enumeration.
//
//	format: keymap format
//	fd: keymap file descriptor
//	size: keymap size, in bytes
func (i *ZwpVirtualKeyboardV1) Keymap(format uint32, fd int, size uint32) error {
	const opcode = 0
	const _reqBufLen = 8 + 4 + 4
	var _reqBuf [_reqBufLen]byte
	l := 0
	client.PutUint32(_reqBuf[l:4], i.ID())
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(_reqBufLen<<16|opcode&0x0000ffff))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(format))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(size))
	l += 4
	oob := unix.UnixRights(int(fd))
	err := i.Context().WriteMsg(_reqBuf[:], oob)
	return err
}

// Key : key event
//
// A key was pressed or released.
// The time argument is a timestamp with millisecond granularity, with an
// undefined base. All requests regarding a single object must share the
// same clock.
//
// Keymap must be set before issuing this request.
//
// State carries a value from the key_state enumeration.
//
//	time: timestamp with millisecond granularity
//	key: key that produced the event
//	state: physical state of the key
func (i *ZwpVirtualKeyboardV1) Key(time, key, state uint32) error {
	const opcode = 1
	const _reqBufLen = 8 + 4 + 4 + 4
	var _reqBuf [_reqBufLen]byte
	l := 0
	client.PutUint32(_reqBuf[l:4], i.ID())
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(_reqBufLen<<16|opcode&0x0000ffff))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(time))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(key))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(state))
	l += 4
	err := i.Context().WriteMsg(_reqBuf[:], nil)
	return err
}

// Modifiers : modifier and group state
//
// Notifies the compositor that the modifier and/or group state has
// changed, and it should update state.
//
// The client should use wl_keyboard.modifiers event to synchronize its
// internal state with seat state.
//
// Keymap must be set before issuing this request.
//
//	modsDepressed: depressed modifiers
//	modsLatched: latched modifiers
//	modsLocked: locked modifiers
//	group: keyboard layout
func (i *ZwpVirtualKeyboardV1) Modifiers(modsDepressed, modsLatched, modsLocked, group uint32) error {
	const opcode = 2
	const _reqBufLen = 8 + 4 + 4 + 4 + 4
	var _reqBuf [_reqBufLen]byte
	l := 0
	client.PutUint32(_reqBuf[l:4], i.ID())
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(_reqBufLen<<16|opcode&0x0000ffff))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(modsDepressed))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(modsLatched))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(modsLocked))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(group))
	l += 4
	err := i.Context().WriteMsg(_reqBuf[:], nil)
	return err
}

// Destroy : destroy the virtual keyboard keyboard object
func (i *ZwpVirtualKeyboardV1) Destroy() error {
	defer i.Context().Unregister(i)
	const opcode = 3
	const _reqBufLen = 8
	var _reqBuf [_reqBufLen]byte
	l := 0
	client.PutUint32(_reqBuf[l:4], i.ID())
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(_reqBufLen<<16|opcode&0x0000ffff))
	l += 4
	err := i.Context().WriteMsg(_reqBuf[:], nil)
	return err
}

type ZwpVirtualKeyboardV1Error uint32

// ZwpVirtualKeyboardV1Error :
const (
	// ZwpVirtualKeyboardV1ErrorNoKeymap : No keymap was set
	ZwpVirtualKeyboardV1ErrorNoKeymap ZwpVirtualKeyboardV1Error = 0
)

func (e ZwpVirtualKeyboardV1Error) Name() string {
	switch e {
	case ZwpVirtualKeyboardV1ErrorNoKeymap:
		return "no_keymap"
	default:
		return ""
	}
}

func (e ZwpVirtualKeyboardV1Error) Value() string {
	switch e {
	case ZwpVirtualKeyboardV1ErrorNoKeymap:
		return "0"
	default:
		return ""
	}
}

func (e ZwpVirtualKeyboardV1Error) String() string {
	return e.Name() + "=" + e.Value()
}

// ZwpVirtualKeyboardManagerV1 : virtual keyboard manager
//
// A virtual keyboard manager allows an application to provide keyboard
// input events as if they came from a physical keyboard.
type ZwpVirtualKeyboardManagerV1 struct {
	client.BaseProxy
}

// NewZwpVirtualKeyboardManagerV1 : virtual keyboard manager
//
// A virtual keyboard manager allows an application to provide keyboard
// input events as if they came from a physical keyboard.
func NewZwpVirtualKeyboardManagerV1(ctx *client.Context) *ZwpVirtualKeyboardManagerV1 {
	zwpVirtualKeyboardManagerV1 := &ZwpVirtualKeyboardManagerV1{}
	ctx.Register(zwpVirtualKeyboardManagerV1)
	return zwpVirtualKeyboardManagerV1
}

// CreateVirtualKeyboard : Create a new virtual keyboard
//
// Creates a new virtual keyboard associated to a seat.
//
// If the compositor enables a keyboard to perform arbitrary actions, it
// should present an error when an untrusted client requests a new
// keyboard.
func (i *ZwpVirtualKeyboardManagerV1) CreateVirtualKeyboard(seat *client.Seat) (*ZwpVirtualKeyboardV1, error) {
	id := NewZwpVirtualKeyboardV1(i.Context())
	const opcode = 0
	const _reqBufLen = 8 + 4 + 4
	var _reqBuf [_reqBufLen]byte
	l := 0
	client.PutUint32(_reqBuf[l:4], i.ID())
	l += 4
	client.PutUint32(_reqBuf[l:l+4], uint32(_reqBufLen<<16|opcode&0x0000ffff))
	l += 4
	client.PutUint32(_reqBuf[l:l+4], seat.ID())
	l += 4
	client.PutUint32(_reqBuf[l:l+4], id.ID())
	l += 4
	err := i.Context().WriteMsg(_reqBuf[:], nil)
	return id, err
}

func (i *ZwpVirtualKeyboardManagerV1) Destroy() error {
	i.Context().Unregister(i)
	return nil
}

type ZwpVirtualKeyboardManagerV1Error uint32

// ZwpVirtualKeyboardManagerV1Error :
const (
	// ZwpVirtualKeyboardManagerV1ErrorUnauthorized : client not authorized to use the interface
	ZwpVirtualKeyboardManagerV1ErrorUnauthorized ZwpVirtualKeyboardManagerV1Error = 0
)

func (e ZwpVirtualKeyboardManagerV1Error) Name() string {
	switch e {
	case ZwpVirtualKeyboardManagerV1ErrorUnauthorized:
		return "unauthorized"
	default:
		return ""
	}
}

func (e ZwpVirtualKeyboardManagerV1Error) Value() string {
	switch e {
	case ZwpVirtualKeyboardManagerV1ErrorUnauthorized:
		return "0"
	default:
		return ""
	}
}

func (e ZwpVirtualKeyboardManagerV1Error) String() string {
	return e.Name() + "=" + e.Value()
}
