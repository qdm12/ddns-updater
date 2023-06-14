package dondominio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

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
	username  string
	password  string
	name      string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	if host == "" {
		host = "@" // default
	}
	p = &Provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		username:  extraSettings.Username,
		password:  extraSettings.Password,
		name:      extraSettings.Name,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.username == "":
		return fmt.Errorf("%w", errors.ErrEmptyUsername)
	case p.password == "":
		return fmt.Errorf("%w", errors.ErrEmptyPassword)
	case p.name == "":
		return fmt.Errorf("%w", errors.ErrEmptyName)
	case p.host != "@":
		return fmt.Errorf("%w", errors.ErrHostOnlyAt)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.DonDominio, p.ipVersion)
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
		Provider:  "<a href=\"https://www.dondominio.com/\">DonDominio</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/x-www-form-urlencoded")
	headers.SetAccept(request, "application/json")
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "simple-api.dondominio.net",
	}
	values := url.Values{}
	values.Set("apiuser", p.username)
	values.Set("apipasswd", p.password)
	values.Set("domain", p.domain)
	values.Set("name", p.name)
	isIPv4 := ip.Is4()
	if isIPv4 {
		values.Set("ipv4", ip.String())
	} else {
		values.Set("ipv6", ip.String())
	}
	encodedValues := values.Encode()
	buffer := strings.NewReader(encodedValues)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
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
		Success          bool   `json:"success"`
		ErrorCode        int    `json:"errorCode"`
		ErrorCodeMessage string `json:"errorCodeMsg"`
		ResponseData     struct {
			GlueRecords []struct {
				IPv4 string `json:"ipv4"`
				IPv6 string `json:"ipv6"`
			} `json:"gluerecords"`
		} `json:"responseData"`
	}
	err = decoder.Decode(&responseData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	if !responseData.Success {
		return netip.Addr{}, fmt.Errorf("%w: %s (error code %d)",
			errors.ErrUnsuccessfulResponse, responseData.ErrorCodeMessage, responseData.ErrorCode)
	}
	ipString := responseData.ResponseData.GlueRecords[0].IPv4
	if !isIPv4 {
		ipString = responseData.ResponseData.GlueRecords[0].IPv6
	}
	newIP, err = netip.ParseAddr(ipString)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if ip.Compare(newIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
