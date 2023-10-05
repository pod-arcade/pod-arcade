package gamepad

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"testing"
)

func TestUnmarshalBinary(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		wantErr   bool
		wantField *GamepadBitfield
	}{
		{
			name:    "Invalid payload size (too small)",
			data:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantErr: true,
		},
		{
			name:    "Invalid payload size (too large)",
			data:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			wantErr: true,
		},
		{
			name: "Valid payload",
			data: func() []byte {
				data := make([]byte, 26)
				data[0] = 0b10101010
				data[1] = 0b01010101
				binary.LittleEndian.PutUint32(data[2:6], 0x41280000)                // 10.5 in float32
				binary.LittleEndian.PutUint32(data[6:10], math.Float32bits(11.5))   // 11.5 in float32
				binary.LittleEndian.PutUint32(data[10:14], math.Float32bits(14.5))  // 14.5 in float32
				binary.LittleEndian.PutUint32(data[14:18], math.Float32bits(16.89)) // 16.89 in float32
				binary.LittleEndian.PutUint32(data[18:22], math.Float32bits(128.3)) // 128.3 in float32
				binary.LittleEndian.PutUint32(data[22:26], 0x0)                     // 0.0 in float32
				fmt.Printf("Hex value is %s", hex.EncodeToString(data))
				return data
			}(),
			wantErr: false,
			wantField: &GamepadBitfield{
				ButtonNorth:       false, // (data[0] >> 0) & 1
				ButtonSouth:       true,  // (data[0] >> 1) & 1
				ButtonWest:        false, // (data[0] >> 2) & 1
				ButtonEast:        true,  // (data[0] >> 3) & 1
				ButtonBumperLeft:  false, // (data[0] >> 4) & 1
				ButtonBumperRight: true,  // (data[0] >> 5) & 1
				ButtonThumbLeft:   false, // (data[0] >> 6) & 1
				ButtonThumbRight:  true,  // (data[0] >> 7) & 1

				ButtonSelect:    true,  // (data[1] >> 0) & 1
				ButtonStart:     false, // (data[1] >> 1) & 1
				ButtonDpadUp:    true,  // (data[1] >> 2) & 1
				ButtonDpadDown:  false, // (data[1] >> 3) & 1
				ButtonDpadLeft:  true,  // (data[1] >> 4) & 1
				ButtonDpadRight: false, // (data[1] >> 5) & 1
				ButtonMode:      true,  // (data[1] >> 6) & 1

				AxisLeftX:        10.5,
				AxisLeftY:        11.5,
				AxisRightX:       14.5,
				AxisRightY:       16.89,
				AxisLeftTrigger:  128.3,
				AxisRightTrigger: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gb := &GamepadBitfield{}
			err := gb.UnmarshalBinary(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalBinary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && *gb != *tt.wantField {
				t.Errorf("UnmarshalBinary() got = %v, want %v", gb, tt.wantField)
			}
		})
	}
}
