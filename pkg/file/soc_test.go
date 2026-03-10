package file_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/file"
)

func TestService_SOC(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/soc/") && r.Method == http.MethodPost {
			if r.URL.Query().Get("sig") == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference": "soc_ref"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)

	ref, err := c.UploadSOC(context.Background(), "batch1", "owner", "id", "sig", []byte("data"), nil)
	if err != nil {
		t.Fatalf("UploadSOC error = %v", err)
	}
	if ref.Value != "soc_ref" {
		t.Errorf("UploadSOC ref = %v, want soc_ref", ref.Value)
	}
}
