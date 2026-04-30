package swarm

import (
	"testing"
	"time"
)

func TestDuration_Constructors(t *testing.T) {
	if got := DurationFromHours(1).ToSeconds(); got != 3600 {
		t.Errorf("hours = %d", got)
	}
	if got := DurationFromDays(1).ToSeconds(); got != 86400 {
		t.Errorf("days = %d", got)
	}
	if got := DurationFromWeeks(1).ToSeconds(); got != 604800 {
		t.Errorf("weeks = %d", got)
	}
	if got := DurationFromMilliseconds(1500).ToSeconds(); got != 2 {
		// Math.ceil(1.5) = 2 — bee-js parity
		t.Errorf("ms ceil = %d", got)
	}
	if got := DurationFromSeconds(-5).ToSeconds(); got != 0 {
		t.Errorf("negative should clamp, got %d", got)
	}
}

func TestDuration_FromString(t *testing.T) {
	cases := map[string]int64{
		"1h":         3600,
		"1.5h":       5400,
		"5d":         5 * 86400,
		"2 weeks":    2 * 7 * 86400,
		"1d 4h":      86400 + 4*3600,
		"30s":        30,
		"500ms":      1, // ceil
		"1Y":         365 * 86400,
		"2 month":    2 * 30 * 86400,
		"1d4h5m30s":  86400 + 4*3600 + 5*60 + 30,
	}
	for in, want := range cases {
		got, err := DurationFromString(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got.ToSeconds() != want {
			t.Errorf("%q = %d, want %d", in, got.ToSeconds(), want)
		}
	}

	if _, err := DurationFromString("garbage"); err == nil {
		t.Errorf("expected error for garbage input")
	}
	if _, err := DurationFromString(""); err == nil {
		t.Errorf("expected error for empty input")
	}
}

func TestDuration_EndDate(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d := DurationFromDays(1)
	got := d.ToEndDate(start)
	if got.Sub(start) != 24*time.Hour {
		t.Errorf("end-start = %v", got.Sub(start))
	}
}

func TestDuration_String(t *testing.T) {
	if got := ZeroDuration.String(); got != "0s" {
		t.Errorf("zero = %q", got)
	}
	d := DurationFromDays(1).
		// 1d 4h 5m 30s
		ToTimeDuration() + 4*time.Hour + 5*time.Minute + 30*time.Second
	if got := DurationFromTime(d).String(); got != "1d 4h 5m 30s" {
		t.Errorf("string = %q", got)
	}
}
