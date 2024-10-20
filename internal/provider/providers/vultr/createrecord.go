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

// https://www.vultr.com/api/#tag/dns/operation/create-dns-domain-record
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
		Data string `json:"data"`
		Name string `json:"name"`
		TTL  uint32 `json:"ttl,omitempty"`
	}{
		Type: recordType,
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

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
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

	switch response.StatusCode {
	case http.StatusCreated:
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, parseJSONErrorOrFullBody(bodyBytes))
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", errors.ErrAuth, parseJSONErrorOrFullBody(bodyBytes))
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", errors.ErrDomainNotFound, parseJSONErrorOrFullBody(bodyBytes))
	default:
		return fmt.Errorf("%w: %s: %s", errors.ErrHTTPStatusNotValid,
			response.Status, parseJSONErrorOrFullBody(bodyBytes))
	}

	errorMessage := parseJSONError(bodyBytes)
	if errorMessage != "" {
		return fmt.Errorf("%w: %s", errors.ErrUnsuccessful, errorMessage)
	}
	return nil
}

// parseJSONErrorOrFullBody parses the json error from a response body
// and returns it if it is not empty. If the json decoding fails OR
// the error parsed is empty, the entire body is returned on a single line.
func parseJSONErrorOrFullBody(body []byte) (message string) {
	var parsedJSON struct {
		Error string `json:"error"`
	}
	err := json.Unmarshal(body, &parsedJSON)
	if err != nil || parsedJSON.Error == "" {
		return utils.ToSingleLine(string(body))
	}
	return parsedJSON.Error
}

// parseJSONError parses the json error from a response body
// and returns it directly. If the json decoding fails, the
// entire body is returned on a single line.
func parseJSONError(body []byte) (message string) {
	var parsedJSON struct {
		Error string `json:"error"`
	}
	err := json.Unmarshal(body, &parsedJSON)
	if err != nil {
		return utils.ToSingleLine(string(body))
	}
	return parsedJSON.Error
}
