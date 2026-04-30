package swarm

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// SingleOwnerChunk is a chunk addressed by (owner || identifier) instead of
// content hash. The owner signs (identifier || cacAddress), where cacAddress
// is the BMT root of (span || payload).
type SingleOwnerChunk struct {
	ID        []byte
	Signature []byte
	Owner     []byte
	Span      []byte
	Payload   []byte
}

// CreateSOC builds a SingleOwnerChunk for the given identifier and payload,
// signed by signer. The span is encoded little-endian per Swarm chunk format
// (matches bee-js cac.ts setUint64(..., true)).
func CreateSOC(id []byte, data []byte, signer *ecdsa.PrivateKey) (*SingleOwnerChunk, error) {
	if int64(len(data)) > ChunkSize {
		return nil, fmt.Errorf("payload too large")
	}

	span := make([]byte, 8)
	binary.LittleEndian.PutUint64(span, uint64(len(data)))

	cacAddress, err := CalculateChunkAddress(append(span, data...))
	if err != nil {
		return nil, err
	}

	toSign := make([]byte, 0, 32+32)
	toSign = append(toSign, id...)
	toSign = append(toSign, cacAddress...)

	signature, err := Sign(toSign, signer)
	if err != nil {
		return nil, err
	}

	ownerAddress := crypto.PubkeyToAddress(signer.PublicKey)

	return &SingleOwnerChunk{
		ID:        id,
		Signature: signature,
		Owner:     ownerAddress.Bytes(),
		Span:      span,
		Payload:   data,
	}, nil
}

// Sign hashes data with keccak256 and signs the digest with privateKey,
// returning the 65-byte [R || S || V] signature with V in {0,1}.
func Sign(data []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	hash := crypto.Keccak256(data)
	return crypto.Sign(hash, privateKey)
}
