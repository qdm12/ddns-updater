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
	// Keep is the number of backups to keep, where zero keeps all backups.
	Keep *uint
}

func (b *Backup) setDefaults() {
	b.Period = gosettings.DefaultPointer(b.Period, 0)
	b.Directory = gosettings.DefaultPointer(b.Directory, "./data")
	b.Keep = gosettings.DefaultPointer(b.Keep, 0)
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
	if *b.Keep == 0 {
		node.Appendf("Keep: all backups")
	} else {
		node.Appendf("Keep: %d backups", *b.Keep)
	}
	return node
}

func (b *Backup) read(reader *reader.Reader) (err error) {
	b.Period, err = reader.DurationPtr("BACKUP_PERIOD")
	if err != nil {
		return err
	}

	b.Directory = reader.Get("BACKUP_DIRECTORY")

	b.Keep, err = reader.UintPtr("BACKUP_KEEP")
	return err
}
