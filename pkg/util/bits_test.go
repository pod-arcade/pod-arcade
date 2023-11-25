package util_test

import (
	"testing"

	"github.com/pod-arcade/pod-arcade/pkg/util"
)

func btb(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestPackBits(t *testing.T) {
	b := util.PackBits(true, false, true, false, true, false, true, false)
	if b != 0b01010101 {
		t.Errorf("Expected 01010101, got %08b", b)
	}
}
func TestUnpackBits(t *testing.T) {
	b1, b2, b3, b4, b5, b6, b7, b8 := util.UnpackBits(0b01010101)
	if !b1 || b2 || !b3 || b4 || !b5 || b6 || !b7 || b8 {
		t.Errorf("Expected 10101010, got %v%v%v%v%v%v%v%v", btb(b1), btb(b2), btb(b3), btb(b4), btb(b5), btb(b6), btb(b7), btb(b8))
	}
}
