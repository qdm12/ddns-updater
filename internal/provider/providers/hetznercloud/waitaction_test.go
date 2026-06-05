package hetznercloud

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc lets us mock the HTTP client transport.
type roundTripFunc func(request *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request), nil
}

// bodyResponse describes a single HTTP response returned by the mock transport.
type bodyResponse struct {
	statusCode int
	body       string
}

func Test_Provider_waitAction(t *testing.T) {
	t.Parallel()

	const actionID = 12345
	runningBody := fmt.Sprintf(`{"action":{"id":%d,"status":"running"}}`, actionID)
	successBody := fmt.Sprintf(`{"action":{"id":%d,"status":"success"}}`, actionID)

	testCases := map[string]struct {
		id           uint64
		responses    []bodyResponse
		wantQueries  int
		errWrapped   error
		errSubstring string
	}{
		"zero_id": {
			id:         0,
			errWrapped: errors.ErrReceivedNoResult,
		},
		"action_success": {
			id:          actionID,
			responses:   []bodyResponse{{http.StatusOK, successBody}},
			wantQueries: 1,
		},
		// The action is asynchronous, so a "running" status must be polled until
		// it reaches "success".
		"running_then_success": {
			id: actionID,
			responses: []bodyResponse{
				{http.StatusOK, runningBody},
				{http.StatusOK, runningBody},
				{http.StatusOK, successBody},
			},
			wantQueries: 3,
		},
		// Regression test for https://github.com/qdm12/ddns-updater/issues/1136:
		// the action originates from a Zone RRSet operation, so querying the
		// Server actions endpoint returned 404 (action not found) and wrongly
		// marked successful updates as failed.
		"not_found_on_action_lookup": {
			id: actionID,
			responses: []bodyResponse{{
				http.StatusNotFound,
				`{"error":{"code":"not_found","message":"action not found"}}`,
			}},
			wantQueries:  1,
			errWrapped:   errors.ErrHTTPStatusNotValid,
			errSubstring: "404",
		},
		// The Hetzner Cloud API returns the action error as an object
		// {"code": ..., "message": ...}, not a bare string.
		// See https://docs.hetzner.cloud/reference/cloud#tag/zone-actions/get_zone_action
		"action_error": {
			id: actionID,
			responses: []bodyResponse{{
				http.StatusOK,
				fmt.Sprintf(`{"action":{"id":%d,"status":"error",`+
					`"error":{"code":"action_failed","message":"Action failed"}}}`, actionID),
			}},
			wantQueries:  1,
			errWrapped:   errors.ErrUnsuccessful,
			errSubstring: "action_failed",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var queries int
			var lastURL string
			client := &http.Client{
				Transport: roundTripFunc(func(request *http.Request) *http.Response {
					lastURL = request.URL.String()
					response := testCase.responses[queries]
					queries++
					return &http.Response{
						StatusCode: response.statusCode,
						Body:       io.NopCloser(strings.NewReader(response.body)),
					}
				}),
			}

			// Use a tiny poll period to keep the test fast.
			provider := &Provider{token: testToken, actionPollPeriod: time.Millisecond}
			err := provider.waitAction(context.Background(), client, testCase.id)

			assert.Equal(t, testCase.wantQueries, queries)
			if testCase.wantQueries > 0 {
				// The action must be queried through the Zone actions endpoint,
				// not the Server actions endpoint (issue #1136).
				assert.Equal(t,
					fmt.Sprintf("https://api.hetzner.cloud/v1/zones/actions/%d", actionID),
					lastURL)
			}

			if testCase.errWrapped != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, testCase.errWrapped)
				if testCase.errSubstring != "" {
					assert.Contains(t, err.Error(), testCase.errSubstring)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test_Provider_waitAction_timeout ensures waitAction gives up with an error
// once the action did not complete within the allotted number of tries.
func Test_Provider_waitAction_timeout(t *testing.T) {
	t.Parallel()

	const actionID = 12345
	runningBody := fmt.Sprintf(`{"action":{"id":%d,"status":"running"}}`, actionID)

	var queries int
	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) *http.Response {
			queries++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(runningBody)),
			}
		}),
	}

	provider := &Provider{token: testToken, actionPollPeriod: time.Millisecond}
	err := provider.waitAction(context.Background(), client, actionID)

	require.Error(t, err)
	assert.ErrorIs(t, err, errors.ErrUnsuccessful)
	assert.Contains(t, err.Error(), "did not complete")
	// The action is polled exactly the maximum number of tries.
	assert.Equal(t, 12, queries)
}
