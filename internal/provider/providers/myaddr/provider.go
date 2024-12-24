package myaddr

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
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	key        string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix,
) (*Provider, error) {
	var providerSpecificSettings struct {
		Key string `json:"key"`
	}
	err := json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}
	err = validateSettings(domain, providerSpecificSettings.Key)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}
	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		key:        providerSpecificSettings.Key,
	}, nil
}

func validateSettings(domain, key string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}
	if key == "" {
		return fmt.Errorf("%w", errors.ErrKeyNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.Domain(), p.Owner(), constants.Myaddr, p.IPVersion())
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Owner() string {
	return p.owner
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.owner, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://myaddr.tools/\">myaddr</a>",
		IPVersion: p.IPVersion().String(),
	}
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (netip.Addr, error) {
	u := &url.URL{
		Scheme: "https",
		Host:   "myaddr.tools",
		Path:   "/update",
	}
	v := url.Values{}
	v.Set("key", p.key)
	v.Set("ip", ip.String())
	buffer := strings.NewReader(v.Encode())
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetContentType(request, "application/x-www-form-urlencoded")
	headers.SetUserAgent(request)
	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
		return ip, nil
	case http.StatusBadRequest:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrBadRequest, utils.BodyToSingleLine(response.Body))
	case http.StatusNotFound:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrKeyNotValid, utils.BodyToSingleLine(response.Body))
	default:
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}
}
