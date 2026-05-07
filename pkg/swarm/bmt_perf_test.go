package swarm

import (
	"bytes"
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/sha3"
)

// naiveBMTRoot is the pre-optimization implementation: pad the payload
// to ChunkSize, reduce 128 segments level-by-level. Used as an oracle
// for [calculateBMTRoot] across short payloads, where the production
// path takes the zero-subtree shortcut.
func naiveBMTRoot(payload []byte) []byte {
	padded := make([]byte, ChunkSize)
	copy(padded, payload)
	segments := make([][]byte, SegmentsCount)
	for i := range SegmentsCount {
		segments[i] = padded[i*SegmentSize : (i+1)*SegmentSize]
	}
	for len(segments) > 1 {
		next := make([][]byte, len(segments)/2)
		for i := 0; i < len(segments); i += 2 {
			h := sha3.NewLegacyKeccak256()
			h.Write(segments[i])
			h.Write(segments[i+1])
			next[i/2] = h.Sum(nil)
		}
		segments = next
	}
	return segments[0]
}

// TestBMT_ZeroSubtreeMatchesNaive verifies the zero-subtree
// optimization produces byte-identical roots to the naive 128-segment
// reduction across a sweep of payload sizes.
func TestBMT_ZeroSubtreeMatchesNaive(t *testing.T) {
	cases := []int{0, 1, 31, 32, 33, 64, 100, 1024, 4096}
	for _, n := range cases {
		payload := make([]byte, n)
		if _, err := rand.Read(payload); err != nil {
			t.Fatalf("rand: %v", err)
		}
		got := calculateBMTRoot(payload)
		want := naiveBMTRoot(payload)
		if !bytes.Equal(got, want) {
			t.Errorf("size=%d: zero-subtree root != naive root\n got=%x\nwant=%x", n, got, want)
		}
	}
}

// TestBMT_AllZeroChunkRoot verifies the well-known root of a 4 KiB
// all-zero chunk: hash(zeroSubtree[7]) at depth 7 of the BMT.
func TestBMT_AllZeroChunkRoot(t *testing.T) {
	got := calculateBMTRoot(nil)
	if !bytes.Equal(got, zeroSubtree[bmtDepth]) {
		t.Errorf("empty payload root != zeroSubtree[depth]\n got=%x\nwant=%x", got, zeroSubtree[bmtDepth])
	}
	if got2 := calculateBMTRoot(make([]byte, ChunkSize)); !bytes.Equal(got2, zeroSubtree[bmtDepth]) {
		t.Errorf("4KiB-zero payload root != zeroSubtree[depth]\n got=%x", got2)
	}
}

// BenchmarkBMT_Small measures the small-payload speedup; the naive
// path always pays for 127 keccak invocations whereas the optimized
// path pays for ceil(log2(K))+1 plus the zero-subtree precompute (one-
// time cost).
func BenchmarkBMT_Small(b *testing.B) {
	payload := []byte("hello swarm")
	b.ReportAllocs()
	for b.Loop() {
		_ = calculateBMTRoot(payload)
	}
}

func BenchmarkBMT_Naive_Small(b *testing.B) {
	payload := []byte("hello swarm")
	b.ReportAllocs()
	for b.Loop() {
		_ = naiveBMTRoot(payload)
	}
}
