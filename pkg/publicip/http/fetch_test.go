package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
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
		version     ipversion.IPVersion
		httpContent []byte
		httpErr     error
		publicIP    netip.Addr
		err         error
	}{
		"canceled context": {
			ctx:     canceledCtx,
			url:     "https://opendns.com/ip",
			version: ipversion.IP4or6,
			err:     errors.New(`Get "https://opendns.com/ip": context canceled`),
		},
		"http error": {
			ctx:     context.Background(),
			url:     "https://opendns.com/ip",
			version: ipversion.IP4or6,
			httpErr: errDummy,
			err:     errors.New(`Get "https://opendns.com/ip": dummy`),
		},
		"empty response": {
			ctx:     context.Background(),
			url:     "https://opendns.com/ip",
			version: ipversion.IP4or6,
			err:     errors.New(`no IP address found: from "https://opendns.com/ip"`),
		},
		"no IP for IP4or6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4or6,
			httpContent: []byte(`abc def`),
			err:         errors.New(`no IP address found: from "https://opendns.com/ip"`),
		},
		"single IPv4 for IP4or6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4or6,
			httpContent: []byte(`1.67.201.251`),
			publicIP:    netip.AddrFrom4([4]byte{1, 67, 201, 251}),
		},
		"single IPv6 for IP4or6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4or6,
			httpContent: []byte(`::1`),
			publicIP: netip.AddrFrom16([16]byte{
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 1,
			}),
		},
		"IPv4 and IPv6 for IP4or6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4or6,
			httpContent: []byte(`1.67.201.251 ::1`),
			publicIP:    netip.AddrFrom4([4]byte{1, 67, 201, 251}),
		},
		"too many IPv4s for IP4or6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4or6,
			httpContent: []byte(`1.67.201.251 1.67.201.252`),
			err:         errors.New("too many IP addresses: found 2 IPv4 addresses instead of 1"),
		},
		"too many IPv6s for IP4or6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4or6,
			httpContent: []byte(`::1 ::2`),
			err:         errors.New("too many IP addresses: found 2 IPv6 addresses instead of 1"),
		},
		"no IP for IP4": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4,
			httpContent: []byte(`abc def`),
			err:         errors.New(`no IP address found: from "https://opendns.com/ip" for version ipv4`),
		},
		"single IPv4 for IP4": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4,
			httpContent: []byte(`1.67.201.251`),
			publicIP:    netip.AddrFrom4([4]byte{1, 67, 201, 251}),
		},
		"too many IPv4s for IP4": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP4,
			httpContent: []byte(`1.67.201.251 1.67.201.252`),
			err:         errors.New("too many IP addresses: found 2 IPv4 addresses instead of 1"),
		},
		"no IP for IP6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP6,
			httpContent: []byte(`abc def`),
			err:         errors.New(`no IP address found: from "https://opendns.com/ip" for version ipv6`),
		},
		"single IPv6 for IP6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP6,
			httpContent: []byte(`::1`),
			publicIP: netip.AddrFrom16([16]byte{
				0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 1,
			}),
		},
		"too many IPv6s for IP6": {
			ctx:         context.Background(),
			url:         "https://opendns.com/ip",
			version:     ipversion.IP6,
			httpContent: []byte(`::1 ::2`),
			err:         errors.New("too many IP addresses: found 2 IPv6 addresses instead of 1"),
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := &http.Client{
				Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					assert.Equal(t, tc.url, r.URL.String())
					err := r.Context().Err()
					if err != nil {
						return nil, err
					} else if tc.httpErr != nil {
						return nil, tc.httpErr
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(tc.httpContent)),
					}, nil
				}),
			}

			publicIP, err := fetch(tc.ctx, client, tc.url, tc.version)

			if tc.err != nil {
				require.Error(t, err)
				assert.Equal(t, tc.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			if tc.publicIP.Compare(publicIP) != 0 {
				t.Errorf("IP address mismatch: expected %s and got %s", tc.publicIP, publicIP)
			}
		})
	}
}
