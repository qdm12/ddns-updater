package settings

import (
	"fmt"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/qdm12/gosettings"
)

type Shoutrrr struct {
	Addresses []string
	Params    types.Params
}

func (s *Shoutrrr) setDefaults() {
	s.Addresses = []string{}
	s.Params = types.Params{
		"title": "DDNS Updater",
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
