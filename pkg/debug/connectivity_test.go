package debug_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/debug"
)

func TestService_Connectivity(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/blocklist" {
			w.Write([]byte(`{"peers": [{"address": "addr1", "fullNode": true}]}`))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/peers/") && r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/pingpong/") && r.Method == http.MethodPost {
			w.Write([]byte(`{"rtt": "50ms"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := debug.NewService(u, http.DefaultClient)

	// GetBlocklist
	peers, err := c.GetBlocklist(context.Background())
	if err != nil {
		t.Fatalf("GetBlocklist error = %v", err)
	}
	if len(peers) != 1 || peers[0].Address != "addr1" {
		t.Errorf("GetBlocklist = %v, want [{addr1 true}]", peers)
	}

	// RemovePeer
	if err := c.RemovePeer(context.Background(), "addr1"); err != nil {
		t.Fatalf("RemovePeer error = %v", err)
	}

	// PingPeer
	rtt, err := c.PingPeer(context.Background(), "addr1")
	if err != nil {
		t.Fatalf("PingPeer error = %v", err)
	}
	if *rtt != "50ms" {
		t.Errorf("PingPeer RTT = %v, want 50ms", *rtt)
	}
}
