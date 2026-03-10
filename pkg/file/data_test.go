package file_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestService_UploadData(t *testing.T) {
	tests := []struct {
		name    string
		batchID string
		data    []byte
		handler http.HandlerFunc
		wantRef string
		wantErr bool
	}{
		{
			name:    "ok",
			batchID: "batch1",
			data:    []byte("hello"),
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Swarm-Postage-Batch-Id") != "batch1" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"reference": "ref1"}`))
			},
			wantRef: "ref1",
		},
		{
			name:    "server error",
			batchID: "batch1",
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
			if !tt.wantErr && got.Value != tt.wantRef {
				t.Errorf("Service.UploadData() = %v, want %v", got.Value, tt.wantRef)
			}
		})
	}
}

func TestService_DownloadData(t *testing.T) {
	tests := []struct {
		name    string
		ref     swarm.Reference
		handler http.HandlerFunc
		want    []byte
		wantErr bool
	}{
		{
			name: "ok",
			ref:  swarm.Reference{Value: "ref1"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/bytes/ref1" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Write([]byte("hello"))
			},
			want: []byte("hello"),
		},
		{
			name: "not found",
			ref:  swarm.Reference{Value: "ref1"},
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
			reader, err := c.DownloadData(context.Background(), tt.ref)
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
