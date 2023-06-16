package shoutrrr

import (
	"fmt"

	"github.com/containrrr/shoutrrr"
	"github.com/qdm12/gosettings"
	"github.com/qdm12/gotree"
)

type Settings struct {
	Addresses    []string
	DefaultTitle string
	Logger       Erroer
}

func (s *Settings) SetDefaults() {
	s.Addresses = gosettings.DefaultSlice(s.Addresses, []string{})
	s.DefaultTitle = gosettings.DefaultString(s.DefaultTitle, "DDNS Updater")
	s.Logger = gosettings.DefaultInterface(s.Logger, &noopLogger{})
}

func (s Settings) MergeWith(other Settings) (merged Settings) {
	merged.Addresses = gosettings.MergeWithSlice(s.Addresses, other.Addresses)
	merged.DefaultTitle = gosettings.MergeWithString(s.DefaultTitle, other.DefaultTitle)
	merged.Logger = gosettings.MergeWithInterface(s.Logger, other.Logger)
	return merged
}

func (s Settings) Validate() (err error) {
	_, err = shoutrrr.CreateSender(s.Addresses...)
	if err != nil {
		return fmt.Errorf("shoutrrr addresses: %w", err)
	}
	return nil
}

func (s Settings) String() string {
	return s.ToLinesNode().String()
}

func (s Settings) ToLinesNode() *gotree.Node {
	if len(s.Addresses) == 0 {
		return nil // no address means shoutrrr is disabled
	}

	node := gotree.New("Shoutrrr")
	node.Appendf("Default title: %s", s.DefaultTitle)

	childNode := node.Appendf("Addresses")
	for _, address := range s.Addresses {
		childNode.Appendf(address)
	}

	return node
}

type noopLogger struct{}

func (l noopLogger) Error(_ string) {}
