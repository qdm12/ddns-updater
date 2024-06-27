package gcp

import (
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain      string
	owner       string
	ipVersion   ipversion.IPVersion
	ipv6Suffix  netip.Prefix
	project     string
	zone        string
	credentials json.RawMessage
}

func New(data json.RawMessage, domain, owner string,
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

	err = validateSettings(domain, extraSettings.Project, extraSettings.Zone, extraSettings.Credentials)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:      domain,
		owner:       owner,
		ipVersion:   ipVersion,
		ipv6Suffix:  ipv6Suffix,
		project:     extraSettings.Project,
		zone:        extraSettings.Zone,
		credentials: extraSettings.Credentials,
	}, nil
}

func validateSettings(domain, project, zone string, credentials json.RawMessage) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case project == "":
		return fmt.Errorf("%w", errors.ErrGCPProjectNotSet)
	case zone == "":
		return fmt.Errorf("%w", errors.ErrZoneIdentifierNotSet)
	case len(credentials) == 0:
		return fmt.Errorf("%w", errors.ErrCredentialsNotSet)
	}
	var creds struct {
		Type string `json:"type"`
	}
	err = json.Unmarshal(credentials, &creds)
	if err != nil || creds.Type == "" {
		return fmt.Errorf("%w: 'type' JSON field value missing",
			errors.ErrCredentialsNotValid)
	}

	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.GCP, p.ipVersion)
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
		Provider:  "<a href=\"https://cloud.google.com/\">Google Cloud</a>",
		IPVersion: p.ipVersion.String(),
	}
}
