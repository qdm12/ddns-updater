package settings

import (
	"github.com/qdm12/gosettings"
	"github.com/qdm12/log"
)

type Logger struct {
	Caller *bool
	Level  *log.Level
}

func (l *Logger) setDefaults() {
	l.Caller = gosettings.DefaultPointer(l.Caller, false)
	l.Level = gosettings.DefaultPointer(l.Level, log.LevelInfo)
}

func (l Logger) mergeWith(other Logger) (merged Logger) {
	merged.Caller = gosettings.MergeWithPointer(l.Caller, other.Caller)
	merged.Level = gosettings.MergeWithPointer(l.Level, other.Level)
	return merged
}

func (l Logger) Validate() (err error) {
	return nil
}
