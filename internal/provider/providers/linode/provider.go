package linode

import (
	"bytes"
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"

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
	token      string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.Token)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		token:      extraSettings.Token,
	}, nil
}

func validateSettings(domain, token string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if token == "" {
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Linode, p.ipVersion)
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
		Provider:  "<a href=\"https://cloud.linode.com/\">Linode</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Using https://www.linode.com/docs/api/domains/
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	domainID, err := p.getDomainID(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting domain id: %w", err)
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	recordID, err := p.getRecordID(ctx, client, domainID, recordType)
	if goerrors.Is(err, errors.ErrRecordNotFound) {
		err := p.createRecord(ctx, client, domainID, recordType, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	} else if err != nil {
		return netip.Addr{}, fmt.Errorf("getting record id: %w", err)
	}

	err = p.updateRecord(ctx, client, domainID, recordID, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("updating record: %w", err)
	}

	return ip, nil
}

type linodeErrors struct {
	Errors []struct {
		Field  string `json:"field"`
		Reason string `json:"reason"`
	} `json:"errors"`
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAuthBearer(request, p.token)
}

func (p *Provider) getDomainID(ctx context.Context, client *http.Client) (domainID int, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   "/v4/domains",
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)
	headers.SetOauth(request, "domains:read_only")
	headers.SetXFilter(request, `{"domain": "`+p.domain+`"}`)

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		return 0, fmt.Errorf("%w: %s", err, p.getErrorMessage(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var obj struct {
		Data []struct {
			ID     *int   `json:"id,omitempty"`
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"data"`
	}
	err = decoder.Decode(&obj)
	if err != nil {
		return 0, err
	}

	domains := obj.Data
	switch len(domains) {
	case 0:
		return 0, fmt.Errorf("%w", errors.ErrDomainIDNotFound)
	case 1:
	default:
		return 0, fmt.Errorf("%w: %d records instead of 1",
			errors.ErrResultsCountReceived, len(domains))
	}

	if domains[0].Status == "disabled" {
		return 0, fmt.Errorf("%w", errors.ErrDomainDisabled)
	}

	if domains[0].ID == nil {
		return 0, fmt.Errorf("%w", errors.ErrDomainIDNotFound)
	}

	return *domains[0].ID, nil
}

func (p *Provider) getRecordID(ctx context.Context, client *http.Client,
	domainID int, recordType string) (recordID int, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   fmt.Sprintf("/v4/domains/%d/records", domainID),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)
	headers.SetOauth(request, "domains:read_only")

	response, err := client.Do(request)
	if err != nil {
		return 0, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		return 0, fmt.Errorf("%w: %s", err, p.getErrorMessage(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var obj struct {
		Data []struct {
			ID   int    `json:"id"`
			Host string `json:"name"`
			Type string `json:"type"`
		} `json:"data"`
	}
	err = decoder.Decode(&obj)
	if err != nil {
		return 0, fmt.Errorf("json decoding response body: %w", err)
	}

	for _, domainRecord := range obj.Data {
		if domainRecord.Type == recordType && domainRecord.Host == p.owner {
			return domainRecord.ID, nil
		}
	}

	return 0, fmt.Errorf("%w", errors.ErrRecordNotFound)
}

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	domainID int, recordType string, ip netip.Addr) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   fmt.Sprintf("/v4/domains/%d/records", domainID),
	}

	type domainRecord struct {
		Type string `json:"type"`
		Host string `json:"name"`
		IP   string `json:"target"`
	}

	requestData := domainRecord{
		Type: recordType,
		Host: p.BuildDomainName(),
		IP:   ip.String(),
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
	headers.SetOauth(request, "domains:read_write")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		return fmt.Errorf("%w: %s", err, p.getErrorMessage(response.Body))
	}

	var responseData domainRecord
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&responseData)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	newIP, err := netip.ParseAddr(responseData.IP)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if newIP.Compare(ip) != 0 {
		return fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}

	return nil
}

func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	domainID, recordID int, ip netip.Addr) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   fmt.Sprintf("/v4/domains/%d/records/%d", domainID, recordID),
	}

	data := struct {
		IP string `json:"target"`
	}{
		IP: ip.String(),
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)
	headers.SetOauth(request, "domains:read_write")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
		return fmt.Errorf("%w: %s", err, p.getErrorMessage(response.Body))
	}

	data.IP = ""
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return fmt.Errorf("json decoding response body: %w", err)
	}

	newIP, err := netip.ParseAddr(data.IP)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if newIP.Compare(ip) != 0 {
		return fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}

	return nil
}

func (p *Provider) getErrorMessage(body io.Reader) (message string) {
	var errorObj linodeErrors
	b, err := io.ReadAll(body)
	if err != nil {
		return fmt.Sprintf("reading body: %s", err)
	}
	err = json.Unmarshal(b, &errorObj)
	if err != nil {
		return utils.ToSingleLine(string(b))
	}
	return fmt.Sprintf("%v", errorObj)
}
