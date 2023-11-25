package util

import (
	"io"

	"github.com/rs/zerolog"
)

var _ io.Writer = (*ProcessLogWrapper)(nil)

type ProcessLogWrapper struct {
	Logger zerolog.Logger
	Level  zerolog.Level
}

func NewProcessLogWrapper(logger zerolog.Logger, level zerolog.Level) *ProcessLogWrapper {
	return &ProcessLogWrapper{
		Logger: logger,
		Level:  level,
	}
}

func (p *ProcessLogWrapper) Write(data []byte) (int, error) {
	p.Logger.WithLevel(p.Level).Msg(string(data))
	return len(data), nil
}
