package log

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
)

// environment variables holding commong log configuration
const (
	EnvLogLevel     = "LOG_LEVEL"
	EnvUseTimestamp = "LOG_USE_TIMESTAMP"
	EnvUseCaller    = "LOG_USE_CALLER"
)

var baseLogger *zerolog.Logger
var cfg LogConfig

type LogConfig struct {
	DefaultLevel zerolog.Level
	Timestamp    bool
	Caller       bool
}

func LevelForComponent(component string) zerolog.Level {
	compEnv := component
	compEnv = strings.ReplaceAll(compEnv, "-", "_")
	compEnv = strings.ReplaceAll(compEnv, ".", "_")
	compEnv = strings.ReplaceAll(compEnv, " ", "_")
	compEnv = strings.ToUpper(EnvLogLevel + "_" + compEnv)
	if os.Getenv(compEnv) != "" {
		return LevelFromString(os.Getenv(compEnv))
	} else {
		return LevelFromString(os.Getenv(EnvLogLevel))
	}
}

func LevelFromString(logLevel string) zerolog.Level {
	switch strings.ToLower(logLevel) {
	case "none":
		return zerolog.Disabled
	case "disabled":
		return zerolog.Disabled
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

func Setup(c LogConfig) {
	cfg = c
	// just in case someone already called GetBaseLogger
	baseLogger = nil
}

func GetBaseLogger() zerolog.Logger {
	if baseLogger == nil {
		ctx := zerolog.New(zerolog.NewConsoleWriter()).
			Level(LevelFromString(os.Getenv(EnvLogLevel))).
			With()

		if cfg.Timestamp {
			ctx = ctx.
				Timestamp()
		}

		if cfg.Caller {
			ctx = ctx.
				Caller()
		}

		l := ctx.Logger()
		baseLogger = &l
	}
	return *baseLogger
}

func NewLogger(component string, context map[string]string) zerolog.Logger {
	ctx := GetBaseLogger().Level(LevelForComponent(component)).With().Str("Component", component)

	for k, v := range context {
		ctx = ctx.Str(k, v)
	}

	return ctx.Logger()
}
