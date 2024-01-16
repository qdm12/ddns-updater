package resolver

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/qdm12/gosettings"
)

type Settings struct {
	Address *string
	Timeout time.Duration
}

func (s *Settings) setDefaults() {
	s.Address = gosettings.DefaultPointer(s.Address, "")
	const defaultTimeout = 5 * time.Second
	s.Timeout = gosettings.DefaultComparable(s.Timeout, defaultTimeout)
}

var (
	ErrAddressHostEmpty = errors.New("address host is empty")
	ErrAddressPortEmpty = errors.New("address port is empty")
	ErrTimeoutTooLow    = errors.New("timeout is too low")
)

func (s Settings) validate() (err error) {
	if *s.Address != "" {
		host, port, err := net.SplitHostPort(*s.Address)
		if err != nil {
			return fmt.Errorf("splitting host and port from address: %w", err)
		}

		switch {
		case host == "":
			return fmt.Errorf("%w: in %s", ErrAddressHostEmpty, *s.Address)
		case port == "":
			return fmt.Errorf("%w: in %s", ErrAddressPortEmpty, *s.Address)
		}
	}

	const minTimeout = 10 * time.Millisecond
	if s.Timeout < minTimeout {
		return fmt.Errorf("%w: %s is below the minimum %s",
			ErrTimeoutTooLow, s.Timeout, minTimeout)
	}

	return nil
}
