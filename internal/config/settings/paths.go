package settings

import (
	"github.com/qdm12/gosettings"
)

type Paths struct {
	DataDir *string
}

func (p *Paths) setDefaults() {
	p.DataDir = gosettings.DefaultPointer(p.DataDir, "./data")
}

func (p Paths) mergeWith(other Paths) (merged Paths) {
	merged.DataDir = gosettings.MergeWithPointer(p.DataDir, other.DataDir)
	return merged
}

func (p Paths) Validate() (err error) {
	return nil
}
