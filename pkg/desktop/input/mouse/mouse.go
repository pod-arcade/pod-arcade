package mouse

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/JohnCMcDonough/uinput"
	"github.com/pilebones/go-udev/netlink"
	"github.com/pod-arcade/pod-arcade/pkg/desktop/input/udev"
	"github.com/pod-arcade/pod-arcade/pkg/logger"
	"github.com/rs/zerolog"
	eventemitter "github.com/vansante/go-event-emitter"
)

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

func NewVirtualMouse(ctx context.Context, udev *udev.UDev) (*VirtualMouse, error) {
	m := &VirtualMouse{
		udev: udev,
		done: make(chan interface{}, 1),
		l: logger.CreateLogger(map[string]string{
			"Component": "Virtual Mouse",
		}),
	}
	m.mtx.Lock()
	defer m.mtx.Unlock()

	udev.KernelEvents.AddListener(eventemitter.EventType(netlink.ADD), eventemitter.HandleFunc(func(arguments ...interface{}) {
		m.mtx.Lock()
		defer m.mtx.Unlock()
		evt := (arguments[0]).(*netlink.UEvent)

		if strings.HasPrefix("/sys"+evt.KObj, m.syspath) {
			m.l.Debug().Msg("Handling event for my device")
			m.handleEvent(evt)
		} else {
			m.l.Debug().Msgf("skipping event for not my device %v does not have prefix %v", "/sys"+evt.KObj, m.syspath)
		}
	}))

	mouse, err := uinput.CreateMouse("/dev/uinput", []byte("[PA] Mouse"))
	if err != nil {
		return nil, err
	}
	m.m = mouse

	syspath, err := mouse.FetchSyspath()
	if err != nil {
		m.l.Error().Err(err).Msg("Failed to get syspath")
		return nil, err
	}

	m.l = m.l.With().Str("syspath", syspath).Logger()
	m.l.Debug().Msg("Fetched syspath")

	context.AfterFunc(ctx, m.Close)

	return m, nil
}

func (m *VirtualMouse) handleEvent(evt *netlink.UEvent) {
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

func (m *VirtualMouse) LeftClick(down bool) {
	if down {
		m.m.LeftPress()
	} else {
		m.m.LeftRelease()
	}
}
func (m *VirtualMouse) RightClick(down bool) {
	if down {
		m.m.RightPress()
	} else {
		m.m.RightRelease()
	}
}
func (m *VirtualMouse) MiddleClick(down bool) {
	if down {
		m.m.MiddlePress()
	} else {
		m.m.MiddleRelease()
	}
}
func (m *VirtualMouse) MoveCursor(x float32, y float32) {
	m.m.Move(int32(x), int32(y))
}
func (m *VirtualMouse) MoveWheel(x float32, y float32) {
	m.m.Wheel(true, int32(x))
	m.m.Wheel(false, int32(y))
}

func (m *VirtualMouse) Close() {
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
	m.done <- nil
}

func (k *VirtualMouse) Done() <-chan interface{} {
	return k.done
}
