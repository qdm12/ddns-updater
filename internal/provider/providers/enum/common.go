package enum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) doRequest(ctx context.Context, client *http.Client, method string,
	requestData any,
) (response *http.Response, err error) {
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)

	err = encoder.Encode(requestData)
	if err != nil {
		return nil, fmt.Errorf("json encoding request data: %w", err)
	}

	url := "https://api.enum.co/enum.api.v1.DnsService/" + method

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buffer)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}

	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.apiKey)

	return client.Do(request)
}

func (p *Provider) recordName() string {
	return strings.ToLower(utils.BuildURLQueryHostname(p.owner, p.domain))
}

func handleErrorResponse(response *http.Response) (err error) {
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("%w: %d: unable to read response body: %w",
			errors.ErrHTTPStatusNotValid, response.StatusCode, err)
	}

	message := utils.ToSingleLine(string(data))

	var parsed struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	err = json.Unmarshal(data, &parsed)
	if err == nil && parsed.Code != "" {
		message = parsed.Code + ": " + parsed.Message
	}

	switch response.StatusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, message)
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", errors.ErrAuth, message)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", errors.ErrRateLimit, message)
	default:
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid, response.StatusCode, message)
	}
}
