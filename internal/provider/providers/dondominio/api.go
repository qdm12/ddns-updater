package dondominio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

func apiCall(ctx context.Context, client *http.Client,
	path string, requestData any) (responseData json.RawMessage, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "simple-api.dondominio.net",
		Path:   path,
	}

	body := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(body)
	err = encoder.Encode(requestData)
	if err != nil {
		return nil, fmt.Errorf("encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	var data struct {
		Success      bool            `json:"success"`
		ErrorCode    int             `json:"errorCode"`
		ErrorCodeMsg string          `json:"errorCodeMsg"`
		ResponseData json.RawMessage `json:"responseData"`
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&data)
	if err != nil {
		_ = response.Body.Close()
		return nil, fmt.Errorf("decoding response body: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("closing response body: %w", err)
	}

	if !data.Success {
		_ = response.Body.Close()
		return nil, makeError(data.ErrorCode, data.ErrorCodeMsg)
	}

	return data.ResponseData, nil
}
