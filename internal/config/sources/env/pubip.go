package env

import (
	"errors"
	"fmt"
	"strings"

	"github.com/qdm12/ddns-updater/internal/config/settings"
	"github.com/qdm12/gosettings/sources/env"
)

func (s *Source) readPubIP() (settings settings.PubIP, err error) {
	settings.HTTPEnabled, settings.DNSEnabled, err = getFetchers(s.env)
	if err != nil {
		return settings, err
	}

	settings.HTTPIPProviders = s.env.CSV("PUBLICIP_HTTP_PROVIDERS",
		env.RetroKeys("IP_METHOD"))
	settings.HTTPIPv4Providers = s.env.CSV("PUBLICIPV4_HTTP_PROVIDERS",
		env.RetroKeys("IPV4_METHOD"))
	settings.HTTPIPv6Providers = s.env.CSV("PUBLICIPV6_HTTP_PROVIDERS",
		env.RetroKeys("IPV6_METHOD"))

	// Retro-compatibility
	for i := range settings.HTTPIPProviders {
		settings.HTTPIPProviders[i] = handleRetroProvider(settings.HTTPIPProviders[i])
	}
	for i := range settings.HTTPIPv4Providers {
		settings.HTTPIPv4Providers[i] = handleRetroProvider(settings.HTTPIPv4Providers[i])
	}
	for i := range settings.HTTPIPv6Providers {
		settings.HTTPIPv6Providers[i] = handleRetroProvider(settings.HTTPIPv6Providers[i])
	}

	// Retro-compatibility for now defunct opendns http provider for ipv4 or ipv6
	if len(settings.HTTPIPProviders) > 0 { // check to avoid transforming `nil` to `[]`
		httpIPProvidersTemp := make([]string, len(settings.HTTPIPProviders))
		copy(httpIPProvidersTemp, settings.HTTPIPProviders)
		settings.HTTPIPProviders = make([]string, 0, len(settings.HTTPIPProviders))
		for _, provider := range httpIPProvidersTemp {
			if provider != "opendns" {
				settings.HTTPIPProviders = append(settings.HTTPIPProviders, provider)
			}
		}
	}

	settings.DNSProviders = s.env.CSV("PUBLICIP_DNS_PROVIDERS")
	settings.DNSTimeout, err = s.env.Duration("PUBLICIP_DNS_TIMEOUT")
	if err != nil {
		return settings, err
	}

	return settings, nil
}

var ErrInvalidFetcher = errors.New("invalid fetcher specified")

func getFetchers(env env.Env) (http, dns *bool, err error) {
	// TODO change to use env.BoolPtr with retro-compatibility
	s := env.String("PUBLICIP_FETCHERS")
	if s == "" {
		return nil, nil, nil
	}

	http, dns = new(bool), new(bool)
	fields := strings.Split(s, ",")
	for i, field := range fields {
		switch strings.ToLower(field) {
		case "all":
			*http = true
			*dns = true
		case "http":
			*http = true
		case "dns":
			*dns = true
		default:
			return nil, nil, fmt.Errorf(
				"%w: %q at position %d of %d",
				ErrInvalidFetcher, field, i+1, len(fields))
		}
	}

	return http, dns, nil
}

func handleRetroProvider(provider string) (updatedProvider string) {
	switch provider {
	case "ipify6":
		return "ipify"
	case "noip4", "noip6", "noip8245_4", "noip8245_6":
		return "noip"
	case "cycle":
		return "all"
	default:
		return provider
	}
}
