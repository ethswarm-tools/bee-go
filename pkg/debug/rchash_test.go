package debug_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/debug"
)

func TestService_RCHash(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// RCHash
		if strings.HasPrefix(r.URL.Path, "/rchash/") {
			w.Write([]byte(`{"durationSeconds": 1.5}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	// RCHash
	rch, err := c.RCHash(context.Background(), 10, "anchor1", "anchor2")
	if err != nil {
		t.Fatalf("RCHash error = %v", err)
	}
	if rch != 1.5 {
		t.Errorf("RCHash = %v, want 1.5", rch)
	}
}
