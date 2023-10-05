package logger

import (
	env "github.com/caarlos0/env/v9"
)

type config struct {
	LogLevel     int  `env:"LOG_LEVEL" envDefault:"0"`
	UseTimestamp bool `env:"LOG_USE_TIMESTAMP" envDefault:"true"`
	UseCaller    bool `env:"LOG_USE_CALLER" envDefault:"true"`
}

var cfg config

func init() {
	env.Parse(&cfg)
}
