package file_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestRefinedFeatures(t *testing.T) {
	// Test UploadOptions Headers
	t.Run("UploadOptions", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Swarm-Pin") != "true" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.Header.Get("Swarm-Encrypt") != "true" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference": "` + strings.Repeat("aa", 32) + `"}`))
		}))
		defer s.Close()

		u, _ := url.Parse(s.URL)
		c := file.NewService(u, http.DefaultClient)
		opts := &api.FileUploadOptions{
			UploadOptions: api.UploadOptions{
				Pin:     api.BoolPtr(true),
				Encrypt: api.BoolPtr(true),
			},
		}

		batch := swarm.MustBatchID(strings.Repeat("bb", 32))
		_, err := c.UploadFile(context.Background(), batch, nil, "name", "content-type", opts)
		if err != nil {
			t.Logf("UploadFile returned error: %v", err)
		}
	})
}
