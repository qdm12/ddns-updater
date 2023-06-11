package settings

import (
	"time"

	"github.com/qdm12/gosettings"
)

type Backup struct {
	Period    *time.Duration
	Directory *string
}

func (b *Backup) setDefaults() {
	b.Period = gosettings.DefaultPointer(b.Period, 0)
	b.Directory = gosettings.DefaultPointer(b.Directory, "./data")
}

func (b Backup) mergeWith(other Backup) (merged Backup) {
	merged.Period = gosettings.MergeWithPointer(b.Period, other.Period)
	merged.Directory = gosettings.MergeWithPointer(b.Directory, other.Directory)
	return merged
}

func (b Backup) Validate() (err error) {
	return nil
}
