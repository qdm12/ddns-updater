package resolver

import (
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

func (s Settings) validate() (err error) {
	if *s.Address != "" {
		host, port, err := net.SplitHostPort(*s.Address)
		if err != nil {
			return fmt.Errorf("splitting host and port from address: %w", err)
		}

		switch {
		case host == "":
			return fmt.Errorf("address host is empty: in %s", *s.Address)
		case port == "":
			return fmt.Errorf("address port is empty: in %s", *s.Address)
		}
	}

	const minTimeout = 10 * time.Millisecond
	if s.Timeout < minTimeout {
		return fmt.Errorf("timeout is too low: %s is below the minimum %s", s.Timeout, minTimeout)
	}

	return nil
}
