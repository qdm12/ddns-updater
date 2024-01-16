package config

import (
	"time"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Backup struct {
	Period    *time.Duration
	Directory *string
}

func (b *Backup) setDefaults() {
	b.Period = gosettings.DefaultPointer(b.Period, 0)
	b.Directory = gosettings.DefaultPointer(b.Directory, "./data")
}

func (b Backup) Validate() (err error) {
	return nil
}

func (b Backup) String() string {
	return b.toLinesNode().String()
}

func (b Backup) toLinesNode() *gotree.Node {
	if *b.Period == 0 {
		return gotree.New("Backup: disabled")
	}
	node := gotree.New("Backup")
	node.Appendf("Period: %s", b.Period)
	node.Appendf("Directory: %s", *b.Directory)
	return node
}

func (b *Backup) read(reader *reader.Reader) (err error) {
	b.Period, err = reader.DurationPtr("BACKUP_PERIOD")
	if err != nil {
		return err
	}

	b.Directory = reader.Get("BACKUP_DIRECTORY")
	return nil
}
