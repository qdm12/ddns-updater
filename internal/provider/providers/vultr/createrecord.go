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

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

func (p *Provider) createRecord(ctx context.Context, client *http.Client, ip netip.Addr) (err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.vultr.com",
		Path:   fmt.Sprintf("/v2/domains/%s/records", p.domain),
	}

	requestData := struct {
		Type string `json:"type"`
		IP   string `json:"data"`
		Name string `json:"name"`
		TTL  uint32 `json:"ttl,omitempty"`
	}{
		Type: recordType,
		IP:   ip.String(),
		Name: p.owner,
		TTL:  p.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
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
	case http.StatusCreated:
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, parseJSONError(response.Body))
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", errors.ErrAuth, parseJSONError(response.Body))
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", errors.ErrDomainNotFound, parseJSONError(response.Body))
	default:
		return fmt.Errorf("%w: %s: %s", errors.ErrHTTPStatusNotValid,
			response.Status, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var parsedJSON struct {
		Error  string
		Status uint32
		Record Record
	}
	err = decoder.Decode(&parsedJSON)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	newIP, err := netip.ParseAddr(parsedJSON.Record.IP)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if newIP.Compare(ip) != 0 {
		return fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return nil
}

func parseJSONError(body io.Reader) (message string) {
	var parsedJSON struct {
		Error string `json:"error"`
	}
	decoder := json.NewDecoder(body)
	err := decoder.Decode(&parsedJSON)
	if err != nil {
		return "<json decoding error: " + err.Error() + ">"
	}
	return parsedJSON.Error
}
