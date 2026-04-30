package swarm

import "testing"

func TestSize_Constructors(t *testing.T) {
	cases := map[string]struct {
		size Size
		want int64
	}{
		"1KB": {mustSize(SizeFromKilobytes(1)), 1000},
		"1MB": {mustSize(SizeFromMegabytes(1)), 1_000_000},
		"1GB": {mustSize(SizeFromGigabytes(1)), 1_000_000_000},
	}
	for name, c := range cases {
		if got := c.size.ToBytes(); got != c.want {
			t.Errorf("%s = %d, want %d", name, got, c.want)
		}
	}
}

func TestSize_NegativeRejected(t *testing.T) {
	if _, err := SizeFromBytes(-1); err == nil {
		t.Errorf("expected error for negative size")
	}
}

func TestSize_FromString(t *testing.T) {
	cases := map[string]int64{
		"28MB":       28 * 1_000_000,
		"1gb":        1_000_000_000,
		"1.5GB":      1_500_000_000,
		"512 kb":     512_000,
		"2megabytes": 2_000_000,
		"1tb":        1_000_000_000_000,
	}
	for in, want := range cases {
		got, err := SizeFromString(in)
		if err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if got.ToBytes() != want {
			t.Errorf("%q = %d, want %d", in, got.ToBytes(), want)
		}
	}

	if _, err := SizeFromString("garbage"); err == nil {
		t.Errorf("expected error for garbage")
	}
}

func TestSize_String(t *testing.T) {
	cases := map[Size]string{
		mustSize(SizeFromBytes(500)):      "500 B",
		mustSize(SizeFromKilobytes(1.5)):  "1.50 kB",
		mustSize(SizeFromMegabytes(2.25)): "2.25 MB",
		mustSize(SizeFromGigabytes(1)):    "1.00 GB",
		mustSize(SizeFromTerabytes(0.5)):  "500.00 GB",
	}
	for s, want := range cases {
		if got := s.String(); got != want {
			t.Errorf("%d bytes -> %q, want %q", s.ToBytes(), got, want)
		}
	}
}

func mustSize(s Size, err error) Size {
	if err != nil {
		panic(err)
	}
	return s
}
