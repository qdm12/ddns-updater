package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	netlib "github.com/qdm12/golibs/network"
)

//nolint:maligned
type donDominio struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	dnsLookup bool
	username  string
	password  string
	name      string
}

func NewDonDominio(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
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
		dnsLookup: !noDNSLookup,
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
		return fmt.Errorf("username cannot be empty")
	case len(d.password) == 0:
		return fmt.Errorf("password cannot be empty")
	case len(d.name) == 0:
		return fmt.Errorf("name cannot be empty")
	case d.host != "@":
		return fmt.Errorf(`host can only be "@"`)
	}
	return nil
}

func (d *donDominio) String() string {
	return toString(d.domain, d.host, constants.DONDOMINIO, d.ipVersion)
}

func (d *donDominio) Domain() string {
	return d.domain
}

func (d *donDominio) Host() string {
	return d.host
}

func (d *donDominio) DNSLookup() bool {
	return d.dnsLookup
}

func (d *donDominio) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *donDominio) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *donDominio) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://www.dondominio.com/\">DonDominio</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *donDominio) Update(ctx context.Context, client netlib.Client, ip net.IP) (newIP net.IP, err error) {
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
	r, err := http.NewRequest(http.MethodPost, u.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentid.mcgaw@gmail.com")
	r = r.WithContext(ctx)
	content, status, err := client.Do(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf(http.StatusText(status))
	}
	response := struct {
		Success          bool   `json:"success"`
		ErrorCode        int    `json:"errorCode"`
		ErrorCodeMessage string `json:"errorCodeMsg"`
		ResponseData     struct {
			GlueRecords []struct {
				IPv4 string `json:"ipv4"`
				IPv6 string `json:"ipv6"`
			} `json:"gluerecords"`
		} `json:"responseData"`
	}{}
	if err := json.Unmarshal(content, &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, fmt.Errorf("%s (error code %d)", response.ErrorCodeMessage, response.ErrorCode)
	}
	ipString := response.ResponseData.GlueRecords[0].IPv4
	if !isIPv4 {
		ipString = response.ResponseData.GlueRecords[0].IPv6
	}
	newIP = net.ParseIP(ipString)
	if newIP == nil {
		return nil, fmt.Errorf("IP address received %q is malformed", ipString)
	}
	return newIP, nil
}
