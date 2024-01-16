package config

import (
	"fmt"
	"strings"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
	"github.com/qdm12/log"
)

type Logger struct {
	Level  string
	Caller string
}

func (l *Logger) setDefaults() {
	l.Level = gosettings.DefaultComparable(l.Level, log.LevelInfo.String())
	l.Caller = gosettings.DefaultComparable(l.Caller, "hidden")
}

func (l Logger) Validate() (err error) {
	_, err = log.ParseLevel(l.Level)
	if err != nil {
		return fmt.Errorf("log level: %w", err)
	}

	err = validate.IsOneOf(l.Caller, "hidden", "short")
	if err != nil {
		return fmt.Errorf("log caller: %w", err)
	}

	return nil
}

func (l Logger) ToOptions() (options []log.Option) {
	level, _ := log.ParseLevel(l.Level)
	options = append(options, log.SetLevel(level))
	if l.Caller == "short" {
		options = append(options, log.SetCallerFile(true), log.SetCallerLine(true))
	}
	return options
}

func (l Logger) String() string {
	return l.toLinesNode().String()
}

func (l Logger) toLinesNode() *gotree.Node {
	node := gotree.New("Logger")
	node.Appendf("Level: %s", l.Level)
	node.Appendf("Caller: %s", l.Caller)
	return node
}

func (l *Logger) read(reader *reader.Reader) {
	l.Level = reader.String("LOG_LEVEL")
	// Retro compatibility
	if strings.ToLower(l.Level) == "warning" {
		l.Level = "warn"
	}
	l.Caller = reader.String("LOG_CALLER")
}
