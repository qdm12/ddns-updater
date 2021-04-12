package info

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
