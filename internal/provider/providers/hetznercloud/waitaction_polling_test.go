package hetznercloud

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	providererrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type actionRoundTripper func(*http.Request) (*http.Response, error)

func (f actionRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type actionResponseBody struct {
	io.Reader
	closed *bool
}

func (b actionResponseBody) Close() error {
	*b.closed = true
	return nil
}

func TestWaitActionAfterMultipleRunningResponses(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		finalResponse string
		wantErr       error
	}{
		"eventual success": {
			finalResponse: `{"action":{"id":42,"status":"success"}}`,
		},
		"eventual action error": {
			finalResponse: `{"action":{"id":42,"status":"error","error":{"code":"failed","message":"boom"}}}`,
			wantErr:       providererrors.ErrUnsuccessful,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				const runningResponses = 4
				requests := 0
				bodyClosed := true
				client := &http.Client{Transport: actionRoundTripper(func(request *http.Request) (*http.Response, error) {
					assert.Equal(t, "/v1/zones/actions/42", request.URL.Path)
					assert.Equal(t, "Bearer token", request.Header.Get("Authorization"))
					assert.True(t, bodyClosed, "each response must close before the next poll")
					require.Less(t, requests, runningResponses+1)
					requests++
					body := `{"action":{"id":42,"status":"running"}}`
					if requests > runningResponses {
						body = testCase.finalResponse
					}
					bodyClosed = false
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       actionResponseBody{Reader: strings.NewReader(body), closed: &bodyClosed},
					}, nil
				})}
				provider := &Provider{token: "token"}

				err := provider.waitAction(t.Context(), client, 42)

				if testCase.wantErr != nil {
					require.ErrorIs(t, err, testCase.wantErr)
					assert.ErrorContains(t, err, "boom")
				} else {
					require.NoError(t, err)
				}
				assert.Equal(t, runningResponses+1, requests)
				assert.True(t, bodyClosed)
			})
		})
	}
}

func TestWaitActionContextLimits(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		parentTimeout time.Duration
		cancelAfter   time.Duration
		inFlight      bool
		wantErr       error
		wantElapsed   time.Duration
		wantRequests  int
	}{
		"maximum polling duration": {
			wantErr: context.DeadlineExceeded, wantElapsed: time.Minute,
		},
		"maximum duration includes pending HTTP request": {
			inFlight: true, wantErr: context.DeadlineExceeded, wantElapsed: time.Minute, wantRequests: 1,
		},
		"parent deadline during delay": {
			parentTimeout: 100 * time.Millisecond,
			wantErr:       context.DeadlineExceeded, wantElapsed: 100 * time.Millisecond, wantRequests: 1,
		},
		"parent deadline during HTTP request": {
			parentTimeout: 100 * time.Millisecond, inFlight: true,
			wantErr: context.DeadlineExceeded, wantElapsed: 100 * time.Millisecond, wantRequests: 1,
		},
		"cancellation during delay": {
			cancelAfter: 100 * time.Millisecond,
			wantErr:     context.Canceled, wantElapsed: 100 * time.Millisecond, wantRequests: 1,
		},
		"cancellation during HTTP request": {
			cancelAfter: 100 * time.Millisecond, inFlight: true,
			wantErr: context.Canceled, wantElapsed: 100 * time.Millisecond, wantRequests: 1,
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) {
				ctx := t.Context()
				if testCase.parentTimeout > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, testCase.parentTimeout)
					defer cancel()
				}
				if testCase.cancelAfter > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithCancel(ctx)
					defer cancel()
					time.AfterFunc(testCase.cancelAfter, cancel)
				}
				requests := 0
				client := &http.Client{Transport: actionRoundTripper(func(request *http.Request) (*http.Response, error) {
					requests++
					if testCase.inFlight {
						_, hasDeadline := request.Context().Deadline()
						require.True(t, hasDeadline, "the request must have a finite deadline")
						<-request.Context().Done()
						return nil, request.Context().Err()
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"action":{"id":42,"status":"running"}}`)),
					}, nil
				})}
				provider := &Provider{token: "token"}
				start := time.Now()

				err := provider.waitAction(ctx, client, 42)

				require.ErrorIs(t, err, testCase.wantErr)
				if testCase.wantRequests > 0 {
					assert.Equal(t, testCase.wantRequests, requests)
				} else {
					// The poll timer and deadline may both become ready at the limit.
					assert.GreaterOrEqual(t, requests, 60)
					assert.LessOrEqual(t, requests, 61)
				}
				assert.Equal(t, testCase.wantElapsed, time.Since(start))
			})
		})
	}
}
