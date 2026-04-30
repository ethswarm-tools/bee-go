package swarm

import (
	"crypto/ecdsa"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/crypto"
)

// MinPayloadSize / MaxPayloadSize bound a Content Addressed Chunk's
// payload. Mirrors bee-js cac.ts.
const (
	MinPayloadSize = 1
	MaxPayloadSize = ChunkSize
)

// Chunk is a Content Addressed Chunk: an immutable 4096-byte (max)
// payload preceded by an 8-byte little-endian span. Address is the
// BMT root over (span || zero-padded payload).
//
// Mirrors bee-js Chunk (cac.ts).
type Chunk struct {
	Address Reference
	Span    [SpanSize]byte
	Payload []byte
}

// Data returns the wire form: span || payload. The caller may append
// it to a request body or hand it to UploadChunk.
func (c Chunk) Data() []byte {
	out := make([]byte, 0, SpanSize+len(c.Payload))
	out = append(out, c.Span[:]...)
	out = append(out, c.Payload...)
	return out
}

// MakeContentAddressedChunk wraps payload in a Chunk: encodes the span
// little-endian, computes the BMT address, and returns the result.
// Mirrors bee-js makeContentAddressedChunk.
func MakeContentAddressedChunk(payload []byte) (Chunk, error) {
	if len(payload) < MinPayloadSize || len(payload) > MaxPayloadSize {
		return Chunk{}, NewBeeArgumentError("payload size out of bounds", len(payload))
	}
	var span [SpanSize]byte
	binary.LittleEndian.PutUint64(span[:], uint64(len(payload)))

	full := make([]byte, 0, SpanSize+len(payload))
	full = append(full, span[:]...)
	full = append(full, payload...)
	addrBytes, err := CalculateChunkAddress(full)
	if err != nil {
		return Chunk{}, err
	}
	addr, err := NewReference(addrBytes)
	if err != nil {
		return Chunk{}, err
	}
	return Chunk{Address: addr, Span: span, Payload: payload}, nil
}

// ToSingleOwnerChunk wraps this CAC in a SOC signed by signer. The SOC
// is addressed by keccak256(identifier || ownerAddress).
func (c Chunk) ToSingleOwnerChunk(id Identifier, signer *ecdsa.PrivateKey) (*SingleOwnerChunk, error) {
	return CreateSOC(id.Raw(), c.Payload, signer)
}

// MakeSingleOwnerChunk is a convenience wrapper around CreateSOC that
// takes a typed Identifier. Mirrors bee-js makeSingleOwnerChunk.
func MakeSingleOwnerChunk(id Identifier, payload []byte, signer *ecdsa.PrivateKey) (*SingleOwnerChunk, error) {
	return CreateSOC(id.Raw(), payload, signer)
}

// CalculateSingleOwnerChunkAddress returns the SOC reference,
// keccak256(identifier || owner), without any HTTP call.
func CalculateSingleOwnerChunkAddress(id Identifier, owner EthAddress) (Reference, error) {
	addr := crypto.Keccak256(id.Raw(), owner.Raw())
	return NewReference(addr)
}
