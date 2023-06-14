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
	"strconv"

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
	token     string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		token:     extraSettings.Token,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	if p.token == "" {
		return fmt.Errorf("%w", errors.ErrEmptyToken)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Linode, p.ipVersion)
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
		Provider:  "<a href=\"https://cloud.linode.com/\">Linode</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

// Using https://www.linode.com/docs/api/domains/
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	domainID, err := p.getDomainID(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrGetDomainID, err)
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	recordID, err := p.getRecordID(ctx, client, domainID, recordType)
	if goerrors.Is(err, errors.ErrNotFound) {
		err := p.createRecord(ctx, client, domainID, recordType, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrCreateRecord, err)
		}
		return ip, nil
	} else if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrGetRecordID, err)
	}

	err = p.updateRecord(ctx, client, domainID, recordID, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUpdateRecord, err)
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
		return 0, err
	}
	p.setHeaders(request)
	headers.SetOauth(request, "domains:read_only")
	headers.SetXFilter(request, `{"domain": "`+p.domain+`"}`)

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
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
		return 0, fmt.Errorf("%w", errors.ErrNotFound)
	case 1:
	default:
		return 0, fmt.Errorf("%w: %d records instead of 1",
			errors.ErrNumberOfResultsReceived, len(domains))
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
		Path:   "/v4/domains/" + strconv.Itoa(domainID) + "/records",
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	p.setHeaders(request)
	headers.SetOauth(request, "domains:read_only")

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
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
		return 0, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	for _, domainRecord := range obj.Data {
		if domainRecord.Type == recordType && domainRecord.Host == p.host {
			return domainRecord.ID, nil
		}
	}

	return 0, fmt.Errorf("%w", errors.ErrNotFound)
}

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	domainID int, recordType string, ip netip.Addr) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   "/v4/domains/" + strconv.Itoa(domainID) + "/records",
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
		return fmt.Errorf("%w: %w", errors.ErrRequestMarshal, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return err
	}
	p.setHeaders(request)
	headers.SetOauth(request, "domains:read_write")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
		return fmt.Errorf("%w: %s", err, p.getErrorMessage(response.Body))
	}

	var responseData domainRecord
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&responseData)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
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
		Path:   "/v4/domains/" + strconv.Itoa(domainID) + "/records/" + strconv.Itoa(recordID),
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
		return fmt.Errorf("%w: %w", errors.ErrRequestMarshal, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return err
	}
	p.setHeaders(request)
	headers.SetOauth(request, "domains:read_write")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
		return fmt.Errorf("%w: %s", err, p.getErrorMessage(response.Body))
	}

	data.IP = ""
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
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
