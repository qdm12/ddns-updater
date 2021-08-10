package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/qdm12/golibs/params"
)

type Update struct {
	Period   time.Duration
	Cooldown time.Duration
}

func (u *Update) get(env params.Env) (warning string, err error) {
	warning, err = u.getPeriod(env)
	if err != nil {
		return warning, err
	}

	u.Cooldown, err = env.Duration("UPDATE_COOLDOWN_PERIOD", params.Default("5m"))
	if err != nil {
		return "", fmt.Errorf("%w: for environment variable UPDATE_COOLDOWN_PERIOD", err)
	}

	return warning, nil
}

func (u *Update) getPeriod(env params.Env) (warning string, err error) {
	// Backward compatibility: DELAY
	s, err := env.Get("DELAY", params.Compulsory())
	if err == nil {
		warning = "the environment variable DELAY should be changed to PERIOD"
		// Backward compatibility: integer only, treated as seconds
		n, err := strconv.Atoi(s)
		if err == nil {
			u.Period = time.Duration(n) * time.Second
			return warning, nil
		}

		period, err := time.ParseDuration(s)
		if err == nil {
			u.Period = period
			return warning, nil
		}
	}

	u.Period, err = env.Duration("PERIOD", params.Default("10m"))
	if err != nil {
		return "", fmt.Errorf("%w: for environment variable PERIOD", err)
	}

	return "", err
}
