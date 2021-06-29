package config

import (
	"time"

	"github.com/qdm12/golibs/params"
)

type Client struct {
	Timeout time.Duration
}

func (c *Client) get(env params.Env) (err error) {
	c.Timeout, err = env.Duration("HTTP_TIMEOUT", params.Default("10s"))
	return err
}
