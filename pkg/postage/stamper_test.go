package postage

import (
	"strings"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func TestStamper(t *testing.T) {
	signer, err := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	if err != nil {
		t.Fatal(err)
	}

	batchID := swarm.MustBatchID(strings.Repeat("00", 32))
	stamper, err := NewStamper(signer, batchID, 20)
	if err != nil {
		t.Fatalf("NewStamper failed: %v", err)
	}

	// Stamp a chunk: zero address maps to bucket 0.
	chunkAddr := make([]byte, 32)
	env, err := stamper.Stamp(chunkAddr)
	if err != nil {
		t.Fatalf("Stamp failed: %v", err)
	}

	if !env.BatchID.Equal(batchID.Bytes) {
		t.Errorf("Wrong batch ID returned: %s", env.BatchID.Hex())
	}
	if env.Signature.Len() != 65 {
		t.Errorf("Invalid signature length: %d", env.Signature.Len())
	}
	if len(env.Index) != 8 {
		t.Errorf("Invalid index length: %d", len(env.Index))
	}
	if env.Issuer.Len() != 20 {
		t.Errorf("Invalid issuer length: %d", env.Issuer.Len())
	}

	// Verify bucket increment.
	if stamper.buckets[0] != 1 {
		t.Errorf("Bucket not incremented: %d", stamper.buckets[0])
	}

	// Stamp again, should be 2.
	if _, err := stamper.Stamp(chunkAddr); err != nil {
		t.Fatal(err)
	}
	if stamper.buckets[0] != 2 {
		t.Errorf("Bucket not incremented: %d", stamper.buckets[0])
	}

	// Signature must verify against the issuer address (sanity check that
	// PrivateKey.Sign + Signature.RecoverPublicKey are mutually consistent).
	toSign := append(append(append(append([]byte{}, chunkAddr...), batchID.Raw()...), env.Index...), env.Timestamp...)
	if !env.Signature.IsValid(toSign, env.Issuer) {
		t.Errorf("signature does not verify against issuer")
	}
}
