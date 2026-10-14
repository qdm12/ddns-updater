package config

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Paths struct {
	DataDir       *string
	Config        *string
	ConfigPersist *bool
	// Umask is the custom umask to use for the system, if different than zero.
	// If it is set to zero, the system umask is unchanged.
	// It cannot be nil in the internal state.
	Umask *fs.FileMode

	// environmentConfig is true if the CONFIG environment variable is set.
	environmentConfig bool
}

// configFileHasSettings reports whether the config file holds the provider
// settings. This is always the case when the settings are read from the file,
// and only the case with CONFIG_PERSIST enabled when they come from CONFIG.
func (p Paths) configFileHasSettings() bool {
	return !p.environmentConfig || *p.ConfigPersist
}

func (p *Paths) setDefaults() {
	p.DataDir = gosettings.DefaultPointer(p.DataDir, "./data")
	defaultConfig := filepath.Join(*p.DataDir, "config.json")
	p.Config = gosettings.DefaultPointer(p.Config, defaultConfig)
	p.ConfigPersist = gosettings.DefaultPointer(p.ConfigPersist, true)
	p.Umask = gosettings.DefaultPointer(p.Umask, fs.FileMode(0))
}

func (p Paths) Validate() (err error) {
	return nil
}

func (p Paths) String() string {
	return p.toLinesNode().String()
}

func (p Paths) toLinesNode() *gotree.Node {
	node := gotree.New("Paths")
	node.Appendf("Data directory: %s", *p.DataDir)
	node.Appendf("Config file: %s", *p.Config)
	node.Appendf("Persist environment config: %t", *p.ConfigPersist)
	umaskString := "system default"
	if *p.Umask != 0 {
		umaskString = p.Umask.String()
	}
	node.Appendf("Umask: %s", umaskString)
	return node
}

func (p *Paths) read(reader *reader.Reader) (err error) {
	p.DataDir = reader.Get("DATADIR")
	p.Config = reader.Get("CONFIG_FILEPATH")
	p.ConfigPersist, err = reader.BoolPtr("CONFIG_PERSIST")
	if err != nil {
		return err
	}
	environmentConfig := reader.Get("CONFIG")
	p.environmentConfig = environmentConfig != nil && *environmentConfig != ""

	umaskString := reader.String("UMASK")
	if umaskString != "" {
		umask, err := parseUmask(umaskString)
		if err != nil {
			return fmt.Errorf("parse umask: %w", err)
		}
		p.Umask = &umask
	}

	return nil
}

func parseUmask(s string) (umask fs.FileMode, err error) {
	const base, bitSize = 8, 32
	umaskUint64, err := strconv.ParseUint(s, base, bitSize)
	if err != nil {
		return 0, err
	}
	return fs.FileMode(umaskUint64), nil
}
