package hetznercloud

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport leitet alle Anfragen (an api.hetzner.cloud) auf den
// Test-Server um, ohne den Provider-Code für Tests anpassen zu müssen.
type rewriteTransport struct {
	host string
	base http.RoundTripper
}

func (t rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.URL.Scheme = "http"
	request.URL.Host = t.host
	return t.base.RoundTrip(request)
}

func Test_waitAction(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		actionID   uint64
		statusCode int
		body       string
		wantPath   string
		wantErr    error
	}{
		// Regression: die Action einer set_records-/create-Operation gehört zu
		// einer Zone, muss also unter /v1/zones/actions/{id} gepollt werden.
		// Zuvor wurde fälschlich /v1/servers/actions/{id} verwendet (404).
		"success_polls_zone_action_endpoint": {
			actionID:   42,
			statusCode: http.StatusOK,
			body:       `{"action":{"id":42,"status":"success"}}`,
			wantPath:   "/v1/zones/actions/42",
		},
		"error_status_is_reported": {
			actionID:   43,
			statusCode: http.StatusOK,
			body:       `{"action":{"id":43,"status":"error","error":{"code":"failed","message":"boom"}}}`,
			wantPath:   "/v1/zones/actions/43",
			wantErr:    errors.ErrUnsuccessful,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var gotPath string
			handler := func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(testCase.statusCode)
				_, _ = io.WriteString(w, testCase.body)
			}
			server := httptest.NewServer(http.HandlerFunc(handler))
			defer server.Close()

			client := &http.Client{Transport: rewriteTransport{
				host: strings.TrimPrefix(server.URL, "http://"),
				base: http.DefaultTransport,
			}}

			provider := &Provider{token: "token"}

			err := provider.waitAction(context.Background(), client, testCase.actionID)

			assert.Equal(t, testCase.wantPath, gotPath)
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
