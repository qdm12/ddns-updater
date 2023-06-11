package settings

import (
	"time"

	"github.com/qdm12/gosettings"
)

type Update struct {
	Period   time.Duration
	Cooldown time.Duration
}

func (u *Update) setDefaults() {
	const defaultPeriod = 10 * time.Minute
	u.Period = gosettings.DefaultNumber(u.Period, defaultPeriod)
	const defaultCooldown = 5 * time.Minute
	u.Cooldown = gosettings.DefaultNumber(u.Cooldown, defaultCooldown)
}

func (u Update) mergeWith(other Update) (merged Update) {
	merged.Period = gosettings.MergeWithNumber(u.Period, other.Period)
	merged.Cooldown = gosettings.MergeWithNumber(u.Cooldown, other.Cooldown)
	return merged
}

func (u Update) Validate() (err error) {
	return nil
}
