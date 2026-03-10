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

	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestService_UploadFile(t *testing.T) {
	tests := []struct {
		name        string
		batchID     string
		data        []byte
		fileName    string
		contentType string
		handler     http.HandlerFunc
		wantRef     string
		wantErr     bool
	}{
		{
			name:        "ok with content type",
			batchID:     "batch1",
			data:        []byte("hello"),
			fileName:    "hello.txt",
			contentType: "text/plain",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Swarm-Postage-Batch-Id") != "batch1" {
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
				w.Write([]byte(`{"reference": "ref1"}`))
			},
			wantRef: "ref1",
		},
		{
			name:        "ok default content type",
			batchID:     "batch1",
			data:        []byte("hello"),
			fileName:    "",
			contentType: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Content-Type") != "application/octet-stream" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"reference": "ref1"}`))
			},
			wantRef: "ref1",
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
			if !tt.wantErr && got.Value != tt.wantRef {
				t.Errorf("Service.UploadFile() = %v, want %v", got.Value, tt.wantRef)
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
			ref:  swarm.Reference{Value: "ref1"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/bzz/ref1" {
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
			reader, ct, err := c.DownloadFile(context.Background(), tt.ref)
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
				if ct != tt.wantCT {
					t.Errorf("Service.DownloadFile() content-type = %s, want %s", ct, tt.wantCT)
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

	tests := []struct {
		name      string
		batchID   string
		dir       string
		indexFile string
		handler   http.HandlerFunc
		wantRef   string
		wantErr   bool
	}{
		{
			name:      "ok",
			batchID:   "batch1",
			dir:       tmpDir,
			indexFile: "index.html",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Swarm-Postage-Batch-Id") != "batch1" {
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
				if r.URL.Query().Get("index") != "index.html" {
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
				w.Write([]byte(`{"reference": "ref1"}`))
			},
			wantRef: "ref1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := file.NewService(u, http.DefaultClient)
			got, err := c.UploadCollection(context.Background(), tt.batchID, tt.dir, tt.indexFile, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.UploadCollection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Value != tt.wantRef {
				t.Errorf("Service.UploadCollection() = %v, want %v", got.Value, tt.wantRef)
			}
		})
	}
}
