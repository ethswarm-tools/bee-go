package swarm_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestSOCCreation(t *testing.T) {
	// SOC Signing Test
	privKey, _ := crypto.GenerateKey()
	id := make([]byte, 32)
	payload := []byte("hello world")
	soc, err := swarm.CreateSOC(id, payload, privKey)
	if err != nil {
		t.Fatalf("CreateSOC error = %v", err)
	}
	if len(soc.Signature) != 65 {
		t.Errorf("Signature length = %d, want 65", len(soc.Signature))
	}
}
