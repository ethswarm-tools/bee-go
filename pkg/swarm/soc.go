package swarm

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// SingleOwnerChunk represents a single owner chunk.
type SingleOwnerChunk struct {
	ID        []byte
	Signature []byte
	Owner     []byte
	Span      []byte
	Payload   []byte
}

// CreateSOC creates a SingleOwnerChunk.
// id: 32 bytes identifier
// data: payload data
// signer: private key
func CreateSOC(id []byte, data []byte, signer *ecdsa.PrivateKey) (*SingleOwnerChunk, error) {
	// 1. Calculate content address (CAC)
	span := make([]byte, 8)
	binary.LittleEndian.PutUint64(span, uint64(len(data))) // Swarm uses LittleEndian for span in chunks? No, updated bee checks say LittleEndian.
	// Wait, Bee uses LittleEndian for span in chunks logic usually.
	// Bee-js `Span` says `binary.numberToUint64(..., 'LE')` is default? No, 'BE' in previous feed code I saw?
	// Let's check feed.ts again. "Binary.numberToUint64(BigInt(Math.floor(at)), 'BE')" is for timestamp.
	// Chunk span... bee-js cac.ts: `span.setUint64(0, BigInt(length), true)` (true for LittleEndian).

	spanLen := int64(len(data))
	if spanLen > ChunkSize {
		return nil, fmt.Errorf("payload too large")
	}
	binary.LittleEndian.PutUint64(span, uint64(spanLen))

	cacData := append(span, data...)
	cacAddress, err := CalculateChunkAddress(cacData)
	if err != nil {
		return nil, err
	}

	// 2. Sign (ID + CAC Address)
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

// Sign signs the data with the private key.
func Sign(data []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	hash := crypto.Keccak256(data)
	signature, err := crypto.Sign(hash, privateKey)
	if err != nil {
		return nil, err
	}
	return signature, nil
}
