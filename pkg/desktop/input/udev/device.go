package udev

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/pilebones/go-udev/netlink"
	"github.com/pod-arcade/pod-arcade/pkg/logger"
	"github.com/rs/zerolog"
	"golang.org/x/sys/unix"
)

func touch(name string, perm fs.FileMode) error {
	file, err := os.OpenFile(name, os.O_RDONLY|os.O_CREATE, perm)
	if err != nil {
		return err
	}
	err = os.Chmod(name, perm)
	if err != nil {
		return err
	}
	return file.Close()
}

type DeviceType int

const (
	KEYBOARD DeviceType = iota
	MOUSE
	TOUCHSCREEN
	GAMEPAD
)

type Device struct {
	OriginalId int
	Id         int
	KObj       string
	Env        map[string]string
	Major      int16
	Minor      int16
	DevPath    string
	DeviceType DeviceType

	initTime int64 // should me usec
	udev     *UDev
	l        zerolog.Logger
}

/*
 * This function should be called after all of the public properties of the device have been set.
 */
func (d *Device) Initialize(udev *UDev) {
	d.udev = udev
	d.l = logger.CreateLogger(map[string]string{
		"Component":  "Device",
		"DevicePath": d.DevPath,
		"DeviceType": string(d.DeviceType),
		"Id":         string(d.Id),
		"Major":      string(d.Major),
		"Minor":      string(d.Minor),
	})

	d.l = logger.CreateLogger(map[string]string{
		"Id":         fmt.Sprint(d.Id),
		"OriginalId": fmt.Sprint(d.OriginalId),
	})
	if err := os.MkdirAll("/run/udev/data", 0o755); err != nil {
		d.l.Error().Err(err).Msg("Failed to create /run/udev/data")
	} else if os.Chmod("/run/udev/data", 0o755); err != nil {
		d.l.Error().Err(err).Msg("Failed to set permissions on /run/udev/data")
	}
	if err := touch("/run/udev/control", 0o755); err != nil {
		d.l.Error().Err(err).Msg("Failed to create /run/udev/control")
	}
	if err := os.RemoveAll(d.DevPath); err != nil {
		d.l.Error().Msgf("Failed to remove existing devpath %s", d.DevPath)
	}

	d.initTime = time.Now().UnixNano() / 1000

	d.MakeDeviceNode()
	d.WriteUDevDatabaseData()
	if err := d.EmitUDevEvent(netlink.ADD); err != nil {
		d.l.Error().Err(err).Msg("Failed to emit udev add event")
	}
	d.l.Debug().Msg("Device created successfully")
}

func (d *Device) GetUDevDBDevicePath() string {
	return fmt.Sprintf("/run/udev/data/c%v:%v", d.Major, d.Minor)
}

func (d *Device) GetUDevDBInputPath() string {
	components := strings.Split(d.KObj, "/")
	if len(components) <= 4 {
		panic("kObj is not in the expected format")
	}
	inputId := components[4]
	return fmt.Sprintf("/run/udev/data/+input:%v", inputId)
}

func (d *Device) WriteUDevDatabaseData() {
	// Write udev database information
	uDevDeviceDBPath := d.GetUDevDBDevicePath()
	uDevInputDBPath := d.GetUDevDBInputPath()
	data := ""
	data += fmt.Sprintf("I:%v\n", d.initTime)
	if d.DeviceType == GAMEPAD {
		data += "E:ID_INPUT_JOYSTICK=1\n"
	}
	if d.DeviceType == MOUSE {
		data += "E:ID_INPUT_MOUSE=1\n"
	}
	data += "E:ID_INPUT=1\n"
	data += "E:ID_SERIAL=noserial\n"
	data += "G:seat\n"
	data += "G:uaccess\n"
	data += "Q:seat\n"
	data += "Q:uaccess\n"
	data += "V:1\n"
	if err := os.WriteFile(uDevDeviceDBPath, []byte(data), 0o755); err != nil {
		d.l.Error().Err(err).Msgf("Failed to write device database to %s", uDevDeviceDBPath)
	}
	if err := os.WriteFile(uDevInputDBPath, []byte(data), 0o755); err != nil {
		d.l.Error().Err(err).Msgf("Failed to write device database to %s", uDevDeviceDBPath)
	}
}

func (d *Device) CleanupUDevDatabaseData() error {
	// Write udev database information
	uDevDeviceDBPath := d.GetUDevDBDevicePath()
	uDevInputDBPath := d.GetUDevDBInputPath()
	return errors.Join(os.Remove(uDevDeviceDBPath),
		os.Remove(uDevInputDBPath))
}

func (d *Device) MakeDeviceNode() {
	devId := unix.Mkdev(uint32(d.Major), uint32(d.Minor))
	if err := unix.Mknod(d.DevPath, syscall.S_IFCHR|0o777, int(devId)); err != nil {
		d.l.Error().Err(err).Msgf("Failed to mknod for device path at %s", d.DevPath)
		return
	}
	if err := os.Chmod(d.DevPath, syscall.S_IFCHR|0o777); err != nil {
		d.l.Error().Err(err).Msgf("Failed to update permissions to be more open at %s", d.DevPath)
	}
}

func (d *Device) EmitUDevEvent(action netlink.KObjAction) error {
	evt := netlink.UEvent{
		Action: action,
		KObj:   d.KObj,
		Env:    d.Env,
	}

	evt.Env["ACTION"] = action.String()
	evt.Env["DEVNAME"] = d.DevPath // overwrite the original devpath with our new one
	evt.Env["SUBSYSTEM"] = "input"
	evt.Env["USEC_INITIALIZED"] = fmt.Sprint(d.initTime)
	if d.DeviceType == GAMEPAD {
		evt.Env["ID_INPUT"] = "1"
		evt.Env["ID_INPUT_JOYSTICK"] = "1"
		evt.Env[".INPUT_CLASS"] = "joystick"
	} else if d.DeviceType == MOUSE {
		evt.Env["ID_INPUT"] = "1"
		evt.Env[".INPUT_CLASS"] = "mouse"
		evt.Env["ID_INPUT_MOUSE"] = "1"
	}
	evt.Env["ID_SERIAL"] = "noserial"
	evt.Env["TAGS"] = ":seat:uaccess:"
	evt.Env["CURRENT_TAGS"] = ":seat:uaccess:"

	evtString := strings.Join(strings.Split(evt.String(), "\000"), "\n\t")

	d.l.Info().Msgf("Emitting UDev Event %s", evtString)
	return d.udev.WriteUDevEvent(evt)
}

func (d *Device) Close() error {
	if d.udev == nil {
		return nil
	}
	return errors.Join(
		d.EmitUDevEvent(netlink.REMOVE),
		os.Remove(d.DevPath),
		d.CleanupUDevDatabaseData(),
	)
}

func (d *Device) GetSysPath() string {
	return "/sys" + d.KObj
}
