package swarm

import (
	"encoding/base32"
	"encoding/hex"
	"errors"
	"strings"
)

// Codecs
const (
	SwarmManifestCodec = 0xfa
	SwarmFeedCodec     = 0xfb
)

// DecodedCID represents a decoded CID.
type DecodedCID struct {
	Type      string
	Reference string
}

// ConvertReferenceToCID converts a Swarm reference to a CID.
// refType: "feed" or "manifest"
func ConvertReferenceToCID(reference string, refType string) (string, error) {
	refBytes, err := hex.DecodeString(reference)
	if err != nil {
		return "", err
	}

	var codec byte
	switch refType {
	case "feed":
		codec = SwarmFeedCodec
	case "manifest":
		codec = SwarmManifestCodec
	default:
		return "", errors.New("invalid reference type")
	}

	version := byte(1)
	unknown := byte(1)   // from bee-js
	sha256 := byte(0x1b) // keccak256 multicodec
	size := byte(32)

	// Concat: version + codec + unknown + sha256 + size
	header := []byte{version, codec, unknown, sha256, size}

	// Base32 encode header
	// bee-js uses standard RFC4648 without padding, but lowercase?
	// bee-js: "b" + base32(header + hash) ?
	// No, bee-js: base32(header) + base32(hash) ?
	// bee-js code:
	// const header = Binary.uint8ArrayToBase32(Binary.concatBytes(version, codec, unknown, sha256, size)).replace(/\=+$/, '')
	// const hash = reference.toBase32().replace(/\=+$/, '')
	// return `${base32}${header}${hash}`.toLowerCase()

	// We need a Base32 encoder. Go has standard StdEncoding.

	enc := base32.StdEncoding.WithPadding(base32.NoPadding)
	headerStr := enc.EncodeToString(header)
	hashStr := enc.EncodeToString(refBytes)

	return strings.ToLower("b" + headerStr + hashStr), nil
}

// ConvertCIDToReference converts a CID to a Swarm reference.
func ConvertCIDToReference(cid string) (*DecodedCID, error) {
	if !strings.HasPrefix(cid, "b") {
		return nil, errors.New("invalid CID prefix")
	}

	// bee-js logic:
	// const bytes = Binary.base32ToUint8Array(cid.toUpperCase().slice(1))
	// const codec = bytes[1]
	// const reference = bytes.slice(-32)

	cidBody := cid[1:]
	data, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(cidBody))
	if err != nil {
		return nil, err
	}

	if len(data) < 34 { // at least header + some bytes? Header is 5 bytes. Hash is 32. Total 37.
		return nil, errors.New("invalid CID length")
	}

	codec := data[1]
	var refType string
	switch codec {
	case SwarmFeedCodec:
		refType = "feed"
	case SwarmManifestCodec:
		refType = "manifest"
	default:
		return nil, errors.New("unknown codec")
	}

	// Ref is last 32 bytes
	if len(data) < 32 {
		return nil, errors.New("invalid data length")
	}
	refBytes := data[len(data)-32:]
	refHex := hex.EncodeToString(refBytes)

	return &DecodedCID{
		Type:      refType,
		Reference: refHex,
	}, nil
}
