package http

import (
	"errors"
	"net/url"
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ListProvidersForVersion(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		version   ipversion.IPVersion
		providers []Provider
	}{
		"ip4or6": {
			version: ipversion.IP4or6,
			providers: []Provider{Google, Ifconfig, Ipify, Ipinfo, Spdyn, Ipleak,
				Icanhazip, Ident, Nnev, Wtfismyip, Seeip, Changeip},
		},
		"ip4": {
			version:   ipversion.IP4,
			providers: []Provider{Ipify, Ipleak, Icanhazip, Ident, Nnev, Wtfismyip, Seeip},
		},
		"ip6": {
			version:   ipversion.IP6,
			providers: []Provider{Ipify, Ipleak, Icanhazip, Ident, Nnev, Wtfismyip, Seeip},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			providers := ListProvidersForVersion(testCase.version)
			assert.Equal(t, testCase.providers, providers)
		})
	}
}

func Test_ValidateProvider(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		provider Provider
		version  ipversion.IPVersion
		err      error
	}{
		"valid": {
			provider: Google,
			version:  ipversion.IP4or6,
		},
		"custom url": {
			provider: Provider("url:https://ip.com"),
			version:  ipversion.IP4or6,
		},
		"invalid for ip version": {
			provider: Google,
			version:  ipversion.IP4,
			err:      errors.New(`provider does not support IP version: "google" for version ipv4`),
		},
		"unknown": {
			provider: Provider("unknown"),
			version:  ipversion.IP4,
			err:      errors.New("unknown public IP echo HTTP provider: unknown"),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := ValidateProvider(testCase.provider, testCase.version)
			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_customurl(t *testing.T) {
	t.Parallel()
	url := &url.URL{
		Scheme: "https",
		Host:   "abc",
	}
	customProvider := CustomProvider(url)
	s, ok := customProvider.url(ipversion.IP4or6)
	assert.True(t, ok)
	assert.Equal(t, "https://abc", s)
}
