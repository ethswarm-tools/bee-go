package swarm

import (
	"math/big"
	"testing"
)

func TestBZZ_FromDecimalAndPLUR(t *testing.T) {
	// 1.5 BZZ at 16 digits = 15000000000000000 PLUR.
	b, err := BZZFromDecimalString("1.5")
	if err != nil {
		t.Fatalf("BZZFromDecimalString: %v", err)
	}
	want := "15000000000000000"
	if got := b.ToPLURString(); got != want {
		t.Errorf("PLUR = %q, want %q", got, want)
	}

	b2, err := BZZFromPLURString(want)
	if err != nil {
		t.Fatalf("BZZFromPLURString: %v", err)
	}
	if !b.Eq(b2) {
		t.Errorf("round-trip mismatch: %s vs %s", b.ToPLURString(), b2.ToPLURString())
	}
	if got := b.ToDecimalString(); got != "1.5000000000000000" {
		t.Errorf("ToDecimalString = %q", got)
	}
	if got := b.ToSignificantDigits(2); got != "1.50" {
		t.Errorf("ToSignificantDigits(2) = %q", got)
	}
}

func TestBZZ_DecimalTruncation(t *testing.T) {
	// 17 fractional digits — last one should be silently truncated.
	b, err := BZZFromDecimalString("0.12345678901234567")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	// First 16 fractional digits = 1234567890123456.
	if got := b.ToPLURString(); got != "1234567890123456" {
		t.Errorf("PLUR = %q", got)
	}
}

func TestBZZ_Arithmetic(t *testing.T) {
	a, _ := BZZFromDecimalString("2")
	b, _ := BZZFromDecimalString("0.5")
	if got := a.Plus(b).ToDecimalString(); got != "2.5000000000000000" {
		t.Errorf("plus = %q", got)
	}
	if got := a.Minus(b).ToDecimalString(); got != "1.5000000000000000" {
		t.Errorf("minus = %q", got)
	}
	if got := a.Divide(big.NewInt(4)).ToDecimalString(); got != "0.5000000000000000" {
		t.Errorf("divide = %q", got)
	}
}

func TestBZZ_Comparisons(t *testing.T) {
	a, _ := BZZFromDecimalString("1")
	b, _ := BZZFromDecimalString("2")
	if !a.Lt(b) || !b.Gt(a) || !a.Lte(a) || !a.Gte(a) || !a.Eq(a) {
		t.Errorf("comparisons broken: a=%s b=%s", a.ToDecimalString(), b.ToDecimalString())
	}
}

func TestBZZ_ExchangeToDAI(t *testing.T) {
	bzz, _ := BZZFromDecimalString("2")
	rate, _ := DAIFromDecimalString("3") // 3 DAI per 1 BZZ
	got := bzz.ExchangeToDAI(rate)
	if got.ToDecimalString() != "6.000000000000000000" {
		t.Errorf("exchange = %s", got.ToDecimalString())
	}
}

func TestDAI_RoundTripAndExchangeToBZZ(t *testing.T) {
	d, _ := DAIFromDecimalString("6")
	if got := d.ToWeiString(); got != "6000000000000000000" {
		t.Errorf("wei = %s", got)
	}
	rate, _ := DAIFromDecimalString("3")
	bzz := d.ExchangeToBZZ(rate)
	if bzz.ToDecimalString() != "2.0000000000000000" {
		t.Errorf("bzz = %s", bzz.ToDecimalString())
	}
}

func TestBZZ_NegativeAndInvalid(t *testing.T) {
	b, err := BZZFromDecimalString("-1.5")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := b.ToDecimalString(); got != "-1.5000000000000000" {
		t.Errorf("negative decimal = %q", got)
	}
	if _, err := BZZFromDecimalString("abc"); err == nil {
		t.Errorf("expected error for invalid string")
	}
}

func TestBZZ_FloatRoundTrip(t *testing.T) {
	b, err := BZZFromFloat(0.25)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got := b.ToFloat(); got != 0.25 {
		t.Errorf("ToFloat = %v", got)
	}
}
