package config

import (
	"fmt"
	"time"

	"github.com/qdm12/golibs/params"
)

type Client struct {
	Timeout time.Duration
}

func (c *Client) get(env params.Interface) (err error) {
	c.Timeout, err = env.Duration("HTTP_TIMEOUT", params.Default("10s"))
	if err != nil {
		return fmt.Errorf("%w: for environment variable HTTP_TIMEOUT", err)
	}

	return nil
}
