package namecom

import (
	"context"
	"encoding/json"
	stderrors "errors"
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
	token     string
	ttl       *uint32
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Username string  `json:"username"`
		Token    string  `json:"token"`
		TTL      *uint32 `json:"ttl,omitempty"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	const minTTL = 300
	switch {
	case extraSettings.Username == "":
		return nil, fmt.Errorf("%w", errors.ErrEmptyUsername)
	case extraSettings.Token == "":
		return nil, fmt.Errorf("%w", errors.ErrEmptyPassword)
	case extraSettings.TTL != nil && *extraSettings.TTL < minTTL:
		return nil, fmt.Errorf("%w: %d must be at least %d",
			errors.ErrTTLTooLow, *extraSettings.TTL, minTTL)
	}

	return &Provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		username:  extraSettings.Username,
		token:     extraSettings.Token,
		ttl:       extraSettings.TTL,
	}, nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.NameCom, p.ipVersion)
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
		Provider:  "<a href=\"https://name.com\">Name.com</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// Documentation at https://www.name.com/api-docs
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	recordID, err := p.getRecordID(ctx, client, recordType)

	if stderrors.Is(err, errors.ErrRecordNotFound) {
		err = p.createRecord(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrCreateRecord, err)
		}

		return ip, nil
	} else if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrGetRecordID, err)
	}

	err = p.updateRecord(ctx, client, recordID, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUpdateRecord, err)
	}

	return ip, nil
}
