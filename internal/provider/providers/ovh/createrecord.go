package ovh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	recordType, subdomain, ipStr string, timestamp int64) (err error) {
	u := url.URL{
		Scheme: p.apiURL.Scheme,
		Host:   p.apiURL.Host,
		Path:   fmt.Sprintf("%s/domain/zone/%s/record", p.apiURL.Path, p.domain),
	}
	postRecordsParams := struct {
		FieldType string `json:"fieldType"`
		SubDomain string `json:"subDomain"`
		Target    string `json:"target"`
	}{
		FieldType: recordType,
		SubDomain: subdomain,
		Target:    ipStr,
	}
	bodyBytes, err := json.Marshal(postRecordsParams)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	request.Header.Add("Content-Type", "application/json;charset=utf-8")
	p.setHeaderCommon(request.Header)
	p.setHeaderAuth(request.Header, timestamp, request.Method, request.URL, bodyBytes)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return extractAPIError(response)
	}

	_ = response.Body.Close()

	return nil
}
