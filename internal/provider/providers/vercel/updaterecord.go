package vercel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

// See https://vercel.com/docs/rest-api/dns/update-an-existing-dns-record
func (p *Provider) updateRecord(ctx context.Context, client *http.Client, recordID string, ip netip.Addr) error {
	u := p.makeURL("/v1/domains/records/" + recordID)

	name := p.owner
	if name == "@" {
		name = ""
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	requestData := struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Value   string `json:"value"`
		TTL     uint32 `json:"ttl,omitempty"`
		Comment string `json:"comment,omitempty"`
	}{
		Name:    name,
		Type:    recordType,
		Value:   ip.String(),
		TTL:     p.ttl,
		Comment: "DDNS Updater automatically manages this record.",
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err := encoder.Encode(requestData)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, u, buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK, http.StatusCreated:
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s",
			errors.ErrBadRequest, utils.BodyToSingleLine(response.Body))
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s",
			errors.ErrAuth, utils.BodyToSingleLine(response.Body))
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s",
			errors.ErrRecordNotFound, utils.BodyToSingleLine(response.Body))
	default:
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var data struct {
		Value string `json:"value"`
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	receivedIP, err := netip.ParseAddr(data.Value)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	}

	if receivedIP != ip {
		return fmt.Errorf("%w: sent %s and received %s",
			errors.ErrIPReceivedMismatch, ip, receivedIP)
	}

	return nil
}
