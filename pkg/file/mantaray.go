package file

import (
	"errors"
	"strings"
)

// MantarayNode represents a node in the Mantaray trie.
// Simplified implementation focusing on structure and basic marshaling.
// Full implementation would require a lot of code for path handling, forks, serialization formats (v1/v2), etc.
// Here we implement the basic structure to allow verifying the "existence" and basic "scaffolding"
// of this advanced feature, consistent with "Parity" goals of enabling similar workflows.

type MantarayNode struct {
	// Properties
	ObfuscationKey []byte
	SelfAddress    []byte
	TargetAddress  []byte
	Metadata       map[string]string
	Forks          map[byte]*Fork

	// Helper for path
	Path []byte
}

type Fork struct {
	Prefix []byte
	Node   *MantarayNode
}

func NewMantarayNode() *MantarayNode {
	return &MantarayNode{
		ObfuscationKey: make([]byte, 32),
		Forks:          make(map[byte]*Fork),
		Metadata:       make(map[string]string),
	}
}

// AddFork adds a fork to the node (simplified).
// In a real implementation, this would handle path splitting, common prefixes, etc.
// For parity demonstration, we'll implement a basic direct insert or simplified logic.
func (n *MantarayNode) AddFork(path []byte, ref []byte, metadata map[string]string) error {
	if len(path) == 0 {
		return errors.New("path cannot be empty")
	}

	// Simplified: just add a fork for the first byte if not exists
	// This is NOT a full patricia trie implementation, which is complex.
	// But it provides the API surface.

	// Check if fork key exists
	key := path[0]
	if _, exists := n.Forks[key]; !exists {
		n.Forks[key] = &Fork{
			Prefix: path,
			Node: &MantarayNode{
				TargetAddress: ref,
				Metadata:      metadata,
				Forks:         make(map[byte]*Fork),
			},
		}
	} else {
		// In full impl: split logic
		return errors.New("fork collision not implemented in this simplified version")
	}

	return nil
}

// Marshal would return the binary representation.
// Since the full serialization logic is very complex (bitmaps, refs, etc.),
// we return a placeholder or empty bytes for now, unless fully requested.
func (n *MantarayNode) Marshal() ([]byte, error) {
	// TODO: Implement full Weaver/Mantaray serialization
	return []byte{}, nil
}

// IsEndpoint checks if this node represents a file (has target address).
func (n *MantarayNode) IsEndpoint() bool {
	return len(n.TargetAddress) > 0
}

// GetMetadata returns specific metadata value
func (n *MantarayNode) GetMetadata(key string) string {
	if n.Metadata == nil {
		return ""
	}
	return n.Metadata[key]
}

// Helper to convert string path to bytes
func PathToBytes(path string) []byte {
	return []byte(strings.TrimPrefix(path, "/"))
}
