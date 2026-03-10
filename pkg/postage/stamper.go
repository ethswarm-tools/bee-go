package postage

import (
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

// Stamper represents a client-side postage stamper.
type Stamper struct {
	signer  *ecdsa.PrivateKey
	batchID []byte
	buckets []uint32
	depth   int
	maxSlot uint32
}

// NewStamper creates a new Stamper.
// batchID: 32-byte hex string or bytes
// depth: batch depth
func NewStamper(signer *ecdsa.PrivateKey, batchIDHex string, depth int) (*Stamper, error) {
	batchID, err := hex.DecodeString(batchIDHex)
	if err != nil {
		return nil, fmt.Errorf("invalid batch ID: %w", err)
	}
	if len(batchID) != 32 {
		return nil, errors.New("invalid batch ID length")
	}

	return &Stamper{
		signer:  signer,
		batchID: batchID,
		buckets: make([]uint32, 65536), // 2^16 buckets
		depth:   depth,
		maxSlot: 1 << (depth - 16), // 2^(depth-16)
	}, nil
}

// Stamp signs a chunk and returns the batch ID, index, and signature.
// chunkAddr: 32-byte hash of the chunk
func (s *Stamper) Stamp(chunkAddr []byte) (batchID []byte, index []byte, signature []byte, err error) {
	if len(chunkAddr) != 32 {
		return nil, nil, nil, errors.New("invalid chunk address length")
	}

	// Calculate bucket index (first 2 bytes of chunk address as BE uint16)
	bucket := binary.BigEndian.Uint16(chunkAddr[:2])

	// Get and increment collision counter
	height := s.buckets[bucket]
	if height >= s.maxSlot {
		return nil, nil, nil, errors.New("bucket is full")
	}
	s.buckets[bucket]++

	// Create index buffer (bucket uint32 BE + height uint32 BE)
	// bee-js uses 8 bytes for index: 4 bytes bucket + 4 bytes height?
	// bee-js:
	// const index = Binary.concatBytes(Binary.numberToUint32(bucket, 'BE'), Binary.numberToUint32(height, 'BE'))
	index = make([]byte, 8)
	binary.BigEndian.PutUint32(index[:4], uint32(bucket))
	binary.BigEndian.PutUint32(index[4:], height)

	// Prepare data to sign: address + batchID + index + timestamp
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().UnixNano()))
	// Bee-js uses Date.now() which is milliseconds. Go UnixNano is nanoseconds.
	// bee-js: Binary.numberToUint64(BigInt(Date.now()), 'BE')
	// We should probably match the millisecond precision to be safe, although it's just a timestamp.
	// Let's use UnixMilli.
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().UnixMilli()))

	toSign := make([]byte, 0, 32+32+8+8)
	toSign = append(toSign, chunkAddr...)
	toSign = append(toSign, s.batchID...)
	toSign = append(toSign, index...)
	toSign = append(toSign, timestamp...)

	// Sign
	sig, err := crypto.Sign(crypto.Keccak256(toSign), s.signer)
	if err != nil {
		return nil, nil, nil, err
	}

	return s.batchID, index, sig, nil
}
