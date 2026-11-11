package health

import (
	"context"
	"fmt"
	"net/http"
)

func CheckHTTP(ctx context.Context, client *http.Client) (err error) {
	const url = "https://github.com"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	_ = response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status code is not OK: %d", response.StatusCode)
	}

	return nil
}
