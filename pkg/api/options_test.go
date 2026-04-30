package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func newReq(t *testing.T) *http.Request {
	t.Helper()
	r, err := http.NewRequest("POST", "http://example/x", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	return r
}

func TestPrepareUploadHeaders_AllFields(t *testing.T) {
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	hist := swarm.MustReference(strings.Repeat("bb", 32))
	opts := &UploadOptions{
		Pin:               BoolPtr(true),
		Encrypt:           BoolPtr(true),
		Tag:               42,
		Deferred:          BoolPtr(false),
		Act:               BoolPtr(true),
		ActHistoryAddress: &hist,
	}
	req := newReq(t)
	PrepareUploadHeaders(req, batch, opts)

	cases := map[string]string{
		"Swarm-Postage-Batch-Id":    batch.Hex(),
		"Swarm-Pin":                 "true",
		"Swarm-Encrypt":             "true",
		"Swarm-Tag":                 "42",
		"Swarm-Deferred-Upload":     "false",
		"Swarm-Act":                 "true",
		"Swarm-Act-History-Address": hist.Hex(),
	}
	for k, want := range cases {
		if got := req.Header.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestPrepareUploadHeaders_NilOpts(t *testing.T) {
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	req := newReq(t)
	PrepareUploadHeaders(req, batch, nil)
	if got := req.Header.Get("Swarm-Postage-Batch-Id"); got != batch.Hex() {
		t.Errorf("batch header missing: %q", got)
	}
	// No other headers should be set when opts is nil.
	if got := req.Header.Get("Swarm-Pin"); got != "" {
		t.Errorf("Swarm-Pin should be unset, got %q", got)
	}
}

func TestPrepareUploadHeaders_BoolPointerSemantics(t *testing.T) {
	// nil pointer -> header omitted; explicit false -> "false" header.
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	cases := []struct {
		name string
		pin  *bool
		want string
	}{
		{"nil", nil, ""},
		{"true", BoolPtr(true), "true"},
		{"false", BoolPtr(false), "false"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			req := newReq(t)
			PrepareUploadHeaders(req, batch, &UploadOptions{Pin: tt.pin})
			if got := req.Header.Get("Swarm-Pin"); got != tt.want {
				t.Errorf("Swarm-Pin = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrepareCollectionUploadHeaders(t *testing.T) {
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	opts := &CollectionUploadOptions{
		IndexDocument:   "index.html",
		ErrorDocument:   "404.html",
		RedundancyLevel: RedundancyLevelMedium,
	}
	req := newReq(t)
	PrepareCollectionUploadHeaders(req, batch, opts)

	if got := req.Header.Get("Swarm-Index-Document"); got != "index.html" {
		t.Errorf("Swarm-Index-Document = %q", got)
	}
	if got := req.Header.Get("Swarm-Error-Document"); got != "404.html" {
		t.Errorf("Swarm-Error-Document = %q", got)
	}
	if got := req.Header.Get("Swarm-Redundancy-Level"); got != "1" {
		t.Errorf("Swarm-Redundancy-Level = %q", got)
	}
}

func TestPrepareDownloadHeaders(t *testing.T) {
	hist := swarm.MustReference(strings.Repeat("11", 32))
	priv, _ := swarm.PrivateKeyFromHex(strings.Repeat("22", 32))
	pub := priv.PublicKey()
	wantPub, _ := pub.CompressedHex()

	opts := &DownloadOptions{
		RedundancyStrategy: RedundancyStrategyPtr(RedundancyStrategyData),
		Fallback:           BoolPtr(false),
		TimeoutMs:          1500,
		ActPublisher:       &pub,
		ActHistoryAddress:  &hist,
		ActTimestamp:       100,
	}
	req := newReq(t)
	PrepareDownloadHeaders(req, opts)

	cases := map[string]string{
		"Swarm-Redundancy-Strategy":      "1",
		"Swarm-Redundancy-Fallback-Mode": "false",
		"Swarm-Chunk-Retrieval-Timeout":  "1500",
		"Swarm-Act-Publisher":            wantPub,
		"Swarm-Act-History-Address":      hist.Hex(),
		"Swarm-Act-Timestamp":            "100",
		"Swarm-Act":                      "true",
	}
	for k, want := range cases {
		if got := req.Header.Get(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}
}

func TestReadUploadResult(t *testing.T) {
	headers := http.Header{}
	headers.Set("Swarm-Tag", "12345")
	hist := strings.Repeat("cd", 32)
	headers.Set("Swarm-Act-History-Address", hist)

	refHex := strings.Repeat("ab", 32)
	res, err := ReadUploadResult(refHex, headers)
	if err != nil {
		t.Fatalf("ReadUploadResult: %v", err)
	}
	if res.Reference.Hex() != refHex {
		t.Errorf("Reference = %q, want %q", res.Reference.Hex(), refHex)
	}
	if res.TagUID != 12345 {
		t.Errorf("TagUID = %d, want 12345", res.TagUID)
	}
	if res.HistoryAddress == nil || res.HistoryAddress.Hex() != hist {
		t.Errorf("HistoryAddress = %v, want %s", res.HistoryAddress, hist)
	}
}

func TestReadUploadResult_NoHeaders(t *testing.T) {
	refHex := strings.Repeat("ab", 32)
	res, err := ReadUploadResult(refHex, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	if res.TagUID != 0 {
		t.Errorf("TagUID should be 0 when header missing")
	}
	if res.HistoryAddress != nil {
		t.Errorf("HistoryAddress should be nil when header missing")
	}
}

func TestParseFileHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("Content-Type", "image/png")
	headers.Set("Content-Disposition", `attachment; filename="hello.png"`)
	headers.Set("Swarm-Tag-Uid", "42")

	got := ParseFileHeaders(headers)
	if got.ContentType != "image/png" {
		t.Errorf("ContentType = %q", got.ContentType)
	}
	if got.Name != "hello.png" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.TagUID != 42 {
		t.Errorf("TagUID = %d", got.TagUID)
	}
}

func TestParseFileHeaders_RFC5987UTF8(t *testing.T) {
	// filename* form: Bee may emit "UTF-8''actual-name" for non-ASCII names.
	headers := http.Header{}
	headers.Set("Content-Disposition", `attachment; filename*=UTF-8''greet%20me.txt`)
	got := ParseFileHeaders(headers)
	if got.Name != "greet%20me.txt" {
		t.Errorf("Name = %q, want greet%%20me.txt", got.Name)
	}
}
