package file_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func TestService_Chunk(t *testing.T) {
	const refHex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	batch := swarm.MustBatchID(strings.Repeat("bb", 32))
	tests := []struct {
		name    string
		batchID swarm.BatchID
		data    []byte
		handler http.HandlerFunc
		wantRef string
		wantErr bool
	}{
		{
			name:    "upload ok",
			batchID: batch,
			data:    []byte("chunk data"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
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

			got, err := c.UploadChunk(context.Background(), tt.batchID, tt.data, nil)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UploadChunk error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got.Reference.Hex() != tt.wantRef {
				t.Errorf("UploadChunk ref = %v, want %v", got.Reference.Hex(), tt.wantRef)
			}

			// Test Download
			if !tt.wantErr {
				s.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodGet {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
					if !strings.HasSuffix(r.URL.Path, tt.wantRef) {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					w.Write(tt.data)
				})

				data, err := c.DownloadChunk(context.Background(), got.Reference, nil)
				if err != nil {
					t.Fatalf("DownloadChunk error = %v", err)
				}
				if !bytes.Equal(data, tt.data) {
					t.Errorf("DownloadChunk data = %v, want %v", data, tt.data)
				}
			}
		})
	}
}
