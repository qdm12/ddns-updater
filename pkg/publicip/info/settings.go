package info

type settings struct {
	providers []Provider
}

func (s *settings) setDefaults() {
	if len(s.providers) == 0 {
		s.providers = ListProviders()
	}
}
