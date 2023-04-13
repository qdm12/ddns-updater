package info

type Option func(s *settings) error

func SetProviders(first Provider, providers ...Provider) Option {
	return func(s *settings) (err error) {
		providers = append(providers, first)
		for _, provider := range providers {
			err = ValidateProvider(provider)
			if err != nil {
				return err
			}
		}
		s.providers = providers
		return nil
	}
}
