package hetznercloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rewriteTransport redirects all requests (to api.hetzner.cloud) to the
// test server, without having to make the provider code test-aware.
type rewriteTransport struct {
	host string
	base http.RoundTripper
}

func (t rewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.URL.Scheme = "http"
	request.URL.Host = t.host
	return t.base.RoundTrip(request)
}

// testToken is a dummy API token used by the hetznercloud provider tests.
const testToken = "token"

// bodyResponse is a single HTTP response served by the test API, in the order
// the requests come in.
type bodyResponse struct {
	statusCode int
	body       string
}

// newTestAPIClient returns an HTTP client sending every request aimed at
// api.hetzner.cloud to the given handler.
func newTestAPIClient(t *testing.T, handler http.HandlerFunc) (client *http.Client) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return &http.Client{Transport: rewriteTransport{
		host: strings.TrimPrefix(server.URL, "http://"),
		base: http.DefaultTransport,
	}}
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
		// Regression: the action of a set_records/create operation belongs to a
		// zone, so it must be polled at /v1/zones/actions/{id}.
		// Previously /v1/servers/actions/{id} was used by mistake, which 404s.
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

func Test_waitAction_polling(t *testing.T) {
	t.Parallel()

	const actionID = 42
	runningBody := fmt.Sprintf(`{"action":{"id":%d,"status":"running"}}`, actionID)
	successBody := fmt.Sprintf(`{"action":{"id":%d,"status":"success"}}`, actionID)

	testCases := map[string]struct {
		actionID    uint64
		responses   []bodyResponse
		wantQueries uint
		errWrapped  error
		errMessage  string
	}{
		"zero_action_id": {
			actionID:   0,
			errWrapped: errors.ErrReceivedNoResult,
		},
		// A Zone action is asynchronous and takes around 20 seconds to reach
		// the success status, so a running status must be polled instead of
		// being given up on.
		"running_until_success": {
			actionID: actionID,
			responses: []bodyResponse{
				{http.StatusOK, runningBody},
				{http.StatusOK, runningBody},
				{http.StatusOK, successBody},
			},
			wantQueries: 3,
		},
		// A 404 on the action lookup must be reported instead of being polled
		// again, since the action id will not appear later.
		"not_found": {
			actionID: actionID,
			responses: []bodyResponse{{
				http.StatusNotFound,
				`{"error":{"code":"not_found","message":"action not found"}}`,
			}},
			wantQueries: 1,
			errWrapped:  errors.ErrHTTPStatusNotValid,
			errMessage:  "404",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var queries uint
			remaining := testCase.responses
			client := newTestAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
				if len(remaining) == 0 {
					t.Errorf("unexpected request number %d", queries+1)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				response := remaining[0]
				remaining = remaining[1:]
				queries++
				w.WriteHeader(response.statusCode)
				_, _ = io.WriteString(w, response.body)
			})

			// The poll period is shortened to keep the test fast.
			provider := &Provider{token: testToken, actionPollPeriod: time.Millisecond}

			err := provider.waitAction(t.Context(), client, testCase.actionID)

			assert.Equal(t, testCase.wantQueries, queries)
			if testCase.errWrapped != nil {
				require.ErrorIs(t, err, testCase.errWrapped)
				if testCase.errMessage != "" {
					assert.Contains(t, err.Error(), testCase.errMessage)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Test_waitAction_timeout checks waitAction gives up with an error once the
// Zone action did not reach a final status within the allotted tries.
func Test_waitAction_timeout(t *testing.T) {
	t.Parallel()

	const actionID = 42
	runningBody := fmt.Sprintf(`{"action":{"id":%d,"status":"running"}}`, actionID)

	var queries uint
	client := newTestAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		queries++
		_, _ = io.WriteString(w, runningBody)
	})

	// The poll period is shortened to keep the test fast.
	provider := &Provider{token: testToken, actionPollPeriod: time.Millisecond}

	err := provider.waitAction(t.Context(), client, actionID)

	require.ErrorIs(t, err, errors.ErrUnsuccessful)
	assert.Contains(t, err.Error(), "did not complete")
	// The action is polled exactly the maximum number of tries.
	const wantQueries uint = 12
	assert.Equal(t, wantQueries, queries)
}
