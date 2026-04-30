package pss_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ethswarm-tools/bee-go/pkg/pss"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// pssWSServer accepts a /pss/subscribe/{topic} WS upgrade and pushes the
// supplied messages, then closes.
func pssWSServer(t *testing.T, msgs [][]byte, holdOpen time.Duration) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/pss/subscribe/") {
			w.WriteHeader(404)
			return
		}
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		for _, m := range msgs {
			if err := c.WriteMessage(websocket.BinaryMessage, m); err != nil {
				return
			}
		}
		time.Sleep(holdOpen)
		_ = c.Close()
	}))
}

func TestPssSubscribe_StreamsMessages(t *testing.T) {
	s := pssWSServer(t, [][]byte{[]byte("a"), []byte("b")}, 50*time.Millisecond)
	defer s.Close()

	u, _ := url.Parse(s.URL)
	svc := pss.NewService(u, http.DefaultClient, websocket.DefaultDialer)
	topic := swarm.TopicFromString("t")

	sub, err := svc.PssSubscribe(context.Background(), topic)
	if err != nil {
		t.Fatalf("PssSubscribe: %v", err)
	}
	defer sub.Cancel()

	got := []string{}
	timeout := time.After(time.Second)
	for len(got) < 2 {
		select {
		case m, ok := <-sub.Messages:
			if !ok {
				t.Fatalf("channel closed early; got=%v", got)
			}
			got = append(got, string(m))
		case <-timeout:
			t.Fatalf("timeout; got=%v", got)
		}
	}
	if got[0] != "a" || got[1] != "b" {
		t.Errorf("messages = %v", got)
	}
}

func TestPssReceive_OneShot(t *testing.T) {
	s := pssWSServer(t, [][]byte{[]byte("hello")}, 50*time.Millisecond)
	defer s.Close()

	u, _ := url.Parse(s.URL)
	svc := pss.NewService(u, http.DefaultClient, websocket.DefaultDialer)
	topic := swarm.TopicFromString("t")

	msg, err := svc.PssReceive(context.Background(), topic, time.Second)
	if err != nil {
		t.Fatalf("PssReceive: %v", err)
	}
	if string(msg) != "hello" {
		t.Errorf("msg = %q", msg)
	}
}

func TestPssReceive_Timeout(t *testing.T) {
	// Server upgrades but never sends — should time out.
	s := pssWSServer(t, nil, time.Second)
	defer s.Close()

	u, _ := url.Parse(s.URL)
	svc := pss.NewService(u, http.DefaultClient, websocket.DefaultDialer)
	topic := swarm.TopicFromString("t")

	_, err := svc.PssReceive(context.Background(), topic, 50*time.Millisecond)
	if err == nil {
		t.Errorf("expected timeout error")
	}
}
