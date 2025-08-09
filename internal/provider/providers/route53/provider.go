package route53

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	zoneID     string
	ttl        uint32
	session    *session.Session
	accessKey  string // For static credentials
	secretKey  string // For static credentials
}

func New(data json.RawMessage, domain, owner string, ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (*Provider, error) {
	var settings struct {
		AccessKey  string  `json:"access_key"`
		SecretKey  string  `json:"secret_key"`
		AWSProfile string  `json:"aws_profile"`
		ZoneID     string  `json:"zone_id"`
		TTL        *uint32 `json:"ttl,omitempty"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("decoding provider specific settings: %w", err)
	}

	if err := validateSettings(domain, settings.AccessKey, settings.SecretKey, settings.AWSProfile, settings.ZoneID); err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	const defaultTTL = 300
	ttl := defaultTTL
	if settings.TTL != nil {
		ttl = int(*settings.TTL)
	}

	var sess *session.Session
	var accessKey, secretKey string

	if settings.AWSProfile != "" {
		fmt.Println("Using AWS profile:", settings.AWSProfile)
		var err error
		sess, err = session.NewSessionWithOptions(session.Options{
			Profile:           settings.AWSProfile,
			SharedConfigState: session.SharedConfigEnable,
		})
		if err != nil {
			return nil, fmt.Errorf("creating AWS session: %w", err)
		}

		// Verify credentials
		_, err = sess.Config.Credentials.Get()
		if err == nil {
			stsSvc := sts.New(sess)
			identity, err := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
			if err == nil {
				fmt.Println("STS Identity:", *identity.Arn)
			} else {
				fmt.Println("Could not verify identity with STS:", err)
			}
		} else {
			return nil, fmt.Errorf("resolving credentials from profile: %w", err)
		}
	} else {
		fmt.Println("Using access key and secret key")
		// Store credentials for static credential creation
		accessKey = settings.AccessKey
		secretKey = settings.SecretKey

		// Create a basic session for non-profile usage
		var err error
		sess, err = session.NewSession()
		if err != nil {
			return nil, fmt.Errorf("creating AWS session: %w", err)
		}
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		zoneID:     settings.ZoneID,
		ttl:        uint32(ttl),
		session:    sess,
		accessKey:  accessKey,
		secretKey:  secretKey,
	}, nil
}

func validateSettings(domain, accessKey, secretKey, awsProfile, zoneID string) error {
	if err := utils.CheckDomain(domain); err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if awsProfile != "" {
		if zoneID == "" {
			return fmt.Errorf("%w", errors.ErrZoneIdentifierNotSet)
		}
		return nil // AWS SDK is expected to resolve credentials
	}

	switch {
	case accessKey == "":
		return fmt.Errorf("%w", errors.ErrAccessKeyNotSet)
	case secretKey == "":
		return fmt.Errorf("%w", errors.ErrSecretKeyNotSet)
	case zoneID == "":
		return fmt.Errorf("%w", errors.ErrZoneIdentifierNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Route53, p.ipVersion)
}

func (p *Provider) Domain() string                 { return p.domain }
func (p *Provider) Owner() string                  { return p.owner }
func (p *Provider) IPVersion() ipversion.IPVersion { return p.ipVersion }
func (p *Provider) IPv6Suffix() netip.Prefix       { return p.ipv6Suffix }
func (p *Provider) Proxied() bool                  { return false }
func (p *Provider) BuildDomainName() string        { return utils.BuildDomainName(p.owner, p.domain) }

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://aws.amazon.com/route53/\">Amazon Route 53</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (netip.Addr, error) {
	signer := p.createSigner()
	return updateRecord(ctx, client, signer, p.zoneID, p.BuildDomainName(), p.ttl, ip)
}

func (p *Provider) createSigner() *Route53Signer {
	if p.accessKey != "" && p.secretKey != "" {
		// Use static credentials
		creds := credentials.NewStaticCredentials(p.accessKey, p.secretKey, "")
		return NewRoute53Signer(creds)
	}
	// Use session credentials (for profile-based authentication)
	return NewRoute53Signer(p.session.Config.Credentials)
}
