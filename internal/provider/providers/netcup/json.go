package netcup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

func doJSONHTTP(ctx context.Context, client *http.Client,
	jsonRequestBody, jsonResponseDataTarget any) (err error) {
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
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL.String(), buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")

	httpResponse, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}

	var commonResponse struct {
		ShortMessage string          `json:"shortmessage"`
		Status       string          `json:"status"`
		StatusCode   uint            `json:"statuscode"`
		ResponseData json.RawMessage `json:"responsedata"`
	}

	decoder := json.NewDecoder(httpResponse.Body)
	err = decoder.Decode(&commonResponse)
	if err != nil {
		_ = httpResponse.Body.Close()
		return fmt.Errorf("decoding json common response: %w", err)
	}

	err = httpResponse.Body.Close()
	if err != nil {
		return fmt.Errorf("closing response body: %w", err)
	}

	if commonResponse.Status == "error" {
		return fmt.Errorf("%w: %s (status %d)", errors.ErrHTTPStatusNotValid,
			commonResponse.ShortMessage, commonResponse.StatusCode)
	}

	err = json.Unmarshal(commonResponse.ResponseData, jsonResponseDataTarget)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	return nil
}
