package uinput

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/pod-arcade/pod-arcade/api"
	"github.com/pod-arcade/pod-arcade/internal/udev"
	"github.com/pod-arcade/pod-arcade/internal/uinput"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rs/zerolog"
	eventemitter "github.com/vansante/go-event-emitter"
)

var _ api.Mouse = (*VirtualMouse)(nil)

type VirtualMouse struct {
	udev        *udev.UDev
	m           uinput.Mouse
	eventdevice *udev.Device
	mousedevice *udev.Device

	syspath string

	done chan interface{}
	l    zerolog.Logger
	mtx  sync.Mutex
}

func NewVirtualMouse(ctx context.Context, uDev *udev.UDev) (*VirtualMouse, error) {
	m := &VirtualMouse{
		udev: uDev,
		l:    log.NewLogger("input-uinput-mouse", nil),
	}

	return m, nil
}

func (m *VirtualMouse) GetName() string {
	return "uinput-mouse"
}

func (m *VirtualMouse) Open() error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	m.udev.KernelEvents.AddListener(eventemitter.EventType(udev.ADD), eventemitter.HandleFunc(func(arguments ...interface{}) {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		evt := (arguments[0]).(*udev.UEvent)

		if strings.HasPrefix("/sys"+evt.KObj, m.syspath) {
			m.l.Debug().Msg("Handling event for my device")
			m.handleEvent(evt)
		} else {
			m.l.Debug().Msgf("skipping event for not my device %v does not have prefix %v", "/sys"+evt.KObj, m.syspath)
		}
	}))

	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("[PA] Mouse"))
	if err != nil {
		return err
	}
	m.m = mouse

	syspath, err := mouse.FetchSyspath()
	if err != nil {
		m.l.Error().Err(err).Msg("Failed to get syspath")
		return err
	}

	m.l = m.l.With().Str("syspath", syspath).Logger()
	m.l.Debug().Msg("Fetched syspath")

	return nil
}

func (m *VirtualMouse) handleEvent(evt *udev.UEvent) {
	comps := strings.Split(evt.KObj, "/")
	last := comps[len(comps)-1]
	isMouseDev := regexp.MustCompile("mouse[0-9]+").MatchString(last)
	isEventDev := regexp.MustCompile("event[0-9]+").MatchString(last)
	if !isMouseDev && !isEventDev {
		m.l.Debug().Msgf("Skipping device that doesn't match %v", last)
		return
	}

	major, err := strconv.ParseInt(evt.Env["MAJOR"], 10, 16)
	if err != nil {
		m.l.Error().Err(err).Msg("Error getting device major number")
	}
	minor, err := strconv.ParseInt(evt.Env["MINOR"], 10, 16)
	if err != nil {
		m.l.Error().Err(err).Msg("Error getting device minor number")
	}

	d := &udev.Device{
		OriginalId: 0,
		Id:         0,
		KObj:       evt.KObj,
		Env:        evt.Env,
		Major:      int16(major),
		Minor:      int16(minor),
		DevPath:    "/dev/input/" + last,
		DeviceType: udev.MOUSE,
	}

	d.Initialize(m.udev)
	if isMouseDev {
		m.mousedevice = d
	} else {
		m.eventdevice = d
	}
}

func (m *VirtualMouse) SetMouseButtonLeft(down bool) error {
	if down {
		return m.m.LeftPress()
	} else {
		return m.m.LeftRelease()
	}
}
func (m *VirtualMouse) SetMouseButtonRight(down bool) error {
	if down {
		return m.m.RightPress()
	} else {
		return m.m.RightRelease()
	}
}
func (m *VirtualMouse) SetMouseButtonMiddle(down bool) error {
	if down {
		return m.m.MiddlePress()
	} else {
		return m.m.MiddleRelease()
	}
}
func (m *VirtualMouse) MoveMouse(x float64, y float64) error {
	return m.m.Move(int32(x), int32(y))
}
func (m *VirtualMouse) MoveMouseWheel(x float64, y float64) error {
	return errors.Join(
		m.m.Wheel(true, int32(x)),
		m.m.Wheel(false, int32(y)),
	)
}

func (m *VirtualMouse) Close() error {
	m.l.Debug().Msg("Closing down keyboard...")
	if m.eventdevice != nil {
		m.eventdevice.Close()
	}
	if m.mousedevice != nil {
		m.mousedevice.Close()
	}
	if m.m != nil {
		m.m.Close()
	}
	return nil
}
