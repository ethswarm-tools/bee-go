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
)

// CalculateChunkAddress calculates the BMT address of a chunk.
// data must be the full chunk data (span + payload).
func CalculateChunkAddress(data []byte) ([]byte, error) {
	if len(data) < SpanSize {
		return nil, fmt.Errorf("data too short")
	}

	span := data[:SpanSize]
	payload := data[SpanSize:]

	if len(payload) > ChunkSize {
		return nil, fmt.Errorf("payload too large")
	}

	// Pad payload to 4096 bytes
	paddedPayload := make([]byte, ChunkSize)
	copy(paddedPayload, payload)

	rootHash := calculateBMTRoot(paddedPayload)

	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(span)
	hasher.Write(rootHash)
	return hasher.Sum(nil), nil
}

func calculateBMTRoot(payload []byte) []byte {
	segments := make([][]byte, SegmentsCount)
	for i := 0; i < SegmentsCount; i++ {
		segments[i] = payload[i*SegmentSize : (i+1)*SegmentSize]
	}

	return reduceBMT(segments)
}

func reduceBMT(segments [][]byte) []byte {
	if len(segments) == 1 {
		return segments[0]
	}

	nextLevel := make([][]byte, len(segments)/2)
	for i := 0; i < len(segments); i += 2 {
		hasher := sha3.NewLegacyKeccak256()
		hasher.Write(segments[i])
		hasher.Write(segments[i+1])
		nextLevel[i/2] = hasher.Sum(nil)
	}

	return reduceBMT(nextLevel)
}

// NewHasher returns a new Keccak256 hasher.
func NewHasher() hash.Hash {
	return sha3.NewLegacyKeccak256()
}

// Keccak256 calculates the Keccak256 hash of the data.
func Keccak256(data ...[]byte) []byte {
	return crypto.Keccak256(data...)
}
