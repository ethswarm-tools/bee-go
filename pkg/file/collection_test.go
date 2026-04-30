package file_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func TestUploadCollectionEntries_TarShape(t *testing.T) {
	const refHex = "abababababababababababababababababababababababababababababababab"
	var receivedTar []byte
	var sawIndexHeader string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/bzz" {
			body, _ := io.ReadAll(r.Body)
			receivedTar = body
			sawIndexHeader = r.Header.Get("Swarm-Index-Document")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference": "` + refHex + `"}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))

	entries := []file.CollectionEntry{
		{Path: "index.html", Data: []byte("<h1>hi</h1>")},
		{Path: "data/items.json", Data: []byte(`{"a": 1}`)},
	}
	if total := file.CollectionSize(entries); total != int64(len(entries[0].Data)+len(entries[1].Data)) {
		t.Errorf("CollectionSize = %d", total)
	}

	res, err := c.UploadCollectionEntries(context.Background(), batch, entries,
		&api.CollectionUploadOptions{IndexDocument: "index.html"})
	if err != nil {
		t.Fatalf("UploadCollectionEntries: %v", err)
	}
	if res.Reference.Hex() != refHex {
		t.Errorf("ref = %s", res.Reference.Hex())
	}
	if sawIndexHeader != "index.html" {
		t.Errorf("Swarm-Index-Document = %q", sawIndexHeader)
	}

	// Verify the tar we sent contains both entries with the right contents.
	tr := tar.NewReader(bytes.NewReader(receivedTar))
	got := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		body, _ := io.ReadAll(tr)
		got[hdr.Name] = body
	}
	if string(got["index.html"]) != "<h1>hi</h1>" {
		t.Errorf("index.html = %q", got["index.html"])
	}
	if string(got["data/items.json"]) != `{"a": 1}` {
		t.Errorf("items.json = %q", got["data/items.json"])
	}
}
