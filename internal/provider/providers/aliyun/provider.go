package aliyun

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
	domain       string
	owner        string
	ipVersion    ipversion.IPVersion
	ipv6Suffix   netip.Prefix
	accessKeyID  string
	accessSecret string
	region       string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		AccessKeyID  string `json:"access_key_id"`
		AccessSecret string `json:"access_secret"`
		Region       string `json:"region"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	region := "cn-hangzhou"
	if extraSettings.Region != "" {
		region = extraSettings.Region
	}

	err = validateSettings(domain, extraSettings.AccessKeyID, extraSettings.AccessSecret)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:       domain,
		owner:        owner,
		ipVersion:    ipVersion,
		ipv6Suffix:   ipv6Suffix,
		accessKeyID:  extraSettings.AccessKeyID,
		accessSecret: extraSettings.AccessSecret,
		region:       region,
	}, nil
}

func validateSettings(domain, accessKeyID, accessSecret string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case accessKeyID == "":
		return fmt.Errorf("%w", errors.ErrAccessKeyIDNotSet)
	case accessSecret == "":
		return fmt.Errorf("%w", errors.ErrAccessKeySecretNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Aliyun, p.ipVersion)
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
		Provider:  "<a href=\"https://www.aliyun.com/\">Aliyun</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// Documentation at https://api.aliyun.com/
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	recordID, err := p.getRecordID(ctx, client, recordType)
	if stderrors.Is(err, errors.ErrRecordNotFound) {
		recordID, err = p.createRecord(ctx, client, ip)
		if err != nil {
			return newIP, fmt.Errorf("creating record: %w", err)
		}
	} else if err != nil {
		return newIP, fmt.Errorf("getting record id: %w", err)
	}

	err = p.updateRecord(ctx, client, recordID, ip)
	if err != nil {
		return newIP, fmt.Errorf("updating record: %w", err)
	}

	return ip, nil
}
