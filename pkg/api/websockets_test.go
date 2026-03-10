package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func TestWebSockets(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/pss/subscribe/") || strings.HasPrefix(r.URL.Path, "/gsoc/subscribe/") {
			c, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Logf("Upgrade error: %v", err)
				return
			}
			defer c.Close()
			c.WriteMessage(websocket.TextMessage, []byte("hello"))
			return
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	dialer := websocket.DefaultDialer

	// PSS
	sub, err := api.PSSSubscribe(context.Background(), u, dialer, "topic")
	if err != nil {
		t.Fatalf("PSSSubscribe error = %v", err)
	}
	defer sub.Close()

	_, msg, err := sub.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage error = %v", err)
	}
	if string(msg) != "hello" {
		t.Errorf("ReadMessage = %s, want hello", msg)
	}

	// GSOC
	subG, err := api.GSOCSubscribe(context.Background(), u, dialer, "addr")
	if err != nil {
		t.Fatalf("GSOCSubscribe error = %v", err)
	}
	defer subG.Close()

	_, msgG, err := subG.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage error = %v", err)
	}
	if string(msgG) != "hello" {
		t.Errorf("ReadMessage = %s, want hello", msgG)
	}
}
