package file_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestService_UploadFile(t *testing.T) {
	const (
		refHex   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		batchHex = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	batch := swarm.MustBatchID(batchHex)
	tests := []struct {
		name        string
		batchID     swarm.BatchID
		data        []byte
		fileName    string
		contentType string
		handler     http.HandlerFunc
		wantRef     string
		wantErr     bool
	}{
		{
			name:        "ok with content type",
			batchID:     batch,
			data:        []byte("hello"),
			fileName:    "hello.txt",
			contentType: "text/plain",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Swarm-Postage-Batch-Id") != batchHex {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.Header.Get("Content-Type") != "text/plain" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.URL.Query().Get("name") != "hello.txt" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"reference": "` + refHex + `"}`))
			},
			wantRef: refHex,
		},
		{
			name:        "ok default content type",
			batchID:     batch,
			data:        []byte("hello"),
			fileName:    "",
			contentType: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Content-Type") != "application/octet-stream" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"reference": "` + refHex + `"}`))
			},
			wantRef: refHex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := file.NewService(u, http.DefaultClient)
			got, err := c.UploadFile(context.Background(), tt.batchID, bytes.NewReader(tt.data), tt.fileName, tt.contentType, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.UploadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Reference.Hex() != tt.wantRef {
				t.Errorf("Service.UploadFile() = %v, want %v", got.Reference.Hex(), tt.wantRef)
			}
		})
	}
}

func TestService_DownloadFile(t *testing.T) {
	tests := []struct {
		name    string
		ref     swarm.Reference
		handler http.HandlerFunc
		want    []byte
		wantCT  string
		wantErr bool
	}{
		{
			name: "ok",
			ref:  swarm.MustReference("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/bzz/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("hello"))
			},
			want:   []byte("hello"),
			wantCT: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := file.NewService(u, http.DefaultClient)
			reader, headers, err := c.DownloadFile(context.Background(), tt.ref, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.DownloadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				defer reader.Close()
				got, _ := io.ReadAll(reader)
				if !bytes.Equal(got, tt.want) {
					t.Errorf("Service.DownloadFile() data = %s, want %s", got, tt.want)
				}
				if headers.ContentType != tt.wantCT {
					t.Errorf("Service.DownloadFile() content-type = %s, want %s", headers.ContentType, tt.wantCT)
				}
			}
		})
	}
}

func TestService_UploadCollection(t *testing.T) {
	// Create a temp dir with some files
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte("<html></html>"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Mkdir(filepath.Join(tmpDir, "css"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "css", "style.css"), []byte("body {}"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	const (
		collRefHex   = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
		collBatchHex = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	)
	collBatch := swarm.MustBatchID(collBatchHex)
	tests := []struct {
		name      string
		batchID   swarm.BatchID
		dir       string
		indexFile string
		handler   http.HandlerFunc
		wantRef   string
		wantErr   bool
	}{
		{
			name:      "ok",
			batchID:   collBatch,
			dir:       tmpDir,
			indexFile: "index.html",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Swarm-Postage-Batch-Id") != collBatchHex {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.Header.Get("Content-Type") != "application/x-tar" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.Header.Get("Swarm-Collection") != "true" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if r.Header.Get("Swarm-Index-Document") != "index.html" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				// Check if body is a valid tar
				tr := tar.NewReader(r.Body)
				filesFound := 0
				for {
					header, err := tr.Next()
					if err == io.EOF {
						break
					}
					if err != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					// Check for relative paths
					if header.Name == "index.html" || header.Name == "css/style.css" {
						filesFound++
					}
				}

				// We expect at least these two files. Directory entries might or might not be present.
				if filesFound < 2 {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"reference": "` + collRefHex + `"}`))
			},
			wantRef: collRefHex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := file.NewService(u, http.DefaultClient)
			opts := &api.CollectionUploadOptions{IndexDocument: tt.indexFile}
			got, err := c.UploadCollection(context.Background(), tt.batchID, tt.dir, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.UploadCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Reference.Hex() != tt.wantRef {
				t.Errorf("Service.UploadCollection() = %v, want %v", got.Reference.Hex(), tt.wantRef)
			}
		})
	}
}
