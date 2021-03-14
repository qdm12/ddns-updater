package dns

import "time"

type settings struct {
	providers []Provider
	timeout   time.Duration
}

func newDefaultSettings() settings {
	return settings{
		providers: ListProviders(),
		timeout:   time.Second,
	}
}

type Option func(s *settings) error

func SetProviders(providers ...Provider) Option {
	return func(s *settings) error {
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
