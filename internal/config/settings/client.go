package settings

import (
	"time"

	"github.com/qdm12/gosettings"
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
