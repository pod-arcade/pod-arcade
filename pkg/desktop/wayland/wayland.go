package wayland

import (
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/internal/wayland/wlr_virtual_keyboard"
	"github.com/pod-arcade/pod-arcade/internal/wayland/wlr_virtual_pointer"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rajveermalviya/go-wayland/wayland/client"
	"github.com/rs/zerolog"
)

var _ api.Mouse = (*WaylandInputClient)(nil)
var _ api.Keyboard = (*WaylandInputClient)(nil)

type WaylandInputClient struct {
	display         *client.Display
	registry        *client.Registry
	pointerManager  *wlr_virtual_pointer.ZwlrVirtualPointerManagerV1
	pointer         *wlr_virtual_pointer.ZwlrVirtualPointerV1
	keyboardManager *wlr_virtual_keyboard.ZwpVirtualKeyboardManagerV1
	keyboard        *wlr_virtual_keyboard.ZwpVirtualKeyboardV1
	seat            *client.Seat
	sync.Once

	// last known mouseState of the mouse buttons
	mouseState struct {
		lmb bool // left mouse button
		mmb bool // middle mouse button
		rmb bool // right mouse button
	}
	keyboardState XKBModifiers

	mtx  sync.RWMutex
	ctx  context.Context
	once sync.Once
	l    zerolog.Logger
}

func NewWaylandInputClient(ctx context.Context) *WaylandInputClient {
	c := &WaylandInputClient{
		ctx: ctx,
		l:   log.NewLogger("input-wayland", nil),
	}

	context.AfterFunc(ctx, func() { c.Close() })
	return c
}

func (c *WaylandInputClient) GetName() string {
	return "Wayland Input Client"
}

func (c *WaylandInputClient) Open() (returnErr error) {
	c.once.Do(func() {
		c.mtx.Lock()
		defer c.mtx.Unlock()
		c.l.Debug().Msgf("Connecting to Wayland")
		if display, err := client.Connect(""); err != nil {
			panic(err)
		} else {
			c.l.Debug().Msgf("...Connected Successfully")
			c.display = display
		}

		c.l.Debug().Msgf("...Setting error handler")
		c.display.SetErrorHandler(func(dee client.DisplayErrorEvent) {
			c.l.Error().Str("err", dee.Message).Msg("The Display encountered an error")
		})

		c.l.Debug().Msgf("...Getting Registry")

		if reg, err := c.display.GetRegistry(); err != nil {
			panic(err)
		} else {
			c.registry = reg
		}

		c.l.Debug().Msgf("...Setting Global Registry Handler")
		c.registry.SetGlobalHandler(c.GlobalRegistryHandler)
		c.l.Debug().Msgf("...Waiting for Interfaces to Register")
		c.WaitForDisplaySync() // Wait for interfaces to register
		c.l.Debug().Msgf("...Waiting for Events to be Called")
		c.WaitForDisplaySync() // Wait for events to be called

		c.l.Debug().Msgf("...Creating virtual pointer")
		if c.pointerManager != nil {
			mouse, err := c.pointerManager.CreateVirtualPointer(c.seat)
			if err != nil {
				returnErr = err
				return
			}
			c.l.Debug().Msgf("...Created virtual pointer")

			c.pointer = mouse
		} else {
			c.l.Debug().Msgf("...display does not support virtual pointer")
			returnErr = fmt.Errorf("display does not support virtual pointers")
			return
		}

		// TODO: Need to actually get a seat. Seats may be optional for mice, but not keyboards.
		c.l.Debug().Msgf("...Creating virtual keyboard")
		if c.keyboardManager != nil {
			keyboard, err := c.keyboardManager.CreateVirtualKeyboard(c.seat)
			if err != nil {
				returnErr = err
				return
			}
			c.l.Debug().Msgf("...Created virtual keyboard")
			c.keyboard = keyboard
		} else {
			c.l.Debug().Msgf("...display does not support virtual keyboards")
			returnErr = fmt.Errorf("display does not support virtual keyboards")
			return
		}
		c.l.Debug().Msgf("...Waiting for Final Display Sync")
		err := c.CreateKeymap()
		if err != nil {
			returnErr = err
			return
		}
		c.WaitForDisplaySync() // Wait for events to be called
	})
	return returnErr
}

// This blog post explains basically everything
// https://medium.com/@damko/a-simple-humble-but-comprehensive-guide-to-xkb-for-linux-6f1ad5e13450
// https://way-cooler.org/docs/wlroots/enum.wlr_keyboard_modifier.html explains modifier keys
func (c *WaylandInputClient) CreateKeymap() error {
	// Define your keymap data (this is a simplified example)
	keymapData := `
	xkb_keymap {
    xkb_keycodes  { include "evdev+aliases(qwerty)" };
    xkb_types     { include "complete" };
    xkb_compat    { include "complete" };
    xkb_symbols   { include "pc+us+inet(evdev)" };
    xkb_geometry  { include "pc(pc105)" };
};
`

	// Create a temporary file for the keymap
	keymapFile, err := os.CreateTemp(os.TempDir(), "keymap-")
	if err != nil {
		return fmt.Errorf("failed to create temp file for keymap: %v", err)
	}
	defer keymapFile.Close()

	// Write the keymap data to the file
	_, err = keymapFile.WriteString(keymapData)
	if err != nil {
		return fmt.Errorf("failed to write to keymap file: %v", err)
	}

	// Get the file size for the keymap
	fileInfo, err := keymapFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat keymap file: %v", err)
	}
	size := uint32(fileInfo.Size())

	// Get the file descriptor
	fd := int(keymapFile.Fd())

	// Duplicate the file descriptor to keep it open after passing to Wayland
	dupFd, err := syscall.Dup(fd)
	if err != nil {
		return fmt.Errorf("failed to duplicate file descriptor: %v", err)
	}

	// Now pass the file descriptor and size to Wayland
	// 0x01 is the xkb format, which is currently the only format supported
	error := c.keyboard.Keymap(0x01, dupFd, size)
	if error != nil {
		return fmt.Errorf("failed to set keymap: %v", err)
	}

	// The file descriptor dupFd should not be closed here, as Wayland will use it.
	// It will be closed automatically when the Wayland server reads it.
	return nil
}
func (c *WaylandInputClient) SetKeyboardKey(i api.KeyboardInput) error {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	c.l.Debug().Msgf("Handling KeyboardInput — %v", i)
	if c.keyboard != nil {
		// handle modifiers
		mods := APIInputToModifierState(i.KeyboardInputModifiers)
		if mods != c.keyboardState {
			c.l.Debug().Msgf("Setting modifiers to %v", mods)
			if err := c.keyboard.Modifiers(uint32(mods), 0, 0, 0); err != nil {
				return err
			}
			c.keyboardState = mods
		}
		// handle keypress
		stateInt := uint32(0)
		if i.State {
			stateInt = 1
		}
		// The Keycode needs to be offset by 8
		// This is just how XKB maps the evdev keycodes to the XKB keycodes
		if err := c.keyboard.Key(uint32(time.Now().UnixMilli()), uint32(i.KeyCode+8), stateInt); err != nil {
			return err
		}
	}
	return nil
}

// This blog post explains basically everything
// https://medium.com/@damko/a-simple-humble-but-comprehensive-guide-to-xkb-for-linux-6f1ad5e13450
// https://way-cooler.org/docs/wlroots/enum.wlr_keyboard_modifier.html explains modifier keys
func (c *WaylandInputClient) CreateKeymap() error {
	// Define your keymap data (this is a simplified example)
	keymapData := `
	xkb_keymap {
    xkb_keycodes  { include "evdev+aliases(qwerty)" };
    xkb_types     { include "complete" };
    xkb_compat    { include "complete" };
    xkb_symbols   { include "pc+us+inet(evdev)" };
    xkb_geometry  { include "pc(pc105)" };
};
`

	// Create a temporary file for the keymap
	keymapFile, err := os.CreateTemp(os.TempDir(), "keymap-")
	if err != nil {
		return fmt.Errorf("failed to create temp file for keymap: %v", err)
	}
	defer keymapFile.Close()

	// Write the keymap data to the file
	_, err = keymapFile.WriteString(keymapData)
	if err != nil {
		return fmt.Errorf("failed to write to keymap file: %v", err)
	}

	// Get the file size for the keymap
	fileInfo, err := keymapFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat keymap file: %v", err)
	}
	size := uint32(fileInfo.Size())

	// Get the file descriptor
	fd := int(keymapFile.Fd())

	// Duplicate the file descriptor to keep it open after passing to Wayland
	dupFd, err := syscall.Dup(fd)
	if err != nil {
		return fmt.Errorf("failed to duplicate file descriptor: %v", err)
	}

	// Now pass the file descriptor and size to Wayland
	// 0x01 is the xkb format, which is currently the only format supported
	error := c.keyboard.Keymap(0x01, dupFd, size)
	if error != nil {
		return fmt.Errorf("failed to set keymap: %v", err)
	}

	// The file descriptor dupFd should not be closed here, as Wayland will use it.
	// It will be closed automatically when the Wayland server reads it.
	return nil
}

func (c *WaylandInputClient) MoveMouse(dx, dy float64) error {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if dx == 0 && dy == 0 {
		return nil
	}
	if c.pointer != nil {
		c.l.Debug().Msgf("Moving Mouse by (%v,%v)", dx, dy)
		err := c.pointer.Motion(uint32(time.Now().UnixMilli()), dx, dy)
		if err != nil {
			return err
		}
		err = c.pointer.Frame()
		if err != nil {
			return err
		}
	}
	return nil
}
func (c *WaylandInputClient) MoveMouseWheel(dx, dy float64) error {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if c.pointer != nil {
		if dx != 0 {
			if err := c.pointer.AxisDiscrete(uint32(time.Now().UnixMilli()), uint32(client.PointerAxisHorizontalScroll), dy*15, int32(dx)); err != nil {
				return err
			}
		}
		if dy != 0 {
			if err := c.pointer.AxisDiscrete(uint32(time.Now().UnixMilli()), uint32(client.PointerAxisVerticalScroll), dy*15, int32(dy)); err != nil {
				return err
			}
		}
		return c.pointer.Frame()
	}
	return nil
}
func (c *WaylandInputClient) SetMouseButton(btn uint32, state bool) error {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if c.pointer != nil {
		stateInt := uint32(0)
		if state {
			stateInt = 1
		}
		if err := c.pointer.Button(uint32(time.Now().UnixMilli()), btn, stateInt); err != nil {
			return err
		}
		return c.pointer.Frame()
	}
	return nil
}
func (c *WaylandInputClient) SetMouseButtonRight(state bool) error {
	// #define BTN_RIGHT		0x111
	if c.mouseState.rmb == state {
		return nil
	}
	c.mouseState.rmb = state
	return c.SetMouseButton(0x111, state)
}
func (c *WaylandInputClient) SetMouseButtonLeft(state bool) error {
	// #define BTN_LEFT		0x110
	if c.mouseState.lmb == state {
		return nil
	}
	c.mouseState.lmb = state
	return c.SetMouseButton(0x110, state)
}
func (c *WaylandInputClient) SetMouseButtonMiddle(state bool) error {
	// #define BTN_MIDDLE		0x112
	if c.mouseState.mmb == state {
		return nil
	}
	c.mouseState.mmb = state
	return c.SetMouseButton(0x112, state)
}

func (c *WaylandInputClient) GlobalRegistryHandler(evt client.RegistryGlobalEvent) {
	c.l.Debug().Msgf("Got Registry Event")
	c.l.Debug().Msgf("Discovered an interface: %v\n", evt.Interface)
	switch evt.Interface {
	case "zwlr_virtual_pointer_manager_v1":

		c.pointerManager = wlr_virtual_pointer.NewZwlrVirtualPointerManagerV1(c.display.Context())
		err := c.registry.Bind(evt.Name, evt.Interface, evt.Version, c.pointerManager)
		if err != nil {
			c.l.Error().Msgf("Unable to bind zwlr_virtual_pointer_manager_v1: %v", err)
			panic(err)
		} else {
			c.l.Debug().Msgf("Bound zwlr_virtual_pointer_manager_v1")
		}
	case "zwp_virtual_keyboard_manager_v1":
		c.keyboardManager = wlr_virtual_keyboard.NewZwpVirtualKeyboardManagerV1(c.display.Context())
		err := c.registry.Bind(evt.Name, evt.Interface, evt.Version, c.keyboardManager)
		if err != nil {
			c.l.Error().Msgf("Unable to bind zwp_virtual_keyboard_manager_v1: %v", err)
			panic(err)
		} else {
			c.l.Debug().Msgf("Bound zwp_virtual_keyboard_manager_v1")
		}
	case "wl_seat":
		c.seat = client.NewSeat(c.display.Context())
		err := c.registry.Bind(evt.Name, evt.Interface, evt.Version, c.seat)
		if err != nil {
			c.l.Error().Msgf("Unable to bind wl_seat: %v", err)
			panic(err)
		} else {
			c.l.Debug().Msgf("Bound wl_seat")
		}
	}
}

func (c *WaylandInputClient) WaitForDisplaySync() {
	c.l.Debug().Msgf("*** Starting Display Sync")
	defer c.l.Debug().Msgf("*** Completed Display Sync")
	// Start Display Sync
	cb, err := c.display.Sync()
	if err != nil {
		c.l.Debug().Msgf("Unable to sync with display — %v\n", err)
		return
	}
	// Destroy callback after sync completed
	defer func() {
		if err := cb.Destroy(); err != nil {
			c.l.Debug().Msgf("Unable to destroy callback: %v", err)
		}
	}()

	// Make a done channel
	done := make(chan interface{}, 1)
	cb.SetDoneHandler(func(cde client.CallbackDoneEvent) {
		done <- nil
	})

	for {
		select {
		case <-done:
			return
		default:
			if err := c.display.Context().Dispatch(); err != nil {
				c.l.Debug().Msgf("There was an error dispatching display events %v", err)
				return
			}
		}
	}
}

func (c *WaylandInputClient) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	defer func() { c.once = sync.Once{} }()

	// Destroy the pointer if we have one
	if c.pointer != nil {
		if err := c.pointer.Destroy(); err != nil {
			c.l.Debug().Msgf("Unable to destroy mouse")
		} else {
			c.pointer = nil
		}
	}

	if c.pointerManager != nil {
		if err := c.pointerManager.Destroy(); err != nil {
			c.l.Debug().Msgf("Unable to destroy mouseManager")
		} else {
			c.pointerManager = nil
		}
	}

	// Destroy the keyboard if we have one
	if c.keyboard != nil {
		if err := c.keyboard.Destroy(); err != nil {
			c.l.Debug().Msgf("Unable to destroy keyboard")
		} else {
			c.keyboard = nil
		}
	}

	if c.keyboardManager != nil {
		if err := c.keyboardManager.Destroy(); err != nil {
			c.l.Debug().Msgf("Unable to destroy keyboardManager")
		} else {
			c.keyboardManager = nil
		}
	}

	if c.seat != nil {
		if err := c.seat.Release(); err != nil {
			c.l.Debug().Msgf("Unable to release seat")
		} else {
			c.seat = nil
		}
	}

	if c.registry != nil {
		if err := c.registry.Destroy(); err != nil {
			c.l.Debug().Msgf("Unable to destroy registry")
		} else {
			c.registry = nil
		}
	}

	if c.display != nil {
		if err := c.display.Destroy(); err != nil {
			c.l.Debug().Msgf("Unable to destroy display")
		}
		if err := c.display.Context().Close(); err != nil {
			c.l.Debug().Msgf("Unable to close wayland context")
		} else {
			c.display = nil
		}
	}

	c.l.Debug().Msgf("closed successfully")
	return nil
}
