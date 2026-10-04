package spaceship

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TTL(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		configTTL uint32
		knownTTL  uint32
		wantTTL   uint32 // 0 means nil
	}{
		"config_ttl_set": {
			configTTL: 300,
			knownTTL:  600,
			wantTTL:   300,
		},
		"known_ttl_fallback": {
			knownTTL: 600,
			wantTTL:  600,
		},
		"unknown": {},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			p := Provider{ttl: testCase.configTTL}
			if testCase.knownTTL != 0 {
				p.knownTTL.Store(testCase.knownTTL)
			}

			ttl := p.TTL()
			if testCase.wantTTL == 0 {
				assert.Nil(t, ttl)
				return
			}
			require.NotNil(t, ttl)
			assert.Equal(t, testCase.wantTTL, *ttl)
		})
	}
}
