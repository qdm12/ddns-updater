package gcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"golang.org/x/oauth2/google"
)

type Provider struct {
	domain      string
	owner       string
	ipVersion   ipversion.IPVersion
	ipv6Suffix  netip.Prefix
	project     string
	zone        string
	credentials json.RawMessage
	credType    google.CredentialsType
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	var extraSettings struct {
		Project     string          `json:"project"`
		Zone        string          `json:"zone"`
		Credentials json.RawMessage `json:"credentials"`
	}

	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding extra settings: %w", err)
	}

	credType, err := validateSettings(domain, extraSettings.Project, extraSettings.Zone, extraSettings.Credentials)
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
		credType:    credType,
	}, nil
}

func validateSettings(domain, project, zone string, credentials json.RawMessage) (
	googleCredType google.CredentialsType, err error,
) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return "", fmt.Errorf("domain is not valid: %w", err)
	}

	switch {
	case project == "":
		return "", errors.New("GCP project is not set")
	case zone == "":
		return "", errors.New("zone identifier is not set")
	case len(credentials) == 0:
		return "", errors.New("credentials are not set")
	}
	var creds struct {
		Type string `json:"type"`
	}
	err = json.Unmarshal(credentials, &creds)
	switch {
	case err != nil:
		return "", fmt.Errorf("credentials are not valid: json malformed: %w", err)
	case creds.Type == "":
		return "", errors.New("credentials are not valid: missing 'type' field in credentials JSON")
	}
	googleCredType, err = parseCredentialsType(creds.Type)
	if err != nil {
		return "", fmt.Errorf("parsing credentials type: %w", err)
	}
	return googleCredType, nil
}

func parseCredentialsType(s string) (credentialsType google.CredentialsType, err error) {
	available := [...]google.CredentialsType{
		google.ServiceAccount,
		google.AuthorizedUser,
		google.ExternalAccount,
		google.ExternalAccountAuthorizedUser,
		google.ImpersonatedServiceAccount,
		google.GDCHServiceAccount,
	}
	for _, credType := range available {
		if strings.EqualFold(s, string(credType)) {
			return credType, nil
		}
	}
	return "", fmt.Errorf("credentials type not valid: %q", s)
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
