package network

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// BuildHTTPPut is used for GoDaddy and Cloudflare only.
func BuildHTTPPut(ctx context.Context, url string, body interface{}) (request *http.Request, err error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	request, err = http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return request, nil
}
