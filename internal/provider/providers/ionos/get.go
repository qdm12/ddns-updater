package ionos

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) getZones(ctx context.Context, client *http.Client) (
	zones []apiZone, err error) {
	err = p.get(ctx, client, "/zones", nil, &zones)
	return zones, err
}

func (p *Provider) getRecords(ctx context.Context, client *http.Client,
	zoneID string, recordType string) (records []apiRecord, err error) {
	queryParams := url.Values{
		"recordName": []string{p.BuildDomainName()},
		"recordType": []string{recordType},
	}
	var responseData struct {
		Records []apiRecord `json:"records"`
	}
	err = p.get(ctx, client, "/zones/"+zoneID, queryParams, &responseData)
	if err != nil {
		return nil, fmt.Errorf("for zone id %s and type %s: %w",
			zoneID, recordType, err)
	}

	return responseData.Records, nil
}

func (p *Provider) get(ctx context.Context, client *http.Client,
	subPath string, queryParams url.Values, responseJSONData any) (err error) {
	u := url.URL{
		Scheme:   "https",
		Host:     "api.hosting.ionos.com",
		Path:     filepath.Join("/dns/v1/", subPath),
		RawQuery: queryParams.Encode(),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", errors.ErrAuth,
			decodeErrorMessage(response.Body))
	default:
		return fmt.Errorf("%w: %s: %s", errors.ErrHTTPStatusNotValid,
			response.Status, decodeErrorMessage(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&responseJSONData)
	if err != nil {
		return fmt.Errorf("decoding JSON response: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return fmt.Errorf("closing response body: %w", err)
	}

	return nil
}

func decodeErrorMessage(body io.Reader) (message string) {
	b, err := io.ReadAll(body)
	if err != nil {
		return fmt.Sprintf("failed reading response body: %s", err)
	}

	if len(b) == 0 {
		return ""
	}

	var data []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(b, &data)
	if err != nil {
		return "failed decoding the following: " + string(b)
	}
	if len(data) == 0 {
		return "no message found"
	}

	messages := make([]string, len(data))
	for i, object := range data {
		if object.Message != "" {
			messages[i] = object.Message
			continue
		}
		messages[i] = fmt.Sprintf("code %q", object.Code)
	}
	return strings.Join(messages, "; ")
}
