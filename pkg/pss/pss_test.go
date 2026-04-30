package pss_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/pss"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestService_PssSend(t *testing.T) {
	topic := swarm.TopicFromString("topic1")
	signer, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	recipient := signer.PublicKey()
	wantRecipient, _ := recipient.CompressedHex()
	batchID, _ := swarm.BatchIDFromHex(strings.Repeat("a1", 32))

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pss/send/"+topic.Hex()+"/target1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("recipient") != wantRecipient {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("Swarm-Postage-Batch-Id") != batchID.Hex() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := pss.NewService(u, http.DefaultClient, nil)

	if err := c.PssSend(context.Background(), batchID, topic, "target1", nil, recipient); err != nil {
		t.Fatalf("PssSend error = %v", err)
	}
}
