package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_GetDurationSinceSuccess(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		h History
		s string
	}{
		"empty history": {
			h: History{},
			s: "N/A",
		},
		"single event": {
			h: History{{}},
			s: "106751d",
		},
		"two events": {
			h: History{{}, {}},
			s: "106751d",
		},
	}
	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			now, _ := time.Parse("2006-01-02", "2000-01-01")
			s := tc.h.GetDurationSinceSuccess(now)
			assert.Equal(t, tc.s, s)
		})
	}
}
