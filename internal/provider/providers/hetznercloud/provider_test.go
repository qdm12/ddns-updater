package hetznercloud

import (
	"errors"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Provider_Update(t *testing.T) {
	t.Parallel()

	type exchange struct {
		method         string
		path           string
		requestBody    string
		responseStatus int
		responseBody   string
	}

	const actionSuccess = `{"action":{"id":42,"status":"success"}}`

	testCases := map[string]struct {
		ttl        uint32
		ip         netip.Addr
		exchanges  []exchange
		wantTTL    uint32 // 0 means nil
		errMessage string
	}{
		"ipv4_rrset_updated": {
			ip: netip.MustParseAddr("203.0.113.2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/A",
				responseStatus: http.StatusOK,
				responseBody:   `{"rrset":{"records":[{"value":"203.0.113.1"}]}}`,
			}, {
				method:         http.MethodPost,
				path:           "/v1/zones/example.com/rrsets/home/A/actions/set_records",
				requestBody:    `{"records":[{"value":"203.0.113.2"}]}`,
				responseStatus: http.StatusCreated,
				responseBody:   actionSuccess,
			}},
		},
		"ipv6_rrset_updated": {
			ip: netip.MustParseAddr("2001:db8::2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/AAAA",
				responseStatus: http.StatusOK,
				responseBody:   `{"rrset":{"records":[{"value":"2001:db8::1"}]}}`,
			}, {
				method:         http.MethodPost,
				path:           "/v1/zones/example.com/rrsets/home/AAAA/actions/set_records",
				requestBody:    `{"records":[{"value":"2001:db8::2"}]}`,
				responseStatus: http.StatusCreated,
				responseBody:   actionSuccess,
			}},
		},
		"ipv4_rrset_created": {
			ttl: 60,
			ip:  netip.MustParseAddr("203.0.113.2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/A",
				responseStatus: http.StatusNotFound,
			}, {
				method:         http.MethodPost,
				path:           "/v1/zones/example.com/rrsets",
				requestBody:    `{"name":"home","type":"A","ttl":60,"records":[{"value":"203.0.113.2"}]}`,
				responseStatus: http.StatusCreated,
				responseBody:   actionSuccess,
			}},
		},
		"ipv6_rrset_created": {
			ttl: 60,
			ip:  netip.MustParseAddr("2001:db8::2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/AAAA",
				responseStatus: http.StatusNotFound,
			}, {
				method:         http.MethodPost,
				path:           "/v1/zones/example.com/rrsets",
				requestBody:    `{"name":"home","type":"AAAA","ttl":60,"records":[{"value":"2001:db8::2"}]}`,
				responseStatus: http.StatusCreated,
				responseBody:   actionSuccess,
			}},
		},
		"rrset_up_to_date": {
			ip: netip.MustParseAddr("203.0.113.2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/A",
				responseStatus: http.StatusOK,
				responseBody:   `{"rrset":{"ttl":300,"records":[{"value":"203.0.113.2"}]}}`,
			}},
			wantTTL: 300,
		},
		// An RRSet with several values is left as is when one of them is the IP address.
		"rrset_with_other_values_up_to_date": {
			ip: netip.MustParseAddr("203.0.113.2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/A",
				responseStatus: http.StatusOK,
				responseBody:   `{"rrset":{"records":[{"value":"192.0.2.10"},{"value":"203.0.113.2"}]}}`,
			}},
		},
		"get_rrset_error": {
			ip: netip.MustParseAddr("203.0.113.2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/A",
				responseStatus: http.StatusUnauthorized,
				responseBody:   `{"error":{"code":"unauthorized","message":"invalid token"}}`,
			}},
			errMessage: "getting record id: HTTP status is not valid: 401",
		},
		"update_rrset_error": {
			ip: netip.MustParseAddr("203.0.113.2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/A",
				responseStatus: http.StatusOK,
				responseBody:   `{"rrset":{"records":[{"value":"203.0.113.1"}]}}`,
			}, {
				method:         http.MethodPost,
				path:           "/v1/zones/example.com/rrsets/home/A/actions/set_records",
				requestBody:    `{"records":[{"value":"203.0.113.2"}]}`,
				responseStatus: http.StatusUnprocessableEntity,
				responseBody:   `{"error":{"code":"invalid_input","message":"invalid input"}}`,
			}},
			errMessage: "updating record: HTTP status is not valid: 422",
		},
		"update_rrset_action_error": {
			ip: netip.MustParseAddr("203.0.113.2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/A",
				responseStatus: http.StatusOK,
				responseBody:   `{"rrset":{"records":[{"value":"203.0.113.1"}]}}`,
			}, {
				method:         http.MethodPost,
				path:           "/v1/zones/example.com/rrsets/home/A/actions/set_records",
				requestBody:    `{"records":[{"value":"203.0.113.2"}]}`,
				responseStatus: http.StatusCreated,
				responseBody:   `{"action":{"id":42,"status":"error"}}`,
			}},
			errMessage: "updating record: handling action response: unsuccessful result: action id 42 failed",
		},
		"create_rrset_error": {
			ip: netip.MustParseAddr("203.0.113.2"),
			exchanges: []exchange{{
				method:         http.MethodGet,
				path:           "/v1/zones/example.com/rrsets/home/A",
				responseStatus: http.StatusNotFound,
			}, {
				method:         http.MethodPost,
				path:           "/v1/zones/example.com/rrsets",
				requestBody:    `{"name":"home","type":"A","records":[{"value":"203.0.113.2"}]}`,
				responseStatus: http.StatusUnprocessableEntity,
				responseBody:   `{"error":{"code":"invalid_input","message":"invalid input"}}`,
			}},
			errMessage: "creating record: HTTP status is not valid: 422",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			exchangeIndex := 0
			client := &http.Client{
				Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					exchangeIndex++
					if exchangeIndex > len(testCase.exchanges) {
						return nil, errors.New("unexpected request")
					}
					expected := testCase.exchanges[exchangeIndex-1]

					assert.Equal(t, expected.method, request.Method)
					assert.Equal(t, "https://api.hetzner.cloud"+expected.path, request.URL.String())
					assert.Equal(t, "Bearer token", request.Header.Get("Authorization"))
					if expected.requestBody == "" {
						assert.Nil(t, request.Body)
					} else {
						requestBody, err := io.ReadAll(request.Body)
						require.NoError(t, err)
						assert.JSONEq(t, expected.requestBody, string(requestBody))
					}

					return &http.Response{
						StatusCode: expected.responseStatus,
						Body:       io.NopCloser(strings.NewReader(expected.responseBody)),
					}, nil
				}),
			}
			provider := &Provider{
				domain: "example.com",
				owner:  "home",
				token:  "token",
				ttl:    testCase.ttl,
			}

			newIP, err := provider.Update(t.Context(), client, testCase.ip)

			assert.Equal(t, len(testCase.exchanges), exchangeIndex)
			if testCase.errMessage != "" {
				assert.ErrorContains(t, err, testCase.errMessage)
				assert.False(t, newIP.IsValid())
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, testCase.ip, newIP)
			if testCase.wantTTL == 0 {
				return
			}
			ttl := provider.TTL()
			require.NotNil(t, ttl)
			assert.Equal(t, testCase.wantTTL, *ttl)
		})
	}
}
