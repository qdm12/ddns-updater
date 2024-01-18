package config

import (
	"time"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gotree"
)

type Client struct {
	Timeout time.Duration
}

func (c *Client) setDefaults() {
	const defaultTimeout = 20 * time.Second
	c.Timeout = gosettings.DefaultComparable(c.Timeout, defaultTimeout)
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

func (c *Client) read(reader *reader.Reader) (err error) {
	c.Timeout, err = reader.Duration("HTTP_TIMEOUT")
	if err != nil {
		return err
	}

	return nil
}
