package noop

import "context"

type Service struct {
	name string
}

func New(name string) *Service {
	return &Service{
		name: name,
	}
}

func (s *Service) String() string {
	return s.name + " (no-op)"
}

func (s *Service) Start(_ context.Context) (_ <-chan error, _ error) {
	return nil, nil //nolint:nilnil
}

func (s *Service) Stop() (stopErr error) {
	return nil
}
