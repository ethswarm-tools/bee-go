package gsoc_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/gsoc"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func TestSOCAddress_Deterministic(t *testing.T) {
	owner, _ := swarm.EthAddressFromHex(strings.Repeat("ab", 20))
	id, _ := swarm.IdentifierFromHex(strings.Repeat("cd", 32))
	a, err := gsoc.SOCAddress(id, owner)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := gsoc.SOCAddress(id, owner)
	if a.Hex() != b.Hex() {
		t.Errorf("not deterministic: %s vs %s", a.Hex(), b.Hex())
	}
}

func TestGsocSend_HitsSOCEndpoint(t *testing.T) {
	const ref = "abababababababababababababababababababababababababababababababab"
	gotPath := ""
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if strings.HasPrefix(r.URL.Path, "/soc/") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference": "` + ref + `"}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	fs := file.NewService(u, http.DefaultClient)
	gs := gsoc.NewService(u, http.DefaultClient, websocket.DefaultDialer, fs)

	signer, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	id, _ := swarm.IdentifierFromHex(strings.Repeat("aa", 32))
	batch := swarm.MustBatchID(strings.Repeat("ee", 32))

	res, err := gs.Send(context.Background(), batch, signer, id, []byte("hi"), nil)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.HasPrefix(gotPath, "/soc/") {
		t.Errorf("expected /soc/ path, got %s", gotPath)
	}
	if res.Reference.Hex() != ref {
		t.Errorf("ref = %s", res.Reference.Hex())
	}
}

func TestGsocSubscribe_StreamsMessages(t *testing.T) {
	upgrader := websocket.Upgrader{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/gsoc/subscribe/") {
			w.WriteHeader(404)
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = c.WriteMessage(websocket.BinaryMessage, []byte("hello"))
		time.Sleep(50 * time.Millisecond)
		_ = c.Close()
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	fs := file.NewService(u, http.DefaultClient)
	gs := gsoc.NewService(u, http.DefaultClient, websocket.DefaultDialer, fs)

	owner, _ := swarm.EthAddressFromHex(strings.Repeat("ab", 20))
	id, _ := swarm.IdentifierFromHex(strings.Repeat("cd", 32))

	sub, err := gs.Subscribe(context.Background(), owner, id)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Cancel()

	select {
	case m, ok := <-sub.Messages:
		if !ok {
			t.Fatal("channel closed without message")
		}
		if string(m) != "hello" {
			t.Errorf("msg = %q", m)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
