package config

import (
	"strconv"
	"time"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Update struct {
	Period   time.Duration
	Cooldown time.Duration
}

func (u *Update) setDefaults() {
	const defaultPeriod = 10 * time.Minute
	u.Period = gosettings.DefaultComparable(u.Period, defaultPeriod)
	const defaultCooldown = 5 * time.Minute
	u.Cooldown = gosettings.DefaultComparable(u.Cooldown, defaultCooldown)
}

func (u Update) Validate() (err error) {
	return nil
}

func (u Update) String() string {
	return u.toLinesNode().String()
}

func (u Update) toLinesNode() *gotree.Node {
	node := gotree.New("Update")
	node.Appendf("Period: %s", u.Period)
	node.Appendf("Cooldown: %s", u.Cooldown)
	return node
}

func (u *Update) read(reader *reader.Reader, warner Warner) (err error) {
	u.Period, err = readUpdatePeriod(reader, warner)
	if err != nil {
		return err
	}

	u.Cooldown, err = reader.Duration("UPDATE_COOLDOWN_PERIOD")
	return err
}

func readUpdatePeriod(r *reader.Reader, warner Warner) (period time.Duration, err error) {
	// Retro-compatibility: DELAY variable name
	delayStringPtr := r.Get("DELAY")
	if delayStringPtr != nil {
		handleDeprecated(warner, "DELAY", "PERIOD")
		// Retro-compatibility: integer only, treated as seconds
		delayInt, err := strconv.Atoi(*delayStringPtr)
		if err == nil {
			return time.Duration(delayInt) * time.Second, nil
		}

		return time.ParseDuration(*delayStringPtr)
	}

	return r.Duration("PERIOD")
}
