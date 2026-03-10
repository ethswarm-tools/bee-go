package file_test

import (
	"testing"

	"github.com/ethersphere/bee-go/pkg/file"
)

func TestMantaray(t *testing.T) {
	node := file.NewMantarayNode()
	err := node.AddFork(file.PathToBytes("/index.html"), []byte("ref1"), map[string]string{"Content-Type": "text/html"})
	if err != nil {
		t.Fatalf("AddFork failed: %v", err)
	}

	if len(node.Forks) == 0 {
		t.Fatal("Node should have forks")
	}

	// Check basic retrieval simulation
	// In a real trie we'd walk it, here we just check the map
	fork := node.Forks[byte('i')]
	if fork == nil {
		t.Fatal("Fork 'i' not found")
	}
}
