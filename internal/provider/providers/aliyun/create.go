package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) createRecord(ctx context.Context,
	client *http.Client, ip netip.Addr) (recordID string, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := &url.URL{
		Scheme: "https",
		Host:   "alidns.aliyuncs.com",
	}
	values := newURLValues(p.accessKeyID)
	values.Set("Action", "AddDomainRecord")
	values.Set("DomainName", p.domain)
	values.Set("RR", p.owner)
	values.Set("Type", recordType)
	values.Set("Value", ip.String())

	sign(http.MethodGet, values, p.accessSecret)

	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("doing HTTP request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			utils.BodyToSingleLine(response.Body))
	}

	var data struct {
		RecordID string `json:"RecordId"`
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return "", fmt.Errorf("json decoding response body: %w", err)
	}

	return data.RecordID, nil
}
