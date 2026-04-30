package swarm_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestMakeContentAddressedChunk(t *testing.T) {
	payload := []byte("hello swarm")
	c, err := swarm.MakeContentAddressedChunk(payload)
	if err != nil {
		t.Fatalf("MakeContentAddressedChunk: %v", err)
	}
	if !bytes.Equal(c.Payload, payload) {
		t.Errorf("payload mismatch")
	}
	// Span is little-endian uint64 of the payload length.
	if c.Span[0] != byte(len(payload)) || c.Span[1] != 0 {
		t.Errorf("span = %v, want first byte = len", c.Span)
	}
	want, err := swarm.CalculateChunkAddress(c.Data())
	if err != nil {
		t.Fatalf("CalculateChunkAddress: %v", err)
	}
	if !bytes.Equal(c.Address.Raw(), want) {
		t.Errorf("address mismatch")
	}
}

func TestMakeContentAddressedChunk_BoundsCheck(t *testing.T) {
	if _, err := swarm.MakeContentAddressedChunk(nil); err == nil {
		t.Errorf("empty payload: expected error")
	}
	if _, err := swarm.MakeContentAddressedChunk(make([]byte, swarm.MaxPayloadSize+1)); err == nil {
		t.Errorf("oversize payload: expected error")
	}
}

func TestCalculateSingleOwnerChunkAddress(t *testing.T) {
	id := swarm.IdentifierFromString("topic")
	swPriv, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	signer, _ := crypto.ToECDSA(swPriv.Raw())
	owner, _ := swarm.NewEthAddress(crypto.PubkeyToAddress(signer.PublicKey).Bytes())

	addr, err := swarm.CalculateSingleOwnerChunkAddress(id, owner)
	if err != nil {
		t.Fatalf("CalculateSingleOwnerChunkAddress: %v", err)
	}
	if len(addr.Raw()) != swarm.ReferenceLength {
		t.Errorf("address length = %d", len(addr.Raw()))
	}

	soc, err := swarm.MakeSingleOwnerChunk(id, []byte("payload"), signer)
	if err != nil {
		t.Fatalf("MakeSingleOwnerChunk: %v", err)
	}
	if !bytes.Equal(soc.Owner, owner.Raw()) {
		t.Errorf("owner mismatch: %x vs %x", soc.Owner, owner.Raw())
	}
}

func TestChunk_ToSingleOwnerChunk_Roundtrip(t *testing.T) {
	swPriv, _ := swarm.PrivateKeyFromHex(strings.Repeat("22", 32))
	signer, _ := crypto.ToECDSA(swPriv.Raw())
	id := swarm.IdentifierFromString("rt")
	owner, _ := swarm.NewEthAddress(crypto.PubkeyToAddress(signer.PublicKey).Bytes())

	cac, err := swarm.MakeContentAddressedChunk([]byte("rt-payload"))
	if err != nil {
		t.Fatalf("CAC: %v", err)
	}
	soc, err := cac.ToSingleOwnerChunk(id, signer)
	if err != nil {
		t.Fatalf("ToSingleOwnerChunk: %v", err)
	}

	addr, _ := swarm.CalculateSingleOwnerChunkAddress(id, owner)
	wire := make([]byte, 0, len(soc.ID)+len(soc.Signature)+len(soc.Span)+len(soc.Payload))
	wire = append(wire, soc.ID...)
	wire = append(wire, soc.Signature...)
	wire = append(wire, soc.Span...)
	wire = append(wire, soc.Payload...)

	parsed, err := swarm.UnmarshalSingleOwnerChunk(wire, addr)
	if err != nil {
		t.Fatalf("UnmarshalSingleOwnerChunk: %v", err)
	}
	if !bytes.Equal(parsed.Payload, soc.Payload) || !bytes.Equal(parsed.Owner, soc.Owner) {
		t.Errorf("round-trip mismatch")
	}
}
