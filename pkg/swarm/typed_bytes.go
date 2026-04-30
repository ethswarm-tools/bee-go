package swarm

import (
	"crypto/ecdsa"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Length constants mirror bee-js typed-bytes.ts.
const (
	ReferenceLength          = 32
	EncryptedReferenceLength = 64
	BatchIDLength            = 32
	TransactionIDLength      = 32
	PeerAddressLength        = 32
	IdentifierLength         = 32
	TopicLength              = 32
	EthAddressLength         = 20
	PrivateKeyLength         = 32
	PublicKeyLength          = 64
	SignatureLength          = 65
	SpanLength               = 8
	FeedIndexLength          = 8
)

// ============================================================================
// Reference — Swarm content reference (32 bytes, or 64 bytes when encrypted)
// ============================================================================

type Reference struct{ Bytes }

func NewReference(b []byte) (Reference, error) {
	bb, err := newBytes(b, ReferenceLength, EncryptedReferenceLength)
	if err != nil {
		return Reference{}, err
	}
	return Reference{Bytes: bb}, nil
}

func ReferenceFromHex(s string) (Reference, error) {
	bb, err := newBytesFromHex(s, ReferenceLength, EncryptedReferenceLength)
	if err != nil {
		return Reference{}, err
	}
	return Reference{Bytes: bb}, nil
}

// MustReference panics if s is not a valid reference hex; for use with
// known-good constants and tests.
func MustReference(s string) Reference {
	r, err := ReferenceFromHex(s)
	if err != nil {
		panic(err)
	}
	return r
}

// ToCID converts the reference to a Swarm CID. refType is "manifest" or "feed".
func (r Reference) ToCID(refType string) (string, error) {
	return ConvertReferenceToCID(r.Hex(), refType)
}

// IsValidReference reports whether s is a syntactically valid reference hex.
func IsValidReference(s string) bool {
	_, err := ReferenceFromHex(s)
	return err == nil
}

// ============================================================================
// BatchId — postage batch identifier (32 bytes)
// ============================================================================

type BatchID struct{ Bytes }

func NewBatchID(b []byte) (BatchID, error) {
	bb, err := newBytes(b, BatchIDLength)
	if err != nil {
		return BatchID{}, err
	}
	return BatchID{Bytes: bb}, nil
}

func BatchIDFromHex(s string) (BatchID, error) {
	bb, err := newBytesFromHex(s, BatchIDLength)
	if err != nil {
		return BatchID{}, err
	}
	return BatchID{Bytes: bb}, nil
}

func MustBatchID(s string) BatchID {
	id, err := BatchIDFromHex(s)
	if err != nil {
		panic(err)
	}
	return id
}

// ============================================================================
// TransactionID (32 bytes)
// ============================================================================

type TransactionID struct{ Bytes }

func NewTransactionID(b []byte) (TransactionID, error) {
	bb, err := newBytes(b, TransactionIDLength)
	if err != nil {
		return TransactionID{}, err
	}
	return TransactionID{Bytes: bb}, nil
}

func TransactionIDFromHex(s string) (TransactionID, error) {
	bb, err := newBytesFromHex(s, TransactionIDLength)
	if err != nil {
		return TransactionID{}, err
	}
	return TransactionID{Bytes: bb}, nil
}

// ============================================================================
// PeerAddress (32 bytes overlay)
// ============================================================================

type PeerAddress struct{ Bytes }

func NewPeerAddress(b []byte) (PeerAddress, error) {
	bb, err := newBytes(b, PeerAddressLength)
	if err != nil {
		return PeerAddress{}, err
	}
	return PeerAddress{Bytes: bb}, nil
}

func PeerAddressFromHex(s string) (PeerAddress, error) {
	bb, err := newBytesFromHex(s, PeerAddressLength)
	if err != nil {
		return PeerAddress{}, err
	}
	return PeerAddress{Bytes: bb}, nil
}

// ============================================================================
// Identifier (32 bytes) — arbitrary identifier, supports keccak256 of UTF-8
// ============================================================================

type Identifier struct{ Bytes }

func NewIdentifier(b []byte) (Identifier, error) {
	bb, err := newBytes(b, IdentifierLength)
	if err != nil {
		return Identifier{}, err
	}
	return Identifier{Bytes: bb}, nil
}

func IdentifierFromHex(s string) (Identifier, error) {
	bb, err := newBytesFromHex(s, IdentifierLength)
	if err != nil {
		return Identifier{}, err
	}
	return Identifier{Bytes: bb}, nil
}

// IdentifierFromString returns keccak256(utf8(s)) as an Identifier.
// Mirrors bee-js Identifier.fromString.
func IdentifierFromString(s string) Identifier {
	hash := crypto.Keccak256([]byte(s))
	bb, _ := newBytes(hash, IdentifierLength)
	return Identifier{Bytes: bb}
}

// ============================================================================
// Topic (32 bytes) — feed/PSS topic, supports keccak256 of UTF-8
// ============================================================================

type Topic struct{ Bytes }

func NewTopic(b []byte) (Topic, error) {
	bb, err := newBytes(b, TopicLength)
	if err != nil {
		return Topic{}, err
	}
	return Topic{Bytes: bb}, nil
}

func TopicFromHex(s string) (Topic, error) {
	bb, err := newBytesFromHex(s, TopicLength)
	if err != nil {
		return Topic{}, err
	}
	return Topic{Bytes: bb}, nil
}

// TopicFromString returns keccak256(utf8(s)) as a Topic.
// Mirrors bee-js Topic.fromString.
func TopicFromString(s string) Topic {
	hash := crypto.Keccak256([]byte(s))
	bb, _ := newBytes(hash, TopicLength)
	return Topic{Bytes: bb}
}

// ============================================================================
// EthAddress (20 bytes) — Ethereum address with EIP-55 checksum
// ============================================================================

type EthAddress struct{ Bytes }

func NewEthAddress(b []byte) (EthAddress, error) {
	bb, err := newBytes(b, EthAddressLength)
	if err != nil {
		return EthAddress{}, err
	}
	return EthAddress{Bytes: bb}, nil
}

func EthAddressFromHex(s string) (EthAddress, error) {
	bb, err := newBytesFromHex(s, EthAddressLength)
	if err != nil {
		return EthAddress{}, err
	}
	return EthAddress{Bytes: bb}, nil
}

// ToChecksum returns the EIP-55 mixed-case representation, with 0x prefix.
func (a EthAddress) ToChecksum() string {
	return common.BytesToAddress(a.raw).Hex()
}

// ============================================================================
// PrivateKey (32 bytes) — secp256k1 private key
// ============================================================================

type PrivateKey struct{ Bytes }

func NewPrivateKey(b []byte) (PrivateKey, error) {
	bb, err := newBytes(b, PrivateKeyLength)
	if err != nil {
		return PrivateKey{}, err
	}
	return PrivateKey{Bytes: bb}, nil
}

func PrivateKeyFromHex(s string) (PrivateKey, error) {
	bb, err := newBytesFromHex(s, PrivateKeyLength)
	if err != nil {
		return PrivateKey{}, err
	}
	return PrivateKey{Bytes: bb}, nil
}

// PublicKey returns the uncompressed (64-byte) public key derived from the
// private key.
func (k PrivateKey) PublicKey() PublicKey {
	priv, err := crypto.ToECDSA(k.raw)
	if err != nil {
		// raw is length-validated; ToECDSA should only fail on malformed key
		// material (e.g. zero scalar). Wrap as a panic since this would be a
		// programming error: a PrivateKey value should always be usable.
		panic(fmt.Errorf("PrivateKey.PublicKey: %w", err))
	}
	// Marshal X || Y (64 bytes), dropping the 0x04 uncompressed prefix.
	uncompressed := crypto.FromECDSAPub(&priv.PublicKey)
	bb, _ := newBytes(uncompressed[1:], PublicKeyLength)
	return PublicKey{Bytes: bb}
}

// Sign signs data using the Ethereum signed-message scheme:
//
//	digest = keccak256("\x19Ethereum Signed Message:\n32" || keccak256(data))
//
// matching bee-js PrivateKey.sign. Returns a 65-byte [R || S || V] signature.
func (k PrivateKey) Sign(data []byte) (Signature, error) {
	priv, err := crypto.ToECDSA(k.raw)
	if err != nil {
		return Signature{}, err
	}
	digest := ethSignedMessageDigest(data)
	sig, err := crypto.Sign(digest, priv)
	if err != nil {
		return Signature{}, err
	}
	// go-ethereum returns V in {0,1}; bee-js stores {27,28}. Normalize to
	// match bee-js wire format.
	if sig[64] < 27 {
		sig[64] += 27
	}
	bb, err := newBytes(sig, SignatureLength)
	if err != nil {
		return Signature{}, err
	}
	return Signature{Bytes: bb}, nil
}

// ToECDSA returns the raw go-ethereum *ecdsa.PrivateKey for callers that need
// it (e.g. SOC creation paths still using go-ethereum primitives directly).
func (k PrivateKey) ToECDSA() (*ecdsa.PrivateKey, error) {
	return crypto.ToECDSA(k.raw)
}

// ============================================================================
// PublicKey (64 bytes uncompressed: X || Y)
// ============================================================================

type PublicKey struct{ Bytes }

func NewPublicKey(b []byte) (PublicKey, error) {
	// Allow 33-byte compressed input by decompressing first.
	if len(b) == 33 {
		pub, err := crypto.DecompressPubkey(b)
		if err != nil {
			return PublicKey{}, fmt.Errorf("decompress public key: %w", err)
		}
		uncompressed := crypto.FromECDSAPub(pub)
		bb, _ := newBytes(uncompressed[1:], PublicKeyLength)
		return PublicKey{Bytes: bb}, nil
	}
	bb, err := newBytes(b, PublicKeyLength)
	if err != nil {
		return PublicKey{}, err
	}
	return PublicKey{Bytes: bb}, nil
}

func PublicKeyFromHex(s string) (PublicKey, error) {
	bb, err := newBytesFromHex(s, PublicKeyLength)
	if err != nil {
		return PublicKey{}, err
	}
	return PublicKey{Bytes: bb}, nil
}

// Address returns the Ethereum address (last 20 bytes of keccak256(pubkey)).
func (p PublicKey) Address() EthAddress {
	hash := crypto.Keccak256(p.raw)
	bb, _ := newBytes(hash[12:], EthAddressLength)
	return EthAddress{Bytes: bb}
}

// CompressedBytes returns the 33-byte compressed SEC1 encoding.
func (p PublicKey) CompressedBytes() ([]byte, error) {
	// Reconstruct an *ecdsa.PublicKey from X||Y to call CompressPubkey.
	if len(p.raw) != PublicKeyLength {
		return nil, errors.New("invalid public key length")
	}
	x := new(big.Int).SetBytes(p.raw[:32])
	y := new(big.Int).SetBytes(p.raw[32:])
	pub := &ecdsa.PublicKey{Curve: crypto.S256(), X: x, Y: y}
	return crypto.CompressPubkey(pub), nil
}

// CompressedHex returns the compressed public key as lower-case hex.
func (p PublicKey) CompressedHex() (string, error) {
	b, err := p.CompressedBytes()
	if err != nil {
		return "", err
	}
	return Bytes{raw: b}.Hex(), nil
}

// ============================================================================
// Signature (65 bytes: R || S || V)
// ============================================================================

type Signature struct{ Bytes }

func NewSignature(b []byte) (Signature, error) {
	bb, err := newBytes(b, SignatureLength)
	if err != nil {
		return Signature{}, err
	}
	return Signature{Bytes: bb}, nil
}

func SignatureFromHex(s string) (Signature, error) {
	bb, err := newBytesFromHex(s, SignatureLength)
	if err != nil {
		return Signature{}, err
	}
	return Signature{Bytes: bb}, nil
}

// RecoverPublicKey recovers the public key that produced this signature for
// the given data, assuming the signature was produced via PrivateKey.Sign
// (Ethereum signed-message scheme).
func (s Signature) RecoverPublicKey(data []byte) (PublicKey, error) {
	digest := ethSignedMessageDigest(data)
	// go-ethereum expects V in {0,1}; normalize from bee-js' {27,28}.
	sig := make([]byte, SignatureLength)
	copy(sig, s.raw)
	if sig[64] >= 27 {
		sig[64] -= 27
	}
	uncompressed, err := crypto.Ecrecover(digest, sig)
	if err != nil {
		return PublicKey{}, err
	}
	bb, _ := newBytes(uncompressed[1:], PublicKeyLength)
	return PublicKey{Bytes: bb}, nil
}

// IsValid reports whether the signature was produced by the holder of
// expected's private key, signing data via the Ethereum signed-message scheme.
func (s Signature) IsValid(data []byte, expected EthAddress) bool {
	pub, err := s.RecoverPublicKey(data)
	if err != nil {
		return false
	}
	return pub.Address().Equal(expected.Bytes)
}

// ethSignedMessageDigest returns
// keccak256("\x19Ethereum Signed Message:\n32" || keccak256(data)).
func ethSignedMessageDigest(data []byte) []byte {
	inner := crypto.Keccak256(data)
	prefix := []byte("\x19Ethereum Signed Message:\n32")
	return crypto.Keccak256(append(prefix, inner...))
}

// ============================================================================
// Span (8 bytes, little-endian uint64) — chunk payload length
// ============================================================================

type Span struct{ Bytes }

func NewSpan(b []byte) (Span, error) {
	bb, err := newBytes(b, SpanLength)
	if err != nil {
		return Span{}, err
	}
	return Span{Bytes: bb}, nil
}

// SpanFromUint64 encodes n as little-endian per Swarm chunk format.
func SpanFromUint64(n uint64) Span {
	buf := make([]byte, SpanLength)
	binary.LittleEndian.PutUint64(buf, n)
	bb, _ := newBytes(buf, SpanLength)
	return Span{Bytes: bb}
}

// ToUint64 returns the decoded little-endian uint64.
func (s Span) ToUint64() uint64 {
	return binary.LittleEndian.Uint64(s.raw)
}

// ============================================================================
// FeedIndex (8 bytes, big-endian uint64); MAX_UINT64 represents "epoch root"
// ============================================================================

type FeedIndex struct{ Bytes }

// FeedIndexMinusOne is the all-0xff sentinel used by bee-js as the "before
// first" / wraparound index.
var FeedIndexMinusOne = func() FeedIndex {
	buf := make([]byte, FeedIndexLength)
	for i := range buf {
		buf[i] = 0xff
	}
	bb, _ := newBytes(buf, FeedIndexLength)
	return FeedIndex{Bytes: bb}
}()

func NewFeedIndex(b []byte) (FeedIndex, error) {
	bb, err := newBytes(b, FeedIndexLength)
	if err != nil {
		return FeedIndex{}, err
	}
	return FeedIndex{Bytes: bb}, nil
}

// FeedIndexFromUint64 encodes n as big-endian per the feed index format.
func FeedIndexFromUint64(n uint64) FeedIndex {
	buf := make([]byte, FeedIndexLength)
	binary.BigEndian.PutUint64(buf, n)
	bb, _ := newBytes(buf, FeedIndexLength)
	return FeedIndex{Bytes: bb}
}

// ToUint64 returns the decoded big-endian uint64. For FeedIndexMinusOne this
// returns math.MaxUint64.
func (f FeedIndex) ToUint64() uint64 {
	return binary.BigEndian.Uint64(f.raw)
}

// Next returns the successor index. The MinusOne sentinel wraps to 0,
// matching bee-js FeedIndex.next behavior.
func (f FeedIndex) Next() FeedIndex {
	if f.Equal(FeedIndexMinusOne.Bytes) {
		return FeedIndexFromUint64(0)
	}
	return FeedIndexFromUint64(f.ToUint64() + 1)
}
