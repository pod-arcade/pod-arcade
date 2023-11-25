package wayland

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/internal/wayland/wlr_virtual_keyboard"
	"github.com/pod-arcade/pod-arcade/internal/wayland/wlr_virtual_pointer"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rajveermalviya/go-wayland/wayland/client"
	"github.com/rs/zerolog"
)

var _ api.Mouse = (*WaylandInputClient)(nil)

type WaylandInputClient struct {
	display         *client.Display
	registry        *client.Registry
	pointerManager  *wlr_virtual_pointer.ZwlrVirtualPointerManagerV1
	pointer         *wlr_virtual_pointer.ZwlrVirtualPointerV1
	keyboardManager *wlr_virtual_keyboard.ZwpVirtualKeyboardManagerV1
	keyboard        *wlr_virtual_keyboard.ZwpVirtualKeyboardV1
	sync.Once

	mtx sync.RWMutex
	ctx context.Context
	l   zerolog.Logger
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

func (c *WaylandInputClient) Open() error {
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
		mouse, err := c.pointerManager.CreateVirtualPointer(nil)
		if err != nil {
			return err
		}
		c.l.Debug().Msgf("...Created virtual pointer")

		c.pointer = mouse
	} else {
		c.l.Debug().Msgf("...display does not support virtual pointer")
		return fmt.Errorf("display does not support virtual pointers")
	}
	// TODO: Need to actually get a seat. Seats may be optional for mice, but not keyboards.
	// c.l.Debug().Msgf("...Creating virtual keyboard")
	// if c.pointerManager != nil {
	// 	keyboard, err := c.keyboardManager.CreateVirtualKeyboard(nil)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	c.l.Debug().Msgf("...Created virtual keyboard")

	// 	c.keyboard = keyboard
	// } else {
	// 	c.l.Debug().Msgf("...display does not support virtual keyboards")
	// 	return fmt.Errorf("display does not support virtual keyboards")
	// }
	c.l.Debug().Msgf("...Waiting for Final Display Sync")
	c.WaitForDisplaySync() // Wait for events to be called

	return nil
}

func (c *WaylandInputClient) MoveMouse(dx, dy float64) error {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
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
	return c.SetMouseButton(0x111, state)
}
func (c *WaylandInputClient) SetMouseButtonLeft(state bool) error {
	// #define BTN_LEFT		0x110
	return c.SetMouseButton(0x110, state)
}
func (c *WaylandInputClient) SetMouseButtonMiddle(state bool) error {
	// #define BTN_MIDDLE		0x112
	return c.SetMouseButton(0x112, state)
}
func (c *WaylandInputClient) SetKeyboardKey(vk int, state bool) error {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if c.keyboard != nil {
		stateInt := uint32(0)
		if state {
			stateInt = 1
		}
		if err := c.keyboard.Key(uint32(time.Now().UnixMilli()), uint32(vk), stateInt); err != nil {
			return err
		}
	}
	return nil
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
