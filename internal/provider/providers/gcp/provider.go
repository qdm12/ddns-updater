package gcp

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain      string
	host        string
	ipVersion   ipversion.IPVersion
	ipv6Suffix  netip.Prefix
	project     string
	zone        string
	credentials json.RawMessage
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	var extraSettings struct {
		Project     string          `json:"project"`
		Zone        string          `json:"zone"`
		Credentials json.RawMessage `json:"credentials"`
	}

	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding extra settings: %w", err)
	}

	p = &Provider{
		domain:      domain,
		host:        host,
		ipVersion:   ipVersion,
		ipv6Suffix:  ipv6Suffix,
		project:     extraSettings.Project,
		zone:        extraSettings.Zone,
		credentials: extraSettings.Credentials,
	}

	err = p.isValid()
	if err != nil {
		return nil, fmt.Errorf("configuration is not valid: %w", err)
	}

	return p, nil
}

func (p *Provider) isValid() error {
	if p.project == "" {
		return fmt.Errorf("%w", ddnserrors.ErrGCPProjectNotSet)
	}

	if p.zone == "" {
		return fmt.Errorf("%w", ddnserrors.ErrZoneIdentifierNotSet)
	}

	if len(p.credentials) == 0 {
		return fmt.Errorf("%w", ddnserrors.ErrCredentialsNotSet)
	}

	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.GCP, p.ipVersion)
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
		Provider:  "<a href=\"https://cloud.google.com/\">Google Cloud</a>",
		IPVersion: p.ipVersion.String(),
	}
}
