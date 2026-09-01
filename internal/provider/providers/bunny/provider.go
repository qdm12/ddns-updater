package bunny

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	apiKey     string
	ttl        uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		APIKey string `json:"api_key"`
		TTL    uint32 `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.APIKey, extraSettings.TTL)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		apiKey:     extraSettings.APIKey,
		ttl:        extraSettings.TTL,
	}, nil
}

func validateSettings(domain, apiKey string, ttl uint32) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	const minTTL, maxTTL = 60, 3600
	switch {
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case ttl != 0 && ttl < minTTL:
		return fmt.Errorf("%w: %d", errors.ErrTTLTooLow, ttl)
	case ttl > maxTTL:
		return fmt.Errorf("%w: %d > %d", errors.ErrTTLTooHigh, ttl, maxTTL)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Bunny, p.ipVersion)
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
		Provider:  "<a href=\"https://bunny.net/\">Bunny</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	request.Header.Set("AccessKey", p.apiKey) //nolint:canonicalheader
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	zoneID, err := p.getZoneID(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting zone id: %w", err)
	}

	recordType := 0
	if ip.Is6() {
		recordType = 1
	}

	record, err := p.getRecord(ctx, client, zoneID, recordType)
	if err != nil {
		if !stderrors.Is(err, errors.ErrRecordNotFound) {
			return netip.Addr{}, fmt.Errorf("getting record: %w", err)
		}

		err = p.createRecord(ctx, client, zoneID, recordType, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	}

	if record.IP == ip {
		return ip, nil
	}

	err = p.updateRecord(ctx, client, zoneID, record.ID, recordType, record.Name, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("updating record: %w", err)
	}
	return ip, nil
}

func (p *Provider) getZoneID(ctx context.Context, client *http.Client) (zoneID int64, err error) {
	values := url.Values{}
	values.Set("page", "1")
	values.Set("perPage", "1000")
	values.Set("search", p.domain)

	u := url.URL{
		Scheme:   "https",
		Host:     "api.bunny.net",
		Path:     "/dnszone",
		RawQuery: values.Encode(),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	bodyBytes, statusCode, err := doRequest(client, request)
	if err != nil {
		return 0, err
	}

	if statusCode != http.StatusOK {
		return 0, statusCodeToError(statusCode, bodyBytes)
	}

	var parsedJSON struct {
		Items []struct {
			ID     int64  `json:"Id"`
			Domain string `json:"Domain"`
		} `json:"Items"`
	}
	err = json.Unmarshal(bodyBytes, &parsedJSON)
	if err != nil {
		return 0, fmt.Errorf("json decoding response body: %w", err)
	}

	for _, item := range parsedJSON.Items {
		if item.Domain == p.domain {
			return item.ID, nil
		}
	}

	return 0, fmt.Errorf("%w: %s", errors.ErrZoneNotFound, p.domain)
}

type dnsRecord struct {
	ID   int64
	Name string
	IP   netip.Addr
}

func (p *Provider) getRecord(ctx context.Context, client *http.Client, zoneID int64,
	recordType int,
) (record dnsRecord, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.bunny.net",
		Path:   "/dnszone/" + strconv.FormatInt(zoneID, 10),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return dnsRecord{}, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	bodyBytes, statusCode, err := doRequest(client, request)
	if err != nil {
		return dnsRecord{}, err
	}

	if statusCode != http.StatusOK {
		return dnsRecord{}, statusCodeToError(statusCode, bodyBytes)
	}

	var parsedJSON struct {
		Records []struct {
			ID    int64  `json:"Id"`
			Type  int    `json:"Type"`
			Value string `json:"Value"`
			Name  string `json:"Name"`
		} `json:"Records"`
	}
	err = json.Unmarshal(bodyBytes, &parsedJSON)
	if err != nil {
		return dnsRecord{}, fmt.Errorf("json decoding response body: %w", err)
	}

	var records []dnsRecord
	for _, record := range parsedJSON.Records {
		if record.Type != recordType || !p.recordNameMatches(record.Name) {
			continue
		}

		ip, err := netip.ParseAddr(record.Value)
		if err != nil {
			return dnsRecord{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
		}

		records = append(records, dnsRecord{
			ID:   record.ID,
			Name: record.Name,
			IP:   ip,
		})
	}

	switch len(records) {
	case 0:
		return dnsRecord{}, fmt.Errorf("%w", errors.ErrRecordNotFound)
	case 1:
		return records[0], nil
	default:
		return dnsRecord{}, fmt.Errorf("%w: %d", errors.ErrResultsCountReceived, len(records))
	}
}

func (p *Provider) createRecord(ctx context.Context, client *http.Client, zoneID int64,
	recordType int, ip netip.Addr,
) (err error) {
	requestBody, err := p.marshalRecordRequest(0, recordType, p.recordName(), ip)
	if err != nil {
		return err
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.bunny.net",
		Path:   "/dnszone/" + strconv.FormatInt(zoneID, 10) + "/records",
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), requestBody)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	bodyBytes, statusCode, err := doRequest(client, request)
	if err != nil {
		return err
	}

	if statusCode != http.StatusCreated {
		return statusCodeToError(statusCode, bodyBytes)
	}
	return nil
}

func (p *Provider) updateRecord(ctx context.Context, client *http.Client, zoneID, recordID int64,
	recordType int, recordName string, ip netip.Addr,
) (err error) {
	requestBody, err := p.marshalRecordRequest(recordID, recordType, recordName, ip)
	if err != nil {
		return err
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.bunny.net",
		Path: "/dnszone/" + strconv.FormatInt(zoneID, 10) +
			"/records/" + strconv.FormatInt(recordID, 10),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), requestBody)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	bodyBytes, statusCode, err := doRequest(client, request)
	if err != nil {
		return err
	}

	if statusCode != http.StatusNoContent {
		return statusCodeToError(statusCode, bodyBytes)
	}
	return nil
}

func (p *Provider) marshalRecordRequest(recordID int64, recordType int, recordName string,
	ip netip.Addr,
) (buffer *bytes.Buffer, err error) {
	type requestData struct {
		Type  int    `json:"Type"`
		TTL   uint32 `json:"Ttl,omitempty"`
		Value string `json:"Value"`
		Name  string `json:"Name"`
		ID    int64  `json:"Id,omitempty"`
	}

	data := requestData{
		Type:  recordType,
		TTL:   p.ttl,
		Value: ip.String(),
		Name:  recordName,
		ID:    recordID,
	}

	buffer = bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(data)
	if err != nil {
		return nil, fmt.Errorf("json encoding request data: %w", err)
	}
	return buffer, nil
}

func (p *Provider) recordName() string {
	if p.owner == "@" {
		return ""
	}
	return p.owner
}

func (p *Provider) recordNameMatches(recordName string) bool {
	if p.owner == "@" {
		return recordName == "" || recordName == "@"
	}
	return recordName == p.owner
}

func doRequest(client *http.Client, request *http.Request) (bodyBytes []byte, statusCode int, err error) {
	response, err := client.Do(request) //nolint:gosec
	if err != nil {
		return nil, 0, err
	}

	bodyBytes, err = io.ReadAll(response.Body)
	if err != nil {
		_ = response.Body.Close()
		return nil, 0, fmt.Errorf("reading response body: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return nil, 0, fmt.Errorf("closing response body: %w", err)
	}
	return bodyBytes, response.StatusCode, nil
}

func statusCodeToError(statusCode int, bodyBytes []byte) (err error) {
	message := parseErrorMessage(bodyBytes)
	switch statusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, message)
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", errors.ErrAuth, message)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", errors.ErrZoneNotFound, message)
	default:
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid, statusCode, message)
	}
}

func parseErrorMessage(bodyBytes []byte) (message string) {
	var parsedJSON struct {
		ErrorKey string `json:"ErrorKey"`
		Field    string `json:"Field"`
		Message  string `json:"Message"`
	}
	err := json.Unmarshal(bodyBytes, &parsedJSON)
	if err != nil {
		return utils.ToSingleLine(string(bodyBytes))
	}

	switch {
	case parsedJSON.Message != "":
		return parsedJSON.Message
	case parsedJSON.Field != "" && parsedJSON.ErrorKey != "":
		return parsedJSON.Field + ": " + parsedJSON.ErrorKey
	case parsedJSON.ErrorKey != "":
		return parsedJSON.ErrorKey
	default:
		return utils.ToSingleLine(string(bodyBytes))
	}
}
