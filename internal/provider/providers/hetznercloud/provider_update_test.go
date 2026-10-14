package hetznercloud

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProviderTestClient(t *testing.T, handler http.Handler) *http.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &http.Client{Transport: rewriteTransport{
		host: strings.TrimPrefix(server.URL, "http://"),
		base: http.DefaultTransport,
	}}
}

func TestUpdateExistingHetznerCloudRRSet(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version    ipversion.IPVersion
		recordType string
		oldIP      string
		newIP      string
	}{
		"IPv4": {ipversion.IP4, "A", "203.0.113.1", "203.0.113.2"},
		"IPv6": {ipversion.IP6, "AAAA", "2001:db8::1", "2001:db8::2"},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			postCount := 0
			getPath := "/v1/zones/example.com/rrsets/home/" + testCase.recordType
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				switch {
				case r.Method == http.MethodGet && r.URL.Path == getPath:
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprintf(w, "{\"rrset\":{\"records\":[{\"value\":%q}]}}", testCase.oldIP)
				case r.Method == http.MethodPost && r.URL.Path == getPath+"/actions/set_records":
					postCount++
					var request struct {
						Records []record `json:"records"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					require.Len(t, request.Records, 1)
					assert.Equal(t, testCase.newIP, request.Records[0].Value)
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"action":{"id":42,"status":"success"}}`))
				default:
					http.Error(w, "unexpected route", http.StatusNotFound)
				}
			})
			provider := &Provider{
				domain:    "example.com",
				owner:     "home",
				ipVersion: testCase.version,
				token:     "token",
			}
			newIP := netip.MustParseAddr(testCase.newIP)

			updatedIP, err := provider.Update(t.Context(), newProviderTestClient(t, handler), newIP)

			require.NoError(t, err)
			assert.Equal(t, newIP, updatedIP)
			assert.Equal(t, 1, postCount)
		})
	}
}

func TestUpdateCreatesMissingHetznerCloudRRSet(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version    ipversion.IPVersion
		recordType string
		ip         string
	}{
		"IPv4": {ipversion.IP4, "A", "203.0.113.2"},
		"IPv6": {ipversion.IP6, "AAAA", "2001:db8::2"},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			postCount := 0
			getPath := "/v1/zones/example.com/rrsets/home/" + testCase.recordType
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				switch {
				case r.Method == http.MethodGet && r.URL.Path == getPath:
					http.NotFound(w, r)
				case r.Method == http.MethodPost && r.URL.Path == "/v1/zones/example.com/rrsets":
					postCount++
					var request struct {
						Name    string   `json:"name"`
						Type    string   `json:"type"`
						TTL     uint32   `json:"ttl"`
						Records []record `json:"records"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
					assert.Equal(t, "home", request.Name)
					assert.Equal(t, testCase.recordType, request.Type)
					assert.Equal(t, uint32(60), request.TTL)
					require.Len(t, request.Records, 1)
					assert.Equal(t, testCase.ip, request.Records[0].Value)
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"action":{"id":42,"status":"success"}}`))
				default:
					http.Error(w, "unexpected route", http.StatusNotFound)
				}
			})
			provider := &Provider{
				domain:    "example.com",
				owner:     "home",
				ipVersion: testCase.version,
				token:     "token",
				ttl:       60,
			}
			ip := netip.MustParseAddr(testCase.ip)

			updatedIP, err := provider.Update(t.Context(), newProviderTestClient(t, handler), ip)

			require.NoError(t, err)
			assert.Equal(t, ip, updatedIP)
			assert.Equal(t, 1, postCount)
		})
	}
}

func TestUpdateSkipsUpToDateHetznerCloudRRSet(t *testing.T) {
	t.Parallel()

	requests := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/zones/example.com/rrsets/home/A", r.URL.Path)
		_, _ = w.Write([]byte(`{"rrset":{"records":[{"value":"203.0.113.2"}]}}`))
	})
	provider := &Provider{
		domain:    "example.com",
		owner:     "home",
		ipVersion: ipversion.IP4,
		token:     "token",
	}
	ip := netip.MustParseAddr("203.0.113.2")

	updatedIP, err := provider.Update(t.Context(), newProviderTestClient(t, handler), ip)

	require.NoError(t, err)
	assert.Equal(t, ip, updatedIP)
	assert.Equal(t, 1, requests)
}

func TestUpdateLeavesMixedRRSetWhenDesiredIPExists(t *testing.T) {
	t.Parallel()

	requests := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/zones/example.com/rrsets/home/A", r.URL.Path)
		_, _ = w.Write([]byte(`{"rrset":{"records":[{"value":"203.0.113.2"},{"value":"192.0.2.10"}]}}`))
	})
	provider := &Provider{
		domain:    "example.com",
		owner:     "home",
		ipVersion: ipversion.IP4,
		token:     "token",
	}
	ip := netip.MustParseAddr("203.0.113.2")

	updatedIP, err := provider.Update(t.Context(), newProviderTestClient(t, handler), ip)

	require.NoError(t, err)
	assert.Equal(t, ip, updatedIP)
	assert.Equal(t, 1, requests, "a matching value currently prevents RRSet reconciliation")
}
