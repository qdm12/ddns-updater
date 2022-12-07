package update

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/qdm12/ddns-updater/internal/update/mock_update"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LogClient(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		requestMethod      string
		requetsHeaders     http.Header
		requestBodyNil     bool
		requestBodyString  string
		requestLineRegex   string
		responseStatusCode int
		responseBodyNil    bool
		responseBodyString string
		responseLineRegex  string
	}{
		"PUT with headers and body": {
			requestMethod: http.MethodPut,
			requetsHeaders: http.Header{
				"Key1": []string{"value 1", "value 2"},
				"Key2": []string{"value 3"},
			},
			requestBodyString: "request body",
			requestLineRegex: "PUT http://127.0.0.1:[0-9]{0,5} | " +
				"headers: Key1: value1,value 2; Key2: value 3 | " +
				"body: request body",
			responseStatusCode: http.StatusAccepted,
			responseBodyString: "response body",
			responseLineRegex: "202 Accepted | " +
				"headers: Content-Length: 9; Content-Type: text/plain; charset=utf-8; Date: .+ | " +
				"body: response body",
		},
		"simple GET": {
			requestMethod:      http.MethodGet,
			requestBodyNil:     true,
			requestLineRegex:   "GET http://127.0.0.1:[0-9]{0,5}",
			responseStatusCode: http.StatusOK,
			responseBodyString: "response body",
			responseLineRegex: "200 OK | " +
				"headers: Date: .+; Content-Length: 13; Content-Type: text/plain; charset=utf-8 | " +
				"body: response body",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			handler := http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
				// Check request matches the request we sent
				assert.Equal(t, testCase.requestMethod, request.Method)
				for key, expectedValues := range testCase.requetsHeaders {
					values := request.Header[key]
					assert.Equal(t, expectedValues, values)
				}

				if testCase.requestBodyNil {
					require.Equal(t, http.NoBody, request.Body)
				} else {
					require.NotNil(t, request.Body)
					b, err := io.ReadAll(request.Body)
					require.NoError(t, err)
					assert.Equal(t, testCase.requestBodyString, string(b))
				}

				// Send the response
				rw.WriteHeader(testCase.responseStatusCode)

				if testCase.responseBodyNil {
					return
				}
				_, err := rw.Write([]byte(testCase.responseBodyString))
				require.NoError(t, err)
			})
			server := httptest.NewServer(handler)

			client := server.Client()

			logger := mock_update.NewMockDebugLogger(ctrl)
			logger.EXPECT().Debug(gomock.AssignableToTypeOf("")).
				DoAndReturn(func(s string) {
					assert.Regexp(t, testCase.requestLineRegex, s)
				})
			logger.EXPECT().Debug(gomock.AssignableToTypeOf("")).
				DoAndReturn(func(s string) {
					assert.Regexp(t, testCase.responseLineRegex, s)
				})

			logClient := makeLogClient(client, logger)

			assert.Equal(t, logClient.Timeout, client.Timeout)

			ctx := context.Background()

			var requestBody io.Reader
			if !testCase.requestBodyNil {
				requestBody = bytes.NewBufferString(testCase.requestBodyString)
			}
			request, err := http.NewRequestWithContext(ctx,
				testCase.requestMethod, server.URL, requestBody)
			require.NoError(t, err)
			request.Header = testCase.requetsHeaders

			response, err := logClient.Do(request)
			require.NoError(t, err)

			defer require.NoError(t, response.Body.Close())

			// Ensure response received is as expected
			assert.Equal(t, testCase.responseStatusCode, response.StatusCode)
			if testCase.responseBodyNil {
				assert.Nil(t, response.Body)
				return
			}
			b, err := io.ReadAll(response.Body)
			require.NoError(t, err)
			assert.Equal(t, testCase.responseBodyString, string(b))
		})
	}
}
