package file_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func TestService_UploadData(t *testing.T) {
	const (
		refHex   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		batchHex = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)
	batch := swarm.MustBatchID(batchHex)
	tests := []struct {
		name    string
		batchID swarm.BatchID
		data    []byte
		handler http.HandlerFunc
		wantRef string
		wantErr bool
	}{
		{
			name:    "ok",
			batchID: batch,
			data:    []byte("hello"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Swarm-Postage-Batch-Id") != batchHex {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"reference": "` + refHex + `"}`))
			},
			wantRef: refHex,
		},
		{
			name:    "server error",
			batchID: batch,
			data:    []byte("hello"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := file.NewService(u, http.DefaultClient)
			got, err := c.UploadData(context.Background(), tt.batchID, bytes.NewReader(tt.data), nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.UploadData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Reference.Hex() != tt.wantRef {
				t.Errorf("Service.UploadData() = %v, want %v", got.Reference.Hex(), tt.wantRef)
			}
		})
	}
}

func TestService_DownloadData(t *testing.T) {
	const refHex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tests := []struct {
		name    string
		ref     swarm.Reference
		handler http.HandlerFunc
		want    []byte
		wantErr bool
	}{
		{
			name: "ok",
			ref:  swarm.MustReference(refHex),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/bytes/"+refHex {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Write([]byte("hello"))
			},
			want: []byte("hello"),
		},
		{
			name: "not found",
			ref:  swarm.MustReference(refHex),
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := file.NewService(u, http.DefaultClient)
			reader, err := c.DownloadData(context.Background(), tt.ref, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.DownloadData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				defer reader.Close()
				got, _ := io.ReadAll(reader)
				if !bytes.Equal(got, tt.want) {
					t.Errorf("Service.DownloadData() = %s, want %s", got, tt.want)
				}
			}
		})
	}
}
