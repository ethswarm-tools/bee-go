// Generate a fresh secp256k1 keypair and print all the useful forms
// (private hex, compressed public hex, EIP-55 address). Pure offline.
//
// Usage:
//
//	go run ./examples/key-gen
//
// Save the printed private key with the same care you'd give an
// Ethereum private key — it controls every feed, SOC, GSOC and ACT
// identity built on top of it.
package main

import (
	"crypto/rand"
	"fmt"
	"os"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 32 bytes from the OS CSPRNG.
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Errorf("rand: %w", err)
	}

	signer, err := swarm.NewPrivateKey(bytes[:])
	if err != nil {
		return fmt.Errorf("private key: %w", err)
	}
	public := signer.PublicKey()
	address := public.Address()
	compressed, err := public.CompressedHex()
	if err != nil {
		return fmt.Errorf("compressed: %w", err)
	}

	fmt.Println("Generated secp256k1 keypair")
	fmt.Println("===========================")
	fmt.Printf("Private key:           0x%s\n", signer.Hex())
	fmt.Printf("Public key (uncomp):   0x%s\n", public.Hex())
	fmt.Printf("Public key (comp):     0x%s\n", compressed)
	fmt.Printf("Ethereum address:      %s\n", address.ToChecksum())
	fmt.Println()
	fmt.Println("Usage hints:")
	fmt.Printf("- Set BEE_SIGNER_HEX=%s to drive feed-update / pss / soc examples\n", signer.Hex())
	fmt.Println("- The address is what readers need to follow your feeds")
	return nil
}
