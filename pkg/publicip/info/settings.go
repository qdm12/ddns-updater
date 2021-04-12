package info

type settings struct {
	providers []Provider
}

func newDefaultSettings() settings {
	return settings{
		providers: ListProviders(),
	}
}
