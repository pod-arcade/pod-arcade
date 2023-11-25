/*
Package uinput is a pure go package that provides access to the userland input device driver uinput on linux systems.
Virtual keyboard devices as well as virtual mouse input devices may be created using this package.
The keycodes and other event definitions, that are available and can be used to trigger input events,
are part of this package ("Key1" for number 1, for example).

In order to use the virtual keyboard, you will need to follow these three steps:

 1. Initialize the device
    Example: vk, err := CreateKeyboard("/dev/uinput", "Virtual Keyboard")

 2. Send Button events to the device
    Example (print a single D):
    err = vk.KeyPress(uinput.KeyD)

    Example (keep moving right by holding down right arrow key):
    err = vk.KeyDown(uinput.KeyRight)

    Example (stop moving right by releasing the right arrow key):
    err = vk.KeyUp(uinput.KeyRight)

 3. Close the device
    Example: err = vk.Close()

A virtual mouse input device is just as easy to create and use:

 1. Initialize the device:
    Example: vm, err := CreateMouse("/dev/uinput", "DangerMouse")

 2. Move the cursor around and issue click events
    Example (move mouse right):
    err = vm.MoveRight(42)

    Example (move mouse left):
    err = vm.MoveLeft(42)

    Example (move mouse up):
    err = vm.MoveUp(42)

    Example (move mouse down):
    err = vm.MoveDown(42)

    Example (trigger a left click):
    err = vm.LeftClick()

    Example (trigger a right click):
    err = vm.RightClick()

 3. Close the device
    Example: err = vm.Close()

If you'd like to use absolute input events (move the cursor to specific positions on screen), use the touch pad.
Note that you'll need to specify the size of the screen area you want to use when you initialize the
device. Here are a few examples of how to use the virtual touch pad:

 1. Initialize the device:
    Example: vt, err := CreateTouchPad("/dev/uinput", "DontTouchThis", 0, 1024, 0, 768)

 2. Move the cursor around and issue click events
    Example (move cursor to the top left corner of the screen):
    err = vt.MoveTo(0, 0)

    Example (move cursor to the position x: 100, y: 250):
    err = vt.MoveTo(100, 250)

    Example (trigger a left click):
    err = vt.LeftClick()

    Example (trigger a right click):
    err = vt.RightClick()

 3. Close the device
    Example: err = vt.Close()
*/
package uinput

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"
)

func validateDevicePath(path string) error {
	if path == "" {
		return errors.New("device path must not be empty")
	}
	_, err := os.Stat(path)
	return err
}

func validateUinputName(name []byte) error {
	if name == nil || len(name) == 0 {
		return errors.New("device name may not be empty")
	}
	if len(name) > uinputMaxNameSize {
		return fmt.Errorf("device name %s is too long (maximum of %d characters allowed)", name, uinputMaxNameSize)
	}
	return nil
}

func toUinputName(name []byte) (uinputName [uinputMaxNameSize]byte) {
	var fixedSizeName [uinputMaxNameSize]byte
	copy(fixedSizeName[:], name)
	return fixedSizeName
}

func createDeviceFile(path string) (fd *os.File, err error) {
	deviceFile, err := os.OpenFile(path, syscall.O_WRONLY|syscall.O_NONBLOCK, 0660)
	if err != nil {
		return nil, errors.New("could not open device file")
	}
	return deviceFile, err
}

func registerDevice(deviceFile *os.File, evType uintptr) error {
	err := ioctl(deviceFile, uiSetEvBit, evType)
	if err != nil {
		defer deviceFile.Close()
		err = releaseDevice(deviceFile)
		if err != nil {
			return fmt.Errorf("failed to close device: %v", err)
		}
		return fmt.Errorf("invalid file handle returned from ioctl: %v", err)
	}
	return nil
}

func createUsbDevice(deviceFile *os.File, dev uinputUserDev) (fd *os.File, err error) {
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.LittleEndian, dev)
	if err != nil {
		_ = deviceFile.Close()
		return nil, fmt.Errorf("failed to write user device buffer: %v", err)
	}
	_, err = deviceFile.Write(buf.Bytes())
	if err != nil {
		_ = deviceFile.Close()
		return nil, fmt.Errorf("failed to write uidev struct to device file: %v", err)
	}

	err = ioctl(deviceFile, uiDevCreate, uintptr(0))
	if err != nil {
		_ = deviceFile.Close()
		return nil, fmt.Errorf("failed to create device: %v", err)
	}

	time.Sleep(time.Millisecond * 200)

	return deviceFile, err
}

func closeDevice(deviceFile *os.File) (err error) {
	err = releaseDevice(deviceFile)
	if err != nil {
		return fmt.Errorf("failed to close device: %v", err)
	}
	return deviceFile.Close()
}

func releaseDevice(deviceFile *os.File) (err error) {
	return ioctl(deviceFile, uiDevDestroy, uintptr(0))
}

func fetchSyspath(deviceFile *os.File) (string, error) {
	sysInputDir := "/sys/devices/virtual/input/"
	// 64 for name + 1 for null byte
	path := make([]byte, 65)
	err := ioctl(deviceFile, uiGetSysname, uintptr(unsafe.Pointer(&path[0])))

	firstNull := bytes.IndexByte(path, 0)
	if firstNull != -1 {
		path = path[0:firstNull]
	}

	sysInputDir = sysInputDir + string(path)
	return sysInputDir, err
}

// Note that mice and touch pads do have buttons as well. Therefore, this function is used
// by all currently available devices and resides in the main source file.
func sendBtnEvent(deviceFile *os.File, keys []int, btnState int) (err error) {
	for _, key := range keys {
		buf, err := inputEventToBuffer(inputEvent{
			Time:  syscall.Timeval{Sec: 0, Usec: 0},
			Type:  evKey,
			Code:  uint16(key),
			Value: int32(btnState)})
		if err != nil {
			return fmt.Errorf("key event could not be set: %v", err)
		}
		_, err = deviceFile.Write(buf)
		if err != nil {
			return fmt.Errorf("writing btnEvent structure to the device file failed: %v", err)
		}
	}
	return syncEvents(deviceFile)
}

func syncEvents(deviceFile *os.File) (err error) {
	buf, err := inputEventToBuffer(inputEvent{
		Time:  syscall.Timeval{Sec: 0, Usec: 0},
		Type:  evSyn,
		Code:  uint16(synReport),
		Value: 0})
	if err != nil {
		return fmt.Errorf("writing sync event failed: %v", err)
	}
	_, err = deviceFile.Write(buf)
	return err
}

func inputEventToBuffer(iev inputEvent) (buffer []byte, err error) {
	buf := bytes.NewBuffer(make([]byte, 0, 24))
	err = binary.Write(buf, binary.LittleEndian, iev)
	if err != nil {
		return nil, fmt.Errorf("failed to write input event to buffer: %v", err)
	}
	return buf.Bytes(), nil
}

// original function taken from: https://github.com/tianon/debian-golang-pty/blob/master/ioctl.go
func ioctl(deviceFile *os.File, cmd, ptr uintptr) error {
	_, _, errorCode := syscall.Syscall(syscall.SYS_IOCTL, deviceFile.Fd(), cmd, ptr)
	if errorCode != 0 {
		return errorCode
	}
	return nil
}
