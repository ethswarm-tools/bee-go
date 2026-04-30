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

func TestService_Envelope(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Envelope
		if strings.HasPrefix(r.URL.Path, "/envelope/") && r.Method == http.MethodPost {
			w.Write([]byte(`{"issuer": "issuer1", "index": "1", "timestamp": "123", "signature": "sig1"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := api.NewService(u, http.DefaultClient)

	// Envelope
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	ref := swarm.MustReference(strings.Repeat("bb", 32))
	env, err := c.PostEnvelope(context.Background(), batch, ref)
	if err != nil {
		t.Fatalf("PostEnvelope error = %v", err)
	}
	if env.Issuer != "issuer1" {
		t.Errorf("PostEnvelope issuer = %v, want issuer1", env.Issuer)
	}
}
