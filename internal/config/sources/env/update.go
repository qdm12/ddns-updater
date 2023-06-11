package env

import (
	"strconv"
	"time"

	"github.com/qdm12/ddns-updater/internal/config/settings"
)

func (s *Source) readUpdate() (settings settings.Update, err error) {
	settings.Period, err = s.readUpdatePeriod()
	if err != nil {
		return settings, err
	}

	settings.Cooldown, err = s.env.Duration("UPDATE_COOLDOWN_PERIOD")
	return settings, err
}

func (s *Source) readUpdatePeriod() (period time.Duration, err error) {
	// Retro-compatibility: DELAY variable name
	delayStringPtr := s.env.Get("DELAY")
	if delayStringPtr != nil {
		s.handleDeprecated("DELAY", "UPDATE_PERIOD")
		// Retro-compatibility: integer only, treated as seconds
		delayInt, err := strconv.Atoi(*delayStringPtr)
		if err == nil {
			return time.Duration(delayInt) * time.Second, nil
		}

		return time.ParseDuration(*delayStringPtr)
	}

	return s.env.Duration("UPDATE_PERIOD")
}
