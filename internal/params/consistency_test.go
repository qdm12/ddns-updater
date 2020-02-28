package params

import (
	"testing"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/stretchr/testify/assert"
)

func Test_ipMethodIsValid(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		ipMethod models.IPMethod
		valid    bool
	}{
		"empty method": {
			ipMethod: "",
			valid:    false,
		},
		"non existing method": {
			ipMethod: "abc",
			valid:    false,
		},
		"existing method": {
			ipMethod: "opendns",
			valid:    true,
		},
		"http url": {
			ipMethod: "http://ipinfo.io/ip",
			valid:    false,
		},
		"https url": {
			ipMethod: "https://ipinfo.io/ip",
			valid:    true,
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			valid := ipMethodIsValid(tc.ipMethod)
			assert.Equal(t, tc.valid, valid)
		})
	}
}
