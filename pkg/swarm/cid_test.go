package swarm_test

import (
	"testing"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestCID(t *testing.T) {
	// CID Tests
	ref := "ca6357a08e317d15ec560fef34e4c45f8f19f01c75d6f20a7021602e9575a617"
	cid, err := swarm.ConvertReferenceToCID(ref, "feed")
	if err != nil {
		t.Fatalf("ConvertReferenceToCID failed: %v", err)
	}

	decoded, err := swarm.ConvertCIDToReference(cid)
	if err != nil {
		t.Fatalf("ConvertCIDToReference failed: %v", err)
	}

	if decoded.Type != "feed" {
		t.Errorf("Decoded type = %s, want feed", decoded.Type)
	}
	if decoded.Reference != ref {
		t.Errorf("Decoded ref = %s, want %s", decoded.Reference, ref)
	}
}
