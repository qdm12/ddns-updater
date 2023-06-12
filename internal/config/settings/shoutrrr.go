package settings

import (
	"fmt"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/qdm12/gosettings"
	"github.com/qdm12/gotree"
)

type Shoutrrr struct {
	Addresses []string
	Params    types.Params
}

func (s *Shoutrrr) setDefaults() {
	s.Addresses = gosettings.DefaultSlice(s.Addresses, []string{})
	if s.Params == nil {
		s.Params = types.Params{
			"title": "DDNS Updater",
		}
	}
}

func (s Shoutrrr) mergeWith(other Shoutrrr) (merged Shoutrrr) {
	merged.Addresses = gosettings.MergeWithSlice(s.Addresses, other.Addresses)
	if s.Params == nil {
		merged.Params = other.Params
	}
	return merged
}

func (s Shoutrrr) Validate() (err error) {
	_, err = shoutrrr.CreateSender(s.Addresses...)
	if err != nil {
		return fmt.Errorf("shoutrrr addresses: %w", err)
	}
	return nil
}

func (s Shoutrrr) String() string {
	return s.toLinesNode().String()
}

func (s Shoutrrr) toLinesNode() *gotree.Node {
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
