package config

import (
	"testing"

	"github.com/qdm12/ddns-updater/pkg/publicip/privateip"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gosettings/reader/sources/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopWarner struct{}

func (noopWarner) Warnf(string, ...any) {}

func readPubIP(t *testing.T, environ ...string) (p PubIP, err error) {
	t.Helper()
	// An empty Environ falls back to the process environment, so always set one entry.
	environ = append([]string{"UNRELATED=1"}, environ...)
	r := reader.New(reader.Settings{Sources: []reader.Source{env.New(env.Settings{Environ: environ})}})
	err = p.read(r, noopWarner{})
	require.NoError(t, err)
	p.setDefaults()
	return p, p.Validate()
}

func Test_PubIP_fetchers(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		environ      []string
		http, dns    bool
		private      bool
		iface        string
		errSentinel  error
		summary      []string
		summaryNot   []string
		privSettings *privateip.Settings
	}{
		"fetchers omitted": {
			http: true, dns: true,
			summary:    []string{"Private IP enabled: no"},
			summaryNot: []string{"Private IP interface"},
		},
		"fetchers all": {
			environ: []string{"PUBLICIP_FETCHERS=all"},
			http:    true, dns: true,
			summary:    []string{"Private IP enabled: no"},
			summaryNot: []string{"Private IP interface"},
		},
		"privateip with mixed case and spaces": {
			environ: []string{"PUBLICIP_FETCHERS=privateip", "PRIVATEIP_INTERFACE=  Enp2S0  "},
			private: true, iface: "Enp2S0",
			summary:      []string{"Private IP enabled: yes", "Private IP interface: Enp2S0"},
			privSettings: &privateip.Settings{Enabled: true, Interface: "Enp2S0"},
		},
		"privateip without interface": {
			environ:     []string{"PUBLICIP_FETCHERS=privateip"},
			private:     true,
			errSentinel: ErrPrivateIPInterfaceNotSet,
		},
		"privateip with whitespace-only interface": {
			environ:     []string{"PUBLICIP_FETCHERS=privateip", "PRIVATEIP_INTERFACE=   "},
			private:     true,
			errSentinel: ErrPrivateIPInterfaceNotSet,
		},
		"http and privateip": {
			environ: []string{"PUBLICIP_FETCHERS=http,privateip", "PRIVATEIP_INTERFACE=enp2s0"},
			http:    true, private: true, iface: "enp2s0",
		},
		"interface set without privateip is ignored": {
			environ: []string{"PUBLICIP_FETCHERS=http", "PRIVATEIP_INTERFACE=enp2s0"},
			http:    true, iface: "enp2s0",
			summaryNot: []string{"Private IP interface"},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p, err := readPubIP(t, testCase.environ...)

			if testCase.errSentinel != nil {
				require.ErrorIs(t, err, testCase.errSentinel)
				assert.Contains(t, err.Error(), "PRIVATEIP_INTERFACE")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.http, *p.HTTPEnabled)
			assert.Equal(t, testCase.dns, *p.DNSEnabled)
			assert.Equal(t, testCase.private, *p.PrivateIPEnabled)
			assert.Equal(t, testCase.iface, p.PrivateIPInterface)
			summary := p.String()
			for _, s := range testCase.summary {
				assert.Contains(t, summary, s)
			}
			for _, s := range testCase.summaryNot {
				assert.NotContains(t, summary, s)
			}
			if testCase.privSettings != nil {
				assert.Equal(t, *testCase.privSettings, p.ToPrivateIPSettings())
			}
		})
	}
}
