package logging

import "strings"

// Mode is the mode of the logger which can be Default, JSON or MODEHUMAN
type Mode uint8

// Different logger modes available
const (
	MODEJSON Mode = iota
	MODEHUMAN
)

// MODEDEFAULT is the default logging mode
const MODEDEFAULT = MODEJSON

// ParseMode returns the corresponding mode from a string
func ParseMode(s string) Mode {
	s = strings.ToLower(s)
	switch s {
		case "json":
			return MODEJSON
		case "human":
			return MODEHUMAN
		case "":
			return MODEDEFAULT
		default:
			// uses the global logger
			Warn("Unrecognized logging mode %s", s)
			return MODEDEFAULT
	}
}