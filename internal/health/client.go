package health

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

func IsClientMode(args []string) bool {
	return len(args) > 1 && args[1] == "healthcheck"
}

type Client interface {
	Query(ctx context.Context, port uint16) error
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

var ErrParseHealthServerAddress = errors.New("cannot parse health server address")

// Query sends an HTTP request to the other instance of
// the program, and to its internal healthcheck server.
func (c *client) Query(ctx context.Context, port uint16) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+strconv.Itoa(int(port)), nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %s", resp.Status, err)
	}

	return fmt.Errorf(string(b))
}
