package file_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethersphere/bee-go/pkg/file"
)

func TestService_Feed(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/feeds/") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"reference": "feed_ref"}`))
				return
			}
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"reference": "feed_update_ref"}`))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)

	// Create Manifest
	ref, err := c.CreateFeedManifest(context.Background(), "batch1", "owner", "topic")
	if err != nil {
		t.Fatalf("CreateFeedManifest error = %v", err)
	}
	if ref.Value != "feed_ref" {
		t.Errorf("CreateFeedManifest ref = %v, want feed_ref", ref.Value)
	}

	// Get Lookup
	ref2, err := c.GetFeedLookup(context.Background(), "owner", "topic")
	if err != nil {
		t.Fatalf("GetFeedLookup error = %v", err)
	}
	if ref2.Value != "feed_update_ref" {
		t.Errorf("GetFeedLookup ref = %v, want feed_update_ref", ref2.Value)
	}
}

func TestService_Feed_Update(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/soc/") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference": "ref123"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)

	privKey, _ := crypto.GenerateKey()
	// Using hex encoded topic (32 bytes zeroed for simplicity or actual hash)
	topic := "0000000000000000000000000000000000000000000000000000000000000000"

	ref, err := c.UpdateFeedWithIndex(context.Background(), "batch1", privKey, topic, 0, []byte("update"))
	if err != nil {
		t.Fatalf("UpdateFeedWithIndex error = %v", err)
	}
	if ref.Value != "ref123" {
		t.Errorf("UpdateFeedWithIndex ref = %s, want ref123", ref.Value)
	}
}
