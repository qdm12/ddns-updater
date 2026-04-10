package update

import (
	"testing"
	"time"
)

func TestJitterDuration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		period   time.Duration
		wantZero bool
		wantMax  time.Duration
	}{
		{"5m interval", 5 * time.Minute, false, 60 * time.Second},
		{"1h interval", time.Hour, false, 12 * time.Minute},
		{"4ns sub-threshold", 4 * time.Nanosecond, true, 0},
		{"zero", 0, true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := jitterDuration(tc.period)
			if tc.wantZero {
				if got != 0 {
					t.Errorf("want 0, got %v", got)
				}
				return
			}
			if got < 0 || got > tc.wantMax {
				t.Errorf("want [0, %v], got %v", tc.wantMax, got)
			}
		})
	}
}
