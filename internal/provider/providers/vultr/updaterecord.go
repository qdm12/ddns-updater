package vultr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

// https://www.vultr.com/api/#tag/dns/operation/update-dns-domain-record
func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	recordID string, ip netip.Addr,
) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.vultr.com",
		Path:   fmt.Sprintf("/v2/domains/%s/records/%s", p.domain, recordID),
	}

	requestData := struct {
		Data string `json:"data"`
		Name string `json:"name"`
		TTL  uint32 `json:"ttl,omitempty"`
	}{
		Data: ip.String(),
		Name: p.owner,
		TTL:  p.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		return fmt.Errorf("reading response body: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return fmt.Errorf("closing response body: %w", err)
	}

	if response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode,
			parseJSONErrorOrFullBody(bodyBytes))
	}

	errorMessage := parseJSONError(bodyBytes)
	if errorMessage != "" {
		return fmt.Errorf("%w: %s", errors.ErrUnsuccessful, errorMessage)
	}

	return nil
}
