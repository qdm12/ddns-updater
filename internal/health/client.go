package health

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

func IsClientMode(args []string) bool {
	return len(args) > 1 && args[1] == "healthcheck"
}

type Client interface {
	Query(ctx context.Context) error
}

type client struct {
	*http.Client
}

func NewClient() Client {
	const timeout = 5 * time.Second
	return &client{
		Client: &http.Client{Timeout: timeout},
	}
}

// Query sends an HTTP request to the other instance of
// the program, and to its internal healthcheck server.
func (c *client) Query(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:9999", nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	} else if resp.StatusCode == http.StatusOK {
		return nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return fmt.Errorf("%s: %s", resp.Status, err)
	}
	return fmt.Errorf("%s (%s)", string(b), err)
}
