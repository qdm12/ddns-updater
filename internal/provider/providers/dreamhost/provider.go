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
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	key       string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Key string `json:"key"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	if host == "" { // TODO-v2 remove default
		host = "@" // default
	}
	p = &Provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		key:       extraSettings.Key,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

var keyRegex = regexp.MustCompile(`^[a-zA-Z0-9]{16}$`)

func (p *Provider) isValid() error {
	if !keyRegex.MatchString(p.key) {
		return fmt.Errorf("%w: %s", errors.ErrMalformedKey, p.key)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Dreamhost, p.ipVersion)
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
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://www.dreamhost.com/\">Dreamhost</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	records, err := p.getRecords(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrListRecords, err)
	}

	var oldIP netip.Addr
	for _, data := range records.Data {
		if data.Type == recordType && data.Record == utils.BuildURLQueryHostname(p.host, p.domain) {
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
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrCreateRecord, err)
	}

	if oldIP.IsValid() { // Found editable record with a different IP address, so remove it
		err = p.removeRecord(ctx, client, oldIP)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrRemoveRecord, err)
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
		return records, err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return records, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return records, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&records)
	if err != nil {
		return records, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	if records.Result != constants.Success {
		return records, fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, records.Result)
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
	values.Set("record", utils.BuildURLQueryHostname(p.host, p.domain))
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var dhResponse dreamhostReponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&dhResponse)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	if dhResponse.Result != constants.Success { // this should not happen
		return fmt.Errorf("%w: %s - %s",
			errors.ErrUnsuccessfulResponse, dhResponse.Result, dhResponse.Data)
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
	values.Set("record", utils.BuildURLQueryHostname(p.host, p.domain))
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var dhResponse dreamhostReponse
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&dhResponse)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	if dhResponse.Result != constants.Success {
		return fmt.Errorf("%w: %s - %s",
			errors.ErrUnsuccessfulResponse, dhResponse.Result, dhResponse.Data)
	}
	return nil
}
