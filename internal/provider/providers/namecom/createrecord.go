package namecom

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	ip netip.Addr) (err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := &url.URL{
		Scheme: "https",
		Host:   "api.name.com",
		Path:   fmt.Sprintf("/v4/domains/%s/records", p.domain),
		User:   url.UserPassword(p.username, p.token),
	}

	postRecordsParams := struct {
		Host   string  `json:"host"`
		Type   string  `json:"type"`
		Answer string  `json:"answer"`
		TTL    *uint32 `json:"ttl,omitempty"`
	}{
		Host:   p.owner,
		Type:   recordType,
		Answer: ip.String(),
		TTL:    p.ttl,
	}

	bodyBytes, err := json.Marshal(postRecordsParams)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)
	headers.SetContentType(request, "application/json")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing HTTP request: %w", err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return verifySuccessResponseBody(response.Body, ip)
	default:
		return parseErrorResponse(response)
	}
}
