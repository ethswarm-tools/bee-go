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

// Sign signs data using the Ethereum signed-message scheme that Bee
// expects for SOC signatures:
//
//	digest = keccak256("\x19Ethereum Signed Message:\n32" || keccak256(data))
//
// matching bee-js PrivateKey.sign. Returns the 65-byte [R || S || V]
// signature with V normalized to {27,28}.
func Sign(data []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	digest := ethSignedMessageDigest(data)
	sig, err := crypto.Sign(digest, privateKey)
	if err != nil {
		return nil, err
	}
	if sig[64] < 27 {
		sig[64] += 27
	}
	return sig, nil
}

// UnmarshalSingleOwnerChunk parses the wire form of a SOC chunk —
// identifier(32) || signature(65) || span(8) || payload(≤4096) — and
// verifies it matches expectedAddress (recovered owner ⊕ identifier
// must hash to expectedAddress). Mirrors bee-js unmarshalSingleOwnerChunk.
func UnmarshalSingleOwnerChunk(data []byte, expectedAddress Reference) (*SingleOwnerChunk, error) {
	const minLen = IdentifierLength + SignatureLength + SpanSize
	if len(data) < minLen {
		return nil, NewBeeArgumentError("SOC data too short", len(data))
	}

	id := data[:IdentifierLength]
	sig := data[IdentifierLength : IdentifierLength+SignatureLength]
	spanStart := IdentifierLength + SignatureLength
	payloadStart := spanStart + SpanSize
	span := data[spanStart:payloadStart]
	payload := data[payloadStart:]

	cacData := make([]byte, 0, len(span)+len(payload))
	cacData = append(cacData, span...)
	cacData = append(cacData, payload...)
	cacAddr, err := CalculateChunkAddress(cacData)
	if err != nil {
		return nil, err
	}

	// Bee signs the eth-signed-message digest of (identifier || cacAddress);
	// reproduce that digest to recover the owner's public key.
	digest := ethSignedMessageDigest(append(append([]byte{}, id...), cacAddr...))
	// Bee stores V∈{27,28}; go-ethereum's SigToPub expects V∈{0,1}.
	// Denormalize on a copy so we don't mutate the caller's bytes.
	recoverSig := append([]byte{}, sig...)
	if recoverSig[64] >= 27 {
		recoverSig[64] -= 27
	}
	pubKey, err := crypto.SigToPub(digest, recoverSig)
	if err != nil {
		return nil, fmt.Errorf("recover SOC owner: %w", err)
	}
	owner := crypto.PubkeyToAddress(*pubKey).Bytes()

	socAddr := crypto.Keccak256(append(append([]byte{}, id...), owner...))
	if !bytesEq(socAddr, expectedAddress.Raw()) {
		return nil, fmt.Errorf("SOC data does not match given address")
	}

	return &SingleOwnerChunk{
		ID:        id,
		Signature: sig,
		Owner:     owner,
		Span:      span,
		Payload:   payload,
	}, nil
}

func bytesEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
