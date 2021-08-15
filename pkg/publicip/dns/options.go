package dns

import "time"

type settings struct {
	providers []Provider
	timeout   time.Duration
}

func newDefaultSettings() settings {
	const defaultTimeout = 3 * time.Second
	return settings{
		providers: ListProviders(),
		timeout:   defaultTimeout,
	}
}

type Option func(s *settings) error

func SetProviders(first Provider, providers ...Provider) Option {
	return func(s *settings) error {
		providers = append(providers, first)
		for _, provider := range providers {
			if err := ValidateProvider(provider); err != nil {
				return err
			}
		}
		s.providers = providers
		return nil
	}
}

func SetTimeout(timeout time.Duration) Option {
	return func(s *settings) error {
		s.timeout = timeout
		return nil
	}
}
