package swarm

import (
	"fmt"
	"hash"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"
)

const (
	ChunkSize     = 4096
	SpanSize      = 8
	SegmentSize   = 32
	SegmentsCount = ChunkSize / SegmentSize
	// bmtDepth is the height of the BMT for a 4 KiB chunk: 128 segments
	// halve down to 1 in 7 steps. Levels run 0..bmtDepth inclusive.
	bmtDepth = 7
)

// zeroSubtree[level] is the BMT value of an all-zero subtree at that
// level. zeroSubtree[0] is 32 zero bytes (a single empty leaf segment);
// zeroSubtree[i+1] = keccak256(zeroSubtree[i] || zeroSubtree[i]).
//
// Precomputed at init time so [calculateBMTRoot] can short-circuit
// pairs whose right half lies entirely in the zero-padding region — a
// big win for small chunks (typical for chunked uploads), where the
// naive 127 hash invocations collapse to log2(K) where K = filled
// segments. Mirrors the bee-py / bee-rs optimization.
var zeroSubtree = func() [bmtDepth + 1][]byte {
	var t [bmtDepth + 1][]byte
	t[0] = make([]byte, SegmentSize)
	for i := range bmtDepth {
		h := sha3.NewLegacyKeccak256()
		h.Write(t[i])
		h.Write(t[i])
		t[i+1] = h.Sum(nil)
	}
	return t
}()

// CalculateChunkAddress calculates the BMT address of a chunk.
// data must be the full chunk data (span + payload). The payload is
// zero-padded to [ChunkSize] internally; the caller does not need to
// pad.
func CalculateChunkAddress(data []byte) ([]byte, error) {
	if len(data) < SpanSize {
		return nil, fmt.Errorf("data too short")
	}

	span := data[:SpanSize]
	payload := data[SpanSize:]

	if len(payload) > ChunkSize {
		return nil, fmt.Errorf("payload too large")
	}

	rootHash := calculateBMTRoot(payload)

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(span)
	hasher.Write(rootHash)
	return hasher.Sum(nil), nil
}

// calculateBMTRoot returns the BMT root of payload, treating the
// implicit zero-padding past len(payload) as if filled with zeros.
// Pairs whose right half lives entirely in the padding region are
// resolved against the precomputed [zeroSubtree] instead of being
// hashed — for a 1-byte payload this is 7 keccak calls instead of
// 127.
func calculateBMTRoot(payload []byte) []byte {
	n := (len(payload) + SegmentSize - 1) / SegmentSize
	if n == 0 {
		return zeroSubtree[bmtDepth]
	}

	segments := make([][]byte, n)
	for i := range n - 1 {
		segments[i] = payload[i*SegmentSize : (i+1)*SegmentSize]
	}
	// The last real segment may be shorter than SegmentSize; pad it.
	last := make([]byte, SegmentSize)
	copy(last, payload[(n-1)*SegmentSize:])
	segments[n-1] = last

	for level := range bmtDepth {
		newCount := (n + 1) / 2
		newSegments := make([][]byte, newCount)
		for i := range newCount {
			h := sha3.NewLegacyKeccak256()
			h.Write(segments[2*i])
			if 2*i+1 < n {
				h.Write(segments[2*i+1])
			} else {
				h.Write(zeroSubtree[level])
			}
			newSegments[i] = h.Sum(nil)
		}
		segments = newSegments
		n = newCount
	}
	return segments[0]
}

// NewHasher returns a new Keccak256 hasher.
func NewHasher() hash.Hash {
	return sha3.NewLegacyKeccak256()
}

// Keccak256 calculates the Keccak256 hash of the data.
func Keccak256(data ...[]byte) []byte {
	return crypto.Keccak256(data...)
}
