package logging

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/kyokomi/emoji"
)

// Level represents the level of the logger
type Level uint8

// Different logger levels available
const (
	LEVELFATAL Level = iota
	LEVELERROR
	LEVELWARNING
	LEVELSUCCESS
	LEVELINFO
)

// LEVELDEFAULT is the default logger level
const LEVELDEFAULT = LEVELINFO

func (level Level) string() string {
	switch level {
	case LEVELFATAL:
		return "Fatal"
	case LEVELERROR:
		return "Error"
	case LEVELWARNING:
		return "Warning"
	case LEVELSUCCESS:
		return "Success"
	case LEVELINFO:
		return "Info"
	default:
		return fmt.Sprintf("Unknown level %d", uint8(level))
	}
}

func (level Level) formatHuman(message string) string {
	switch level {
	case LEVELFATAL:
		return color.RedString(emoji.Sprintf(":x: %s: %s", level.string(), message))
	case LEVELERROR:
		return color.HiRedString(emoji.Sprintf(":x: %s: %s", level.string(), message))
	case LEVELWARNING:
		return color.HiYellowString(emoji.Sprintf(":warning: %s: %s", level.string(), message))
	case LEVELSUCCESS:
		return color.HiGreenString(emoji.Sprintf(":heavy_check_mark: %s: %s", level.string(), message))
	case LEVELINFO:
		return fmt.Sprintf("%s: %s", level.string(), message)
	default:
		return fmt.Sprintf("%s: %s", level.string(), message)
	}
}

// ParseLevel returns the corresponding level from a string
func ParseLevel(s string) Level {
	s = strings.ToLower(s)
	switch s {
		case "info":
			return LEVELINFO
		case "success":
			return LEVELSUCCESS
		case "warning":
			return LEVELWARNING
		case "error":
			return LEVELERROR
		case "":
			return LEVELDEFAULT
		default:
			Warn("Unrecognized logging level %s", s)
			return LEVELDEFAULT
	}
}
