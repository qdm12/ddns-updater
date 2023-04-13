package config

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/resolver"
	"github.com/qdm12/golibs/params"
)

type Config struct {
	Client   Client
	Update   Update
	PubIP    PubIP
	Resolver resolver.Settings
	IPv6     IPv6
	Server   Server
	Health   Health
	Paths    Paths
	Backup   Backup
	Logger   Logger
	Shoutrrr Shoutrrr
}

func (c *Config) Get(env params.Interface) (warnings []string, err error) {
	err = c.Client.get(env)
	if err != nil {
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

	c.Resolver, err = readResolver()
	if err != nil {
		return warnings, fmt.Errorf("reading resolver settings: %w", err)
	}

	err = c.IPv6.get(env)
	if err != nil {
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

	err = c.Paths.get(env)
	if err != nil {
		return warnings, err
	}

	err = c.Backup.get(env)
	if err != nil {
		return warnings, err
	}

	c.Logger, err = readLog()
	if err != nil {
		return warnings, err
	}

	newWarnings, err = c.Shoutrrr.get(env)
	warnings = append(warnings, newWarnings...)
	if err != nil {
		return warnings, err
	}

	return warnings, nil
}
