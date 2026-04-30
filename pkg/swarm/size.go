package swarm

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Size is a non-negative size in bytes. Mirrors bee-js Size: uses 1000
// (decimal) units instead of 1024 to stay aligned with the Swarm
// theoretical/effective storage tables. Negative inputs return an error,
// fractional inputs are rounded up.
type Size struct {
	bytes int64
}

const (
	bytesInKilobyte = int64(1000)
	bytesInMegabyte = bytesInKilobyte * 1000
	bytesInGigabyte = bytesInMegabyte * 1000
	bytesInTerabyte = bytesInGigabyte * 1000
)

// SizeFromBytes returns a Size of the given bytes.
func SizeFromBytes(b float64) (Size, error) { return newSize(b) }

// SizeFromKilobytes returns a Size of the given kilobytes (1 kB = 1000 B).
func SizeFromKilobytes(k float64) (Size, error) { return newSize(k * float64(bytesInKilobyte)) }

// SizeFromMegabytes returns a Size of the given megabytes.
func SizeFromMegabytes(m float64) (Size, error) { return newSize(m * float64(bytesInMegabyte)) }

// SizeFromGigabytes returns a Size of the given gigabytes.
func SizeFromGigabytes(g float64) (Size, error) { return newSize(g * float64(bytesInGigabyte)) }

// SizeFromTerabytes returns a Size of the given terabytes.
func SizeFromTerabytes(t float64) (Size, error) { return newSize(t * float64(bytesInTerabyte)) }

// SizeFromString parses strings like "28MB", "1gb", "512 kb", "2megabytes".
// Case-insensitive. Whitespace ignored. Decimal numbers OK ("1.5gb").
// Uses 1000 as the base for conversions (stays aligned with Swarm
// theoretical/effective tables).
func SizeFromString(s string) (Size, error) {
	clean := strings.ToLower(strings.ReplaceAll(s, " ", ""))
	if clean == "" {
		return Size{}, NewBeeArgumentError("empty size string", s)
	}
	re := regexp.MustCompile(`([0-9]*\.?[0-9]+)([a-z]+)`)
	matches := re.FindAllStringSubmatch(clean, -1)
	if len(matches) == 0 {
		return Size{}, NewBeeArgumentError("unrecognized size string", s)
	}
	var totalBytes float64
	for _, m := range matches {
		value, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return Size{}, NewBeeArgumentError("invalid size number", m[1])
		}
		mult, err := sizeUnitToBytes(m[2])
		if err != nil {
			return Size{}, err
		}
		totalBytes += value * mult
	}
	return newSize(totalBytes)
}

// ToBytes returns the size in bytes.
func (s Size) ToBytes() int64 { return s.bytes }

// ToKilobytes returns the size in kilobytes (fractional).
func (s Size) ToKilobytes() float64 { return float64(s.bytes) / float64(bytesInKilobyte) }

// ToMegabytes returns the size in megabytes (fractional).
func (s Size) ToMegabytes() float64 { return float64(s.bytes) / float64(bytesInMegabyte) }

// ToGigabytes returns the size in gigabytes (fractional).
func (s Size) ToGigabytes() float64 { return float64(s.bytes) / float64(bytesInGigabyte) }

// ToTerabytes returns the size in terabytes (fractional).
func (s Size) ToTerabytes() float64 { return float64(s.bytes) / float64(bytesInTerabyte) }

// String renders the size as a human-readable formatted string with
// auto-scaled unit (e.g. "1.50 GB"). Uses base 1000.
func (s Size) String() string {
	switch {
	case s.bytes >= bytesInTerabyte:
		return fmt.Sprintf("%.2f TB", s.ToTerabytes())
	case s.bytes >= bytesInGigabyte:
		return fmt.Sprintf("%.2f GB", s.ToGigabytes())
	case s.bytes >= bytesInMegabyte:
		return fmt.Sprintf("%.2f MB", s.ToMegabytes())
	case s.bytes >= bytesInKilobyte:
		return fmt.Sprintf("%.2f kB", s.ToKilobytes())
	default:
		return fmt.Sprintf("%d B", s.bytes)
	}
}

func newSize(b float64) (Size, error) {
	if math.IsNaN(b) {
		return Size{}, NewBeeArgumentError("size is NaN", b)
	}
	if b < 0 {
		return Size{}, NewBeeArgumentError("size must be at least 0", b)
	}
	return Size{bytes: int64(math.Ceil(b))}, nil
}

func sizeUnitToBytes(unit string) (float64, error) {
	switch unit {
	case "b", "byte", "bytes":
		return 1, nil
	case "kb", "kilobyte", "kilobytes":
		return float64(bytesInKilobyte), nil
	case "mb", "megabyte", "megabytes":
		return float64(bytesInMegabyte), nil
	case "gb", "gigabyte", "gigabytes":
		return float64(bytesInGigabyte), nil
	case "tb", "terabyte", "terabytes":
		return float64(bytesInTerabyte), nil
	}
	return 0, NewBeeArgumentError("unsupported size unit", unit)
}
