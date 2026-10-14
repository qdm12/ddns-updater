package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRejectsUnexpectedHTTPStatus(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewBufferString("1.67.201.251")),
			}, nil
		}),
	}

	_, err := fetch(context.Background(), client, "https://example.com/ip", ipversion.IP4)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrHTTPStatus)
	assert.Contains(t, err.Error(), "500")
}

func TestFetchTreatsRateLimitAsBan(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(bytes.NewBufferString("slow down")),
			}, nil
		}),
	}

	_, err := fetch(context.Background(), client, "https://example.com/ip", ipversion.IP4)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBanned)
}

func TestFetchRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(bytes.NewReader(
					bytes.Repeat([]byte("x"), int(maxPublicIPResponseBytes)+1))),
			}, nil
		}),
	}

	_, err := fetch(context.Background(), client, "https://example.com/ip", ipversion.IP4)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrResponseTooLarge)
}
