package hetznercloud

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_waitAction(t *testing.T) {
	t.Parallel()

	const running = `{"action":{"id":42,"status":"running"}}`
	const success = `{"action":{"id":42,"status":"success"}}`
	const failed = `{"action":{"id":42,"status":"error","error":{"code":"failed","message":"boom"}}}`

	testCases := map[string]struct {
		actionID uint64
		// contextTimeout is the timeout of the context given, if it is not zero.
		contextTimeout time.Duration
		statusCode     int
		// bodies are the response bodies returned in order, the last one being
		// returned for any further request. Requests block until their context
		// is done if there is no body.
		bodies   []string
		wantPath string
		// wantRequests lists the numbers of requests accepted.
		wantRequests []int
		wantElapsed  time.Duration
		wantErr      string
		wantErrIs    error
	}{
		// Regression: the action of a set_records/create operation belongs to a
		// zone, so it must be polled at /v1/zones/actions/{id}.
		// Previously /v1/servers/actions/{id} was used by mistake, which 404s.
		"success_polls_zone_action_endpoint": {
			actionID:     42,
			statusCode:   http.StatusOK,
			bodies:       []string{success},
			wantPath:     "/v1/zones/actions/42",
			wantRequests: []int{1},
		},
		"error_status_is_reported": {
			actionID:     42,
			statusCode:   http.StatusOK,
			bodies:       []string{failed},
			wantPath:     "/v1/zones/actions/42",
			wantRequests: []int{1},
			wantErr:      "unsuccessful result: action id 42 failed",
		},
		"success_after_running": {
			actionID:     42,
			statusCode:   http.StatusOK,
			bodies:       []string{running, running, running, running, success},
			wantPath:     "/v1/zones/actions/42",
			wantRequests: []int{5},
			wantElapsed:  4 * time.Second,
		},
		"error_after_running": {
			actionID:     42,
			statusCode:   http.StatusOK,
			bodies:       []string{running, failed},
			wantPath:     "/v1/zones/actions/42",
			wantRequests: []int{2},
			wantElapsed:  time.Second,
			wantErr:      "unsuccessful result: action id 42 failed",
		},
		"http_error_status": {
			actionID:     42,
			statusCode:   http.StatusInternalServerError,
			bodies:       []string{`{"error":{"code":"server_error","message":"boom"}}`},
			wantPath:     "/v1/zones/actions/42",
			wantRequests: []int{1},
			wantErr:      "getting action: HTTP status is not valid: 500",
		},
		"context_done_between_requests": {
			actionID:       42,
			contextTimeout: 1500 * time.Millisecond,
			statusCode:     http.StatusOK,
			bodies:         []string{running},
			wantPath:       "/v1/zones/actions/42",
			wantRequests:   []int{2},
			wantElapsed:    1500 * time.Millisecond,
			wantErrIs:      context.DeadlineExceeded,
		},
		"context_done_during_request": {
			actionID:       42,
			contextTimeout: 1500 * time.Millisecond,
			wantPath:       "/v1/zones/actions/42",
			wantRequests:   []int{1},
			wantElapsed:    1500 * time.Millisecond,
			wantErrIs:      context.DeadlineExceeded,
		},
		"maximum_wait_during_request": {
			actionID:     42,
			wantPath:     "/v1/zones/actions/42",
			wantRequests: []int{1},
			wantElapsed:  time.Minute,
			wantErr:      "unsuccessful result: action id 42 did not complete after 1m0s",
		},
		// At the maximum wait, both the timer between requests and the context
		// are done, so one more request may be sent.
		"maximum_wait_between_requests": {
			actionID:     42,
			statusCode:   http.StatusOK,
			bodies:       []string{running},
			wantPath:     "/v1/zones/actions/42",
			wantRequests: []int{60, 61},
			wantElapsed:  time.Minute,
			wantErr:      "unsuccessful result: action id 42 did not complete after 1m0s",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// The fake clock of synctest makes the test instant and its
			// elapsed durations exact.
			synctest.Test(t, func(t *testing.T) {
				ctx := t.Context()
				if testCase.contextTimeout != 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, testCase.contextTimeout)
					defer cancel()
				}

				var gotPath string
				requests := 0
				client := &http.Client{
					Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
						gotPath = request.URL.Path
						requests++
						if len(testCase.bodies) == 0 {
							<-request.Context().Done()
							return nil, request.Context().Err()
						}
						body := testCase.bodies[min(requests, len(testCase.bodies))-1]
						return &http.Response{
							StatusCode: testCase.statusCode,
							Body:       io.NopCloser(strings.NewReader(body)),
						}, nil
					}),
				}
				provider := &Provider{token: "token"}
				start := time.Now()

				err := provider.waitAction(ctx, client, testCase.actionID)

				assert.Equal(t, testCase.wantPath, gotPath)
				assert.Equal(t, testCase.wantElapsed, time.Since(start))
				assert.Contains(t, testCase.wantRequests, requests)
				switch {
				case testCase.wantErrIs != nil:
					require.ErrorIs(t, err, testCase.wantErrIs)
				case testCase.wantErr != "":
					require.ErrorContains(t, err, testCase.wantErr)
				default:
					require.NoError(t, err)
				}
			})
		})
	}
}
