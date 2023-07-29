package hetzner

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain         string
	host           string
	ipVersion      ipversion.IPVersion
	key            string
	token          string
	zoneIdentifier string
	ttl            uint
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Key            string `json:"key"`
		Token          string `json:"token"`
		ZoneIdentifier string `json:"zone_identifier"`
		TTL            uint   `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:         domain,
		host:           host,
		ipVersion:      ipVersion,
		key:            extraSettings.Key,
		token:          extraSettings.Token,
		zoneIdentifier: extraSettings.ZoneIdentifier,
		ttl:            extraSettings.TTL,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

var (
	keyRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
)

func (p *Provider) isValid() error {
	switch {
	case p.key != "":
		if !keyRegex.MatchString(p.key) {
			return fmt.Errorf("%w: key %q does not match regex %q",
				errors.ErrKeyNotValid, p.key, keyRegex)
		}
	default: // constants.API token only
	}
	switch {
	case p.zoneIdentifier == "":
		return fmt.Errorf("%w", errors.ErrZoneIdentifierNotSet)
	case p.ttl == 0:
		return fmt.Errorf("%w", errors.ErrTTLNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Hetzner, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Host() string {
	return p.host
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:      p.Host(),
		Provider:  "<a href=\"https://www.hetzner.com\">Hetzner</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	switch {
	case p.token != "":
		headers.SetAuthAPIToken(request, p.token)
	}
}

// Obtain domain ID.
// See https://dns.hetzner.com/api-docs#operation/GetZones.
func (p *Provider) getRecordID(ctx context.Context, client *http.Client, newIP netip.Addr) (
	identifier string, upToDate bool, err error) {
	recordType := constants.A
	if newIP.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "dns.hetzner.com",
		Path:   "/api/v1/records",
	}

	values := url.Values{}
	values.Set("zone_id", p.zoneIdentifier)
	values.Set("name", p.host)
	values.Set("type", recordType)
	values.Set("page", "1")
	values.Set("per_page", "1")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", false, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", false, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	listRecordsResponse := struct {
		Records []struct {
			ID    string `json:"id"`
			Value string `json:"value"`
		} `json:"records"`
	}{}
	err = decoder.Decode(&listRecordsResponse)
	if err != nil {
		return "", false, fmt.Errorf("json decoding response body: %w", err)
	}

	switch {
	case len(listRecordsResponse.Records) == 0:
		return "", false, fmt.Errorf("%w", errors.ErrReceivedNoResult)
	case len(listRecordsResponse.Records) > 1:
		return "", false, fmt.Errorf("%w: %d instead of 1",
			errors.ErrResultsCountReceived, len(listRecordsResponse.Records))
	case listRecordsResponse.Records[0].Value == newIP.String(): // up to date
		return "", true, nil
	}
	return listRecordsResponse.Records[0].ID, false, nil
}

func (p *Provider) CreateRecord(ctx context.Context, client *http.Client, ip netip.Addr) (recordID string, err error) {
	recordType := constants.A

	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "dns.hetzner.com",
		Path:   fmt.Sprintf("/api/v1/records"),
	}

	requestData := struct {
		Type           string `json:"type"`  // constants.A or constants.AAAA depending on ip address given
		Name           string `json:"name"`  // DNS record name i.e. example.com
		Value          string `json:"value"` // ip address
		ZoneIdentifier string `json:"zone_id"`
		TTL            uint   `json:"ttl"`
	}{
		Type:           recordType,
		Name:           p.host,
		Value:          ip.String(),
		ZoneIdentifier: p.zoneIdentifier,
		TTL:            p.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return "", fmt.Errorf("JSON encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}

	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode > http.StatusUnsupportedMediaType {
		return "", fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var parsedJSON struct {
		Record struct {
			ID string `json:"id"`
		} `json:"record"`
	}
	err = decoder.Decode(&parsedJSON)
	if err != nil {
		return "", fmt.Errorf("json decoding response body: %w", err)
	}

	return parsedJSON.Record.ID, nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	identifier, upToDate, err := p.getRecordID(ctx, client, ip)

	switch {
	case stderrors.Is(err, errors.ErrReceivedNoResult):
		identifier, err = p.CreateRecord(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
	case err != nil:
		return netip.Addr{}, fmt.Errorf("getting record id: %w", err)
	case upToDate:
		return ip, nil
	}

	u := url.URL{
		Scheme: "https",
		Host:   "dns.hetzner.com",
		Path:   fmt.Sprintf("/api/v1/records/%s", identifier),
	}

	requestData := struct {
		Type           string `json:"type"`  // constants.A or constants.AAAA depending on ip address given
		Name           string `json:"name"`  // DNS record name i.e. example.com
		Value          string `json:"value"` // ip address
		ZoneIdentifier string `json:"zone_id"`
		TTL            uint   `json:"ttl"`
	}{
		Type:           recordType,
		Name:           p.host,
		Value:          ip.String(),
		ZoneIdentifier: p.zoneIdentifier,
		TTL:            p.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode > http.StatusUnsupportedMediaType {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var parsedJSON struct {
		Record struct {
			Value string `json:"value"`
		} `json:"record"`
	}
	err = decoder.Decode(&parsedJSON)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	newIP, err = netip.ParseAddr(parsedJSON.Record.Value)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if newIP.Compare(ip) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
