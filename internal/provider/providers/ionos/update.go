package ionos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	zoneID string, existingRecord apiRecord, ip netip.Addr) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.hosting.ionos.com",
		Path:   "/dns/v1/zones/" + zoneID + "/records/" + existingRecord.ID,
	}

	recordUpdate := struct {
		Content  string `json:"content"`
		TTL      uint32 `json:"ttl"`
		Prio     uint32 `json:"prio"`
		Disabled bool   `json:"disabled"`
	}{
		Content:  ip.String(),
		TTL:      existingRecord.TTL,
		Prio:     existingRecord.Prio,
		Disabled: existingRecord.Disabled,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(recordUpdate)
	if err != nil {
		return fmt.Errorf("encoding request data to JSON: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
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
		err = response.Body.Close()
		if err != nil {
			return fmt.Errorf("closing response body: %w", err)
		}
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", errors.ErrBadRequest,
			decodeErrorMessage(response.Body))
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", errors.ErrAuth,
			decodeErrorMessage(response.Body))
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", errors.ErrRecordNotFound,
			decodeErrorMessage(response.Body))
	default:
		return fmt.Errorf("%w: %s: %s", errors.ErrHTTPStatusNotValid,
			response.Status, decodeErrorMessage(response.Body))
	}
}
