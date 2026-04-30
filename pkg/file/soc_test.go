package file_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

func TestService_SOC(t *testing.T) {
	const refHex = "4444444444444444444444444444444444444444444444444444444444444444"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/soc/") && r.Method == http.MethodPost {
			if r.URL.Query().Get("sig") == "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference": "` + refHex + `"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)

	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	owner, _ := swarm.EthAddressFromHex(strings.Repeat("bb", 20))
	id := swarm.IdentifierFromString("test-id")
	sig, _ := swarm.SignatureFromHex(strings.Repeat("cc", 65))

	ref, err := c.UploadSOC(context.Background(), batch, owner, id, sig, []byte("data"), nil)
	if err != nil {
		t.Fatalf("UploadSOC error = %v", err)
	}
	if ref.Reference.Hex() != refHex {
		t.Errorf("UploadSOC ref = %v, want %s", ref.Reference.Hex(), refHex)
	}
}
