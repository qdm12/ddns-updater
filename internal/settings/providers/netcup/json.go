package netcup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"golang.org/x/net/context"
)

func doJSONHTTP(ctx context.Context, client *http.Client,
	jsonRequestBody, jsonResponseBody any) (err error) {
	endpointURL := url.URL{
		Scheme:   "https",
		Host:     "ccp.netcup.net",
		Path:     "/run/webservice/servers/endpoint.php",
		RawQuery: "JSON",
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(jsonRequestBody)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrRequestEncode, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL.String(), buffer)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}
	headers.SetUserAgent(request)

	httpResponse, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnsuccessfulResponse, err)
	}

	responseBytes, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		_ = httpResponse.Body.Close()
		return fmt.Errorf("reading response body data: %w", err)
	}

	err = httpResponse.Body.Close()
	if err != nil {
		return fmt.Errorf("closing response body: %w", err)
	}

	var commonResponse struct {
		ShortMessage string `json:"shortmessage"`
		Status       string `json:"status"`
		StatusCode   uint   `json:"statuscode"`
	}
	err = json.Unmarshal(responseBytes, &commonResponse)
	if err != nil {
		return fmt.Errorf("json decoding common response: %w", err)
	}

	if commonResponse.Status == "error" {
		return fmt.Errorf("%w: %s (status %d)", errors.ErrBadHTTPStatus,
			commonResponse.ShortMessage, commonResponse.StatusCode)
	}

	err = json.Unmarshal(responseBytes, jsonResponseBody)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	return nil
}
