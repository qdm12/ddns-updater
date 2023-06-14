package digitalocean

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	return utils.ToString(p.domain, p.host, constants.DigitalOcean, p.ipVersion)
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
		Provider:  "<a href=\"https://www.digitalocean.com/\">DigitalOcean</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.token)
}

func (p *Provider) getRecordID(ctx context.Context, recordType string, client *http.Client) (
	recordID int, err error) {
	values := url.Values{}
	values.Set("name", utils.BuildURLQueryHostname(p.host, p.domain))
	values.Set("type", recordType)
	u := url.URL{
		Scheme:   "https",
		Host:     "api.digitalocean.com",
		Path:     "/v2/domains/" + p.domain + "/records",
		RawQuery: values.Encode(),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var result struct {
		DomainRecords []struct {
			ID int `json:"id"`
		} `json:"domain_records"`
	}
	err = decoder.Decode(&result)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	if len(result.DomainRecords) == 0 {
		return 0, fmt.Errorf("%w", errors.ErrNotFound)
	} else if result.DomainRecords[0].ID == 0 {
		return 0, fmt.Errorf("%w", errors.ErrDomainIDNotFound)
	}

	return result.DomainRecords[0].ID, nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	recordID, err := p.getRecordID(ctx, recordType, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrGetRecordID, err)
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.digitalocean.com",
		Path:   fmt.Sprintf("/v2/domains/%s/records/%d", p.domain, recordID),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	requestData := struct {
		Type string `json:"type"`
		Name string `json:"name"`
		Data string `json:"data"`
	}{
		Type: recordType,
		Name: p.host,
		Data: ip.String(),
	}
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrRequestEncode, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var responseData struct {
		DomainRecord struct {
			Data string `json:"data"`
		} `json:"domain_record"`
	}
	err = decoder.Decode(&responseData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	newIP, err = netip.ParseAddr(responseData.DomainRecord.Data)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if newIP.Compare(ip) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
