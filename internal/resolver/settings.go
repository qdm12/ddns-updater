package resolver

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/validate"
)

type Settings struct {
	Address *string
	Timeout time.Duration
}

func (s *Settings) SetDefaults() {
	s.Address = gosettings.DefaultPointer(s.Address, "")
	const defaultTimeout = 5 * time.Second
	s.Timeout = gosettings.DefaultNumber(s.Timeout, defaultTimeout)
}

func (s Settings) MergeWith(other Settings) (merged Settings) {
	merged.Address = gosettings.MergeWithPointer(s.Address, other.Address)
	merged.Timeout = gosettings.MergeWithNumber(s.Timeout, other.Timeout)
	return merged
}

var (
	ErrTimeoutTooLow = errors.New("timeout is too low")
)

func (s Settings) Validate() (err error) {
	if *s.Address != "" {
		err = validate.ListeningAddress(*s.Address, os.Getuid())
		if err != nil {
			return fmt.Errorf("splitting host and port from address: %w", err)
		}
	}

	const minTimeout = 10 * time.Millisecond
	if s.Timeout < minTimeout {
		return fmt.Errorf("%w: %s is below the minimum %s",
			ErrTimeoutTooLow, s.Timeout, minTimeout)
	}

	return nil
}
