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
		providersIP4: []Provider{Ipify},
		providersIP6: []Provider{Ipify},
		timeout:      defaultTimeout,
	}
}

type Option func(s *settings) error

func SetProvidersIP(first Provider, providers ...Provider) Option {
	providers = append(providers, first)
	return func(s *settings) (err error) {
		for _, provider := range providers {
			err = ValidateProvider(provider, ipversion.IP4or6)
			if err != nil {
				return err
			}
		}
		s.providersIP = providers
		return nil
	}
}

func SetProvidersIP4(first Provider, providers ...Provider) Option {
	providers = append(providers, first)
	return func(s *settings) (err error) {
		for _, provider := range providers {
			err = ValidateProvider(provider, ipversion.IP4)
			if err != nil {
				return err
			}
		}
		s.providersIP4 = providers
		return nil
	}
}

func SetProvidersIP6(first Provider, providers ...Provider) Option {
	providers = append(providers, first)
	return func(s *settings) (err error) {
		for _, provider := range providers {
			err = ValidateProvider(provider, ipversion.IP6)
			if err != nil {
				return err
			}
		}
		s.providersIP6 = providers
		return nil
	}
}

func SetTimeout(timeout time.Duration) Option {
	return func(s *settings) (err error) {
		s.timeout = timeout
		return nil
	}
}
