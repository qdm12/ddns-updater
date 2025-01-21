package transip

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	username   string
	key        string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Username string `json:"username"`
		Key      string `json:"key"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.Username, extraSettings.Key)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		username:   extraSettings.Username,
		key:        extraSettings.Key,
	}, nil
}

func validateSettings(domain, username string, key string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if username == "" {
		return errors.ErrUsernameNotSet
	}

	if key == "" {
		return errors.ErrKeyNotSet
	}

	if _, err := parsePrivateKey(key); err != nil {
		return fmt.Errorf("%w: %w", errors.ErrKeyNotValid, err)
	}

	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.TransIP, p.ipVersion)
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
		Provider:  "<a href=\"https://www.transip.nl/\">TransIP</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func parsePrivateKey(keyString string) (*rsa.PrivateKey, error) {
	pemData := strings.ReplaceAll(keyString, "\n", "")
	pemData = strings.ReplaceAll(pemData, "-----BEGIN PRIVATE KEY-----", "")
	pemData = strings.ReplaceAll(pemData, "-----END PRIVATE KEY-----", "")
	pemData = strings.TrimSpace(pemData)

	decodedKey, err := base64.StdEncoding.DecodeString(pemData)
	if err != nil {
		return nil, err
	}

	key, err := x509.ParsePKCS8PrivateKey(decodedKey)
	if err != nil {
		return nil, err
	}

	if rsaKey, ok := key.(*rsa.PrivateKey); ok {
		return rsaKey, nil
	}

	return nil, fmt.Errorf("not an RSA private key")
}

func (p *Provider) createAccessToken(ctx context.Context, client *http.Client) (string, error) {
	requestBody, err := json.Marshal(map[string]any{
		"login":      p.username,
		"nonce":      strconv.FormatInt(time.Now().UnixNano(), 10),
		"global_key": true,
		"read_only":  false,
		"label":      fmt.Sprintf("ddns-updater %d", time.Now().Unix()),
	})
	if err != nil {
		return "", fmt.Errorf("json encoding request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.transip.nl/v6/auth", bytes.NewReader(requestBody))
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")

	privateKey, err := parsePrivateKey(p.key)
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}

	hashedBody := sha512.Sum512(requestBody)
	signature, err := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA512, hashedBody[:])
	if err != nil {
		return "", fmt.Errorf("signing request: %w", err)
	}
	request.Header.Set("Signature", base64.StdEncoding.EncodeToString(signature))

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var result struct {
		Token string `json:"token"`
	}
	err = decoder.Decode(&result)
	if err != nil {
		return "", fmt.Errorf("json decoding response body: %w", err)
	}

	return result.Token, nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	token, err := p.createAccessToken(ctx, client)
	if err != nil {
		return netip.Addr{}, err
	}

	// TODO: List DNS entries, check if a new one should be created or one should be updated.

	requestBody, err := json.Marshal(map[string]any{
		"dnsEntry": map[string]any{
			"name":    p.owner,
			"expire":  300, // TODO: Use expire from existing entry.
			"type":    recordType,
			"content": ip.String(),
		},
	})
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request body: %w", err)
	}

	url := fmt.Sprintf("https://api.transip.nl/v6/domains/%s/dns", p.domain)
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(requestBody))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAuthBearer(request, token)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	return ip, nil
}
