package myaddr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	key        string
	name       string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
}

func New(data json.RawMessage, ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (p *Provider, err error) {
	var providerSpecificSettings struct {
		Key string `json:"key"`
	}
	if err = json.Unmarshal(data, &providerSpecificSettings); err != nil {
		err = fmt.Errorf("json decoding provider specific settings (myaddr): %w", err)
		return
	}
	if providerSpecificSettings.Key == "" {
		err = fmt.Errorf("validating provider specific settings (myaddr): %w", errors.ErrKeyNotSet)
		return
	}
	p = &Provider{
		key:        providerSpecificSettings.Key,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
	}
	return
}

func (p *Provider) Init(ctx context.Context, client *http.Client) (err error) {
	// generate HTTP request to get the name corresponding to the key
	v := url.Values{}
	v.Set("key", p.key)
	u := &url.URL{
		Scheme:   "https",
		Host:     "myaddr.tools",
		Path:     "/reg",
		RawQuery: v.Encode(),
	}
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil); err != nil {
		err = fmt.Errorf("initializing provider (myaddr): creating http request: %w", err)
		return
	}
	headers.SetUserAgent(req)

	// execute HTTP request
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		err = fmt.Errorf("initializing provider (myaddr): %w", err)
		return
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body) // ensure body is read to completion

	// handle HTTP response
	switch resp.StatusCode {
	case http.StatusOK:
		// key is valid, decode response to get name
		var respBody struct {
			Name string `json:"name"`
		}
		if err = json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			err = fmt.Errorf("initializing provider (myaddr): json decoding http response: %w", err)
			return
		}
		p.name = respBody.Name
	case http.StatusBadRequest, http.StatusNotFound:
		// key is invalid
		err = fmt.Errorf("initializing provider (myaddr): %w", errors.ErrKeyNotValid)
	default:
		// unexpected HTTP status
		err = fmt.Errorf("initializing provider (myaddr): %w: %d: %s",
			errors.ErrHTTPStatusNotValid, resp.StatusCode, utils.BodyToSingleLine(resp.Body))
	}
	return
}

func (p *Provider) String() string {
	return utils.ToString(p.Domain(), p.Owner(), constants.Myaddr, p.IPVersion())
}

func (p *Provider) Domain() string {
	return p.name + ".myaddr.tools"
}

func (p *Provider) Owner() string {
	return "@"
}

func (p *Provider) BuildDomainName() string {
	return p.Domain()
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.Domain(), p.Domain()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://myaddr.tools/\">myaddr</a>",
		IPVersion: p.IPVersion().String(),
	}
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// generate HTTP request
	u := &url.URL{
		Scheme: "https",
		Host:   "myaddr.tools",
		Path:   "/update",
	}
	v := url.Values{}
	v.Set("key", p.key)
	v.Set("ip", ip.String())
	b := strings.NewReader(v.Encode())
	var req *http.Request
	if req, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), b); err != nil {
		err = fmt.Errorf("creating http request: %w", err)
		return
	}
	headers.SetContentType(req, "application/x-www-form-urlencoded")
	headers.SetUserAgent(req)

	// execute HTTP request
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()
	defer io.Copy(io.Discard, resp.Body) // ensure body is read to completion

	// handle HTTP response
	switch resp.StatusCode {
	case http.StatusOK:
		// update was successful
		newIP = ip
	case http.StatusBadRequest:
		// bad request
		err = fmt.Errorf("%w: %s", errors.ErrBadRequest, utils.BodyToSingleLine(resp.Body))
	case http.StatusNotFound:
		// registration not found
		err = fmt.Errorf("%w: %s", errors.ErrKeyNotValid, utils.BodyToSingleLine(resp.Body))
	default:
		// unexpected HTTP status
		err = fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, resp.StatusCode, utils.BodyToSingleLine(resp.Body))
	}
	return
}
