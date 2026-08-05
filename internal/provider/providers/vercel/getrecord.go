package vercel

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"strings"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

// See https://vercel.com/docs/rest-api/dns/list-existing-dns-records
func (p *Provider) getRecord(ctx context.Context, client *http.Client, recordType string) (
	id string, ip netip.Addr, err error,
) {
	u := p.makeURL("/v5/domains/" + p.domain + "/records")

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", netip.Addr{}, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusBadRequest:
		return "", netip.Addr{}, fmt.Errorf("%w: %s",
			errors.ErrBadRequest, utils.BodyToSingleLine(response.Body))
	case http.StatusUnauthorized, http.StatusForbidden:
		return "", netip.Addr{}, fmt.Errorf("%w: %s",
			errors.ErrAuth, utils.BodyToSingleLine(response.Body))
	default:
		return "", netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var result struct {
		Records []struct {
			ID    string     `json:"id"`
			Name  string     `json:"name"`
			Type  string     `json:"type"`
			Value netip.Addr `json:"value"`
			TTL   uint32     `json:"ttl"`
		} `json:"records"`
	}
	err = decoder.Decode(&result)
	if err != nil {
		return "", netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	targetName := p.owner
	if targetName == "@" {
		targetName = ""
	}

	for _, r := range result.Records {
		if r.Name == targetName && strings.EqualFold(r.Type, recordType) {
			return r.ID, r.Value, nil
		}
	}

	return "", netip.Addr{}, fmt.Errorf("%w", errors.ErrRecordNotFound)
}
