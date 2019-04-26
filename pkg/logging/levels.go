package logging

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
)

// Level represents the level of the logger
type Level uint8

// Different logger levels available
const (
	FatalLevel Level = iota
	ErrorLevel
	WarningLevel
	SuccessLevel
	InfoLevel
)

func (level Level) string() string {
	switch level {
	case FatalLevel:
		return "Fatal"
	case ErrorLevel:
		return "Error"
	case WarningLevel:
		return "Warning"
	case SuccessLevel:
		return "Success"
	case InfoLevel:
		return "Info"
	default:
		return fmt.Sprintf("Unknown level %d", uint8(level))
	}
}

func (level Level) formatHuman(message string) string {
	switch level {
	case FatalLevel:
		return color.RedString(emoji.Sprintf(":x: %s: %s", level.string(), message))
	case ErrorLevel:
		return color.HiRedString(emoji.Sprintf(":x: %s: %s", level.string(), message))
	case WarningLevel:
		return color.HiYellowString(emoji.Sprintf(":warning: %s: %s", level.string(), message))
	case SuccessLevel:
		return color.HiGreenString(emoji.Sprintf(":heavy_check_mark: %s: %s", level.string(), message))
	case InfoLevel:
		return fmt.Sprintf("%s: %s", level.string(), message)
	default:
		return fmt.Sprintf("%s: %s", level.string(), message)
	}
}
