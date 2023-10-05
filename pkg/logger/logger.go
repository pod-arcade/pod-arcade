package logger

import (
	"github.com/rs/zerolog"
)

var baseLogger *zerolog.Logger

func GetBaseLogger() zerolog.Logger {
	if baseLogger == nil {
		ctx := zerolog.New(zerolog.NewConsoleWriter()).
			Level(zerolog.Level(cfg.LogLevel)).
			With()

		if cfg.UseTimestamp {
			ctx = ctx.
				Timestamp()
		}

		if cfg.UseCaller {
			ctx = ctx.
				Caller()
		}

		l := ctx.Logger()
		baseLogger = &l
	}
	return *baseLogger
}

func WithContext(logger zerolog.Logger, context map[string]string) zerolog.Logger {
	ctx := logger.With()
	for k, v := range context {
		ctx = ctx.Str(k, v)
	}
	return ctx.Logger()
}

func CreateLogger(context map[string]string) zerolog.Logger {
	return WithContext(GetBaseLogger(), context)
}
