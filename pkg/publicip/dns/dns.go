package dns

import "time"

type Fetcher struct {
	ring    ring
	timeout time.Duration
}

type ring struct {
	// counter is used to get an index in the providers slice
	counter   *uint32 // uint32 for 32 bit systems atomic operations
	providers []Provider
}

func New(options ...Option) (f *Fetcher, err error) {
	settings := newDefaultSettings()
	for _, option := range options {
		err = option(&settings)
		if err != nil {
			return nil, err
		}
	}

	return &Fetcher{
		ring: ring{
			counter:   new(uint32),
			providers: settings.providers,
		},
		timeout: settings.timeout,
	}, nil
}
