package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type donDominio struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	username  string
	password  string
	name      string
}

func NewDonDominio(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion) (s Settings, err error) {
	extraSettings := struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	if len(host) == 0 {
		host = "@" // default
	}
	d := &donDominio{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		username:  extraSettings.Username,
		password:  extraSettings.Password,
		name:      extraSettings.Name,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *donDominio) isValid() error {
	switch {
	case len(d.username) == 0:
		return errors.ErrEmptyUsername
	case len(d.password) == 0:
		return errors.ErrEmptyPassword
	case len(d.name) == 0:
		return errors.ErrEmptyName
	case d.host != "@":
		return errors.ErrHostOnlyAt
	}
	return nil
}

func (d *donDominio) String() string {
	return utils.ToString(d.domain, d.host, constants.DonDominio, d.ipVersion)
}

func (d *donDominio) Domain() string {
	return d.domain
}

func (d *donDominio) Host() string {
	return d.host
}

func (d *donDominio) IPVersion() ipversion.IPVersion {
	return d.ipVersion
}

func (d *donDominio) Proxied() bool {
	return false
}

func (d *donDominio) BuildDomainName() string {
	return utils.BuildDomainName(d.host, d.domain)
}

func (d *donDominio) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://www.dondominio.com/\">DonDominio</a>",
		IPVersion: models.HTML(d.ipVersion.String()),
	}
}

func (d *donDominio) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/x-www-form-urlencoded")
	headers.SetAccept(request, "application/json")
}

func (d *donDominio) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "simple-api.dondominio.net",
	}
	values := url.Values{}
	values.Set("apiuser", d.username)
	values.Set("apipasswd", d.password)
	values.Set("domain", d.domain)
	values.Set("name", d.name)
	isIPv4 := ip.To4() != nil
	if isIPv4 {
		values.Set("ipv4", ip.String())
	} else {
		values.Set("ipv6", ip.String())
	}
	buffer := strings.NewReader(values.Encode())

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return nil, err
	}
	d.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
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
	if err := decoder.Decode(&responseData); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if !responseData.Success {
		return nil, fmt.Errorf("%w: %s (error code %d)",
			errors.ErrUnsuccessfulResponse, responseData.ErrorCodeMessage, responseData.ErrorCode)
	}
	ipString := responseData.ResponseData.GlueRecords[0].IPv4
	if !isIPv4 {
		ipString = responseData.ResponseData.GlueRecords[0].IPv6
	}
	newIP = net.ParseIP(ipString)
	if newIP == nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, ipString)
	} else if !ip.Equal(newIP) {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
	}
	return newIP, nil
}
