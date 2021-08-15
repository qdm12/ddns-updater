package linode

import (
	"bytes"
	"context"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	token     string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		token:     extraSettings.Token,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	if len(p.token) == 0 {
		return errors.ErrEmptyToken
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Linode, p.ipVersion)
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
		Provider:  "<a href=\"https://cloud.linode.com/\">Linode</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

// Using https://www.linode.com/docs/api/domains/
func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	domainID, err := p.getDomainID(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrGetDomainID, err)
	}

	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}

	recordID, err := p.getRecordID(ctx, client, domainID, recordType)
	if goerrors.Is(err, errors.ErrNotFound) {
		err := p.createRecord(ctx, client, domainID, recordType, ip)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errors.ErrCreateRecord, err)
		}
		return ip, nil
	} else if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrGetRecordID, err)
	}

	if err := p.updateRecord(ctx, client, domainID, recordID, ip); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUpdateRecord, err)
	}

	return ip, nil
}

type linodeError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (p *provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAuthBearer(request, p.token)
}

func (p *provider) getDomainID(ctx context.Context, client *http.Client) (domainID int, err error) {
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
		return 0, fmt.Errorf("%w: %s", err, p.getError(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var obj struct {
		Data []struct {
			ID     *int   `json:"id,omitempty"`
			Type   string `json:"type"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := decoder.Decode(&obj); err != nil {
		return 0, err
	}

	domains := obj.Data
	switch len(domains) {
	case 0:
		return 0, errors.ErrNotFound
	case 1:
	default:
		return 0, fmt.Errorf("%w: %d records instead of 1",
			errors.ErrNumberOfResultsReceived, len(domains))
	}

	if domains[0].Status == "disabled" {
		return 0, errors.ErrDomainDisabled
	}

	if domains[0].ID == nil {
		return 0, errors.ErrDomainIDNotFound
	}

	return *domains[0].ID, nil
}

func (p *provider) getRecordID(ctx context.Context, client *http.Client,
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
		return 0, fmt.Errorf("%w: %s", err, p.getError(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var obj struct {
		Data []struct {
			ID   int    `json:"id"`
			Host string `json:"name"`
			Type string `json:"type"`
		} `json:"data"`
	}
	if err := decoder.Decode(&obj); err != nil {
		return 0, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	for _, domainRecord := range obj.Data {
		if domainRecord.Type == recordType && domainRecord.Host == p.host {
			return domainRecord.ID, nil
		}
	}

	return 0, errors.ErrNotFound
}

func (p *provider) createRecord(ctx context.Context, client *http.Client,
	domainID int, recordType string, ip net.IP) (err error) {
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
		Host: p.host,
		IP:   ip.String(),
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(requestData); err != nil {
		return fmt.Errorf("%w: %s", errors.ErrRequestMarshal, err)
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
		return fmt.Errorf("%w: %s", err, p.getError(response.Body))
	}

	var responseData domainRecord
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&responseData); err != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	newIP := net.ParseIP(responseData.IP)
	if newIP == nil {
		return fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, responseData.IP)
	} else if !newIP.Equal(ip) {
		return fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
	}

	return nil
}

func (p *provider) updateRecord(ctx context.Context, client *http.Client,
	domainID, recordID int, ip net.IP) (err error) {
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
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("%w: %s", errors.ErrRequestMarshal, err)
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
		return fmt.Errorf("%w: %s", err, p.getError(response.Body))
	}

	data.IP = ""
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	newIP := net.ParseIP(data.IP)
	if newIP == nil {
		return fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, data.IP)
	} else if !newIP.Equal(ip) {
		return fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
	}

	return nil
}

func (p *provider) getError(body io.Reader) (err error) {
	var errorObj linodeError
	b, err := io.ReadAll(body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &errorObj); err != nil {
		return fmt.Errorf("%s", utils.ToSingleLine(string(b)))
	}
	return fmt.Errorf("%s: %s", errorObj.Field, errorObj.Reason)
}
