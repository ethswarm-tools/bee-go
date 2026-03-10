package swarm_test

import (
	"testing"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestBMT(t *testing.T) {
	// BMT Test
	data := make([]byte, 4096)
	addr, err := swarm.CalculateChunkAddress(append(make([]byte, 8), data...))
	if err != nil {
		t.Fatalf("CalculateChunkAddress error = %v", err)
	}
	if len(addr) != 32 {
		t.Errorf("CalculateChunkAddress length = %d, want 32", len(addr))
	}
}
