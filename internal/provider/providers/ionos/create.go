package ionos

import (
	"bytes"
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

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	zoneID string, ip netip.Addr) (err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.hosting.ionos.com",
		Path:   "/dns/v1/zones/" + zoneID + "/records",
	}

	const defaultTTL = 3600
	const defaultPrio = 0
	recordsList := []apiRecord{
		{
			Name:     utils.BuildURLQueryHostname(p.owner, p.domain),
			Type:     recordType,
			Content:  ip.String(),
			TTL:      defaultTTL,
			Prio:     defaultPrio,
			Disabled: false,
		},
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(recordsList)
	if err != nil {
		return fmt.Errorf("encoding request data to JSON: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
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
	case http.StatusCreated:
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
	default:
		return fmt.Errorf("%w: %s: %s", errors.ErrHTTPStatusNotValid,
			response.Status, decodeErrorMessage(response.Body))
	}
}
