package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) getRecordID(ctx context.Context, client *http.Client,
	recordType string) (recordID string, err error) {
	u := &url.URL{
		Scheme: "https",
		Host:   "dns.aliyuncs.com",
	}
	values := newURLValues(p.accessKeyID)
	values.Set("Action", "DescribeDomainRecords")
	values.Set("DomainName", p.domain)
	values.Set("RRKeyWord", p.host)
	values.Set("Type", recordType)

	sign(http.MethodGet, values, p.accessSecret)

	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return "", fmt.Errorf("%w", errors.ErrRecordNotFound)
	case http.StatusBadRequest:
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return "", fmt.Errorf("reading response body: %w", err)
		}

		var data struct {
			Code string `json:"Code"`
		}
		err = json.Unmarshal(bodyBytes, &data)
		if err != nil || data.Code != "InvalidDomainName.NoExist" {
			return "", fmt.Errorf("%w: %d: %s",
				errors.ErrBadHTTPStatus, response.StatusCode,
				utils.BodyToSingleLine(response.Body))
		}

		return "", fmt.Errorf("%w", errors.ErrRecordNotFound)
	default:
		return "", fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var data struct {
		DomainRecords struct {
			Record []struct {
				RecordID string `json:"RecordId"`
			} `json:"Record"`
		} `json:"DomainRecords"`
	}
	err = decoder.Decode(&data)
	if err != nil {
		return "", fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	switch len(data.DomainRecords.Record) {
	case 0:
		return "", fmt.Errorf("%w", errors.ErrRecordNotFound)
	case 1:
	default:
		return "", fmt.Errorf("%w: %d records found instead of 1",
			errors.ErrNumberOfResultsReceived, len(data.DomainRecords.Record))
	}

	return data.DomainRecords.Record[0].RecordID, nil
}
