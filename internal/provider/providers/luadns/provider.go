package luadns

import (
	"bytes"
	"context"
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
	domain     string
	host       string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	email      string
	token      string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:     domain,
		host:       host,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		email:      extraSettings.Email,
		token:      extraSettings.Token,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

var (
	regexEmail = regexp.MustCompile(`[a-zA-Z0-9-_.+]+@[a-zA-Z0-9-_.]+\.[a-zA-Z]{2,10}`)
)

func (p *Provider) isValid() error {
	switch {
	case !regexEmail.MatchString(p.email):
		return fmt.Errorf("%w: email %q does not match regex %s",
			errors.ErrEmailNotValid, p.email, regexEmail)
	case p.token == "":
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.LuaDNS, p.ipVersion)
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

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
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
		Provider:  "<a href=\"https://www.luadns.com/\">LuaDNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")
}

// Using https://www.luadns.com/api.html
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	zoneID, err := p.getZoneID(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting zone id: %w", err)
	}

	record, err := p.getRecord(ctx, client, zoneID, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting record: %w", err)
	}

	newRecord := record
	newRecord.Content = ip.String()
	err = p.updateRecord(ctx, client, zoneID, newRecord)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("updating record: %w", err)
	}
	return ip, nil
}

type luaDNSRecord struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

type luaDNSError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (p *Provider) getZoneID(ctx context.Context, client *http.Client) (zoneID int, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   "/v1/zones",
		User:   url.UserPassword(p.email, p.token),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("reading response body: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		var errorObj luaDNSError
		if jsonErr := json.Unmarshal(b, &errorObj); jsonErr != nil {
			return 0, fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
		}
		return 0, fmt.Errorf("%w: %s: %s", err, errorObj.Status, errorObj.Message)
	}
	type zone struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var zones []zone

	err = json.Unmarshal(b, &zones)
	if err != nil {
		return 0, fmt.Errorf("json decoding response body: %w", err)
	}
	for _, zone := range zones {
		if zone.Name == p.domain {
			return zone.ID, nil
		}
	}
	return 0, fmt.Errorf("%w", errors.ErrZoneNotFound)
}

func (p *Provider) getRecord(ctx context.Context, client *http.Client, zoneID int, ip netip.Addr) (
	record luaDNSRecord, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   fmt.Sprintf("/v1/zones/%d/records", zoneID),
		User:   url.UserPassword(p.email, p.token),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return record, fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return record, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return record, fmt.Errorf("reading response body: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		var errorObj luaDNSError
		if jsonErr := json.Unmarshal(b, &errorObj); jsonErr != nil {
			return record, fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
		}
		return record, fmt.Errorf("%w: %s: %s",
			err, errorObj.Status, errorObj.Message)
	}
	var records []luaDNSRecord

	err = json.Unmarshal(b, &records)
	if err != nil {
		return record, fmt.Errorf("json decoding response body: %w", err)
	}
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	recordName := utils.BuildURLQueryHostname(p.host, p.domain) + "."
	for _, record := range records {
		if record.Type == recordType && record.Name == recordName {
			return record, nil
		}
	}
	return record, fmt.Errorf("%w: %s record in zone %d",
		errors.ErrRecordNotFound, recordType, zoneID)
}

func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	zoneID int, newRecord luaDNSRecord) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   fmt.Sprintf("/v1/zones/%d/records/%d", zoneID, newRecord.ID),
		User:   url.UserPassword(p.email, p.token),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(newRecord)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)
	headers.SetContentType(request, "application/json")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		var errorObj luaDNSError
		if jsonErr := json.Unmarshal(b, &errorObj); jsonErr != nil {
			return fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
		}
		return fmt.Errorf("%w: %s: %s",
			err, errorObj.Status, errorObj.Message)
	}

	var updatedRecord luaDNSRecord
	if jsonErr := json.Unmarshal(b, &updatedRecord); jsonErr != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	if updatedRecord.Content != newRecord.Content {
		return fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, newRecord.Content, updatedRecord.Content)
	}
	return nil
}
