package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestService_Pin(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/ref1") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := api.NewService(u, http.DefaultClient)
	ref := swarm.Reference{Value: "ref1"}

	// Test Pin
	if err := c.Pin(context.Background(), ref); err != nil {
		t.Errorf("Pin error = %v", err)
	}

	// Test GetPin
	exists, err := c.GetPin(context.Background(), ref)
	if err != nil {
		t.Errorf("GetPin error = %v", err)
	}
	if !exists {
		t.Errorf("GetPin exists = false, want true")
	}

	// Test Unpin
	if err := c.Unpin(context.Background(), ref); err != nil {
		t.Errorf("Unpin error = %v", err)
	}
}
