package resolver

import (
	"errors"
	"fmt"
	"net"
	"strings"
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
	ErrAddressHostEmpty  = errors.New("address host is empty")
	ErrTimeoutTooLow     = errors.New("timeout is too low")
	ErrUnsupportedScheme = errors.New("address scheme is not supported; supported schemes are udp://, tcp:// and tls://")
)

// stripScheme removes a known transport scheme prefix (udp://, tcp://,
// tls://) from the address and reports whether it was present. An
// unknown scheme prefix makes it return an error instead, so that
// malformed addresses fail fast with an actionable message rather than
// being silently mis-parsed downstream.
func stripScheme(address string) (scheme string, rest string, err error) {
	i := strings.Index(address, "://")
	if i < 0 {
		return "", address, nil
	}
	candidate := address[:i]
	switch candidate {
	case "udp", "tcp", "tls":
		return candidate, strings.TrimPrefix(address[i:], "://"), nil
	default:
		return "", "", fmt.Errorf("%w: got %q", ErrUnsupportedScheme, candidate+"://")
	}
}

func (s Settings) validate() (err error) {
	if *s.Address != "" {
		_, rest, schemeErr := stripScheme(*s.Address)
		if schemeErr != nil {
			return schemeErr
		}

		host, _, err := net.SplitHostPort(rest)
		if err != nil {
			host = rest // no port specified: New applies the transport default
		}

		if host == "" {
			return fmt.Errorf("%w: in %s", ErrAddressHostEmpty, *s.Address)
		}
	}

	const minTimeout = 10 * time.Millisecond
	if s.Timeout < minTimeout {
		return fmt.Errorf("%w: %s is below the minimum %s",
			ErrTimeoutTooLow, s.Timeout, minTimeout)
	}

	return nil
}
