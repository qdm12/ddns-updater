package hetznercloud

import (
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateReportsHetznerCloudAPIErrors(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		statusCode int
		code       string
		message    string
	}{
		"unauthorized": {
			statusCode: http.StatusUnauthorized,
			code:       "unauthorized",
			message:    "invalid token",
		},
		"rate limited": {
			statusCode: http.StatusTooManyRequests,
			code:       "rate_limit_exceeded",
			message:    "too many requests",
		},
		"server error": {
			statusCode: http.StatusInternalServerError,
			code:       "server_error",
			message:    "temporary failure",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/v1/zones/example.com/rrsets/home/A", r.URL.Path)
				w.WriteHeader(testCase.statusCode)
				_, _ = fmt.Fprintf(w,
					"{\"error\":{\"code\":%q,\"message\":%q}}",
					testCase.code, testCase.message)
			})
			provider := &Provider{
				domain:    "example.com",
				owner:     "home",
				ipVersion: ipversion.IP4,
				token:     "token",
			}

			_, err := provider.Update(t.Context(), newProviderTestClient(t, handler),
				netip.MustParseAddr("203.0.113.2"))

			require.Error(t, err)
			assert.Contains(t, err.Error(), strconv.Itoa(testCase.statusCode))
			assert.Contains(t, err.Error(), testCase.message)
		})
	}
}
