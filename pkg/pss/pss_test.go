package pss_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ethersphere/bee-go/pkg/pss"
)

func TestService_PssSend(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pss/send/topic1/target1" && r.URL.Query().Get("recipient") == "rec1" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	// NewService takes (baseURL, httpClient, dialer)
	c := pss.NewService(u, http.DefaultClient, nil)

	if err := c.PssSend(context.Background(), "topic1", "target1", nil, "rec1"); err != nil {
		t.Fatalf("PssSend error = %v", err)
	}
}
