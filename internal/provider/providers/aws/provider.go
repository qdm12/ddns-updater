package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	awsAccessKeyID     string
	awsSecretAccessKey string
	domain             string
	host               string
	hostedZoneID       string
	ipVersion          ipversion.IPVersion
	ipv6Suffix         netip.Prefix
	region             string
	ttl                int64
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error) {
	var providerSpecificSettings settings
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("decoding provider specific settings: %w", err)
	}

	err = validateSettings(providerSpecificSettings, domain, host)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}
	ttl := int64(300)
	if providerSpecificSettings.TTL != nil {
		ttl = *providerSpecificSettings.TTL
	}

	region := "us-east-1"
	if providerSpecificSettings.Region != "" {
		region = providerSpecificSettings.Region
	}

	return &Provider{
		domain:             domain,
		host:               host,
		ipVersion:          ipVersion,
		ipv6Suffix:         ipv6Suffix,
		awsAccessKeyID:     providerSpecificSettings.AwsAccessKey,
		awsSecretAccessKey: providerSpecificSettings.AwsSecretAccessKey,
		hostedZoneID:       providerSpecificSettings.HostedZoneID,
		region:             region,
		ttl:                ttl,
	}, nil
}

type settings struct {
	AwsAccessKey       string `json:"aws_access_key"`
	AwsSecretAccessKey string `json:"aws_secret_access_key"`
	Region             string `json:"region"`
	HostedZoneID       string `json:"hosted_zone_id"`
	TTL                *int64 `json:"ttl"`
}

func validateSettings(providerSpecificSettings settings, domain, host string) error {
	switch {
	case domain == "":
		return fmt.Errorf("%w", errors.ErrDomainNotSet)
	case host == "":
		return fmt.Errorf("%w", errors.ErrHostNotSet)
	case providerSpecificSettings.AwsAccessKey == "":
		return fmt.Errorf("%w", errors.ErrAccessKeyIDNotSet)
	case providerSpecificSettings.AwsSecretAccessKey == "":
		return fmt.Errorf("%w", errors.ErrSecretAccessKeyNotSet)
	case providerSpecificSettings.HostedZoneID == "":
		return fmt.Errorf("%w", errors.ErrHostedZoneIDNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.AWS, p.ipVersion)
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
		Provider:  "<a href=\"https://aws.amazon.com/route53/\">Amazon Route 53</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// API details https://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html
	// GO API https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/route53#Client.ChangeTagsForResource

	recordType := types.RRTypeA
	if p.ipVersion == ipversion.IP6 {
		recordType = types.RRTypeAaaa
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: []types.Change{
				{
					Action: types.ChangeActionUpsert, // Uses update or insert feature from Route53
					ResourceRecordSet: &types.ResourceRecordSet{
						Name: aws.String(p.BuildDomainName()),
						Type: recordType,
						ResourceRecords: []types.ResourceRecord{
							{
								Value: aws.String(ip.String()),
							},
						},
						TTL: aws.Int64(p.ttl),
						// SetIdentifier not set means Simple Routing,
					},
				},
			},
			Comment: aws.String("Record updated by ddns-updater"),
		},
		HostedZoneId: aws.String(p.hostedZoneID),
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(p.awsAccessKeyID, p.awsSecretAccessKey, "")),
		config.WithRegion(p.region),
	)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("failed loading default config %w", err)
	}

	// TODO: implement option to wait for propagation
	svcRoute53 := route53.NewFromConfig(cfg)
	_, err = svcRoute53.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w", err)
	}
	return ip, nil
}
