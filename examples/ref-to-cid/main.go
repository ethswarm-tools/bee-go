// Convert a Swarm reference to a CID and back. Pure offline. Useful
// for IPFS-style addressing or interop with tooling that consumes CIDs.
//
// Usage:
//
//	go run ./examples/ref-to-cid <reference> [manifest|feed]
//
// Defaults: kind = "manifest". The same hex reference yields different
// CIDs for "manifest" (codec 0xfa) and "feed" (codec 0xfb).
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: ref-to-cid <reference> [manifest|feed]")
	}
	refHex := os.Args[1]
	kind := "manifest"
	if len(os.Args) >= 3 {
		kind = strings.ToLower(os.Args[2])
	}
	if kind != "manifest" && kind != "feed" {
		return fmt.Errorf("unknown kind %q — expected manifest or feed", kind)
	}

	reference, err := swarm.ReferenceFromHex(refHex)
	if err != nil {
		return fmt.Errorf("invalid reference: %w", err)
	}

	cid, err := swarm.ConvertReferenceToCID(reference.Hex(), kind)
	if err != nil {
		return fmt.Errorf("convert: %w", err)
	}

	fmt.Println("Reference → CID")
	fmt.Println("---------------")
	fmt.Printf("Reference: %s\n", reference.Hex())
	fmt.Printf("Kind:      %s\n", kind)
	fmt.Printf("CID:       %s\n", cid)

	decoded, err := swarm.ConvertCIDToReference(cid)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	fmt.Println("\nCID → Reference (round-trip)")
	fmt.Println("----------------------------")
	fmt.Printf("Reference: %s\n", decoded.Reference)
	fmt.Printf("Type:      %s\n", decoded.Type)
	return nil
}
