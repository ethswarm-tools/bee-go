package postage

import (
	"encoding/binary"
	"errors"
	"time"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// Stamper represents a client-side postage stamper.
type Stamper struct {
	signer  swarm.PrivateKey
	batchID swarm.BatchID
	buckets []uint32
	depth   int
	maxSlot uint32
}

// NewStamper creates a new Stamper.
func NewStamper(signer swarm.PrivateKey, batchID swarm.BatchID, depth int) (*Stamper, error) {
	if depth <= 16 {
		return nil, errors.New("depth must be greater than 16")
	}
	return &Stamper{
		signer:  signer,
		batchID: batchID,
		buckets: make([]uint32, 65536), // 2^16 buckets
		depth:   depth,
		maxSlot: 1 << (depth - 16), // 2^(depth-16)
	}, nil
}

// Envelope is the per-chunk postage envelope returned by Stamp. Mirrors
// bee-js EnvelopeWithBatchId.
type Envelope struct {
	BatchID   swarm.BatchID
	Index     []byte // 8 bytes: bucket (BE u32) || height (BE u32)
	Issuer    swarm.EthAddress
	Signature swarm.Signature
	Timestamp []byte // 8 bytes: Unix milliseconds (BE u64), matching bee-js Date.now()
}

// Stamp signs a chunk and returns its postage envelope.
//
// Signing input (per bee-js): chunkAddr || batchID || index || timestamp,
// signed via PrivateKey.Sign (Ethereum signed-message scheme).
func (s *Stamper) Stamp(chunkAddr []byte) (Envelope, error) {
	if len(chunkAddr) != 32 {
		return Envelope{}, errors.New("invalid chunk address length")
	}

	// Bucket = first 2 bytes of chunk address (BE u16).
	bucket := binary.BigEndian.Uint16(chunkAddr[:2])
	height := s.buckets[bucket]
	if height >= s.maxSlot {
		return Envelope{}, errors.New("bucket is full")
	}
	s.buckets[bucket]++

	// Index is 8 bytes: bucket (BE u32) || height (BE u32).
	index := make([]byte, 8)
	binary.BigEndian.PutUint32(index[:4], uint32(bucket))
	binary.BigEndian.PutUint32(index[4:], height)

	// Timestamp is Unix milliseconds (matches bee-js Date.now()).
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().UnixMilli()))

	batchIDBytes := s.batchID.Raw()
	toSign := make([]byte, 0, 32+32+8+8)
	toSign = append(toSign, chunkAddr...)
	toSign = append(toSign, batchIDBytes...)
	toSign = append(toSign, index...)
	toSign = append(toSign, timestamp...)

	sig, err := s.signer.Sign(toSign)
	if err != nil {
		return Envelope{}, err
	}

	return Envelope{
		BatchID:   s.batchID,
		Index:     index,
		Issuer:    s.signer.PublicKey().Address(),
		Signature: sig,
		Timestamp: timestamp,
	}, nil
}
