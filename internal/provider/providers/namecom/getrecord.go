package namecom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) getRecordID(ctx context.Context, client *http.Client,
	recordType string) (recordID int, err error) {
	u := &url.URL{
		Scheme: "https",
		Host:   "api.name.com",
		Path:   fmt.Sprintf("/v4/domains/%s/records", p.domain),
		User:   url.UserPassword(p.username, p.password),
	}

	// by default GET request will return 1000 records.
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return 0, fmt.Errorf("%w", errors.ErrDomainIDNotFound)
	default:
		return 0, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var data struct {
		Records []struct {
			RecordID int    `json:"id"`
			Host     string `json:"host"`
			Type     string `json:"type"`
		} `json:"records"`
	}
	err = decoder.Decode(&data)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	for _, record := range data.Records {
		if record.Host == p.host && record.Type == recordType {
			return record.RecordID, nil
		}
	}

	return 0, fmt.Errorf("%w", errors.ErrRecordNotFound)
}
