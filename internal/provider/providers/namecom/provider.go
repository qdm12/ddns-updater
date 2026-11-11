package namecom

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"sync/atomic"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	username   string
	token      string
	ttl        *uint32
	knownTTL   atomic.Uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Username string  `json:"username"`
		Token    string  `json:"token"`
		TTL      *uint32 `json:"ttl,omitempty"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.Username, extraSettings.Token, extraSettings.TTL)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		username:   extraSettings.Username,
		token:      extraSettings.Token,
		ttl:        extraSettings.TTL,
	}, nil
}

func validateSettings(domain, username, token string, ttl *uint32) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("domain is not valid: %w", err)
	}

	const minTTL = uint32(300)
	switch {
	case username == "":
		return errors.New("username is not set")
	case token == "":
		return errors.New("password is not set")
	case ttl != nil && *ttl < minTTL:
		return fmt.Errorf("TTL is too low: %d must be at least %d", *ttl, minTTL)
	default:
		return nil
	}
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.NameCom, p.ipVersion)
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
		Provider:  "<a href=\"https://name.com\">Name.com</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// TTL returns the record TTL in seconds if it is set or known.
func (p *Provider) TTL() (ttl *uint32) {
	if p.ttl != nil {
		return p.ttl
	}
	if knownTTL := p.knownTTL.Load(); knownTTL != 0 {
		return &knownTTL
	}
	return nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// Documentation at https://www.name.com/api-docs
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	recordID, err := p.getRecordID(ctx, client, recordType)

	if errors.Is(err, ddnserrors.ErrRecordNotFound) {
		err = p.createRecord(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}

		return ip, nil
	} else if err != nil {
		return netip.Addr{}, fmt.Errorf("getting record id: %w", err)
	}

	err = p.updateRecord(ctx, client, recordID, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("updating record: %w", err)
	}

	return ip, nil
}
