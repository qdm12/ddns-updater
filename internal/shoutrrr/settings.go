package shoutrrr

import (
	"fmt"

	"github.com/containrrr/shoutrrr"
	"github.com/qdm12/gosettings"
)

type Settings struct {
	Addresses    []string
	DefaultTitle string
	Logger       Erroer
}

func (s *Settings) setDefaults() {
	s.Addresses = gosettings.DefaultSlice(s.Addresses, []string{})
	s.DefaultTitle = gosettings.DefaultComparable(s.DefaultTitle, "DDNS Updater")
	s.Logger = gosettings.DefaultComparable[Erroer](s.Logger, &noopLogger{})
}

func (s Settings) validate() (err error) {
	_, err = shoutrrr.CreateSender(s.Addresses...)
	if err != nil {
		return fmt.Errorf("shoutrrr addresses: %w", err)
	}
	return nil
}

type noopLogger struct{}

func (l noopLogger) Error(_ string) {}
