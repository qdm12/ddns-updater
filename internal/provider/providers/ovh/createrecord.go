package ovh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	recordType, subdomain, ipStr string, timestamp int64) (err error) {
	u := url.URL{
		Scheme: p.apiURL.Scheme,
		Host:   p.apiURL.Host,
		Path:   p.apiURL.Path + "/domain/zone/" + p.domain + "/record",
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
		return fmt.Errorf("%w: %w", errors.ErrRequestMarshal, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}
	request.Header.Add("Content-Type", "application/json;charset=utf-8")
	p.setHeaderCommon(request.Header)
	p.setHeaderAuth(request.Header, timestamp, request.Method, request.URL, bodyBytes)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnsuccessfulResponse, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return extractAPIError(response)
	}

	_ = response.Body.Close()

	return nil
}
