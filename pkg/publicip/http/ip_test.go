package http

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
)

func Test_fetcher_IP(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const url = "c"
	httpBytes := []byte(`55.55.55.55`)
	expectedPublicIP := net.IP{55, 55, 55, 55}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, url, r.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(httpBytes)),
			}, nil
		}),
	}

	initialFetcher := &Fetcher{
		client:  client,
		timeout: time.Hour,
		ip4or6: &urlsRing{
			index: 1,
			urls:  []string{"a", "b", "c"},
		},
	}
	expectedFetcher := &Fetcher{
		client:  client,
		timeout: time.Hour,
		ip4or6: &urlsRing{
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

func Test_fetcher_IP4(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const url = "c"
	httpBytes := []byte(`55.55.55.55`)
	expectedPublicIP := net.IP{55, 55, 55, 55}

	client := &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			assert.Equal(t, url, r.URL.String())
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(httpBytes)),
			}, nil
		}),
	}

	initialFetcher := &Fetcher{
		client:  client,
		timeout: time.Hour,
		ip4: &urlsRing{
			index: 1,
			urls:  []string{"a", "b", "c"},
		},
	}
	expectedFetcher := &Fetcher{
		client:  client,
		timeout: time.Hour,
		ip4: &urlsRing{
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
	const url = "c"
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
				Body:       io.NopCloser(bytes.NewReader(httpBytes)),
			}, nil
		}),
	}

	initialFetcher := &Fetcher{
		client:  client,
		timeout: time.Hour,
		ip6: &urlsRing{
			index: 1,
			urls:  []string{"a", "b", "c"},
		},
	}
	expectedFetcher := &Fetcher{
		client:  client,
		timeout: time.Hour,
		ip6: &urlsRing{
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

	newTestClient := func(expectedURL string, status int, httpBytes []byte, httpErr error) *http.Client {
		return &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				assert.Equal(t, expectedURL, r.URL.String())
				err := r.Context().Err()
				if err != nil {
					return nil, err
				}
				if httpErr != nil {
					return nil, httpErr
				}
				return &http.Response{
					StatusCode: status,
					Body:       io.NopCloser(bytes.NewReader(httpBytes)),
				}, nil
			}),
		}
	}

	testCases := map[string]struct {
		initialFetcher *Fetcher
		ctx            context.Context
		publicIP       net.IP
		err            error
		errMessage     string
		finalFetcher   *Fetcher // client is ignored when comparing the two
	}{
		"first run": {
			ctx: context.Background(),
			initialFetcher: &Fetcher{
				timeout: time.Hour,
				client:  newTestClient("b", http.StatusOK, []byte(`55.55.55.55`), nil),
				ip4or6: &urlsRing{
					index: 0,
					urls:  []string{"a", "b"},
				},
			},
			publicIP: net.IP{55, 55, 55, 55},
			finalFetcher: &Fetcher{
				timeout: time.Hour,
				ip4or6: &urlsRing{
					index: 1,
					urls:  []string{"a", "b"},
				},
			},
		},
		"second run": {
			ctx: context.Background(),
			initialFetcher: &Fetcher{
				timeout: time.Hour,
				client:  newTestClient("a", http.StatusOK, []byte(`55.55.55.55`), nil),
				ip4or6: &urlsRing{
					index: 1,
					urls:  []string{"a", "b"},
				},
			},
			publicIP: net.IP{55, 55, 55, 55},
			finalFetcher: &Fetcher{
				timeout: time.Hour,
				ip4or6: &urlsRing{
					index: 0,
					urls:  []string{"a", "b"},
				},
			},
		},
		"zero timeout": {
			ctx: context.Background(),
			initialFetcher: &Fetcher{
				client: newTestClient("a", 0, nil, nil),
				ip4or6: &urlsRing{
					index: 1,
					urls:  []string{"a", "b"},
				},
			},
			finalFetcher: &Fetcher{
				ip4or6: &urlsRing{
					index: 0,
					urls:  []string{"a", "b"},
				},
			},
			err:        context.DeadlineExceeded,
			errMessage: `Get "a": context deadline exceeded`,
		},
		"canceled context": {
			ctx: canceledCtx,
			initialFetcher: &Fetcher{
				timeout: time.Hour,
				client:  newTestClient("a", 0, nil, nil),
				ip4or6: &urlsRing{
					index: 1,
					urls:  []string{"a", "b"},
				},
			},
			finalFetcher: &Fetcher{
				timeout: time.Hour,
				ip4or6: &urlsRing{
					index: 0,
					urls:  []string{"a", "b"},
				},
			},
			err:        context.Canceled,
			errMessage: `Get "a": context canceled`,
		},
		"try next if banned": {
			ctx: context.Background(),
			initialFetcher: &Fetcher{
				timeout: time.Hour,
				client:  newTestClient("a", http.StatusOK, []byte(`55.55.55.55`), nil),
				ip4or6: &urlsRing{
					index:  0,
					urls:   []string{"a", "b"},
					banned: map[int]string{1: "banned"},
				},
			},
			finalFetcher: &Fetcher{
				timeout: time.Hour,
				ip4or6: &urlsRing{
					index:  0,
					urls:   []string{"a", "b"},
					banned: map[int]string{1: "banned"},
				},
			},
			publicIP: net.IP{55, 55, 55, 55},
		},
		"all banned": {
			ctx: context.Background(),
			initialFetcher: &Fetcher{
				ip4or6: &urlsRing{
					index:  1,
					urls:   []string{"a", "b"},
					banned: map[int]string{0: "banned", 1: "banned again"},
				},
			},
			finalFetcher: &Fetcher{
				ip4or6: &urlsRing{
					index:  1,
					urls:   []string{"a", "b"},
					banned: map[int]string{0: "banned", 1: "banned again"},
				},
			},
			err:        ErrBanned,
			errMessage: "we got banned: banned (a), banned again (b)",
		},
		"record banned": {
			ctx: context.Background(),
			initialFetcher: &Fetcher{
				timeout: time.Hour,
				client:  newTestClient("a", http.StatusTooManyRequests, []byte(`get out`), nil),
				ip4or6: &urlsRing{
					index:  1,
					urls:   []string{"a", "b"},
					banned: map[int]string{},
				},
			},
			finalFetcher: &Fetcher{
				timeout: time.Hour,
				ip4or6: &urlsRing{
					index:  0,
					urls:   []string{"a", "b"},
					banned: map[int]string{0: "429 (get out)"},
				},
			},
			err:        ErrBanned,
			errMessage: "we got banned: 429 (get out)",
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			urlRing := testCase.initialFetcher.ip4or6

			publicIP, err := testCase.initialFetcher.ip(testCase.ctx, urlRing, ipversion.IP4or6)

			assert.ErrorIs(t, err, testCase.err)
			if testCase.err != nil {
				assert.EqualError(t, err, testCase.errMessage)
			}

			if !testCase.publicIP.Equal(publicIP) {
				t.Errorf("IP address mismatch: expected %s and got %s", testCase.publicIP, publicIP)
			}

			testCase.initialFetcher.client = nil
			assert.Equal(t, testCase.finalFetcher, testCase.initialFetcher)
		})
	}
}
