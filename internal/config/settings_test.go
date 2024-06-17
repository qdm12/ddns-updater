package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Settings_String(t *testing.T) {
	t.Parallel()

	var defaultSettings Config
	defaultSettings.SetDefaults()

	s := defaultSettings.String()

	const expected = `Settings summary:
├── HTTP client
|   └── Timeout: 20s
├── Update
|   ├── Period: 10m0s
|   └── Cooldown: 5m0s
├── Public IP fetching
|   ├── HTTP enabled: yes
|   ├── HTTP IP providers
|   |   └── all
|   ├── HTTP IPv4 providers
|   |   └── all
|   ├── HTTP IPv6 providers
|   |   └── all
|   ├── DNS enabled: yes
|   ├── DNS timeout: 3s
|   └── DNS over TLS providers
|       └── all
├── Resolver: use Go default resolver
├── Server
|   ├── Listening address: :8000
|   └── Root URL: /
├── Health
|   └── Server listening address: 127.0.0.1:9999
├── Paths
|   ├── Data directory: ./data
|   └── Config file: data/config.json
├── Backup: disabled
└── Logger
    ├── Level: INFO
    └── Caller: hidden`
	assert.Equal(t, expected, s)
}
