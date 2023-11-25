package uinput

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/pod-arcade/pod-arcade/internal/udev"
	"github.com/pod-arcade/pod-arcade/internal/uinput"
	"github.com/pod-arcade/pod-arcade/pkg/log"
	"github.com/rs/zerolog"
	eventemitter "github.com/vansante/go-event-emitter"
)

type VirtualKeyboard struct {
	udev    *udev.UDev
	kb      uinput.Keyboard
	dev     *udev.Device
	syspath string

	done chan interface{}
	l    zerolog.Logger
	mtx  sync.Mutex
}

func NewVirtualKeyboard(ctx context.Context, uDev *udev.UDev) (*VirtualKeyboard, error) {
	k := &VirtualKeyboard{
		udev: uDev,
		done: make(chan interface{}, 1),
		l:    log.NewLogger("input-uinput-keyboard", nil),
	}
	k.mtx.Lock()
	defer k.mtx.Unlock()
	uDev.KernelEvents.AddListener(eventemitter.EventType(udev.ADD), eventemitter.HandleFunc(func(arguments ...interface{}) {
		k.mtx.Lock()
		defer k.mtx.Unlock()
		evt := (arguments[0]).(*udev.UEvent)
		if strings.HasPrefix("/sys"+evt.KObj, k.syspath) {
			k.l.Debug().Msg("Handling event for my device")
			k.handleEvent(evt)
		} else {
			k.l.Debug().Msgf("skipping event for not my device %v does not have prefix %v", "/sys"+evt.KObj, k.syspath)
		}
	}))
	kb, err := uinput.CreateKeyboard("/dev/uinput", []byte("[PA] Keyboard"))
	if err != nil {
		return nil, err
	}
	k.kb = kb

	syspath, err := kb.FetchSyspath()
	if err != nil {
		k.l.Error().Err(err).Msg("Failed to get syspath")
		return nil, err
	}
	k.syspath = syspath
	k.l = k.l.With().Str("syspath", syspath).Logger()
	k.l.Debug().Msg("Fetched syspath")

	context.AfterFunc(ctx, k.Close)

	return k, nil
}

func (kb *VirtualKeyboard) handleEvent(evt *udev.UEvent) {
	comps := strings.Split(evt.KObj, "/")
	last := comps[len(comps)-1]
	if !regexp.MustCompile("event[0-9]+").MatchString(last) {
		kb.l.Debug().Msgf("Skipping device that doesn't match %v", last)
		return
	}

	major, err := strconv.ParseInt(evt.Env["MAJOR"], 10, 16)
	if err != nil {
		kb.l.Error().Err(err).Msg("Error getting device major number")
	}
	minor, err := strconv.ParseInt(evt.Env["MINOR"], 10, 16)
	if err != nil {
		kb.l.Error().Err(err).Msg("Error getting device minor number")
	}

	d := &udev.Device{
		OriginalId: 0, // doesn't matter
		Id:         0, // also doesn't matter
		KObj:       evt.KObj,
		Env:        evt.Env,
		Major:      int16(major),
		Minor:      int16(minor),
		DevPath:    "/dev/input/" + last,
		DeviceType: udev.KEYBOARD,
	}

	d.Initialize(kb.udev)
	kb.dev = d
}

func (kb *VirtualKeyboard) KeyEvent(down bool, code int) {
	if down {
		kb.kb.KeyDown(code)
	} else {
		kb.kb.KeyUp(code)
	}
}

func (kb *VirtualKeyboard) Close() {
	kb.l.Debug().Msg("Closing down keyboard...")
	if kb.dev != nil {
		kb.dev.Close()
	}
	if kb.kb != nil {
		kb.kb.Close()
	}
	kb.done <- nil
}

func (k *VirtualKeyboard) Done() <-chan interface{} {
	return k.done
}
