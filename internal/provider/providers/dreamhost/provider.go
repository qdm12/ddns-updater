package dreamhost

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	key        string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		Key string `json:"key"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	if owner == "" { // TODO-v2 remove default
		owner = "@" // default
	}

	err = validateSettings(domain, extraSettings.Key)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		key:        extraSettings.Key,
	}, nil
}

var keyRegex = regexp.MustCompile(`^[a-zA-Z0-9]{16}$`)

func validateSettings(domain, key string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if !keyRegex.MatchString(key) {
		return fmt.Errorf("%w: key %q does not match regex %s",
			errors.ErrKeyNotValid, key, keyRegex)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Dreamhost, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Owner() string {
	return p.owner
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.owner, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://www.dreamhost.com/\">Dreamhost</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	records, err := p.getRecords(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("listing records: %w", err)
	}

	var oldIP netip.Addr
	for _, data := range records.Data {
		if data.Type == recordType && data.Record == utils.BuildURLQueryHostname(p.owner, p.domain) {
			if data.Editable == "0" {
				return netip.Addr{}, fmt.Errorf("%w", errors.ErrRecordNotEditable)
			}
			oldIP, err = netip.ParseAddr(data.Value)
			if err == nil && ip.Compare(oldIP) == 0 { // constants.Success, nothing to change
				return ip, nil
			}
			break
		}
	}

	// Create the record with the new IP before removing the old one if it exists.
	err = p.createRecord(ctx, client, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating record: %w", err)
	}

	if oldIP.IsValid() { // Found editable record with a different IP address, so remove it
		err = p.removeRecord(ctx, client, oldIP)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("removing record: %w", err)
		}
	}

	return ip, nil
}

type (
	dreamHostRecords struct {
		Result string `json:"result"`
		Data   []struct {
			Editable string `json:"editable"`
			Type     string `json:"type"`
			Record   string `json:"record"`
			Value    string `json:"value"`
		} `json:"data"`
	}
	dreamhostReponse struct {
		Result string `json:"result"`
		Data   string `json:"data"`
	}
)

func (p *Provider) defaultURLValues() (values url.Values) {
	uuid := make([]byte, 16) //nolint:gomnd
	_, _ = io.ReadFull(rand.Reader, uuid)
	//nolint:gomnd
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	//nolint:gomnd
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	values = url.Values{}
	values.Set("key", p.key)
	values.Set("unique_id", string(uuid))
	values.Set("format", "json")
	return values
}

func (p *Provider) getRecords(ctx context.Context, client *http.Client) (
	records dreamHostRecords, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := p.defaultURLValues()
	values.Set("cmd", "dns-list_records")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return records, fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return records, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return records, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&records)
	if err != nil {
		return records, fmt.Errorf("json decoding response body: %w", err)
	}

	if records.Result != constants.Success {
		return records, fmt.Errorf("%w: %s", errors.ErrUnsuccessful, records.Result)
	}
	return records, nil
}

func (p *Provider) removeRecord(ctx context.Context, client *http.Client, ip netip.Addr) error { //nolint:dupl
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := p.defaultURLValues()
	values.Set("cmd", "dns-remove_record")
	values.Set("record", utils.BuildURLQueryHostname(p.owner, p.domain))
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var dhResponse dreamhostReponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&dhResponse)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	if dhResponse.Result != constants.Success { // this should not happen
		return fmt.Errorf("%w: %s - %s",
			errors.ErrUnsuccessful, dhResponse.Result, dhResponse.Data)
	}
	return nil
}

func (p *Provider) createRecord(ctx context.Context, client *http.Client, ip netip.Addr) error { //nolint:dupl
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := p.defaultURLValues()
	values.Set("cmd", "dns-add_record")
	values.Set("record", utils.BuildURLQueryHostname(p.owner, p.domain))
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var dhResponse dreamhostReponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&dhResponse)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	if dhResponse.Result != constants.Success {
		return fmt.Errorf("%w: %s - %s",
			errors.ErrUnsuccessful, dhResponse.Result, dhResponse.Data)
	}
	return nil
}
