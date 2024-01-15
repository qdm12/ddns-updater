package healthchecksio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// New creates a new healthchecks.io client.
// If passed an empty uuid string, it acts as no-op implementation.
func New(httpClient *http.Client, uuid string) *Client {
	return &Client{
		httpClient: httpClient,
		uuid:       uuid,
	}
}

type Client struct {
	httpClient *http.Client
	uuid       string
}

var (
	ErrStatusCode = errors.New("bad status code")
)

func (c *Client) Ping(ctx context.Context) (err error) {
	if c.uuid == "" {
		return nil
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://hc-ping.com/"+c.uuid, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}

	switch response.StatusCode {
	case http.StatusOK:
	default:
		return fmt.Errorf("%w: %d %s", ErrStatusCode, response.StatusCode, response.Status)
	}

	err = response.Body.Close()
	if err != nil {
		return fmt.Errorf("closing response body: %w", err)
	}

	return nil
}
