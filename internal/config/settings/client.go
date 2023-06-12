package settings

import (
	"time"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gotree"
)

type Client struct {
	Timeout time.Duration
}

func (c *Client) setDefaults() {
	const defaultTimeout = 10 * time.Second
	c.Timeout = gosettings.DefaultNumber(c.Timeout, defaultTimeout)
}

func (c Client) mergeWith(other Client) (merged Client) {
	merged.Timeout = gosettings.MergeWithNumber(c.Timeout, other.Timeout)
	return merged
}

func (c Client) Validate() (err error) {
	return nil
}

func (c Client) String() string {
	return c.toLinesNode().String()
}

func (c Client) toLinesNode() *gotree.Node {
	node := gotree.New("HTTP client")
	node.Appendf("Timeout: %s", c.Timeout)
	return node
}
