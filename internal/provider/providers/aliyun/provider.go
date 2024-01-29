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
	host         string
	ipVersion    ipversion.IPVersion
	ipv6Suffix   netip.Prefix
	accessKeyID  string
	accessSecret string
	region       string
}

func New(data json.RawMessage, domain, host string,
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
	p = &Provider{
		domain:       domain,
		host:         host,
		ipVersion:    ipVersion,
		ipv6Suffix:   ipv6Suffix,
		accessKeyID:  extraSettings.AccessKeyID,
		accessSecret: extraSettings.AccessSecret,
		region:       "cn-hangzhou",
	}
	if extraSettings.Region != "" {
		p.region = extraSettings.Region
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.accessKeyID == "":
		return fmt.Errorf("%w", errors.ErrAccessKeyIDNotSet)
	case p.accessSecret == "":
		return fmt.Errorf("%w", errors.ErrAccessKeySecretNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Aliyun, p.ipVersion)
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
