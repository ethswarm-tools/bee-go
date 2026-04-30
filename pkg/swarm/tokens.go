package swarm

import (
	"fmt"
	"math/big"
	"strings"
)

// BZZDigits is the number of decimal digits used by the BZZ token (1 BZZ =
// 10^16 PLUR). DAIDigits is the same for DAI (1 DAI = 10^18 wei). Mirrors
// bee-js BZZ.DIGITS and DAI.DIGITS.
const (
	BZZDigits = 16
	DAIDigits = 18
)

// BZZ is a fixed-point amount of BZZ tokens stored as base units (PLUR).
// Mirrors bee-js BZZ. The zero value is 0 BZZ; all operations are pure (no
// in-place mutation).
type BZZ struct {
	plur *big.Int
}

// DAI is a fixed-point amount of DAI/native-token stored as base units
// (wei). Mirrors bee-js DAI.
type DAI struct {
	wei *big.Int
}

// --- BZZ constructors ---

// NewBZZ builds a BZZ amount from base units (PLUR) given as *big.Int.
// A nil argument is treated as zero. The input is copied; mutating it
// afterwards does not affect the BZZ.
func NewBZZ(plur *big.Int) BZZ {
	if plur == nil {
		return BZZ{plur: new(big.Int)}
	}
	return BZZ{plur: new(big.Int).Set(plur)}
}

// BZZFromPLURString parses a decimal integer string of base units.
func BZZFromPLURString(s string) (BZZ, error) {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return BZZ{}, NewBeeArgumentError("invalid PLUR string", s)
	}
	return BZZ{plur: v}, nil
}

// BZZFromDecimalString parses a human-readable BZZ amount such as "1.5".
// Up to BZZDigits fractional digits are kept; extra digits are silently
// truncated (matching bee-js / FixedPointNumber).
func BZZFromDecimalString(s string) (BZZ, error) {
	plur, err := decimalToBaseUnits(s, BZZDigits)
	if err != nil {
		return BZZ{}, err
	}
	return BZZ{plur: plur}, nil
}

// BZZFromFloat builds a BZZ amount from a float. The float is rendered
// with enough precision to capture BZZDigits and then parsed as a decimal
// string — same approach as bee-js FixedPointNumber.fromFloat.
func BZZFromFloat(f float64) (BZZ, error) {
	return BZZFromDecimalString(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.*f", BZZDigits, f), "0"), "."))
}

// --- BZZ accessors ---

// ToPLURBigInt returns a copy of the underlying base-unit value.
func (b BZZ) ToPLURBigInt() *big.Int {
	if b.plur == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(b.plur)
}

// ToPLURString returns the base-unit value as a base-10 integer string.
func (b BZZ) ToPLURString() string {
	if b.plur == nil {
		return "0"
	}
	return b.plur.String()
}

// ToDecimalString renders the amount with BZZDigits fractional digits
// (e.g. "1.5000000000000000"). Trailing zeros are kept so the format is
// stable.
func (b BZZ) ToDecimalString() string { return baseUnitsToDecimal(b.plur, BZZDigits) }

// ToFloat returns the amount as a float64. Lossy for large/precise values.
func (b BZZ) ToFloat() float64 {
	f, _ := new(big.Float).SetInt(b.toPLUR()).Float64()
	return f / pow10Float(BZZDigits)
}

// ToSignificantDigits returns ToDecimalString truncated to `digits`
// fractional digits (matches bee-js BZZ.toSignificantDigits).
func (b BZZ) ToSignificantDigits(digits int) string {
	return truncateDecimal(b.ToDecimalString(), digits)
}

// --- BZZ arithmetic ---

// Plus returns a new BZZ equal to b + other.
func (b BZZ) Plus(other BZZ) BZZ {
	return BZZ{plur: new(big.Int).Add(b.toPLUR(), other.toPLUR())}
}

// Minus returns a new BZZ equal to b - other.
func (b BZZ) Minus(other BZZ) BZZ {
	return BZZ{plur: new(big.Int).Sub(b.toPLUR(), other.toPLUR())}
}

// Divide returns a new BZZ equal to b / divisor (integer division on
// PLUR). divisor must be non-zero.
func (b BZZ) Divide(divisor *big.Int) BZZ {
	return BZZ{plur: new(big.Int).Quo(b.toPLUR(), divisor)}
}

// --- BZZ comparison ---

// Cmp compares b to other: -1 if b < other, 0 if equal, 1 if b > other.
func (b BZZ) Cmp(other BZZ) int { return b.toPLUR().Cmp(other.toPLUR()) }

// Gt reports whether b > other.
func (b BZZ) Gt(other BZZ) bool { return b.Cmp(other) > 0 }

// Gte reports whether b >= other.
func (b BZZ) Gte(other BZZ) bool { return b.Cmp(other) >= 0 }

// Lt reports whether b < other.
func (b BZZ) Lt(other BZZ) bool { return b.Cmp(other) < 0 }

// Lte reports whether b <= other.
func (b BZZ) Lte(other BZZ) bool { return b.Cmp(other) <= 0 }

// Eq reports whether b == other.
func (b BZZ) Eq(other BZZ) bool { return b.Cmp(other) == 0 }

// ExchangeToDAI converts a BZZ amount using a rate expressed as DAI per
// 1 BZZ. Returns the resulting DAI amount.
func (b BZZ) ExchangeToDAI(daiPerBZZ DAI) DAI {
	// daiAmount = (plur * daiPerBZZ.wei) / 10^BZZDigits
	num := new(big.Int).Mul(b.toPLUR(), daiPerBZZ.toWei())
	denom := pow10(BZZDigits)
	return DAI{wei: new(big.Int).Quo(num, denom)}
}

// --- DAI constructors ---

// NewDAI builds a DAI amount from base units (wei) given as *big.Int.
func NewDAI(wei *big.Int) DAI {
	if wei == nil {
		return DAI{wei: new(big.Int)}
	}
	return DAI{wei: new(big.Int).Set(wei)}
}

// DAIFromWeiString parses a decimal integer string of wei.
func DAIFromWeiString(s string) (DAI, error) {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return DAI{}, NewBeeArgumentError("invalid wei string", s)
	}
	return DAI{wei: v}, nil
}

// DAIFromDecimalString parses a human-readable DAI amount such as "0.5".
func DAIFromDecimalString(s string) (DAI, error) {
	wei, err := decimalToBaseUnits(s, DAIDigits)
	if err != nil {
		return DAI{}, err
	}
	return DAI{wei: wei}, nil
}

// DAIFromFloat builds a DAI amount from a float.
func DAIFromFloat(f float64) (DAI, error) {
	return DAIFromDecimalString(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.*f", DAIDigits, f), "0"), "."))
}

// --- DAI accessors ---

// ToWeiBigInt returns a copy of the underlying wei value.
func (d DAI) ToWeiBigInt() *big.Int {
	if d.wei == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(d.wei)
}

// ToWeiString returns the wei value as a base-10 integer string.
func (d DAI) ToWeiString() string {
	if d.wei == nil {
		return "0"
	}
	return d.wei.String()
}

// ToDecimalString renders the amount with DAIDigits fractional digits.
func (d DAI) ToDecimalString() string { return baseUnitsToDecimal(d.wei, DAIDigits) }

// ToFloat returns the amount as a float64. Lossy for large/precise values.
func (d DAI) ToFloat() float64 {
	f, _ := new(big.Float).SetInt(d.toWei()).Float64()
	return f / pow10Float(DAIDigits)
}

// ToSignificantDigits returns ToDecimalString truncated to `digits`
// fractional digits.
func (d DAI) ToSignificantDigits(digits int) string {
	return truncateDecimal(d.ToDecimalString(), digits)
}

// --- DAI arithmetic ---

// Plus returns a new DAI equal to d + other.
func (d DAI) Plus(other DAI) DAI {
	return DAI{wei: new(big.Int).Add(d.toWei(), other.toWei())}
}

// Minus returns a new DAI equal to d - other.
func (d DAI) Minus(other DAI) DAI {
	return DAI{wei: new(big.Int).Sub(d.toWei(), other.toWei())}
}

// Divide returns a new DAI equal to d / divisor.
func (d DAI) Divide(divisor *big.Int) DAI {
	return DAI{wei: new(big.Int).Quo(d.toWei(), divisor)}
}

// --- DAI comparison ---

// Cmp compares d to other: -1, 0, 1.
func (d DAI) Cmp(other DAI) int { return d.toWei().Cmp(other.toWei()) }

// Gt reports whether d > other.
func (d DAI) Gt(other DAI) bool { return d.Cmp(other) > 0 }

// Gte reports whether d >= other.
func (d DAI) Gte(other DAI) bool { return d.Cmp(other) >= 0 }

// Lt reports whether d < other.
func (d DAI) Lt(other DAI) bool { return d.Cmp(other) < 0 }

// Lte reports whether d <= other.
func (d DAI) Lte(other DAI) bool { return d.Cmp(other) <= 0 }

// Eq reports whether d == other.
func (d DAI) Eq(other DAI) bool { return d.Cmp(other) == 0 }

// ExchangeToBZZ converts a DAI amount to BZZ using a rate expressed as
// DAI per 1 BZZ.
func (d DAI) ExchangeToBZZ(daiPerBZZ DAI) BZZ {
	// bzzPLUR = (wei * 10^BZZDigits) / daiPerBZZ.wei
	num := new(big.Int).Mul(d.toWei(), pow10(BZZDigits))
	return BZZ{plur: new(big.Int).Quo(num, daiPerBZZ.toWei())}
}

// --- helpers ---

func (b BZZ) toPLUR() *big.Int {
	if b.plur == nil {
		return new(big.Int)
	}
	return b.plur
}

func (d DAI) toWei() *big.Int {
	if d.wei == nil {
		return new(big.Int)
	}
	return d.wei
}

// decimalToBaseUnits parses "1.5" into 15 * 10^(digits-1). Extra
// fractional digits beyond `digits` are truncated silently. Empty,
// "+/-" sign and leading zeros are accepted.
func decimalToBaseUnits(s string, digits int) (*big.Int, error) {
	if s == "" {
		return nil, NewBeeArgumentError("empty decimal string", s)
	}
	negative := false
	switch s[0] {
	case '-':
		negative = true
		s = s[1:]
	case '+':
		s = s[1:]
	}
	intPart, fracPart, _ := strings.Cut(s, ".")
	if intPart == "" {
		intPart = "0"
	}
	if len(fracPart) > digits {
		fracPart = fracPart[:digits]
	}
	for len(fracPart) < digits {
		fracPart += "0"
	}
	combined := intPart + fracPart
	v, ok := new(big.Int).SetString(combined, 10)
	if !ok {
		return nil, NewBeeArgumentError("invalid decimal string", s)
	}
	if negative {
		v.Neg(v)
	}
	return v, nil
}

// baseUnitsToDecimal renders a base-unit *big.Int as "intPart.fracPart"
// with exactly `digits` fractional digits. nil renders as 0.
func baseUnitsToDecimal(v *big.Int, digits int) string {
	if v == nil {
		v = new(big.Int)
	}
	negative := v.Sign() < 0
	abs := new(big.Int).Abs(v)
	denom := pow10(digits)
	intPart := new(big.Int).Quo(abs, denom).String()
	fracPart := new(big.Int).Rem(abs, denom).String()
	for len(fracPart) < digits {
		fracPart = "0" + fracPart
	}
	out := intPart + "." + fracPart
	if negative {
		out = "-" + out
	}
	return out
}

// truncateDecimal keeps `digits` fractional digits from a decimal string.
func truncateDecimal(s string, digits int) string {
	dot := strings.Index(s, ".")
	if dot < 0 {
		return s
	}
	end := dot + 1 + digits
	if end > len(s) {
		end = len(s)
	}
	return s[:end]
}

func pow10(n int) *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil) }

func pow10Float(n int) float64 {
	f := 1.0
	for i := 0; i < n; i++ {
		f *= 10
	}
	return f
}
