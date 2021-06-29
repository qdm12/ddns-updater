package config

import (
	"github.com/qdm12/golibs/params"
)

type Config struct {
	Client Client
	Update Update
	PubIP  PubIP
	IPv6   IPv6
	Server Server
	Health Health
	Paths  Paths
	Backup Backup
	Logger Logger
	Gotify Gotify
}

func (c *Config) Get(env params.Env) (warnings []string, err error) {
	if err := c.Client.get(env); err != nil {
		return warnings, err
	}

	warning, err := c.Update.get(env)
	warnings = appendIfNotEmpty(warnings, warning)
	if err != nil {
		return warnings, err
	}

	newWarnings, err := c.PubIP.get(env)
	warnings = append(warnings, newWarnings...)
	if err != nil {
		return warnings, err
	}

	if err := c.IPv6.get(env); err != nil {
		return warnings, err
	}

	warning, err = c.Server.get(env)
	warnings = appendIfNotEmpty(warnings, warning)
	if err != nil {
		return warnings, err
	}

	warning, err = c.Health.Get(env)
	warnings = appendIfNotEmpty(warnings, warning)
	if err != nil {
		return warnings, err
	}

	if err := c.Paths.get(env); err != nil {
		return warnings, err
	}

	if err := c.Backup.get(env); err != nil {
		return warnings, err
	}

	if err := c.Logger.get(env); err != nil {
		return warnings, err
	}

	if err := c.Gotify.get(env); err != nil {
		return warnings, err
	}

	return warnings, nil
}
