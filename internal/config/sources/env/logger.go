package env

import (
	"errors"
	"fmt"
	"strings"

	"github.com/qdm12/ddns-updater/internal/config/settings"
	"github.com/qdm12/gosettings/sources/env"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/log"
)

func (s *Source) readLogger() (settings settings.Logger, err error) {
	settings.Caller, err = readCaller(s.env)
	if err != nil {
		return settings, err
	}

	settings.Level, err = readLogLevel(s.env)
	if err != nil {
		return settings, err
	}

	return settings, nil
}

func readCaller(env env.Env) (caller *bool, err error) {
	callerString := env.String("LOG_CALLER")
	switch callerString {
	case "":
		return nil, nil //nolint:nilnil
	case "hidden":
		return ptrTo(false), nil
	case "short":
		return ptrTo(true), nil
	default:
		err = validate.IsOneOf(callerString, "", "hidden", "short")
		return nil, fmt.Errorf("environment variable LOG_CALLER: %w", err)
	}
}

func readLogLevel(env env.Env) (level *log.Level, err error) {
	s := env.String("LOG_LEVEL")
	if s == "" {
		return nil, nil //nolint:nilnil
	}

	level = new(log.Level)
	*level, err = parseLogLevel(s)
	if err != nil {
		return nil, fmt.Errorf("environment variable LOG_LEVEL: %w", err)
	}

	return level, nil
}

var ErrLogLevelUnknown = errors.New("log level is unknown")

func parseLogLevel(s string) (level log.Level, err error) {
	switch strings.ToLower(s) {
	case "debug":
		return log.LevelDebug, nil
	case "info":
		return log.LevelInfo, nil
	case "warning":
		return log.LevelWarn, nil
	case "error":
		return log.LevelError, nil
	default:
		return level, fmt.Errorf(
			"%w: %q is not valid and can be one of debug, info, warning or error",
			ErrLogLevelUnknown, s)
	}
}
