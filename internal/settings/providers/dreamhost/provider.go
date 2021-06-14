package dreamhost

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	key       string
	matcher   regex.Matcher
	logger    log.Logger
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		Key string `json:"key"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	if len(host) == 0 {
		host = "@" // default
	}
	p = &provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		key:       extraSettings.Key,
		matcher:   matcher,
		logger:    logger,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case !p.matcher.DreamhostKey(p.key):
		return fmt.Errorf("invalid key format")
	case p.host != "@":
		return fmt.Errorf(`host can only be "@"`)
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Dreamhost, p.ipVersion)
}

func (p *provider) Domain() string {
	return p.domain
}

func (p *provider) Host() string {
	return p.host
}

func (p *provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *provider) Proxied() bool {
	return false
}

func (p *provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://www.dreamhost.com/\">Dreamhost</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}

	records, err := p.getRecords(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrListRecords, err)
	}

	var oldIP net.IP
	for _, data := range records.Data {
		if data.Type == recordType && data.Record == utils.BuildURLQueryHostname(p.host, p.domain) {
			if data.Editable == "0" {
				return nil, errors.ErrRecordNotEditable
			}
			oldIP = net.ParseIP(data.Value)
			if ip.Equal(oldIP) { // constants.Success, nothing to change
				return ip, nil
			}
			break
		}
	}

	// Create the record with the new IP before removing the old one if it exists.
	if err := p.createRecord(ctx, client, ip); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrCreateRecord, err)
	}

	if oldIP != nil { // Found editable record with a different IP address, so remove it
		if err := p.removeRecord(ctx, client, oldIP); err != nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrRemoveRecord, err)
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

func (p *provider) defaultURLValues() (values url.Values) {
	uuid := make([]byte, 16)
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

func (p *provider) getRecords(ctx context.Context, client *http.Client) (
	records dreamHostRecords, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := p.defaultURLValues()
	values.Set("cmd", "dns-list_records")
	u.RawQuery = values.Encode()

	p.logger.Debug("HTTP GET " + u.String())

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
	if err := decoder.Decode(&records); err != nil {
		return records, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if records.Result != constants.Success {
		return records, fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, records.Result)
	}
	return records, nil
}

func (p *provider) removeRecord(ctx context.Context, client *http.Client, ip net.IP) error { //nolint:dupl
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := p.defaultURLValues()
	values.Set("cmd", "dns-remove_record")
	values.Set("record", p.domain)
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()

	p.logger.Debug("HTTP GET " + u.String())

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
	if err := decoder.Decode(&dhResponse); err != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if dhResponse.Result != constants.Success { // this should not happen
		return fmt.Errorf("%w: %s - %s",
			errors.ErrUnsuccessfulResponse, dhResponse.Result, dhResponse.Data)
	}
	return nil
}

func (p *provider) createRecord(ctx context.Context, client *http.Client, ip net.IP) error { //nolint:dupl
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := p.defaultURLValues()
	values.Set("cmd", "dns-add_record")
	values.Set("record", p.domain)
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()

	p.logger.Debug("HTTP GET " + u.String())

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
	if err := decoder.Decode(&dhResponse); err != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if dhResponse.Result != constants.Success {
		return fmt.Errorf("%w: %s - %s",
			errors.ErrUnsuccessfulResponse, dhResponse.Result, dhResponse.Data)
	}
	return nil
}
