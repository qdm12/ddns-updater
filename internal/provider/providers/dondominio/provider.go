package dondominio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
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
		return fmt.Errorf("%w", errors.ErrUsernameNotSet)
	case p.password == "":
		return fmt.Errorf("%w", errors.ErrPasswordNotSet)
	case p.name == "":
		return fmt.Errorf("%w", errors.ErrNameNotSet)
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
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:      p.Host(),
		Provider:  "<a href=\"https://www.dondominio.com/\">DonDominio</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	aIDs, aaaaIDs, err := p.list(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("listing records: %w", err)
	}

	recordType := constants.A
	ids := aIDs
	if ip.Is6() {
		recordType = constants.AAAA
		ids = aaaaIDs
	}

	if len(ids) == 0 {
		err = p.create(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating %s record for %s: %w",
				recordType, p.BuildDomainName(), err)
		}
		return ip, nil
	}

	for _, id := range ids {
		err = p.update(ctx, client, id, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("updating %s record for %s: %w",
				recordType, p.BuildDomainName(), err)
		}
	}

	return ip, nil
}
