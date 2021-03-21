package http

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_newDefaultSettings(t *testing.T) {
	t.Parallel()

	settings := newDefaultSettings()

	assert.NotEmpty(t, settings.providersIP)
	assert.NotEmpty(t, settings.providersIP4)
	assert.NotEmpty(t, settings.providersIP6)
	assert.GreaterOrEqual(t, int(settings.timeout), int(time.Millisecond))
}

func Test_SetProvidersIP(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		initialSettings  settings
		providers        []Provider
		expectedSettings settings
		err              error
	}{
		"Google": {
			initialSettings: settings{
				providersIP: []Provider{Opendns},
			},
			providers: []Provider{Google},
			expectedSettings: settings{
				providersIP: []Provider{Google},
			},
		},
		"bad provider for IP version": {
			initialSettings: settings{
				providersIP: []Provider{Opendns},
			},
			providers: []Provider{Noip},
			expectedSettings: settings{
				providersIP: []Provider{Opendns},
			},
			err: errors.New(`provider does not support IP version: "noip" for version ipv4 or ipv6`),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			settings := testCase.initialSettings

			option := SetProvidersIP(testCase.providers[0], testCase.providers[1:]...)
			err := option(&settings)

			assert.Equal(t, testCase.expectedSettings, settings)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_SetProvidersIP4(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		initialSettings  settings
		providers        []Provider
		expectedSettings settings
		err              error
	}{
		"NoIP": {
			initialSettings: settings{
				providersIP4: []Provider{Ipify},
			},
			providers: []Provider{Noip},
			expectedSettings: settings{
				providersIP4: []Provider{Noip},
			},
		},
		"bad provider for IP version": {
			initialSettings: settings{
				providersIP4: []Provider{Ipify},
			},
			providers: []Provider{Opendns},
			expectedSettings: settings{
				providersIP4: []Provider{Ipify},
			},
			err: errors.New(`provider does not support IP version: "opendns" for version ipv4`),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			settings := testCase.initialSettings

			option := SetProvidersIP4(testCase.providers[0], testCase.providers[1:]...)
			err := option(&settings)

			assert.Equal(t, testCase.expectedSettings, settings)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_SetProvidersIP6(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		initialSettings  settings
		providers        []Provider
		expectedSettings settings
		err              error
	}{
		"NoIP": {
			initialSettings: settings{
				providersIP6: []Provider{Ipify},
			},
			providers: []Provider{Noip},
			expectedSettings: settings{
				providersIP6: []Provider{Noip},
			},
		},
		"bad provider for IP version": {
			initialSettings: settings{
				providersIP6: []Provider{Ipify},
			},
			providers: []Provider{Opendns},
			expectedSettings: settings{
				providersIP6: []Provider{Ipify},
			},
			err: errors.New(`provider does not support IP version: "opendns" for version ipv6`),
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			settings := testCase.initialSettings

			option := SetProvidersIP6(testCase.providers[0], testCase.providers[1:]...)
			err := option(&settings)

			assert.Equal(t, testCase.expectedSettings, settings)

			if testCase.err != nil {
				require.Error(t, err)
				assert.Equal(t, testCase.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_SetTimeout(t *testing.T) {
	t.Parallel()

	initialSettings := settings{}
	expectedSettings := settings{
		timeout: time.Hour,
	}

	option := SetTimeout(time.Hour)
	err := option(&initialSettings)

	require.NoError(t, err)
	assert.Equal(t, expectedSettings, initialSettings)
}
