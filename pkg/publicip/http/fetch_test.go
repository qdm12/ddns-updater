package http

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_fetch(t *testing.T) {
	t.Parallel()

	errDummy := errors.New("dummy")

	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := map[string]struct {
		ctx         context.Context
		url         string
		httpContent []byte
		httpErr     error
		publicIP    net.IP
		err         error
	}{
		"canceled context": {
			ctx: canceledCtx,
			url: "https://opendns.com/ip",
			err: errors.New(`Get "https://opendns.com/ip": context canceled`),
		},
		"http error": {
			ctx:     context.Background(),
			url:     "https://opendns.com/ip",
			httpErr: errDummy,
			err:     errors.New(`Get "https://opendns.com/ip": dummy`),
		},
		"empty response": {
			ctx: context.Background(),
			url: "https://opendns.com/ip",
			err: errors.New(`no IP address found`),
		},
		"no IP in response": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			httpContent: []byte(`abc def`),
			err:         errors.New(`no IP address found`),
		},
		"ipv4 and ipv6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			httpContent: []byte(`1.67.201.251 ::0`),
			err:         errors.New(`too many IP addresses: found 1 IPv4 addresses and 1 IPv6 addresses, instead of a single one`), //nolint:lll
		},
		"too many ipv4": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			httpContent: []byte(`1.67.201.251 1.67.201.251`),
			err:         errors.New(`too many IP addresses: found 2 IP addresses instead of a single one`),
		},
		"too many ipv6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			httpContent: []byte(`::0 ::0`),
			err:         errors.New(`too many IP addresses: found 2 IP addresses instead of a single one`),
		},
		"ipv4": {
			ctx:         context.Background(),
			httpContent: []byte("blabla 1.67.201.251 25.35.55 sds"),
			publicIP:    net.IP{1, 67, 201, 251},
		},
		"ipv6": {
			ctx:         context.Background(),
			httpContent: []byte("blabla ad07:e846:51ac:6cd0:0000:0000:0000:0000 sds"),
			publicIP:    net.IP{0xad, 0x7, 0xe8, 0x46, 0x51, 0xac, 0x6c, 0xd0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, tc.url, r.URL.String())
					if err := r.Context().Err(); err != nil {
						return nil, err
					} else if tc.httpErr != nil {
						return nil, tc.httpErr
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       ioutil.NopCloser(bytes.NewReader(tc.httpContent)),
					}, nil
				}),
			}

			publicIP, err := fetch(tc.ctx, client, tc.url)

			if tc.err != nil {
				require.Error(t, err)
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			if !tc.publicIP.Equal(publicIP) {
				t.Errorf("IP address mismatch: expected %s and got %s", tc.publicIP, publicIP)
			}
		})
	}
}
