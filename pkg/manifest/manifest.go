// Package manifest implements Mantaray, the patricia-trie file manifest used
// by Swarm. Ported from bee-js src/manifest/manifest.ts; the binary format and
// trie semantics match bee-js exactly.
//
// A MantarayNode represents one node of the trie. Forks index children by the
// first byte of the path edge. The marshaled form is a 32-byte obfuscation
// key followed by an XOR-encrypted body containing a 31-byte version hash, a
// 1-byte target-address length, the target address, a 256-bit fork bitmap,
// and the per-fork records.
package manifest

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Type bitfield used in fork records.
const (
	TypeValue             = 2
	TypeEdge              = 4
	TypeWithPathSeparator = 8
	TypeWithMetadata      = 16
)

// PathSeparator is the slash byte used by the Mantaray path encoding.
const PathSeparator byte = '/'

// MaxPrefixLength is the per-fork prefix length cap used by the wire format.
// AddFork chunks longer paths into chained nodes of at most this many bytes.
const MaxPrefixLength = 30

// Version02HashHex is the version marker that identifies the v0.2 Mantaray
// binary format. Only the first 31 bytes are written into the marshaled
// header; the 32nd byte slot is repurposed as the target-address length.
const Version02HashHex = "5768b3b6a7db56d21d1abff40d41cebfc83448fed8d7e9b06ec0d3b073f28f7b"

var version02Hash = mustHex(Version02HashHex)

// nullAddress is 32 zero bytes, denoting "this node has no file target".
var nullAddress = make([]byte, 32)

// IsNullAddress reports whether b is the canonical 32-byte zero NULL_ADDRESS,
// or empty/short (treated equivalently by the wire format check).
func IsNullAddress(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	return bytes.Equal(b, nullAddress)
}

// Fork is one edge of the trie. Prefix is the path bytes that select this
// child; Node is the child node. Forks are stored in MantarayNode.Forks under
// the key Prefix[0].
type Fork struct {
	Prefix []byte
	Node   *MantarayNode
}

// MantarayNode is one node in the Mantaray trie. Field names mirror bee-js so
// the porting story stays obvious.
//
//   - ObfuscationKey is the 32-byte XOR mask used when serializing this node;
//     freshly-constructed nodes use a zero key. Real-world manifests can
//     populate it with a random key for privacy.
//   - SelfAddress is the chunk address of the marshaled node, populated after
//     CalculateSelfAddress / save. nil means "not computed".
//   - TargetAddress is the file reference at this node (or all-zero / nil for
//     pure directory nodes).
//   - Path is the edge label inherited from the parent fork's prefix; the
//     root node has an empty path.
//   - Forks is the map of child edges keyed by first prefix byte.
//   - Metadata is optional key/value annotations carried alongside the fork
//     in the wire format.
//   - Parent is set during AddFork / Unmarshal so FullPath can climb back up.
type MantarayNode struct {
	ObfuscationKey []byte
	SelfAddress    []byte
	TargetAddress  []byte
	Metadata       map[string]string
	Path           []byte
	Forks          map[byte]*Fork
	Parent         *MantarayNode
}

// New returns an empty Mantaray root.
func New() *MantarayNode {
	return &MantarayNode{
		ObfuscationKey: make([]byte, 32),
		TargetAddress:  make([]byte, 32),
		Forks:          make(map[byte]*Fork),
	}
}

// FullPath walks up the parent chain and concatenates each node's Path, so a
// leaf node added via AddFork("/foo/bar.txt", ...) returns "/foo/bar.txt".
func (n *MantarayNode) FullPath() []byte {
	if n.Parent == nil {
		return append([]byte(nil), n.Path...)
	}
	return append(n.Parent.FullPath(), n.Path...)
}

// FullPathString is FullPath as a Go string. Mantaray paths are UTF-8 by
// convention but the trie operates on raw bytes.
func (n *MantarayNode) FullPathString() string {
	return string(n.FullPath())
}

// DetermineType packs the type bitfield used in a fork record. See the
// TypeXxx constants for the bits.
func (n *MantarayNode) DetermineType() byte {
	var t byte
	if !IsNullAddress(n.TargetAddress) || (len(n.Path) == 1 && n.Path[0] == PathSeparator) {
		t |= TypeValue
	}
	if len(n.Forks) > 0 {
		t |= TypeEdge
	}
	if bytes.IndexByte(n.Path, PathSeparator) != -1 && (len(n.Path) != 1 || n.Path[0] != PathSeparator) {
		t |= TypeWithPathSeparator
	}
	if n.Metadata != nil {
		t |= TypeWithMetadata
	}
	return t
}

// ============================================================================
// Marshal / Unmarshal
// ============================================================================

// Marshal returns the wire format of this node. Children must already have
// SelfAddress populated (call CalculateSelfAddress / SaveRecursively first).
func (n *MantarayNode) Marshal() ([]byte, error) {
	for _, fork := range n.Forks {
		if len(fork.Node.SelfAddress) == 0 {
			addr, err := fork.Node.CalculateSelfAddress()
			if err != nil {
				return nil, fmt.Errorf("calculate child self-address: %w", err)
			}
			fork.Node.SelfAddress = addr
		}
	}

	// Header: 31-byte version hash + 1-byte target-address length.
	header := make([]byte, 32)
	copy(header, version02Hash)
	rootDirNode := IsNullAddress(n.TargetAddress) && len(n.Path) == 1 && n.Path[0] == PathSeparator
	if rootDirNode {
		header[31] = 0
	} else {
		header[31] = byte(len(n.TargetAddress))
	}

	forkBitmap := make([]byte, 32)
	for k := range n.Forks {
		setBitLE(forkBitmap, int(k))
	}

	body := append([]byte{}, header...)
	if !rootDirNode {
		body = append(body, n.TargetAddress...)
	}
	body = append(body, forkBitmap...)

	// Forks are emitted in ascending bitmap order so encoders agree.
	for i := 0; i < 256; i++ {
		if !getBitLE(forkBitmap, i) {
			continue
		}
		f, ok := n.Forks[byte(i)]
		if !ok {
			continue
		}
		fb, err := f.marshal()
		if err != nil {
			return nil, err
		}
		body = append(body, fb...)
	}

	xorInPlace(body, n.ObfuscationKey)
	out := make([]byte, 0, 32+len(body))
	out = append(out, n.ObfuscationKey...)
	out = append(out, body...)
	return out, nil
}

// Unmarshal parses a marshaled node. selfAddress is the chunk address that
// produced data; it determines the fork-record address length (32 bytes for
// standard Swarm references). Children are referenced by selfAddress only and
// remain unloaded — call LoadRecursively (when implemented) to follow them.
func Unmarshal(data, selfAddress []byte) (*MantarayNode, error) {
	if len(data) < 32 {
		return nil, errors.New("mantaray: data too short")
	}
	obfuscationKey := append([]byte(nil), data[:32]...)
	body := append([]byte(nil), data[32:]...)
	xorInPlace(body, obfuscationKey)

	r := newReader(body)
	versionHash, err := r.read(31)
	if err != nil {
		return nil, fmt.Errorf("read version hash: %w", err)
	}
	if !bytes.Equal(versionHash, version02Hash[:31]) {
		return nil, errors.New("mantaray: invalid version hash")
	}
	targetAddressLength, err := r.readByte()
	if err != nil {
		return nil, err
	}
	target := make([]byte, 32)
	if targetAddressLength > 0 {
		raw, err := r.read(int(targetAddressLength))
		if err != nil {
			return nil, fmt.Errorf("read target address: %w", err)
		}
		copy(target, raw)
	}
	node := &MantarayNode{
		ObfuscationKey: obfuscationKey,
		SelfAddress:    append([]byte(nil), selfAddress...),
		TargetAddress:  target,
		Forks:          make(map[byte]*Fork),
	}

	forkBitmap, err := r.read(32)
	if err != nil {
		return nil, fmt.Errorf("read fork bitmap: %w", err)
	}
	for i := 0; i < 256; i++ {
		if !getBitLE(forkBitmap, i) {
			continue
		}
		f, err := unmarshalFork(r, len(selfAddress))
		if err != nil {
			return nil, fmt.Errorf("read fork %d: %w", i, err)
		}
		f.Node.Parent = node
		node.Forks[byte(i)] = f
	}
	return node, nil
}

// marshal emits a single fork record. See Fork format in the package doc.
func (f *Fork) marshal() ([]byte, error) {
	if len(f.Node.SelfAddress) == 0 {
		return nil, errors.New("fork: child SelfAddress not set")
	}
	if len(f.Prefix) > MaxPrefixLength {
		return nil, fmt.Errorf("fork: prefix length %d exceeds max %d", len(f.Prefix), MaxPrefixLength)
	}
	out := make([]byte, 0, 1+1+30+32)
	out = append(out, f.Node.DetermineType())
	out = append(out, byte(len(f.Prefix)))
	out = append(out, f.Prefix...)
	if len(f.Prefix) < MaxPrefixLength {
		out = append(out, make([]byte, MaxPrefixLength-len(f.Prefix))...)
	}
	out = append(out, f.Node.SelfAddress...)

	if f.Node.Metadata != nil {
		j, err := json.Marshal(f.Node.Metadata)
		if err != nil {
			return nil, fmt.Errorf("marshal metadata: %w", err)
		}
		// Layout: 2-byte BE length + JSON, then padded with 0x0a to a multiple
		// of 32. The length is over the (JSON + padding) bytes only.
		body := make([]byte, 2+len(j))
		copy(body[2:], j)
		// Pad to 32-byte multiple; padding bytes are 0x0a, length excludes the
		// 2-byte length prefix.
		padded := padEndToMultiple(body, 32, 0x0a)
		binary.BigEndian.PutUint16(padded[:2], uint16(len(padded)-2))
		out = append(out, padded...)
	}
	return out, nil
}

// unmarshalFork inverts marshal. addressLength is selfAddress length.
func unmarshalFork(r *reader, addressLength int) (*Fork, error) {
	t, err := r.readByte()
	if err != nil {
		return nil, err
	}
	prefixLength, err := r.readByte()
	if err != nil {
		return nil, err
	}
	prefix, err := r.read(int(prefixLength))
	if err != nil {
		return nil, err
	}
	if int(prefixLength) < MaxPrefixLength {
		if _, err := r.read(MaxPrefixLength - int(prefixLength)); err != nil {
			return nil, err
		}
	}
	selfAddress, err := r.read(addressLength)
	if err != nil {
		return nil, err
	}
	var metadata map[string]string
	if hasType(t, TypeWithMetadata) {
		lenBytes, err := r.read(2)
		if err != nil {
			return nil, err
		}
		mlen := binary.BigEndian.Uint16(lenBytes)
		metaBytes, err := r.read(int(mlen))
		if err != nil {
			return nil, err
		}
		// Strip the 0x0a padding before parsing JSON.
		trimmed := bytes.TrimRight(metaBytes, "\x0a")
		metadata = make(map[string]string)
		if len(trimmed) > 0 {
			if err := json.Unmarshal(trimmed, &metadata); err != nil {
				return nil, fmt.Errorf("parse metadata json: %w", err)
			}
		}
	}
	prefixCopy := append([]byte(nil), prefix...)
	return &Fork{
		Prefix: prefixCopy,
		Node: &MantarayNode{
			ObfuscationKey: make([]byte, 32),
			SelfAddress:    append([]byte(nil), selfAddress...),
			TargetAddress:  make([]byte, 32),
			Path:           prefixCopy,
			Forks:          make(map[byte]*Fork),
			Metadata:       metadata,
		},
	}, nil
}

// ============================================================================
// Trie operations: AddFork, Find, FindClosest, RemoveFork, Fork.split
// ============================================================================

// AddFork inserts a (path, ref, metadata) entry into the trie. Long paths are
// chunked into chained nodes of up to MaxPrefixLength bytes each. Metadata is
// attached only at the terminal node.
//
// AddFork invalidates SelfAddress on every node it touches; call
// CalculateSelfAddress / SaveRecursively before marshaling.
func (n *MantarayNode) AddFork(path []byte, ref swarm.Reference, metadata map[string]string) {
	n.SelfAddress = nil
	tip := n
	remaining := append([]byte(nil), path...)
	for len(remaining) > 0 {
		prefix := remaining
		if len(prefix) > MaxPrefixLength {
			prefix = remaining[:MaxPrefixLength]
		}
		remaining = remaining[len(prefix):]
		isLast := len(remaining) == 0

		bestMatch, matchedPath := tip.FindClosest(prefix)
		remainingPrefix := prefix[len(matchedPath):]

		if len(matchedPath) > 0 {
			tip = bestMatch
		}

		if len(remainingPrefix) == 0 {
			continue
		}

		newNode := &MantarayNode{
			ObfuscationKey: make([]byte, 32),
			TargetAddress:  make([]byte, 32),
			Path:           append([]byte(nil), remainingPrefix...),
			Forks:          make(map[byte]*Fork),
		}
		if isLast {
			if !ref.IsZero() {
				newNode.TargetAddress = ref.Raw()
			}
			newNode.Metadata = metadata
		}
		newFork := &Fork{Prefix: append([]byte(nil), remainingPrefix...), Node: newNode}

		if existing, ok := bestMatch.Forks[remainingPrefix[0]]; ok {
			merged := splitForks(newFork, existing)
			tip.Forks[remainingPrefix[0]] = merged
			merged.Node.Parent = tip
			tip.SelfAddress = nil
			tip = newFork.Node
		} else {
			tip.Forks[remainingPrefix[0]] = newFork
			newFork.Node.Parent = tip
			tip.SelfAddress = nil
			tip = newFork.Node
		}
	}
}

// Find returns the node whose FullPath equals path, or nil.
func (n *MantarayNode) Find(path []byte) *MantarayNode {
	closest, matched := n.FindClosest(path)
	if len(matched) == len(path) {
		return closest
	}
	return nil
}

// FindClosest walks down forks while the path matches. It returns the deepest
// node reached and the bytes of path that were matched along the way.
func (n *MantarayNode) FindClosest(path []byte) (*MantarayNode, []byte) {
	return n.findClosest(path, nil)
}

func (n *MantarayNode) findClosest(path, current []byte) (*MantarayNode, []byte) {
	if len(path) == 0 {
		return n, current
	}
	fork, ok := n.Forks[path[0]]
	if !ok {
		return n, current
	}
	common := commonPrefix(fork.Prefix, path)
	if len(common) == len(fork.Prefix) {
		return fork.Node.findClosest(path[len(fork.Prefix):], append(current, fork.Prefix...))
	}
	return n, current
}

// RemoveFork deletes the fork rooted at path. If the removed node had its own
// children, they are re-inserted under the parent so the trie remains
// consistent.
func (n *MantarayNode) RemoveFork(path []byte) error {
	n.SelfAddress = nil
	if len(path) == 0 {
		return errors.New("RemoveFork: path cannot be empty")
	}
	match := n.Find(path)
	if match == nil {
		return errors.New("RemoveFork: not found")
	}
	parent, matchedPath := n.FindClosest(path[:len(path)-1])
	rest := path[len(matchedPath):]
	if len(rest) == 0 {
		return errors.New("RemoveFork: cannot remove root edge")
	}
	delete(parent.Forks, rest[0])
	for _, fork := range match.Forks {
		newPath := append(append([]byte(nil), match.Path...), fork.Prefix...)
		var ref swarm.Reference
		if !IsNullAddress(fork.Node.TargetAddress) {
			r, err := swarm.NewReference(fork.Node.TargetAddress)
			if err != nil {
				return err
			}
			ref = r
		}
		parent.AddFork(newPath, ref, fork.Node.Metadata)
	}
	return nil
}

// splitForks resolves a prefix collision between two new/existing forks at
// the same first byte. It returns the fork that the parent should store,
// reshuffling node parents and paths to maintain the patricia invariant.
//
// Mirrors bee-js Fork.split.
func splitForks(a, b *Fork) *Fork {
	common := commonPrefix(a.Prefix, b.Prefix)

	if len(common) == len(a.Prefix) {
		// b lives under a.
		remaining := b.Prefix[len(common):]
		b.Node.Path = remaining
		b.Prefix = remaining
		b.Node.Parent = a.Node
		a.Node.Forks[remaining[0]] = b
		return a
	}
	if len(common) == len(b.Prefix) {
		// a lives under b.
		remaining := a.Prefix[len(common):]
		a.Node.Path = remaining
		a.Prefix = remaining
		a.Node.Parent = b.Node
		b.Node.Forks[remaining[0]] = a
		return b
	}

	// Neither contains the other: insert a new branching node.
	branch := &MantarayNode{
		ObfuscationKey: make([]byte, 32),
		TargetAddress:  make([]byte, 32),
		Path:           append([]byte(nil), common...),
		Forks:          make(map[byte]*Fork),
	}
	a.Node.Path = a.Prefix[len(common):]
	b.Node.Path = b.Prefix[len(common):]
	newAFork := &Fork{Prefix: a.Prefix[len(common):], Node: a.Node}
	newBFork := &Fork{Prefix: b.Prefix[len(common):], Node: b.Node}
	branch.Forks[newAFork.Prefix[0]] = newAFork
	branch.Forks[newBFork.Prefix[0]] = newBFork
	newAFork.Node.Parent = branch
	newBFork.Node.Parent = branch

	a.Prefix = newAFork.Prefix
	b.Prefix = newBFork.Prefix

	return &Fork{Prefix: append([]byte(nil), common...), Node: branch}
}

// ============================================================================
// SelfAddress (single-chunk BMT for now)
// ============================================================================

// CalculateSelfAddress returns the chunk address of this node's marshaled
// form. If a SelfAddress was already populated (e.g. by a prior save), it is
// returned unchanged.
//
// Multi-chunk marshaled nodes are streamed through swarm.FileChunker:
// the marshaled bytes are treated as a file, hashed at each level of
// the BMT, and the root chunk's address is returned. For the common
// case where Marshal fits in one chunk the chunker still produces the
// same single-leaf BMT address as a direct CalculateChunkAddress call.
func (n *MantarayNode) CalculateSelfAddress() ([]byte, error) {
	if len(n.SelfAddress) > 0 {
		return append([]byte(nil), n.SelfAddress...), nil
	}
	data, err := n.Marshal()
	if err != nil {
		return nil, err
	}
	chunker := swarm.NewFileChunker(nil)
	if _, err := chunker.Write(data); err != nil {
		return nil, err
	}
	root, err := chunker.Finalize()
	if err != nil {
		return nil, err
	}
	return root.Address.Raw(), nil
}

// ============================================================================
// Tree traversal helpers
// ============================================================================

// Collect returns every descendant node that has a non-null TargetAddress.
// Useful for "list all files in this manifest" operations after a manifest
// has been fully loaded.
func (n *MantarayNode) Collect() []*MantarayNode {
	var out []*MantarayNode
	n.collect(&out)
	return out
}

func (n *MantarayNode) collect(out *[]*MantarayNode) {
	for _, fork := range n.Forks {
		if !IsNullAddress(fork.Node.TargetAddress) {
			*out = append(*out, fork.Node)
		}
		fork.Node.collect(out)
	}
}

// CollectAndMap returns {fullPath: hex(targetAddress)} for every file leaf.
// Nodes whose TargetAddress fails Reference validation are skipped.
func (n *MantarayNode) CollectAndMap() map[string]string {
	out := make(map[string]string)
	for _, node := range n.Collect() {
		ref, err := swarm.NewReference(node.TargetAddress)
		if err != nil {
			continue
		}
		out[node.FullPathString()] = ref.Hex()
	}
	return out
}

// ============================================================================
// Internal helpers (XOR cipher, bit ops, prefix math, byte reader)
// ============================================================================

// xorInPlace XORs each byte of dst with key, repeating key as needed. dst is
// modified in place. Empty key is a no-op (matches bee-js xorCypher behavior).
func xorInPlace(dst, key []byte) {
	if len(key) == 0 {
		return
	}
	for i := range dst {
		dst[i] ^= key[i%len(key)]
	}
}

// setBitLE / getBitLE: bee-js stores fork-bitmap bits in little-endian
// per-byte order, so bit i is byte (i>>3), bit (i & 7) — same as standard
// LSB-first packing.
func setBitLE(buf []byte, idx int) { buf[idx>>3] |= 1 << uint(idx&7) }
func getBitLE(buf []byte, idx int) bool {
	return buf[idx>>3]&(1<<uint(idx&7)) != 0
}

func hasType(t byte, mask byte) bool { return t&mask == mask }

// commonPrefix returns the longest leading byte slice shared by a and b.
func commonPrefix(a, b []byte) []byte {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:n]
}

// padEndToMultiple right-pads buf with pad bytes so its length is a multiple
// of m. If buf is already aligned, it is returned unchanged.
func padEndToMultiple(buf []byte, m int, pad byte) []byte {
	r := len(buf) % m
	if r == 0 {
		return buf
	}
	out := make([]byte, len(buf)+m-r)
	copy(out, buf)
	for i := len(buf); i < len(out); i++ {
		out[i] = pad
	}
	return out
}

func mustHex(s string) []byte {
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		hi, ok1 := hexNibble(s[2*i])
		lo, ok2 := hexNibble(s[2*i+1])
		if !ok1 || !ok2 {
			panic("manifest: invalid hex constant")
		}
		out[i] = hi<<4 | lo
	}
	return out
}

func hexNibble(c byte) (byte, bool) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', true
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, true
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

// reader is a tiny cursor over a byte slice that returns io.ErrUnexpectedEOF
// on short reads. We don't pull in bytes.Reader because we want a single error
// type for all the wire-format reads.
type reader struct {
	buf []byte
	pos int
}

func newReader(b []byte) *reader { return &reader{buf: b} }

func (r *reader) read(n int) ([]byte, error) {
	if r.pos+n > len(r.buf) {
		return nil, io.ErrUnexpectedEOF
	}
	out := r.buf[r.pos : r.pos+n]
	r.pos += n
	return out, nil
}

func (r *reader) readByte() (byte, error) {
	b, err := r.read(1)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}
