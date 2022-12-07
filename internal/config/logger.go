package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/qdm12/log"
)

type Logger struct {
	Caller bool
	Level  log.Level
}

var (
	ErrLogCallerNotValid = errors.New("LOG_CALLER value is not valid")
)

func readLog() (settings Logger, err error) {
	callerString := os.Getenv("LOG_CALLER")
	switch callerString {
	case "":
	case "hidden":
	case "short":
		settings.Caller = true
	default:
		return settings, fmt.Errorf("%w: "+
			`%q must be one of "", "hidden" or "short"`,
			ErrLogCallerNotValid, callerString)
	}

	settings.Level, err = readLogLevel()
	if err != nil {
		return settings, err
	}

	return settings, nil
}

func readLogLevel() (level log.Level, err error) {
	s := os.Getenv("LOG_LEVEL")
	if s == "" {
		return log.LevelInfo, nil
	}

	level, err = parseLogLevel(s)
	if err != nil {
		return level, fmt.Errorf("environment variable LOG_LEVEL: %w", err)
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
