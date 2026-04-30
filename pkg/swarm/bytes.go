package swarm

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"
)

// Bytes is the base type for all typed byte sequences in the Swarm protocol.
// Typed wrappers (Reference, BatchId, PrivateKey, ...) embed Bytes to inherit
// hex / equality / length helpers.
//
// Bytes is value-typed: copies are independent. The underlying slice is kept
// unexported so callers go through Raw() (which returns a copy) or Hex().
type Bytes struct {
	raw []byte
}

// newBytes builds a Bytes from a raw slice, optionally validating the length
// against an allow-list. If lengths is empty, no length check is performed.
// The slice is copied so the caller cannot mutate the stored bytes.
func newBytes(b []byte, lengths ...int) (Bytes, error) {
	if len(lengths) > 0 {
		ok := false
		for _, l := range lengths {
			if len(b) == l {
				ok = true
				break
			}
		}
		if !ok {
			return Bytes{}, fmt.Errorf("invalid length: got %d, expected %v", len(b), lengths)
		}
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return Bytes{raw: cp}, nil
}

// newBytesFromHex decodes a hex string (with or without 0x prefix) and
// validates the length.
func newBytesFromHex(s string, lengths ...int) (Bytes, error) {
	s = strings.TrimPrefix(s, "0x")
	decoded, err := hex.DecodeString(s)
	if err != nil {
		return Bytes{}, fmt.Errorf("invalid hex: %w", err)
	}
	return newBytes(decoded, lengths...)
}

// Hex returns the lower-case hex encoding without 0x prefix.
func (b Bytes) Hex() string { return hex.EncodeToString(b.raw) }

// String implements fmt.Stringer; returns the hex encoding.
func (b Bytes) String() string { return b.Hex() }

// Raw returns a copy of the underlying bytes. Mutating the result does not
// affect the Bytes value.
func (b Bytes) Raw() []byte {
	cp := make([]byte, len(b.raw))
	copy(cp, b.raw)
	return cp
}

// Len returns the byte length.
func (b Bytes) Len() int { return len(b.raw) }

// IsZero reports whether the value is uninitialized (zero length).
func (b Bytes) IsZero() bool { return len(b.raw) == 0 }

// Equal reports whether two Bytes hold the same byte sequence.
func (b Bytes) Equal(other Bytes) bool { return bytes.Equal(b.raw, other.raw) }

// MarshalJSON encodes Bytes as a JSON hex string.
func (b Bytes) MarshalJSON() ([]byte, error) {
	return []byte(`"` + b.Hex() + `"`), nil
}

// UnmarshalJSON decodes a JSON hex string into Bytes (no length validation
// here; typed wrappers re-validate via their own UnmarshalJSON).
func (b *Bytes) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	out, err := newBytesFromHex(s)
	if err != nil {
		return err
	}
	*b = out
	return nil
}
