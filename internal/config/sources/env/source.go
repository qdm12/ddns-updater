package env

import (
	"fmt"
	"os"

	"github.com/qdm12/ddns-updater/internal/config/settings"
	"github.com/qdm12/gosettings/sources/env"
)

type Source struct {
	env              env.Env
	handleDeprecated func(deprecatedKey, currentKey string)
}

func New(warner Warner) *Source {
	handleDeprecated := func(deprecatedKey, currentKey string) {
		warner.Warnf("You are using an old environment variable %s, please change it to %s",
			deprecatedKey, currentKey)
	}
	return &Source{
		env:              *env.New(os.Environ(), handleDeprecated),
		handleDeprecated: handleDeprecated,
	}
}

func (s *Source) Read() (settings settings.Settings, err error) {
	settings.Client, err = s.readClient()
	if err != nil {
		return settings, fmt.Errorf("reading client settings: %w", err)
	}

	settings.Update, err = s.readUpdate()
	if err != nil {
		return settings, fmt.Errorf("reading update settings: %w", err)
	}

	settings.PubIP, err = s.readPubIP()
	if err != nil {
		return settings, fmt.Errorf("reading public IP settings: %w", err)
	}

	settings.Resolver, err = s.readResolver()
	if err != nil {
		return settings, fmt.Errorf("reading resolver settings: %w", err)
	}

	settings.IPv6, err = s.readIPv6()
	if err != nil {
		return settings, fmt.Errorf("reading IPv6 settings: %w", err)
	}

	settings.Server, err = s.readServer()
	if err != nil {
		return settings, fmt.Errorf("reading server settings: %w", err)
	}

	settings.Health = s.ReadHealth()
	settings.Paths = s.readPaths()

	settings.Backup, err = s.readBackup()
	if err != nil {
		return settings, fmt.Errorf("reading backup settings: %w", err)
	}

	settings.Logger, err = s.readLogger()
	if err != nil {
		return settings, fmt.Errorf("reading logger settings: %w", err)
	}

	settings.Shoutrrr, err = s.readShoutrrr()
	if err != nil {
		return settings, fmt.Errorf("reading shoutrrr settings: %w", err)
	}

	return settings, nil
}
