package postage

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestStamper(t *testing.T) {
	// Generate random key
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}

	batchID := "0000000000000000000000000000000000000000000000000000000000000000"
	stamper, err := NewStamper(privKey, batchID, 20)
	if err != nil {
		t.Fatalf("NewStamper failed: %v", err)
	}

	// Stamp a chunk
	chunkAddr := make([]byte, 32) // zero address, maps to bucket 0
	bID, idx, sig, err := stamper.Stamp(chunkAddr)
	if err != nil {
		t.Fatalf("Stamp failed: %v", err)
	}

	if hex.EncodeToString(bID) != batchID {
		t.Errorf("Wrong batch ID returned: %s", hex.EncodeToString(bID))
	}

	if len(sig) != 65 {
		t.Errorf("Invalid signature length: %d", len(sig))
	}

	if len(idx) != 8 {
		t.Errorf("Invalid index length: %d", len(idx))
	}

	// Verify bucket increment
	// Bucket 0 should now be 1
	if stamper.buckets[0] != 1 {
		t.Errorf("Bucket not incremented: %d", stamper.buckets[0])
	}

	// Stamp again, should be 2
	_, _, _, err = stamper.Stamp(chunkAddr)
	if err != nil {
		t.Fatal(err)
	}
	if stamper.buckets[0] != 2 {
		t.Errorf("Bucket not incremented: %d", stamper.buckets[0])
	}
}
