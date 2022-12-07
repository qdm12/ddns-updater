package config

import (
	"fmt"
	"os"
	"time"

	"github.com/qdm12/ddns-updater/internal/resolver"
)

func readResolver() (settings resolver.Settings, err error) {
	address := os.Getenv("RESOLVER_ADDRESS")
	if address != "" {
		settings.Address = &address
	}

	timeoutString := os.Getenv("RESOLVER_TIMEOUT")
	if timeoutString != "" {
		timeout, err := time.ParseDuration(timeoutString)
		if err != nil {
			return settings, fmt.Errorf("environment variable RESOLVER_TIMEOUT: %w", err)
		}
		settings.Timeout = timeout
	}

	return settings, nil
}
