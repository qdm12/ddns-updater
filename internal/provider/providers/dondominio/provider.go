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
	domain     string
	host       string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	username   string
	key        string
	name       string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		Username string `json:"username"`
		Password string `json:"password"` // retro-compatibility
		Key      string `json:"key"`
		Name     string `json:"name"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	if host == "" {
		host = "@" // default
	}
	if extraSettings.Password != "" { // retro-compatibility
		extraSettings.Key = extraSettings.Password
	}

	p = &Provider{
		domain:     domain,
		host:       host,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		username:   extraSettings.Username,
		key:        extraSettings.Key,
		name:       extraSettings.Name,
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
		return fmt.Errorf("%w", errors.ErrUsernameNotSet)
	case p.key == "":
		return fmt.Errorf("%w", errors.ErrKeyNotSet)
	case p.name == "":
		return fmt.Errorf("%w", errors.ErrNameNotSet)
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
		Provider:  "<a href=\"https://www.dondominio.com/\">DonDominio</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "dondns.dondominio.com",
		Path:   "/json/",
		RawQuery: url.Values{
			"user":   {p.username},
			"apikey": {p.key},
			"host":   {p.BuildDomainName()},
			"ip":     {ip.String()},
			"lang":   {"en"},
		}.Encode(),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}

	var data struct {
		Success  bool     `json:"success"`
		Messages []string `json:"messages"`
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&data)
	if err != nil {
		_ = response.Body.Close()
		return netip.Addr{}, fmt.Errorf("decoding response body: %w", err)
	}

	err = response.Body.Close()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("closing response body: %w", err)
	}

	if !data.Success {
		_ = response.Body.Close()
		return netip.Addr{}, fmt.Errorf("%w: %s",
			errors.ErrUnsuccessful, strings.Join(data.Messages, ", "))
	}

	return ip, nil
}
