package config

import (
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
		return err
	}

	l.Level, err = env.LogLevel("LOG_LEVEL", params.Default("info"))
	return err
}
