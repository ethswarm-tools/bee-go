package swarm

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Duration is a non-negative whole-second duration. Mirrors bee-js
// Duration: negative inputs clamp to zero, fractional seconds are rounded
// up. Use ToTimeDuration() to convert to the standard library type when
// needed.
type Duration struct {
	seconds int64
}

// ZeroDuration is the zero-length Duration. Equivalent to bee-js
// `Duration.ZERO`.
var ZeroDuration = Duration{}

const (
	secondsInMinute = int64(60)
	secondsInHour   = secondsInMinute * 60
	secondsInDay    = secondsInHour * 24
	secondsInWeek   = secondsInDay * 7
	secondsInMonth  = secondsInDay * 30
	secondsInYear   = secondsInDay * 365
)

// DurationFromSeconds returns a Duration of the given whole seconds.
func DurationFromSeconds(seconds float64) Duration { return newDuration(seconds) }

// DurationFromMilliseconds returns a Duration of the given milliseconds.
func DurationFromMilliseconds(ms float64) Duration { return newDuration(ms / 1000.0) }

// DurationFromMinutes returns a Duration of the given minutes.
func DurationFromMinutes(minutes float64) Duration {
	return newDuration(minutes * float64(secondsInMinute))
}

// DurationFromHours returns a Duration of the given hours.
func DurationFromHours(hours float64) Duration {
	return newDuration(hours * float64(secondsInHour))
}

// DurationFromDays returns a Duration of the given days.
func DurationFromDays(days float64) Duration {
	return newDuration(days * float64(secondsInDay))
}

// DurationFromWeeks returns a Duration of the given weeks.
func DurationFromWeeks(weeks float64) Duration {
	return newDuration(weeks * float64(secondsInWeek))
}

// DurationFromYears returns a Duration of 365-day years.
func DurationFromYears(years float64) Duration {
	return newDuration(years * float64(secondsInYear))
}

// DurationFromEndDate returns the Duration from `start` (defaults to now
// when zero) to `end`.
func DurationFromEndDate(end time.Time, start time.Time) Duration {
	if start.IsZero() {
		start = time.Now()
	}
	return newDuration(end.Sub(start).Seconds())
}

// DurationFromTime returns a Duration from a time.Duration.
func DurationFromTime(d time.Duration) Duration { return newDuration(d.Seconds()) }

// DurationFromString parses strings like "1.5h", "5 d", "2weeks", "30s".
// Case-insensitive. Whitespace is ignored. Supported unit families:
// ms / s / m / h / d / w / month / y. Unknown units yield an error.
//
// Mirrors bee-js Duration.parseFromString. The bee-js implementation
// delegates to cafe-utility's Dates.make; we implement the same surface
// directly to avoid a JS dependency.
func DurationFromString(s string) (Duration, error) {
	clean := strings.ToLower(strings.ReplaceAll(s, " ", ""))
	if clean == "" {
		return ZeroDuration, NewBeeArgumentError("empty duration string", s)
	}
	re := regexp.MustCompile(`([0-9]*\.?[0-9]+)([a-z]+)`)
	matches := re.FindAllStringSubmatch(clean, -1)
	if len(matches) == 0 {
		return ZeroDuration, NewBeeArgumentError("unrecognized duration string", s)
	}
	var totalSeconds float64
	for _, m := range matches {
		value, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return ZeroDuration, NewBeeArgumentError("invalid duration number", m[1])
		}
		mult, err := durationUnitToSeconds(m[2])
		if err != nil {
			return ZeroDuration, err
		}
		totalSeconds += value * mult
	}
	return newDuration(totalSeconds), nil
}

// ToSeconds returns the duration in whole seconds.
func (d Duration) ToSeconds() int64 { return d.seconds }

// ToMilliseconds returns the duration in milliseconds.
func (d Duration) ToMilliseconds() int64 { return d.seconds * 1000 }

// ToMinutes returns the duration in minutes (fractional).
func (d Duration) ToMinutes() float64 { return float64(d.seconds) / float64(secondsInMinute) }

// ToHours returns the duration in hours (fractional).
func (d Duration) ToHours() float64 { return float64(d.seconds) / float64(secondsInHour) }

// ToDays returns the duration in days (fractional).
func (d Duration) ToDays() float64 { return float64(d.seconds) / float64(secondsInDay) }

// ToWeeks returns the duration in weeks (fractional).
func (d Duration) ToWeeks() float64 { return float64(d.seconds) / float64(secondsInWeek) }

// ToYears returns the duration in 365-day years (fractional).
func (d Duration) ToYears() float64 { return float64(d.seconds) / float64(secondsInYear) }

// ToTimeDuration converts to a stdlib time.Duration.
func (d Duration) ToTimeDuration() time.Duration { return time.Duration(d.seconds) * time.Second }

// ToEndDate returns the time obtained by adding this duration to `start`.
// A zero `start` defaults to time.Now().
func (d Duration) ToEndDate(start time.Time) time.Time {
	if start.IsZero() {
		start = time.Now()
	}
	return start.Add(d.ToTimeDuration())
}

// IsZero reports whether the duration is zero.
func (d Duration) IsZero() bool { return d.seconds == 0 }

// String renders the duration in a human-friendly way (e.g. "1d 4h 5s").
func (d Duration) String() string {
	if d.seconds == 0 {
		return "0s"
	}
	parts := []struct {
		unit string
		size int64
	}{
		{"y", secondsInYear},
		{"w", secondsInWeek},
		{"d", secondsInDay},
		{"h", secondsInHour},
		{"m", secondsInMinute},
		{"s", 1},
	}
	remaining := d.seconds
	var sb strings.Builder
	for _, p := range parts {
		if remaining >= p.size {
			n := remaining / p.size
			remaining -= n * p.size
			if sb.Len() > 0 {
				sb.WriteByte(' ')
			}
			sb.WriteString(strconv.FormatInt(n, 10))
			sb.WriteString(p.unit)
		}
	}
	return sb.String()
}

func newDuration(seconds float64) Duration {
	if seconds < 0 || math.IsNaN(seconds) {
		return ZeroDuration
	}
	return Duration{seconds: int64(math.Ceil(seconds))}
}

func durationUnitToSeconds(unit string) (float64, error) {
	switch unit {
	case "ms", "milli", "millis", "millisecond", "milliseconds":
		return 0.001, nil
	case "s", "sec", "second", "seconds":
		return 1, nil
	case "m", "min", "minute", "minutes":
		return float64(secondsInMinute), nil
	case "h", "hour", "hours":
		return float64(secondsInHour), nil
	case "d", "day", "days":
		return float64(secondsInDay), nil
	case "w", "week", "weeks":
		return float64(secondsInWeek), nil
	case "month", "months":
		return float64(secondsInMonth), nil
	case "y", "year", "years":
		return float64(secondsInYear), nil
	}
	return 0, NewBeeArgumentError("unsupported duration unit", unit)
}
