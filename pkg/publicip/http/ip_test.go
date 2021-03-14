package http

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fetcher_IP(t *testing.T) { //nolint:dupl
	t.Parallel()

	ctx := context.Background()
	const url = "b"
	httpBytes := []byte(`55.55.55.55`)
	expectedPublicIP := net.IP{55, 55, 55, 55}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, url, r.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(httpBytes)),
			}, nil
		}),
	}

	initialFetcher := &fetcher{
		client:  client,
		timeout: time.Hour,
		ip4or6: urlsRing{
			index: 1,
			urls:  []string{"a", "b", "c"},
		},
	}
	expectedFetcher := &fetcher{
		client:  client,
		timeout: time.Hour,
		ip4or6: urlsRing{
			index: 2,
			urls:  []string{"a", "b", "c"},
		},
	}

	publicIP, err := initialFetcher.IP(ctx)

	assert.NoError(t, err)
	if !expectedPublicIP.Equal(publicIP) {
		t.Errorf("IP address mismatch: expected %s and got %s", expectedPublicIP, publicIP)
	}
	assert.Equal(t, expectedFetcher, initialFetcher)
}

func Test_fetcher_IP4(t *testing.T) { //nolint:dupl
	t.Parallel()

	ctx := context.Background()
	const url = "b"
	httpBytes := []byte(`55.55.55.55`)
	expectedPublicIP := net.IP{55, 55, 55, 55}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, url, r.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(httpBytes)),
			}, nil
		}),
	}

	initialFetcher := &fetcher{
		client:  client,
		timeout: time.Hour,
		ip4: urlsRing{
			index: 1,
			urls:  []string{"a", "b", "c"},
		},
	}
	expectedFetcher := &fetcher{
		client:  client,
		timeout: time.Hour,
		ip4: urlsRing{
			index: 2,
			urls:  []string{"a", "b", "c"},
		},
	}

	publicIP, err := initialFetcher.IP4(ctx)

	assert.NoError(t, err)
	if !expectedPublicIP.Equal(publicIP) {
		t.Errorf("IP address mismatch: expected %s and got %s", expectedPublicIP, publicIP)
	}
	assert.Equal(t, expectedFetcher, initialFetcher)
}

func Test_fetcher_IP6(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const url = "b"
	httpBytes := []byte(`::1`)
	expectedPublicIP := net.IP{
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
	}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, url, r.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewReader(httpBytes)),
			}, nil
		}),
	}

	initialFetcher := &fetcher{
		client:  client,
		timeout: time.Hour,
		ip6: urlsRing{
			index: 1,
			urls:  []string{"a", "b", "c"},
		},
	}
	expectedFetcher := &fetcher{
		client:  client,
		timeout: time.Hour,
		ip6: urlsRing{
			index: 2,
			urls:  []string{"a", "b", "c"},
		},
	}

	publicIP, err := initialFetcher.IP6(ctx)

	assert.NoError(t, err)
	if !expectedPublicIP.Equal(publicIP) {
		t.Errorf("IP address mismatch: expected %s and got %s", expectedPublicIP, publicIP)
	}
	assert.Equal(t, expectedFetcher, initialFetcher)
}

func Test_fetcher_ip(t *testing.T) {
	t.Parallel()

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	newTestClient := func(expectedURL string, httpBytes []byte, httpErr error) *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, expectedURL, r.URL.String())
				if err := r.Context().Err(); err != nil {
					return nil, err
				}
				if httpErr != nil {
					return nil, httpErr
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       ioutil.NopCloser(bytes.NewReader(httpBytes)),
				}, nil
			}),
		}
	}

	testCases := map[string]struct {
		initialFetcher *fetcher
		ctx            context.Context
		publicIP       net.IP
		err            error
		finalFetcher   *fetcher // client is ignored when comparing the two
	}{
		"first run": {
			ctx: context.Background(),
			initialFetcher: &fetcher{
				timeout: time.Hour,
				client:  newTestClient("a", []byte(`55.55.55.55`), nil),
				ip4or6: urlsRing{
					index: 0,
					urls:  []string{"a", "b"},
				},
			},
			publicIP: net.IP{55, 55, 55, 55},
			finalFetcher: &fetcher{
				timeout: time.Hour,
				ip4or6: urlsRing{
					index: 1,
					urls:  []string{"a", "b"},
				},
			},
		},
		"second run": {
			ctx: context.Background(),
			initialFetcher: &fetcher{
				timeout: time.Hour,
				client:  newTestClient("b", []byte(`55.55.55.55`), nil),
				ip4or6: urlsRing{
					index: 1,
					urls:  []string{"a", "b"},
				},
			},
			publicIP: net.IP{55, 55, 55, 55},
			finalFetcher: &fetcher{
				timeout: time.Hour,
				ip4or6: urlsRing{
					index: 0,
					urls:  []string{"a", "b"},
				},
			},
		},
		"zero timeout": {
			ctx: context.Background(),
			initialFetcher: &fetcher{
				client: newTestClient("a", nil, nil),
				ip4or6: urlsRing{
					index: 0,
					urls:  []string{"a", "b"},
				},
			},
			finalFetcher: &fetcher{
				ip4or6: urlsRing{
					index: 1,
					urls:  []string{"a", "b"},
				},
			},
			err: errors.New(`Get "a": context deadline exceeded`),
		},
		"canceled context": {
			ctx: canceledCtx,
			initialFetcher: &fetcher{
				timeout: time.Hour,
				client:  newTestClient("a", nil, nil),
				ip4or6: urlsRing{
					index: 0,
					urls:  []string{"a", "b"},
				},
			},
			finalFetcher: &fetcher{
				timeout: time.Hour,
				ip4or6: urlsRing{
					index: 1,
					urls:  []string{"a", "b"},
				},
			},
			err: errors.New(`Get "a": context canceled`),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			publicIP, err := testCase.initialFetcher.ip(testCase.ctx, ipversion.IP4or6)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			if !testCase.publicIP.Equal(publicIP) {
				t.Errorf("IP address mismatch: expected %s and got %s", testCase.publicIP, publicIP)
			}

			testCase.initialFetcher.client = nil
			assert.Equal(t, testCase.finalFetcher, testCase.initialFetcher)
		})
	}
}
