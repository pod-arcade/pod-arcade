package uinput

import (
	"errors"
	"fmt"
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

func CreateVirtualGamepad(ud *udev.UDev, gamepadId int, vendorId int16, productId int16) *VirtualGamepad {
	var gp = &VirtualGamepad{}
	gp.busy.Lock()
	defer gp.busy.Unlock()
	gp.gamepadId = gamepadId
	gp.vendorId = vendorId
	gp.productId = productId
	gp.udev = ud

	gp.l = log.NewLogger("input-uinput-gamepad", nil)

	return gp
}

func (gp *VirtualGamepad) GetName() string {
	return "uinput-gamepad"
}

func (gp *VirtualGamepad) SetGamepadRumbleHandler(handler api.GamepadRumbleHandler) {
	// TODO: Actually set this up correctly
}

func (gp *VirtualGamepad) OpenGamepad() error {
	gamepadName := fmt.Sprintf("[PA] Gamepad %v", gp.gamepadId)

	// register Udev listeners to create our devices
	// when the kernel has finished what it needs to
	gp.udevListeners = append(gp.udevListeners, gp.udev.KernelEvents.AddListener(eventemitter.EventType(udev.ADD), eventemitter.HandleFunc(func(arguments ...interface{}) {
		evt := (arguments[0]).(*udev.UEvent)
		gp.handleEvent(evt)
	})))

	if gamepad, err := uinput.CreateGamepad("/dev/uinput", []byte(gamepadName), uint16(gp.vendorId), uint16(gp.productId)); err != nil {
		return err
	} else {
		gp.gamepad = gamepad
		gp.l.Info().Msgf("Gamepad created successfully — Gamepad %v", gp.gamepadId)
	}
	if syspath, err := gp.gamepad.FetchSyspath(); err != nil {
		gp.Close()
		return err
	} else {
		gp.syspath = syspath
	}
	gp.l = gp.l.With().Str("syspath", gp.syspath).Logger()
	gp.l.Debug().Msg("Fetched syspath")
	return nil
}

func (pad *VirtualGamepad) handleEvent(evt *udev.UEvent) {
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

func (pad *VirtualGamepad) createJSDevice(evt *udev.UEvent, originalId int) {
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

func (pad *VirtualGamepad) createEventDevice(evt *udev.UEvent, originalId int) {
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

func (pad *VirtualGamepad) SetGamepadInputState(state api.GamepadInput) error {
	pad.setButtonState(uinput.ButtonNorth, state.North)
	pad.setButtonState(uinput.ButtonSouth, state.South)
	pad.setButtonState(uinput.ButtonWest, state.West)
	pad.setButtonState(uinput.ButtonEast, state.East)
	pad.setButtonState(uinput.ButtonBumperLeft, state.L1)
	pad.setButtonState(uinput.ButtonBumperRight, state.R1)
	pad.setButtonState(uinput.ButtonThumbLeft, state.LZ)
	pad.setButtonState(uinput.ButtonThumbRight, state.RZ)
	pad.setButtonState(uinput.ButtonSelect, state.Select)
	pad.setButtonState(uinput.ButtonStart, state.Start)
	pad.setButtonState(uinput.ButtonDpadUp, state.DPadUp)
	pad.setButtonState(uinput.ButtonDpadDown, state.DPadDown)
	pad.setButtonState(uinput.ButtonDpadLeft, state.DPadLeft)
	pad.setButtonState(uinput.ButtonDpadRight, state.DPadRight)
	pad.setButtonState(uinput.ButtonMode, state.Home)

	pad.gamepad.LeftStickMove(state.AxisLeftX, state.AxisLeftY)
	pad.gamepad.RightStickMove(state.AxisRightX, state.AxisRightY)
	pad.gamepad.LeftTriggerMove(state.AxisLeftTrigger)
	pad.gamepad.RightTriggerMove(state.AxisRightTrigger)

	pad.setButtonState(uinput.ButtonTriggerLeft, state.AxisLeftTrigger > 0.5)
	pad.setButtonState(uinput.ButtonTriggerRight, state.AxisRightTrigger > 0.5)
	return nil
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
