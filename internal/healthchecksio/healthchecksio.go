package healthchecksio

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// New creates a new healthchecks.io client.
// If passed an empty uuid string, it acts as no-op implementation.
func New(httpClient *http.Client, baseURL, uuid string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		uuid:       uuid,
	}
}

type Client struct {
	httpClient *http.Client
	baseURL    string
	uuid       string
}

var (
	ErrStatusCode = errors.New("bad status code")
)

type State string

const (
	Ok    State = "ok"
	Start State = "start"
	Fail  State = "fail"
	Exit0 State = "0"
	Exit1 State = "1"
)

func (c *Client) Ping(ctx context.Context, state State) (err error) {
	if c.uuid == "" {
		return nil
	}

	url := c.baseURL + "/" + c.uuid
	if state != Ok {
		url += "/" + string(state)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
