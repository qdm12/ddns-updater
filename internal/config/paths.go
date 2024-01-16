package config

import (
	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Paths struct {
	DataDir *string
}

func (p *Paths) setDefaults() {
	p.DataDir = gosettings.DefaultPointer(p.DataDir, "./data")
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
	return node
}

func (p *Paths) read(reader *reader.Reader) {
	p.DataDir = reader.Get("DATADIR")
}
