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
	DataDir *string
	Config  *string
	Umask   fs.FileMode
}

func (p *Paths) setDefaults() {
	p.DataDir = gosettings.DefaultPointer(p.DataDir, "./data")
	defaultConfig := filepath.Join(*p.DataDir, "config.json")
	p.Config = gosettings.DefaultPointer(p.Config, defaultConfig)
	p.Umask = gosettings.DefaultComparable(p.Umask, getCurrentUmask())
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
	node.Appendf("Umask: %s", p.Umask.String())
	return node
}

func (p *Paths) read(reader *reader.Reader) (err error) {
	p.DataDir = reader.Get("DATADIR")
	p.Config = reader.Get("CONFIG_FILEPATH")

	umaskString := reader.String("UMASK")
	if umaskString != "" {
		p.Umask, err = parseUmask(umaskString)
		if err != nil {
			return fmt.Errorf("parse umask: %w", err)
		}
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
