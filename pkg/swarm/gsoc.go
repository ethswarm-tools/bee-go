package swarm

import (
	"crypto/ecdsa"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// GSOCMine mines a signer for the target overlay address.
// target: target overlay address (32 bytes)
// identifier: identifier (32 bytes)
// proximity: target proximity (default 12 in bee-js)
func GSOCMine(target []byte, identifier []byte, proximity int) (*ecdsa.PrivateKey, error) {
	// Implementation based on bee-js `gsocMine`
	// start = 0xb33n (2867)
	start := int64(0xb33)

	// Bee-js `PeerAddress` is 32 bytes (HexString).
	// `common.BytesToAddress` creates 20 byte address. We need raw bytes comparison.

	for i := int64(0); i < 0xffff; i++ {
		// val = start + i
		val := start + i

		// Private Key from number? `Binary.numberToUint256(start + i, 'BE')`
		// We need to create a private key where D = val.
		privKeyBytes := make([]byte, 32)
		binary.BigEndian.PutUint64(privKeyBytes[24:], uint64(val)) // padded

		privKey, err := crypto.ToECDSA(privKeyBytes)
		if err != nil {
			continue
		}

		signerAddr := crypto.PubkeyToAddress(privKey.PublicKey)

		// Calculate SOC Address: Keccak(Identifier + SignerAddress)
		// Identifier (32) + SignerAddress (20)
		socAddr := crypto.Keccak256(identifier, signerAddr.Bytes())

		// Calculate Proximity
		po := Proximity(socAddr, target)

		if po >= proximity {
			return privKey, nil
		}
	}

	return nil, fmt.Errorf("could not mine a valid signer")
}

// Proximity calculates the proximity order between two chunks.
// Returns the number of matching prefix bits.
func Proximity(one, two []byte) int {
	// Check common prefix bits
	// implementation skipped for brevity in thought, but required here.
	// standard xor and count leading zeros.

	if len(one) != len(two) {
		return 0
	}
	var po int
	for i := 0; i < len(one); i++ {
		b := one[i] ^ two[i]
		if b == 0 {
			po += 8
		} else {
			// count leading zeros in byte b
			for j := 0; j < 8; j++ {
				if (b & (0x80 >> j)) == 0 {
					po++
				} else {
					break
				}
			}
			break
		}
	}
	return po
}
