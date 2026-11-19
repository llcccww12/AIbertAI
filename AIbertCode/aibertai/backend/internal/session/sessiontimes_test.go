package session

import (
	"testing"
	"time"
)

func TestSessionRetryAcceptsPostgresTimestampPrecision(t *testing.T) {
	stored := time.Date(2026, 11, 19, 0, 0, 0, 123457000, time.UTC)
	row := sessionRow{StartedAt: stored}
	for _, tc := range []struct {
		before    time.Duration
		wantError bool
	}{
		{0, false},
		{time.Nanosecond, false},
		{time.Microsecond / 2, false},
		{time.Microsecond, true},
	} {
		input := sessionInput{StartedAt: &timeValue{stored.Add(-tc.before)}}
		err := validateSessionTimes(input, row)
		if (err != nil) != tc.wantError {
			t.Errorf("start %v before stored time: got %v, want error: %v", tc.before, err, tc.wantError)
		}
	}
}
