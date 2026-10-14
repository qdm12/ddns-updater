package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type updateForcerFunc func(ctx context.Context) []error

func (f updateForcerFunc) ForceUpdate(ctx context.Context) []error {
	return f(ctx)
}

func TestUpdateGETIsNotAllowed(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"root URL":    "/update",
		"URL subpath": "/ddns/update",
	}
	for name, path := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			called := false
			runner := updateForcerFunc(func(_ context.Context) []error {
				called = true
				return nil
			})
			rootURL := path[:len(path)-len("/update")]
			handler := newHandler(context.Background(), rootURL, nil, runner)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
			assert.Equal(t, http.MethodPost, response.Header().Get("Allow"))
			assert.False(t, called)
		})
	}
}

func TestUpdatePOST(t *testing.T) {
	t.Parallel()

	testCases := map[string]string{
		"root URL":    "/update",
		"URL subpath": "/ddns/update",
	}
	for name, path := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			called := false
			runner := updateForcerFunc(func(_ context.Context) []error {
				called = true
				return nil
			})
			rootURL := path[:len(path)-len("/update")]
			handler := newHandler(context.Background(), rootURL, nil, runner)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assert.Equal(t, http.StatusAccepted, response.Code)
			assert.True(t, called)
		})
	}
}
