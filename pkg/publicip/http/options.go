package http

import (
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type settings struct {
	providersIP  []Provider
	providersIP4 []Provider
	providersIP6 []Provider
	timeout      time.Duration
}

func newDefaultSettings() settings {
	const defaultTimeout = 5 * time.Second
	return settings{
		providersIP:  []Provider{Google},
		providersIP4: []Provider{Noip},
		providersIP6: []Provider{Noip},
		timeout:      defaultTimeout,
	}
}

type Option func(s *settings) error

func SetProvidersIP(providers ...Provider) Option {
	return func(s *settings) error {
		for _, provider := range providers {
			if err := ValidateProvider(provider, ipversion.IP4or6); err != nil {
				return err
			}
		}
		s.providersIP = providers
		return nil
	}
}

func SetProvidersIP4(providers ...Provider) Option {
	return func(s *settings) error {
		for _, provider := range providers {
			if err := ValidateProvider(provider, ipversion.IP4); err != nil {
				return err
			}
		}
		s.providersIP4 = providers
		return nil
	}
}

func SetProvidersIP6(providers ...Provider) Option {
	return func(s *settings) error {
		for _, provider := range providers {
			if err := ValidateProvider(provider, ipversion.IP6); err != nil {
				return err
			}
		}
		s.providersIP6 = providers
		return nil
	}
}

func SetTimeout(timeout time.Duration) Option {
	return func(s *settings) error {
		s.timeout = timeout
		return nil
	}
}
