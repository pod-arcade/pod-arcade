package uinput

import (
	"fmt"
	"io"
	"os"
)

// A Keyboard is an key event output device. It is used to
// enable a program to simulate HID keyboard input events.
type Keyboard interface {
	// KeyPress will cause the key to be pressed and immediately released.
	KeyPress(key int) error

	// KeyDown will send a keypress event to an existing keyboard device.
	// The key can be any of the predefined keycodes from keycodes.go.
	// Note that the key will be "held down" until "KeyUp" is called.
	KeyDown(key int) error

	// KeyUp will send a keyrelease event to an existing keyboard device.
	// The key can be any of the predefined keycodes from keycodes.go.
	KeyUp(key int) error

	// FetchSysPath will return the syspath to the device file.
	FetchSyspath() (string, error)

	io.Closer
}

type vKeyboard struct {
	name       []byte
	deviceFile *os.File
}

// CreateKeyboard will create a new keyboard using the given uinput
// device path of the uinput device.
func CreateKeyboard(path string, name []byte) (Keyboard, error) {
	err := validateDevicePath(path)
	if err != nil {
		return nil, err
	}
	err = validateUinputName(name)
	if err != nil {
		return nil, err
	}

	fd, err := createVKeyboardDevice(path, name)
	if err != nil {
		return nil, err
	}

	return vKeyboard{name: name, deviceFile: fd}, nil
}

// KeyPress will issue a single key press (push down a key and then immediately release it).
func (vk vKeyboard) KeyPress(key int) error {
	if !keyCodeInRange(key) {
		return fmt.Errorf("failed to perform KeyPress. Code %d is not in range", key)
	}
	err := sendBtnEvent(vk.deviceFile, []int{key}, btnStatePressed)
	if err != nil {
		return fmt.Errorf("failed to issue the KeyDown event: %v", err)
	}

	return sendBtnEvent(vk.deviceFile, []int{key}, btnStateReleased)
}

// KeyDown will send the key code passed (see keycodes.go for available keycodes). Note that unless a key release
// event is sent to the device, the key will remain pressed and therefore input will continuously be generated. Therefore,
// do not forget to call "KeyUp" afterwards.
func (vk vKeyboard) KeyDown(key int) error {
	if !keyCodeInRange(key) {
		return fmt.Errorf("failed to perform KeyDown. Code %d is not in range", key)
	}
	return sendBtnEvent(vk.deviceFile, []int{key}, btnStatePressed)
}

// KeyUp will release the given key passed as a parameter (see keycodes.go for available keycodes). In most
// cases it is recommended to call this function immediately after the "KeyDown" function in order to only issue a
// single key press.
func (vk vKeyboard) KeyUp(key int) error {
	if !keyCodeInRange(key) {
		return fmt.Errorf("failed to perform KeyUp. Code %d is not in range", key)
	}

	return sendBtnEvent(vk.deviceFile, []int{key}, btnStateReleased)
}

// Close will close the device and free resources.
// It's usually a good idea to use defer to call this function.
func (vk vKeyboard) Close() error {
	return closeDevice(vk.deviceFile)
}

func createVKeyboardDevice(path string, name []byte) (fd *os.File, err error) {
	deviceFile, err := createDeviceFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual keyboard device: %v", err)
	}

	err = registerDevice(deviceFile, uintptr(evKey))
	if err != nil {
		deviceFile.Close()
		return nil, fmt.Errorf("failed to register virtual keyboard device: %v", err)
	}

	// register key events
	for i := 0; i <= keyMax; i++ {
		err = ioctl(deviceFile, uiSetKeyBit, uintptr(i))
		if err != nil {
			deviceFile.Close()
			return nil, fmt.Errorf("failed to register key number %d: %v", i, err)
		}
	}

	return createUsbDevice(deviceFile,
		uinputUserDev{
			Name: toUinputName(name),
			ID: inputID{
				Bustype: busUsb,
				Vendor:  0x4711,
				Product: 0x0815,
				Version: 1}})
}

func keyCodeInRange(key int) bool {
	return key >= keyReserved && key <= keyMax
}

func (vk vKeyboard) FetchSyspath() (string, error) {
	return fetchSyspath(vk.deviceFile)
}
