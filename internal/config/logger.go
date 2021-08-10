package config

import (
	"fmt"

	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/params"
)

type Logger struct {
	Caller logging.Caller
	Level  logging.Level
}

func (l *Logger) get(env params.Env) (err error) {
	l.Caller, err = env.LogCaller("LOG_CALLER", params.Default("hidden"))
	if err != nil {
		return fmt.Errorf("%w: for environment variable LOG_CALLER", err)
	}

	l.Level, err = env.LogLevel("LOG_LEVEL", params.Default("info"))
	if err != nil {
		return fmt.Errorf("%w: for environment variable LOG_LEVEL", err)
	}

	return err
}
