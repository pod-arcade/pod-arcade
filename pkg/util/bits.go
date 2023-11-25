package util

// PackBits packs 8 bools into a byte.
// NOTE: d0 is the LSB, d7 is the MSB.
func PackBits(d0, d1, d2, d3, d4, d5, d6, d7 bool) byte {
	b := byte(0)
	d := [8]bool{d0, d1, d2, d3, d4, d5, d6, d7}
	for i := 0; i < len(d); i++ {
		if d[i] {
			b |= 1 << i
		}
	}
	return b
}

// PackBits unpacks 8 bools from a byte.
// NOTE: d0 is the LSB, d7 is the MSB.
func UnpackBits(b byte) (d0, d1, d2, d3, d4, d5, d6, d7 bool) {
	d0 = b&(1<<0) != 0
	d1 = b&(1<<1) != 0
	d2 = b&(1<<2) != 0
	d3 = b&(1<<3) != 0
	d4 = b&(1<<4) != 0
	d5 = b&(1<<5) != 0
	d6 = b&(1<<6) != 0
	d7 = b&(1<<7) != 0
	return
}
