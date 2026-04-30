package postage_test

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/ethersphere/bee-go/pkg/postage"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestMarshalStamp_RoundTrip(t *testing.T) {
	batchID := swarm.MustBatchID(strings.Repeat("aa", 32))

	signer, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	st, err := postage.NewStamper(signer, batchID, 17)
	if err != nil {
		t.Fatalf("NewStamper: %v", err)
	}
	chunkAddr := bytes.Repeat([]byte{0x42}, 32)
	env, err := st.Stamp(chunkAddr)
	if err != nil {
		t.Fatalf("Stamp: %v", err)
	}

	marshaled, err := postage.ConvertEnvelopeToMarshaledStamp(env)
	if err != nil {
		t.Fatalf("ConvertEnvelopeToMarshaledStamp: %v", err)
	}
	if len(marshaled) != postage.MarshaledStampLength {
		t.Errorf("length = %d, want %d", len(marshaled), postage.MarshaledStampLength)
	}
	// Layout: batchID(32) || index(8) || timestamp(8) || signature(65).
	if !bytes.Equal(marshaled[:32], batchID.Raw()) {
		t.Errorf("batchID prefix mismatch")
	}
	if !bytes.Equal(marshaled[32:40], env.Index) {
		t.Errorf("index slice mismatch")
	}
	if !bytes.Equal(marshaled[40:48], env.Timestamp) {
		t.Errorf("timestamp slice mismatch")
	}
	if !bytes.Equal(marshaled[48:113], env.Signature.Raw()) {
		t.Errorf("signature suffix mismatch")
	}
	// Sanity: timestamp decodes as a sane Unix ms value (within ~1 day of now).
	ts := binary.BigEndian.Uint64(marshaled[40:48])
	delta := time.Now().UnixMilli() - int64(ts)
	if delta < 0 || delta > 24*60*60*1000 {
		t.Errorf("timestamp out of range: %d (now-then=%d ms)", ts, delta)
	}
}

func TestMarshalStamp_Validation(t *testing.T) {
	batchID := swarm.MustBatchID(strings.Repeat("aa", 32))
	sig, _ := swarm.NewSignature(bytes.Repeat([]byte{0xab}, swarm.SignatureLength))

	cases := []struct {
		name      string
		index, ts []byte
	}{
		{"short index", []byte{1, 2, 3}, bytes.Repeat([]byte{0}, 8)},
		{"short timestamp", bytes.Repeat([]byte{0}, 8), []byte{1, 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := postage.MarshalStamp(batchID, tc.index, tc.ts, sig)
			if err == nil {
				t.Fatal("expected error")
			}
			if _, ok := swarm.IsBeeArgumentError(err); !ok {
				t.Errorf("expected BeeArgumentError, got %T", err)
			}
		})
	}
}
