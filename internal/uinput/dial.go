package uinput

import (
	"fmt"
	"io"
	"os"
	"syscall"
)

// A Dial is a device that will trigger rotation events.
// For details see: https://www.kernel.org/doc/Documentation/input/event-codes.txt
type Dial interface {
	// Turn will simulate a dial movement.
	Turn(delta int32) error

	io.Closer
}

type vDial struct {
	name       []byte
	deviceFile *os.File
}

// CreateDial will create a new dial input device. A dial is a device that can trigger rotation events.
func CreateDial(path string, name []byte) (Dial, error) {
	err := validateDevicePath(path)
	if err != nil {
		return nil, err
	}
	err = validateUinputName(name)
	if err != nil {
		return nil, err
	}

	fd, err := createDial(path, name)
	if err != nil {
		return nil, err
	}

	return vDial{name: name, deviceFile: fd}, nil
}

// Turn will simulate a dial movement.
func (vRel vDial) Turn(delta int32) error {
	return sendDialEvent(vRel.deviceFile, delta)
}

// Close closes the device and releases the device.
func (vRel vDial) Close() error {
	return closeDevice(vRel.deviceFile)
}

func createDial(path string, name []byte) (fd *os.File, err error) {
	deviceFile, err := createDeviceFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not create dial input device: %v", err)
	}

	err = registerDevice(deviceFile, uintptr(evRel))
	if err != nil {
		deviceFile.Close()
		return nil, fmt.Errorf("failed to register dial input device: %v", err)
	}

	// register dial events
	err = ioctl(deviceFile, uiSetRelBit, uintptr(relDial))
	if err != nil {
		deviceFile.Close()
		return nil, fmt.Errorf("failed to register dial events: %v", err)
	}

	return createUsbDevice(deviceFile,
		uinputUserDev{
			Name: toUinputName(name),
			ID: inputID{
				Bustype: busUsb,
				Vendor:  0x4711,
				Product: 0x0816,
				Version: 1}})
}

func sendDialEvent(deviceFile *os.File, delta int32) error {
	iev := inputEvent{
		Time:  syscall.Timeval{Sec: 0, Usec: 0},
		Type:  evRel,
		Code:  relDial,
		Value: delta}

	buf, err := inputEventToBuffer(iev)
	if err != nil {
		return fmt.Errorf("writing abs event failed: %v", err)
	}

	_, err = deviceFile.Write(buf)
	if err != nil {
		return fmt.Errorf("failed to write rel event to device file: %v", err)
	}

	return syncEvents(deviceFile)
}
