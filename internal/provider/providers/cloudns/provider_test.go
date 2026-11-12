package cloudns_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/providers/cloudns"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_New(t *testing.T) {
	t.Parallel()

	const dynamicURL = "https://ipv4.cloudns.net/api/dynamicURL/?q=SECRET_TOKEN"
	testCases := map[string]struct {
		url        string
		rawData    string
		domain     string
		ipVersion  ipversion.IPVersion
		errMessage string
	}{
		"valid_ipv4":       {url: dynamicURL, ipVersion: ipversion.IP4},
		"valid_ipv6":       {url: "https://ipv6.cloudns.net/api/dynamicURL/?q=SECRET_TOKEN", ipVersion: ipversion.IP6},
		"opaque_token":     {url: dynamicURL + "%2B%2F%3D", ipVersion: ipversion.IP4},
		"missing_url":      {errMessage: "url is not set"},
		"invalid_json":     {rawData: `{"url":"SECRET_TOKEN"`, errMessage: "json decoding"},
		"invalid_url_type": {rawData: `{"url":{"SECRET_TOKEN":true}}`, errMessage: "json decoding"},
		"invalid_url":      {url: "https://%SECRET_TOKEN", errMessage: "url is malformed"},
		"http":             {url: strings.Replace(dynamicURL, "https:", "http:", 1), errMessage: "must use HTTPS"},
		"unsupported_host": {
			url:        strings.Replace(dynamicURL, "ipv4.cloudns.net", "evil.example", 1),
			errMessage: "url hostname must be",
		},
		"host_suffix": {
			url:        strings.Replace(dynamicURL, "ipv4.cloudns.net", "ipv4.cloudns.net.evil.example", 1),
			errMessage: "url hostname must be",
		},
		"port": {
			url:        strings.Replace(dynamicURL, "ipv4.cloudns.net", "ipv4.cloudns.net:444", 1),
			errMessage: "url hostname must be",
		},
		"userinfo": {
			url:        strings.Replace(dynamicURL, "https://", "https://SECRET_TOKEN@", 1),
			errMessage: "user information",
		},
		"fragment":   {url: dynamicURL + "#SECRET_TOKEN", errMessage: "fragment"},
		"wrong_path": {url: strings.Replace(dynamicURL, "/api/dynamicURL/", "/", 1), errMessage: "endpoint"},
		"missing_q": {
			url: "https://ipv4.cloudns.net/api/dynamicURL/", ipVersion: ipversion.IP4,
			errMessage: "single non-empty q token",
		},
		"empty_q": {
			url: "https://ipv4.cloudns.net/api/dynamicURL/?q=", ipVersion: ipversion.IP4,
			errMessage: "single non-empty q token",
		},
		"duplicate_q": {
			url: dynamicURL + "&q=SECOND_SECRET", ipVersion: ipversion.IP4,
			errMessage: "single non-empty q token",
		},
		"malformed_query": {url: dynamicURL + "%zz", ipVersion: ipversion.IP4, errMessage: "query is malformed"},
		"extra_ip":        {url: dynamicURL + "&ip=192.0.2.1", ipVersion: ipversion.IP4, errMessage: "only the q token"},
		"extra_ipv4":      {url: dynamicURL + "&ipv4=192.0.2.1", ipVersion: ipversion.IP4, errMessage: "only the q token"},
		"extra_ipv6": {
			url: dynamicURL + "&ipv6=2001:db8::1", ipVersion: ipversion.IP4,
			errMessage: "only the q token",
		},
		"extra_myip":        {url: dynamicURL + "&myip=192.0.2.1", ipVersion: ipversion.IP4, errMessage: "only the q token"},
		"extra_proxy":       {url: dynamicURL + "&proxy=1", ipVersion: ipversion.IP4, errMessage: "only the q token"},
		"automatic_version": {url: dynamicURL, errMessage: "ip_version must explicitly match"},
		"ipv4_mismatch":     {url: dynamicURL, ipVersion: ipversion.IP6, errMessage: "ip_version must explicitly match"},
		"ipv6_mismatch": {
			url: strings.Replace(dynamicURL, "ipv4.", "ipv6.", 1), ipVersion: ipversion.IP4,
			errMessage: "ip_version must explicitly match",
		},
		"invalid_domain": {url: dynamicURL, domain: "invalid domain", errMessage: "domain is not valid"},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			domain := testCase.domain
			if domain == "" {
				domain = "example.com"
			}
			data, err := json.Marshal(map[string]string{"url": testCase.url})
			require.NoError(t, err)
			if testCase.rawData != "" {
				data = json.RawMessage(testCase.rawData)
			}
			result, err := cloudns.New(data, domain, "@", testCase.ipVersion, netip.Prefix{})
			if testCase.errMessage == "" {
				require.NoError(t, err)
				require.NotNil(t, result)
				return
			}
			require.ErrorContains(t, err, testCase.errMessage)
			assert.Nil(t, result)
			assert.NotContains(t, err.Error(), "SECRET_TOKEN")
			assert.NotContains(t, err.Error(), "SECOND_SECRET")
			assert.NotContains(t, err.Error(), "https://")
		})
	}
}

func Test_Provider_metadata(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{"@": "example.com", "home": "home.example.com", "*": "any.example.com"}
	for owner, hostname := range testCases {
		t.Run(owner, func(t *testing.T) {
			t.Parallel()
			suffix := netip.MustParsePrefix("::1234/64")
			result, err := provider.New(constants.ClouDNS,
				json.RawMessage(`{"url":"https://ipv6.cloudns.net/api/dynamicURL/?q=SECRET_TOKEN"}`),
				"example.com", owner, ipversion.IP6, suffix)
			require.NoError(t, err)
			assert.Contains(t, constants.ProviderChoices(), models.Provider("cloudns"))
			assert.Equal(t, "example.com", result.Domain())
			assert.Equal(t, owner, result.Owner())
			assert.Equal(t, hostname, result.BuildDomainName())
			assert.Equal(t, ipversion.IP6, result.IPVersion())
			assert.Equal(t, suffix, result.IPv6Suffix())
			assert.False(t, result.Proxied())
			assert.Contains(t, result.String(), "cloudns")
			assert.NotContains(t, result.String(), "SECRET_TOKEN")
			assert.Equal(t, models.HTMLRow{
				Domain: `<a href="http://` + hostname + `">` + hostname + `</a>`,
				Owner:  owner, Provider: `<a href="https://www.cloudns.net/">ClouDNS</a>`, IPVersion: "ipv6",
			}, result.HTML())
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type responseBody struct {
	reader io.Reader
	closed bool
}

func (body *responseBody) Read(data []byte) (int, error) {
	return body.reader.Read(data)
}

func (body *responseBody) Close() error {
	body.closed = true
	return nil
}

func Test_Provider_Update(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		body       string
		statusCode int
		ipv6       bool
		networkErr bool
		readErr    bool
		canceled   bool
		deadline   bool
		contextErr error
		errMessage string
	}{
		"success_ipv4":       {body: "OK"},
		"success_ipv6":       {body: "OK", ipv6: true},
		"success_whitespace": {body: "OK\n"},
		"invalid_request":    {body: "Invalid request.", errMessage: "invalid Dynamic URL request"},
		"unauthorized": {
			statusCode: http.StatusUnauthorized, body: "SECRET_TOKEN",
			errMessage: "authorization rejected",
		},
		"forbidden": {statusCode: http.StatusForbidden, body: "SECRET_TOKEN", errMessage: "authorization rejected"},
		"server_error": {
			statusCode: http.StatusInternalServerError, body: "OK",
			errMessage: "HTTP status is not valid: 500",
		},
		"unexpected":       {body: "SECRET_TOKEN", errMessage: "unexpected Dynamic URL response"},
		"empty":            {errMessage: "unexpected Dynamic URL response"},
		"lowercase_ok":     {body: "ok", errMessage: "unexpected Dynamic URL response"},
		"ok_prefix":        {body: "OK error", errMessage: "unexpected Dynamic URL response"},
		"network_error":    {networkErr: true, errMessage: "network or HTTP client error"},
		"reading_error":    {readErr: true, errMessage: "reading response body"},
		"context_canceled": {canceled: true, contextErr: context.Canceled, errMessage: "context canceled"},
		"context_deadline": {deadline: true, contextErr: context.DeadlineExceeded, errMessage: "context deadline exceeded"},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ipVersion := ipversion.IP4
			ip := netip.MustParseAddr("192.0.2.1")
			dynamicURL := "https://ipv4.cloudns.net/api/dynamicURL/?q=SECRET_TOKEN%2B%2F%3D"
			if testCase.ipv6 {
				ipVersion = ipversion.IP6
				ip = netip.MustParseAddr("2001:db8::1")
				dynamicURL = strings.Replace(dynamicURL, "ipv4.", "ipv6.", 1)
			}
			data, err := json.Marshal(map[string]string{"url": dynamicURL})
			require.NoError(t, err)
			result, err := cloudns.New(data, "example.com", "@", ipVersion, netip.Prefix{})
			require.NoError(t, err)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if testCase.canceled {
				cancel()
			}
			body := &responseBody{reader: strings.NewReader(testCase.body)}
			if testCase.readErr {
				body.reader = errorReader{}
			}
			calls := 0
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				calls++
				assert.Equal(t, ctx, request.Context())
				assert.Equal(t, http.MethodGet, request.Method)
				assert.Equal(t, dynamicURL, request.URL.String())
				assert.NotContains(t, request.URL.String(), ip.String())
				assert.Nil(t, request.Body)
				assert.NotEmpty(t, request.UserAgent())
				assert.Empty(t, request.Header.Get("X-Forwarded-For"))
				assert.Empty(t, request.Header.Get("Authorization"))
				for _, key := range [...]string{"ip", "ipv4", "ipv6", "myip", "proxy"} {
					assert.False(t, request.URL.Query().Has(key))
				}
				switch {
				case testCase.canceled:
					return nil, request.Context().Err()
				case testCase.deadline:
					return nil, context.DeadlineExceeded
				case testCase.networkErr:
					return nil, errors.New("network error " + dynamicURL)
				}
				statusCode := testCase.statusCode
				if statusCode == 0 {
					statusCode = http.StatusOK
				}
				return &http.Response{StatusCode: statusCode, Body: body, Header: make(http.Header)}, nil
			})}

			newIP, err := result.Update(ctx, client, ip)
			assert.Equal(t, 1, calls)
			if testCase.errMessage == "" {
				require.NoError(t, err)
				assert.Equal(t, ip, newIP)
			} else {
				require.ErrorContains(t, err, testCase.errMessage)
				assert.NotContains(t, err.Error(), "SECRET_TOKEN")
				assert.NotContains(t, err.Error(), dynamicURL)
				assert.False(t, newIP.IsValid())
			}
			if testCase.contextErr != nil {
				assert.ErrorIs(t, err, testCase.contextErr)
			}
			assert.Equal(t, !testCase.networkErr && !testCase.canceled && !testCase.deadline, body.closed)
		})
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) {
	return 0, errors.New("read error containing SECRET_TOKEN")
}
