package util

import "math/rand"

type LossyChannel[C any] struct {
	LossRate     float32
	ChannelDepth int
	// Write messages to the in
	In chan C
	// Read messages from the out
	Out chan C
}

func MakeLossyChannel[C any](lossRate float32, channelDepth int) *LossyChannel[C] {
	lc := &LossyChannel[C]{
		LossRate:     lossRate,
		ChannelDepth: channelDepth,
		In:           make(chan C, channelDepth),
		Out:          make(chan C, channelDepth),
	}

	go func() {
		for i := range lc.In {
			if rand.Float32() < lossRate {
				continue
			}
			lc.Out <- i
		}
	}()

	return lc
}

func (c *LossyChannel[C]) Close() {
	close(c.In)
	close(c.Out)
}
