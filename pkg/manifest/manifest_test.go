package manifest

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

func mustRef(t *testing.T, hex string) swarm.Reference {
	t.Helper()
	r, err := swarm.ReferenceFromHex(hex)
	if err != nil {
		t.Fatalf("ReferenceFromHex(%q): %v", hex, err)
	}
	return r
}

func TestAddFork_FindRoundTrip(t *testing.T) {
	root := New()
	ref := mustRef(t, strings.Repeat("ab", 32))
	root.AddFork([]byte("/index.html"), ref, map[string]string{"Content-Type": "text/html"})

	got := root.Find([]byte("/index.html"))
	if got == nil {
		t.Fatal("Find: expected node, got nil")
	}
	if !bytes.Equal(got.TargetAddress, ref.Raw()) {
		t.Errorf("TargetAddress = %x, want %x", got.TargetAddress, ref.Raw())
	}
	if got.Metadata["Content-Type"] != "text/html" {
		t.Errorf("metadata Content-Type = %q, want text/html", got.Metadata["Content-Type"])
	}
}

func TestAddFork_FullPathClimbsParent(t *testing.T) {
	root := New()
	ref := mustRef(t, strings.Repeat("cd", 32))
	root.AddFork([]byte("/css/style.css"), ref, nil)

	leaf := root.Find([]byte("/css/style.css"))
	if leaf == nil {
		t.Fatal("Find leaf: nil")
	}
	if got := leaf.FullPathString(); got != "/css/style.css" {
		t.Errorf("FullPathString = %q, want /css/style.css", got)
	}
}

func TestAddFork_PrefixSplit(t *testing.T) {
	// Both paths share "/co", differ at byte 3. Insertion must split into a
	// branching node so both leaves remain reachable.
	root := New()
	r1 := mustRef(t, strings.Repeat("11", 32))
	r2 := mustRef(t, strings.Repeat("22", 32))
	root.AddFork([]byte("/contact"), r1, nil)
	root.AddFork([]byte("/content"), r2, nil)

	contact := root.Find([]byte("/contact"))
	content := root.Find([]byte("/content"))
	if contact == nil || content == nil {
		t.Fatalf("Find returned nil: contact=%v content=%v", contact, content)
	}
	if !bytes.Equal(contact.TargetAddress, r1.Raw()) {
		t.Errorf("contact target wrong")
	}
	if !bytes.Equal(content.TargetAddress, r2.Raw()) {
		t.Errorf("content target wrong")
	}
}

func TestAddFork_LongPathChunked(t *testing.T) {
	// A path longer than MaxPrefixLength (30) bytes must be split across
	// chained nodes; Find should still resolve it.
	root := New()
	long := []byte("/" + strings.Repeat("a", 60))
	ref := mustRef(t, strings.Repeat("33", 32))
	root.AddFork(long, ref, nil)

	leaf := root.Find(long)
	if leaf == nil {
		t.Fatalf("Find long path: nil")
	}
	if !bytes.Equal(leaf.TargetAddress, ref.Raw()) {
		t.Errorf("long-path target wrong")
	}
	if leaf.FullPathString() != string(long) {
		t.Errorf("FullPathString = %q, want %q", leaf.FullPathString(), string(long))
	}
}

func TestMarshal_RoundTrip(t *testing.T) {
	root := New()
	r1 := mustRef(t, strings.Repeat("11", 32))
	r2 := mustRef(t, strings.Repeat("22", 32))
	root.AddFork([]byte("/a"), r1, map[string]string{"k": "v"})
	root.AddFork([]byte("/b"), r2, nil)

	addr, err := root.CalculateSelfAddress()
	if err != nil {
		t.Fatalf("CalculateSelfAddress: %v", err)
	}
	root.SelfAddress = addr

	data, err := root.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	out, err := Unmarshal(data, addr)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	// Top-level fork bytes preserved.
	if len(out.Forks) != len(root.Forks) {
		t.Errorf("fork count = %d, want %d", len(out.Forks), len(root.Forks))
	}
	for k, fork := range root.Forks {
		got, ok := out.Forks[k]
		if !ok {
			t.Errorf("fork %d missing in unmarshal", k)
			continue
		}
		if !bytes.Equal(got.Prefix, fork.Prefix) {
			t.Errorf("fork %d prefix = %x, want %x", k, got.Prefix, fork.Prefix)
		}
		// Children's selfAddress is what's persisted on the wire; make sure
		// it survives the round trip.
		if !bytes.Equal(got.Node.SelfAddress, fork.Node.SelfAddress) {
			t.Errorf("fork %d selfAddress mismatch", k)
		}
	}
}

func TestMarshal_MetadataSurvivesRoundTrip(t *testing.T) {
	root := New()
	r1 := mustRef(t, strings.Repeat("11", 32))
	root.AddFork([]byte("/x"), r1, map[string]string{"Content-Type": "text/plain", "extra": "y"})

	addr, err := root.CalculateSelfAddress()
	if err != nil {
		t.Fatalf("CalculateSelfAddress: %v", err)
	}
	root.SelfAddress = addr
	data, err := root.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	out, err := Unmarshal(data, addr)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	fork, ok := out.Forks['/']
	if !ok {
		t.Fatal("expected fork '/'")
	}
	if fork.Node.Metadata["Content-Type"] != "text/plain" {
		t.Errorf("metadata not preserved: %v", fork.Node.Metadata)
	}
	if fork.Node.Metadata["extra"] != "y" {
		t.Errorf("metadata extra not preserved: %v", fork.Node.Metadata)
	}
}

func TestMarshal_ObfuscationKeyXOR(t *testing.T) {
	// A non-zero obfuscation key means the body must NOT contain the version
	// hash in cleartext. After Unmarshal, the version check still passes.
	root := New()
	for i := range root.ObfuscationKey {
		root.ObfuscationKey[i] = 0x5a
	}
	r := mustRef(t, strings.Repeat("44", 32))
	root.AddFork([]byte("/f"), r, nil)
	addr, err := root.CalculateSelfAddress()
	if err != nil {
		t.Fatal(err)
	}
	root.SelfAddress = addr
	data, err := root.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data[32:], version02Hash[:31]) {
		t.Errorf("encrypted body should not contain plaintext version hash")
	}
	out, err := Unmarshal(data, addr)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !bytes.Equal(out.ObfuscationKey, root.ObfuscationKey) {
		t.Errorf("obfuscation key not preserved")
	}
}

func TestDetermineType(t *testing.T) {
	cases := []struct {
		name string
		mut  func(n *MantarayNode)
		want byte
	}{
		{
			name: "default empty",
			mut:  func(n *MantarayNode) {},
			want: 0,
		},
		{
			name: "with target",
			mut: func(n *MantarayNode) {
				n.TargetAddress = bytes.Repeat([]byte{1}, 32)
			},
			want: TypeValue,
		},
		{
			name: "root slash path",
			mut: func(n *MantarayNode) {
				n.Path = []byte{'/'}
			},
			want: TypeValue,
		},
		{
			name: "with metadata",
			mut: func(n *MantarayNode) {
				n.Metadata = map[string]string{"a": "b"}
			},
			want: TypeWithMetadata,
		},
		{
			name: "with edge",
			mut: func(n *MantarayNode) {
				n.Forks['x'] = &Fork{}
			},
			want: TypeEdge,
		},
		{
			name: "with path separator",
			mut: func(n *MantarayNode) {
				n.Path = []byte("/foo")
			},
			want: TypeWithPathSeparator,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			n := New()
			tt.mut(n)
			if got := n.DetermineType(); got != tt.want {
				t.Errorf("DetermineType = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRemoveFork(t *testing.T) {
	root := New()
	r1 := mustRef(t, strings.Repeat("11", 32))
	r2 := mustRef(t, strings.Repeat("22", 32))
	root.AddFork([]byte("/a"), r1, nil)
	root.AddFork([]byte("/b"), r2, nil)

	if err := root.RemoveFork([]byte("/a")); err != nil {
		t.Fatalf("RemoveFork: %v", err)
	}
	if root.Find([]byte("/a")) != nil {
		t.Error("Find /a after remove: expected nil")
	}
	if root.Find([]byte("/b")) == nil {
		t.Error("Find /b after remove /a: expected node")
	}
}

func TestCollectAndMap(t *testing.T) {
	root := New()
	r1 := mustRef(t, strings.Repeat("11", 32))
	r2 := mustRef(t, strings.Repeat("22", 32))
	root.AddFork([]byte("/index.html"), r1, nil)
	root.AddFork([]byte("/css/style.css"), r2, nil)

	got := root.CollectAndMap()
	if got["/index.html"] != r1.Hex() {
		t.Errorf("index.html: got %q, want %q", got["/index.html"], r1.Hex())
	}
	if got["/css/style.css"] != r2.Hex() {
		t.Errorf("css/style.css: got %q, want %q", got["/css/style.css"], r2.Hex())
	}
}

func TestFindClosest_PartialMatch(t *testing.T) {
	root := New()
	r := mustRef(t, strings.Repeat("11", 32))
	root.AddFork([]byte("/foo/bar"), r, nil)

	// FindClosest of a prefix-only path returns the closest matching node and
	// the bytes that did match — this is what AddFork relies on internally.
	node, matched := root.FindClosest([]byte("/foo/zzz"))
	if node == nil {
		t.Fatal("FindClosest: nil node")
	}
	if string(matched) != "/foo/" && string(matched) != "/foo" {
		// Either is acceptable depending on how the trie chunked the prefix;
		// what matters is that we got *some* prefix that is contained in the
		// requested path.
		//nolint:gocritic // testing that `matched` (the trie's returned prefix) is a prefix of the queried path; the asymmetry is intentional.
		if !bytes.HasPrefix([]byte("/foo/zzz"), matched) {
			t.Errorf("matched %q is not a prefix of /foo/zzz", matched)
		}
	}
}

func TestNullAddress_Helpers(t *testing.T) {
	if !IsNullAddress(nil) {
		t.Error("IsNullAddress(nil) = false")
	}
	if !IsNullAddress(make([]byte, 32)) {
		t.Error("IsNullAddress(zeros) = false")
	}
	if IsNullAddress(bytes.Repeat([]byte{1}, 32)) {
		t.Error("IsNullAddress(non-zero) = true")
	}
}

func TestCommonPrefix(t *testing.T) {
	cases := []struct {
		a, b, want string
	}{
		{"", "", ""},
		{"abc", "abd", "ab"},
		{"abc", "abc", "abc"},
		{"abc", "xyz", ""},
		{"abc", "abcdef", "abc"},
	}
	for _, tt := range cases {
		got := commonPrefix([]byte(tt.a), []byte(tt.b))
		if string(got) != tt.want {
			t.Errorf("commonPrefix(%q,%q) = %q, want %q", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPadEndToMultiple(t *testing.T) {
	got := padEndToMultiple([]byte("abc"), 8, 0x0a)
	if len(got) != 8 {
		t.Errorf("len = %d, want 8", len(got))
	}
	if !bytes.Equal(got, []byte{'a', 'b', 'c', 0x0a, 0x0a, 0x0a, 0x0a, 0x0a}) {
		t.Errorf("padding wrong: %x", got)
	}
	already := padEndToMultiple([]byte("abcd"), 4, 0x00)
	if len(already) != 4 {
		t.Errorf("aligned input should not grow: %d", len(already))
	}
}

func TestBitmap(t *testing.T) {
	buf := make([]byte, 32)
	for _, idx := range []int{0, 7, 8, 100, 255} {
		setBitLE(buf, idx)
	}
	for i := 0; i < 256; i++ {
		want := false
		switch i {
		case 0, 7, 8, 100, 255:
			want = true
		}
		if got := getBitLE(buf, i); got != want {
			t.Errorf("bit %d = %v, want %v", i, got, want)
		}
	}
}
