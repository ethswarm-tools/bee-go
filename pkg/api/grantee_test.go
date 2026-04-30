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

func TestService_Grantee(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Grantee
		if strings.HasPrefix(r.URL.Path, "/grantee/") && r.Method == http.MethodGet {
			w.Write([]byte(`{"grantees": ["key1", "key2"]}`))
			return
		}
		if r.URL.Path == "/grantee" && r.Method == http.MethodPost {
			w.Write([]byte(`{"ref": "ref1", "historyref": "hist1"}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/grantee/") && r.Method == http.MethodPatch {
			w.Write([]byte(`{"ref": "ref2", "historyref": "hist2"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := api.NewService(u, http.DefaultClient)

	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	ref := swarm.MustReference(strings.Repeat("bb", 32))
	hist := swarm.MustReference(strings.Repeat("cc", 32))

	// Grantee
	grantees, err := c.GetGrantees(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetGrantees error = %v", err)
	}
	if len(grantees) != 2 || grantees[0] != "key1" {
		t.Errorf("GetGrantees = %v, want [key1 key2]", grantees)
	}

	res, err := c.CreateGrantees(context.Background(), batch, []string{"key1"})
	if err != nil {
		t.Fatalf("CreateGrantees error = %v", err)
	}
	if res.Ref != "ref1" {
		t.Errorf("CreateGrantees ref = %v, want ref1", res.Ref)
	}

	res, err = c.PatchGrantees(context.Background(), batch, ref, hist, []string{"key2"}, nil)
	if err != nil {
		t.Fatalf("PatchGrantees error = %v", err)
	}
	if res.Ref != "ref2" {
		t.Errorf("PatchGrantees ref = %v, want ref2", res.Ref)
	}
}
