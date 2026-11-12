package hetznercloud

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_handleErrorResponse(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		statusCode int
		body       string
		wantErr    string
	}{
		"error_with_code_and_message": {
			statusCode: http.StatusUnauthorized,
			body:       `{"error":{"code":"unauthorized","message":"invalid token"}}`,
			wantErr:    "HTTP status is not valid: 401: code: unauthorized: message: invalid token",
		},
		"error_with_details": {
			statusCode: http.StatusUnprocessableEntity,
			body:       `{"error":{"code":"invalid_input","message":"invalid input","details":{"fields":["ttl"]}}}`,
			wantErr:    "HTTP status is not valid: 422: code: invalid_input: message: invalid input: details: map[fields:[ttl]]",
		},
		"body_not_json": {
			statusCode: http.StatusBadGateway,
			body:       "<html>\nBad Gateway\n</html>",
			wantErr:    "HTTP status is not valid: 502: <html>Bad Gateway</html>",
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			response := &http.Response{
				StatusCode: testCase.statusCode,
				Body:       io.NopCloser(strings.NewReader(testCase.body)),
			}

			err := handleErrorResponse(response)

			assert.ErrorContains(t, err, testCase.wantErr)
		})
	}
}
