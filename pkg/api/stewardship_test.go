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

func TestService_Stewardship(t *testing.T) {
	const batchHex = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/stewardship/") {
			if r.Method == http.MethodPut {
				if r.Header.Get("swarm-postage-batch-id") != batchHex {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
				return
			}
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"isRetrievable": true}`))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := api.NewService(u, http.DefaultClient)
	ref := swarm.MustReference("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	batch := swarm.MustBatchID(batchHex)

	// Reupload
	if err := c.Reupload(context.Background(), ref, batch); err != nil {
		t.Fatalf("Reupload error = %v", err)
	}

	// IsRetrievable
	retrievable, err := c.IsRetrievable(context.Background(), ref)
	if err != nil {
		t.Fatalf("IsRetrievable error = %v", err)
	}
	if !retrievable {
		t.Errorf("IsRetrievable = false, want true")
	}
}
