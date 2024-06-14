package models

import (
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_GetPreviousIPs(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		h           History
		previousIPs []netip.Addr
	}{
		"empty_history": {
			h: History{},
		},
		"single_event": {
			h: History{
				{IP: netip.MustParseAddr("1.2.3.4")},
			},
		},
		"two_events": {
			h: History{
				{IP: netip.MustParseAddr("1.2.3.4")},
				{IP: netip.MustParseAddr("5.6.7.8")}, // last one
			},
			previousIPs: []netip.Addr{
				netip.MustParseAddr("1.2.3.4"),
			},
		},
		"three_events": {
			h: History{
				{IP: netip.MustParseAddr("1.2.3.4")},
				{IP: netip.MustParseAddr("5.6.7.8")},
				{IP: netip.MustParseAddr("9.6.7.8")}, // last one
			},
			previousIPs: []netip.Addr{
				netip.MustParseAddr("5.6.7.8"),
				netip.MustParseAddr("1.2.3.4"),
			},
		},
	}
	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			previousIPs := testCase.h.GetPreviousIPs()
			assert.Equal(t, testCase.previousIPs, previousIPs)
		})
	}
}

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
