package swarm

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestBytes_HexRoundTrip(t *testing.T) {
	hexStr := "0123456789abcdef"
	b, err := newBytesFromHex(hexStr)
	if err != nil {
		t.Fatalf("newBytesFromHex: %v", err)
	}
	if got := b.Hex(); got != hexStr {
		t.Errorf("Hex() = %q, want %q", got, hexStr)
	}
	// 0x prefix should be accepted.
	b2, err := newBytesFromHex("0x" + hexStr)
	if err != nil {
		t.Fatalf("0x prefix: %v", err)
	}
	if !b.Equal(b2) {
		t.Errorf("0x-prefixed value not equal to bare hex value")
	}
}

func TestBytes_LengthValidation(t *testing.T) {
	// 31 bytes should fail when 32 is required.
	if _, err := newBytes(make([]byte, 31), 32); err == nil {
		t.Errorf("expected length error, got nil")
	}
	// Allow-list (32 or 64) should accept either.
	if _, err := newBytes(make([]byte, 32), 32, 64); err != nil {
		t.Errorf("32 should be valid: %v", err)
	}
	if _, err := newBytes(make([]byte, 64), 32, 64); err != nil {
		t.Errorf("64 should be valid: %v", err)
	}
	if _, err := newBytes(make([]byte, 48), 32, 64); err == nil {
		t.Errorf("48 should be invalid in {32,64}")
	}
}

func TestBytes_RawIsCopy(t *testing.T) {
	b, _ := newBytes([]byte{1, 2, 3})
	out := b.Raw()
	out[0] = 99
	if b.Raw()[0] != 1 {
		t.Errorf("Raw() should return a copy; original mutated")
	}
}

func TestReference_LengthAndHex(t *testing.T) {
	hex32 := strings.Repeat("ab", 32)
	r, err := ReferenceFromHex(hex32)
	if err != nil {
		t.Fatalf("ReferenceFromHex(32): %v", err)
	}
	if r.Hex() != hex32 {
		t.Errorf("Hex() = %q, want %q", r.Hex(), hex32)
	}
	hex64 := strings.Repeat("cd", 64)
	if _, err := ReferenceFromHex(hex64); err != nil {
		t.Errorf("64-byte (encrypted) reference should be accepted: %v", err)
	}
	// 31 bytes should be rejected.
	if _, err := ReferenceFromHex(strings.Repeat("ab", 31)); err == nil {
		t.Errorf("31-byte reference should be rejected")
	}
}

func TestReference_IsValid(t *testing.T) {
	if !IsValidReference(strings.Repeat("00", 32)) {
		t.Errorf("32-byte zeros should be valid")
	}
	if IsValidReference("not-hex") {
		t.Errorf("non-hex should be invalid")
	}
	if IsValidReference(strings.Repeat("ab", 30)) {
		t.Errorf("wrong-length should be invalid")
	}
}

func TestMustReference_PanicsOnInvalid(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on invalid input")
		}
	}()
	MustReference("not-a-reference")
}

func TestIdentifierFromString_DeterministicKeccak(t *testing.T) {
	id := IdentifierFromString("hello")
	expected := crypto.Keccak256([]byte("hello"))
	if !bytes.Equal(id.Raw(), expected) {
		t.Errorf("IdentifierFromString != keccak256(utf8)")
	}
	if id.Len() != IdentifierLength {
		t.Errorf("length = %d, want %d", id.Len(), IdentifierLength)
	}
}

func TestTopicFromString_DeterministicKeccak(t *testing.T) {
	topic := TopicFromString("my-topic")
	expected := crypto.Keccak256([]byte("my-topic"))
	if !bytes.Equal(topic.Raw(), expected) {
		t.Errorf("TopicFromString != keccak256(utf8)")
	}
}

func TestEthAddress_Checksum(t *testing.T) {
	// EIP-55 example: 0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359
	rawHex := "fb6916095ca1df60bb79ce92ce3ea74c37c5d359"
	addr, err := EthAddressFromHex(rawHex)
	if err != nil {
		t.Fatalf("EthAddressFromHex: %v", err)
	}
	want := "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359"
	if got := addr.ToChecksum(); got != want {
		t.Errorf("ToChecksum() = %q, want %q", got, want)
	}
}

func TestPrivateKey_PublicKey_Address(t *testing.T) {
	// Generate a deterministic private key.
	keyHex := strings.Repeat("11", 32)
	priv, err := PrivateKeyFromHex(keyHex)
	if err != nil {
		t.Fatalf("PrivateKeyFromHex: %v", err)
	}
	pub := priv.PublicKey()
	if pub.Len() != PublicKeyLength {
		t.Errorf("public key length = %d, want %d", pub.Len(), PublicKeyLength)
	}
	// Cross-check against go-ethereum.
	ecdsaKey, _ := priv.ToECDSA()
	wantAddr := crypto.PubkeyToAddress(ecdsaKey.PublicKey).Bytes()
	if !bytes.Equal(pub.Address().Raw(), wantAddr) {
		t.Errorf("Address() does not match go-ethereum reference")
	}
}

func TestPublicKey_Compressed_RoundTrip(t *testing.T) {
	priv, _ := PrivateKeyFromHex(strings.Repeat("22", 32))
	pub := priv.PublicKey()
	compressed, err := pub.CompressedBytes()
	if err != nil {
		t.Fatalf("CompressedBytes: %v", err)
	}
	if len(compressed) != 33 {
		t.Errorf("compressed length = %d, want 33", len(compressed))
	}
	// Decompress via NewPublicKey and verify equality.
	pub2, err := NewPublicKey(compressed)
	if err != nil {
		t.Fatalf("NewPublicKey(compressed): %v", err)
	}
	if !pub.Equal(pub2.Bytes) {
		t.Errorf("compress/decompress did not round-trip")
	}
}

func TestSignature_SignRecover_RoundTrip(t *testing.T) {
	priv, _ := PrivateKeyFromHex(strings.Repeat("33", 32))
	data := []byte("hello swarm")
	sig, err := priv.Sign(data)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if sig.Len() != SignatureLength {
		t.Errorf("sig length = %d, want %d", sig.Len(), SignatureLength)
	}
	// V should be normalized to 27 or 28 (matching bee-js).
	if v := sig.Raw()[64]; v != 27 && v != 28 {
		t.Errorf("V = %d, want 27 or 28", v)
	}
	pub, err := sig.RecoverPublicKey(data)
	if err != nil {
		t.Fatalf("RecoverPublicKey: %v", err)
	}
	if !pub.Equal(priv.PublicKey().Bytes) {
		t.Errorf("recovered pubkey != signer pubkey")
	}
	if !sig.IsValid(data, priv.PublicKey().Address()) {
		t.Errorf("IsValid returned false for own signature")
	}
	// Tamper with data: should not validate.
	if sig.IsValid([]byte("not the same"), priv.PublicKey().Address()) {
		t.Errorf("IsValid returned true for tampered data")
	}
}

func TestSpan_RoundTrip(t *testing.T) {
	for _, n := range []uint64{0, 1, 4096, 1 << 40} {
		s := SpanFromUint64(n)
		if got := s.ToUint64(); got != n {
			t.Errorf("Span(%d) round-trip = %d", n, got)
		}
	}
	// Encoding must be little-endian.
	s := SpanFromUint64(1)
	if !bytes.Equal(s.Raw(), []byte{1, 0, 0, 0, 0, 0, 0, 0}) {
		t.Errorf("Span LE encoding wrong: %x", s.Raw())
	}
}

func TestFeedIndex_RoundTripAndNext(t *testing.T) {
	for _, n := range []uint64{0, 1, 100, 1<<32 - 1} {
		f := FeedIndexFromUint64(n)
		if got := f.ToUint64(); got != n {
			t.Errorf("FeedIndex(%d) round-trip = %d", n, got)
		}
	}
	// Encoding must be big-endian.
	f := FeedIndexFromUint64(1)
	if !bytes.Equal(f.Raw(), []byte{0, 0, 0, 0, 0, 0, 0, 1}) {
		t.Errorf("FeedIndex BE encoding wrong: %x", f.Raw())
	}
	// Next() on a normal index increments.
	if FeedIndexFromUint64(5).Next().ToUint64() != 6 {
		t.Errorf("Next(5) != 6")
	}
	// Next() on MinusOne wraps to 0.
	if FeedIndexMinusOne.Next().ToUint64() != 0 {
		t.Errorf("MinusOne.Next() did not wrap to 0")
	}
}

func TestBatchID_LengthValidation(t *testing.T) {
	if _, err := BatchIDFromHex(strings.Repeat("aa", 32)); err != nil {
		t.Errorf("32-byte batch id should be valid: %v", err)
	}
	if _, err := BatchIDFromHex(strings.Repeat("aa", 31)); err == nil {
		t.Errorf("31-byte batch id should be invalid")
	}
}

func TestBytes_JSON(t *testing.T) {
	hexStr := strings.Repeat("ab", 32)
	r, _ := ReferenceFromHex(hexStr)
	j, err := r.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	want := `"` + hexStr + `"`
	if string(j) != want {
		t.Errorf("MarshalJSON = %s, want %s", j, want)
	}
}

// Sanity: ensure all 32-byte typed wrappers reject 31-byte input.
func TestAllSimple32_RejectShort(t *testing.T) {
	bad := make([]byte, 31)
	if _, err := NewReference(bad); err == nil {
		t.Errorf("NewReference accepted 31 bytes")
	}
	if _, err := NewBatchID(bad); err == nil {
		t.Errorf("NewBatchID accepted 31 bytes")
	}
	if _, err := NewTransactionID(bad); err == nil {
		t.Errorf("NewTransactionID accepted 31 bytes")
	}
	if _, err := NewPeerAddress(bad); err == nil {
		t.Errorf("NewPeerAddress accepted 31 bytes")
	}
	if _, err := NewIdentifier(bad); err == nil {
		t.Errorf("NewIdentifier accepted 31 bytes")
	}
	if _, err := NewTopic(bad); err == nil {
		t.Errorf("NewTopic accepted 31 bytes")
	}
}

// Decode/encode parity with hex package for sanity.
func TestBytes_HexParity(t *testing.T) {
	src := []byte{0xde, 0xad, 0xbe, 0xef}
	b, _ := newBytes(src)
	if got, want := b.Hex(), hex.EncodeToString(src); got != want {
		t.Errorf("Hex parity: got %s, want %s", got, want)
	}
}
