package shoutrrr

import (
	"fmt"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/qdm12/gosettings"
	"github.com/qdm12/gotree"
)

type Settings struct {
	Addresses []string
	Params    types.Params
	Logger    Erroer
}

func (s *Settings) SetDefaults() {
	s.Addresses = gosettings.DefaultSlice(s.Addresses, []string{})
	if s.Params == nil {
		s.Params = types.Params{
			"title": "DDNS Updater",
		}
	}
	s.Logger = gosettings.DefaultInterface(s.Logger, &noopLogger{})
}

func (s Settings) MergeWith(other Settings) (merged Settings) {
	merged.Addresses = gosettings.MergeWithSlice(s.Addresses, other.Addresses)
	if s.Params == nil {
		merged.Params = other.Params
	}
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
	childNode := node.Appendf("Addresses")
	for _, address := range s.Addresses {
		childNode.Appendf(address)
	}

	childNode = node.Appendf("Parameters")
	for key, value := range s.Params {
		childNode.Appendf("%s=%s", key, value)
	}

	return node
}

type noopLogger struct{}

func (l noopLogger) Error(_ string) {}
