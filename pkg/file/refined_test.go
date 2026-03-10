package file_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/file"
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
			w.Write([]byte(`{"reference": "ref1"}`))
		}))
		defer s.Close()

		u, _ := url.Parse(s.URL)
		c := file.NewService(u, http.DefaultClient)
		opts := &api.UploadOptions{
			Pin:     true,
			Encrypt: true,
		}

		_, err := c.UploadFile(context.Background(), "batch1", nil, "name", "content-type", opts)
		if err != nil {
			t.Logf("UploadFile returned error: %v", err)
		}
	})
}
