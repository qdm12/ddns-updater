package ovh

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) getRecords(ctx context.Context, client *http.Client,
	recordType, subdomain string, timestamp int64) (recordIDs []uint64, err error) {
	values := url.Values{}
	values.Set("fieldType", recordType)
	values.Set("subDomain", subdomain)
	u := url.URL{
		Scheme:   p.apiURL.Scheme,
		Host:     p.apiURL.Host,
		Path:     p.apiURL.Path + "/domain/zone/" + p.domain + "/record",
		RawQuery: values.Encode(),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}
	p.setHeaderCommon(request.Header)
	p.setHeaderAuth(request.Header, timestamp, request.Method, request.URL, nil)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrUnsuccessfulResponse, err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, extractAPIError(response)
	}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&recordIDs)
	if err != nil {
		_ = response.Body.Close()
		return nil, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	_ = response.Body.Close()

	return recordIDs, nil
}
