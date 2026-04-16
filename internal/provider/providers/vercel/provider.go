package vercel

import (
	"context"
	"encoding/json"
	stderrors "errors"
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
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	token      string
	teamID     string
	ttl        uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Token  string `json:"token"`
		TeamID string `json:"team_id"`
		TTL    uint32 `json:"ttl"`
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
		teamID:     extraSettings.TeamID,
		ttl:        extraSettings.TTL,
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
	return utils.ToString(p.domain, p.owner, constants.Vercel, p.ipVersion)
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
		Provider:  "<a href=\"https://vercel.com/\">Vercel</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	id, value, err := p.getRecord(ctx, client, recordType)
	switch {
	case stderrors.Is(err, errors.ErrRecordNotFound):
		err = p.createRecord(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	case err != nil:
		return netip.Addr{}, fmt.Errorf("getting record: %w", err)
	case value == ip:
		return ip, nil
	}

	err = p.updateRecord(ctx, client, id, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("updating record: %w", err)
	}

	return ip, nil
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")
	headers.SetContentType(request, "application/json")
	headers.SetAuthBearer(request, p.token)
}

func (p *Provider) makeURL(path string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "api.vercel.com",
		Path:   path,
	}
	if p.teamID != "" {
		values := url.Values{}
		values.Set("teamId", p.teamID)
		u.RawQuery = values.Encode()
	}
	return u.String()
}
