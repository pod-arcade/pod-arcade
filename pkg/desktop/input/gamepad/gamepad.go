package gamepad

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/JohnCMcDonough/pod-arcade/pkg/desktop/input/udev"
	"github.com/JohnCMcDonough/pod-arcade/pkg/logger"
	"github.com/JohnCMcDonough/uinput"
	"github.com/pilebones/go-udev/netlink"
	"github.com/rs/zerolog"
	eventemitter "github.com/vansante/go-event-emitter"
)

type VirtualGamepad struct {
	l zerolog.Logger

	gamepadId int
	vendorId  int16
	productId int16

	udev          *udev.UDev
	udevListeners []*eventemitter.Listener

	syspath        string
	gamepad        uinput.Gamepad
	eventDevice    udev.Device
	joystickDevice udev.Device

	busy sync.Mutex
}

func CreateVirtualGamepad(udev *udev.UDev, gamepadId int, vendorId int16, productId int16) (*VirtualGamepad, error) {
	var gp = &VirtualGamepad{}
	gp.busy.Lock()
	defer gp.busy.Unlock()
	gp.gamepadId = gamepadId
	gp.vendorId = vendorId
	gp.productId = productId
	gp.udev = udev
	gamepadName := fmt.Sprintf("Gamepad %v", gamepadId)

	gp.l = logger.CreateLogger(map[string]string{
		"gamepadId": fmt.Sprint(gamepadId),
	})

	// register Udev listeners to create our devices
	// when the kernel has finished what it needs to
	gp.udevListeners = append(gp.udevListeners, udev.KernelEvents.AddListener(eventemitter.EventType(netlink.ADD), eventemitter.HandleFunc(func(arguments ...interface{}) {
		evt := (arguments[0]).(*netlink.UEvent)
		gp.handleEvent(evt)
	})))

	if gamepad, err := uinput.CreateGamepad("/dev/uinput", []byte(gamepadName), uint16(vendorId), uint16(productId)); err != nil {
		return nil, err
	} else {
		gp.gamepad = gamepad
		gp.l.Info().Msgf("Gamepad created successfully — Gamepad %v", gp.gamepadId)
	}
	if syspath, err := gp.gamepad.FetchSyspath(); err != nil {
		gp.Close()
		return nil, err
	} else {
		gp.syspath = syspath
	}
	return gp, nil
}

func (pad *VirtualGamepad) handleEvent(evt *netlink.UEvent) {
	pad.busy.Lock()
	defer pad.busy.Unlock()
	// syspath should include the /jsX
	// or the /eventX appended to the end
	sysPath := "/sys" + evt.KObj
	if !strings.HasPrefix(sysPath, pad.syspath) {
		pad.l.Trace().Msgf("Path %s does not match prefix %s", sysPath, pad.syspath)
		return
	}
	// device belongs to us. Let's determine if it's a js or event device (or neither)
	comps := strings.Split(sysPath, "/")
	last := comps[len(comps)-1]

	if strings.HasPrefix(last, "js") {
		// We found a JS device
		jsIdString, _ := strings.CutPrefix(last, "js")
		if jsId, err := strconv.ParseInt(jsIdString, 10, 32); err != nil {
			pad.l.Error().Err(err).Msg("Failed to parse js id")
		} else {
			pad.createJSDevice(evt, int(jsId))
		}
	} else if strings.HasPrefix(last, "event") {
		// We found an event device
		eventIdString, _ := strings.CutPrefix(last, "event")
		if eventId, err := strconv.ParseInt(eventIdString, 10, 32); err != nil {
			pad.l.Error().Err(err).Msg("Failed to parse event id")
		} else {
			pad.createEventDevice(evt, int(eventId))
		}
	} else {
		pad.l.Info().Msgf("Found matching device without a type %v", last)
	}
}

func (pad *VirtualGamepad) createJSDevice(evt *netlink.UEvent, originalId int) {
	pad.l.Info().Msgf("Creating JS Device js%v", pad.gamepadId)
	dev := &pad.joystickDevice
	dev.OriginalId = originalId

	dev.Id = pad.gamepadId // remap this to our id
	dev.KObj = evt.KObj
	dev.Env = evt.Env
	dev.DeviceType = udev.GAMEPAD
	if major, err := strconv.ParseInt(evt.Env["MAJOR"], 10, 16); err != nil {
		pad.l.Error().Err(err).Msg("Error getting device major number")
	} else {
		dev.Major = int16(major)
	}
	if minor, err := strconv.ParseInt(evt.Env["MINOR"], 10, 16); err == nil {
		dev.Minor = int16(minor)
	} else {
		pad.l.Error().Err(err).Msg("Error getting device minor number")
	}
	dev.DevPath = fmt.Sprintf("/dev/input/js%v", dev.Id)
	pad.createDevice(dev)
}

func (pad *VirtualGamepad) createEventDevice(evt *netlink.UEvent, originalId int) {
	pad.l.Info().Msgf("Creating Event Device event%v", originalId)
	dev := &pad.eventDevice
	dev.OriginalId = originalId

	dev.Id = originalId
	dev.KObj = evt.KObj
	dev.Env = evt.Env
	dev.DeviceType = udev.GAMEPAD
	if major, err := strconv.ParseInt(evt.Env["MAJOR"], 10, 16); err != nil {
		pad.l.Error().Err(err).Msg("Error getting device major number")
	} else {
		dev.Major = int16(major)
	}
	if minor, err := strconv.ParseInt(evt.Env["MINOR"], 10, 16); err == nil {
		dev.Minor = int16(minor)
	} else {
		pad.l.Error().Err(err).Msg("Error getting device minor number")
	}
	dev.DevPath = fmt.Sprintf("/dev/input/event%v", dev.Id)
	pad.createDevice(dev)
}

func (pad *VirtualGamepad) createDevice(dev *udev.Device) {
	dev.Initialize(pad.udev)
}

func (pad *VirtualGamepad) setButtonState(key int, pressed bool) {
	var err error
	if pressed {
		err = pad.gamepad.ButtonDown(key)
	} else {
		err = pad.gamepad.ButtonUp(key)
	}
	if err != nil {
		pad.l.Error().Err(err).Msg(fmt.Sprintf("Unable to set button state for gamepad %d %v", key, pressed))
	}
}

func (pad *VirtualGamepad) SendInput(state GamepadBitfield) {
	pad.setButtonState(uinput.ButtonNorth, state.ButtonNorth)
	pad.setButtonState(uinput.ButtonSouth, state.ButtonSouth)
	pad.setButtonState(uinput.ButtonWest, state.ButtonWest)
	pad.setButtonState(uinput.ButtonEast, state.ButtonEast)
	pad.setButtonState(uinput.ButtonBumperLeft, state.ButtonBumperLeft)
	pad.setButtonState(uinput.ButtonBumperRight, state.ButtonBumperRight)
	pad.setButtonState(uinput.ButtonThumbLeft, state.ButtonThumbLeft)
	pad.setButtonState(uinput.ButtonThumbRight, state.ButtonThumbRight)
	pad.setButtonState(uinput.ButtonSelect, state.ButtonSelect)
	pad.setButtonState(uinput.ButtonStart, state.ButtonStart)
	pad.setButtonState(uinput.ButtonDpadUp, state.ButtonDpadUp)
	pad.setButtonState(uinput.ButtonDpadDown, state.ButtonDpadDown)
	pad.setButtonState(uinput.ButtonDpadLeft, state.ButtonDpadLeft)
	pad.setButtonState(uinput.ButtonDpadRight, state.ButtonDpadRight)
	pad.setButtonState(uinput.ButtonMode, state.ButtonMode)

	pad.gamepad.LeftStickMove(state.AxisLeftX, state.AxisLeftY)
	pad.gamepad.RightStickMove(state.AxisRightX, state.AxisRightY)
	pad.gamepad.LeftTriggerMove(state.AxisLeftTrigger)
	pad.gamepad.RightTriggerMove(state.AxisRightTrigger)

	pad.setButtonState(uinput.ButtonTriggerLeft, state.AxisLeftTrigger > 0.5)
	pad.setButtonState(uinput.ButtonTriggerRight, state.AxisRightTrigger > 0.5)
}

func (pad *VirtualGamepad) Close() error {
	for _, l := range pad.udevListeners {
		pad.udev.KernelEvents.RemoveListener("ADD", l)
	}
	return errors.Join(
		pad.eventDevice.Close(),
		pad.joystickDevice.Close(),
	)
}
