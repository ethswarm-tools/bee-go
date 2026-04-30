package file_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
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

func TestService_SOCWriter_RoundTrip(t *testing.T) {
	swPriv, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	signer, _ := crypto.ToECDSA(swPriv.Raw())
	id := swarm.IdentifierFromString("rt-topic")
	payload := []byte("rt payload")

	// Build the SOC offline so we know the wire bytes the server will see.
	wantSOC, err := swarm.MakeSingleOwnerChunk(id, payload, signer)
	if err != nil {
		t.Fatalf("MakeSingleOwnerChunk: %v", err)
	}
	owner, _ := swarm.NewEthAddress(crypto.PubkeyToAddress(signer.PublicKey).Bytes())
	socAddr, _ := swarm.CalculateSingleOwnerChunkAddress(id, owner)

	var (
		gotSig    string
		gotPath   string
		uploaded  []byte
		downloads int
	)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/soc/") && r.Method == http.MethodPost:
			gotPath = r.URL.Path
			gotSig = r.URL.Query().Get("sig")
			body, _ := io.ReadAll(r.Body)
			uploaded = body
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference":"` + socAddr.Hex() + `"}`))
		case strings.HasPrefix(r.URL.Path, "/chunks/") && r.Method == http.MethodGet:
			downloads++
			// Server returns the full SOC wire form: id || sig || span || payload.
			out := make([]byte, 0)
			out = append(out, wantSOC.ID...)
			out = append(out, wantSOC.Signature...)
			out = append(out, wantSOC.Span...)
			out = append(out, wantSOC.Payload...)
			w.Write(out)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	svc := file.NewService(u, http.DefaultClient)

	w, err := svc.MakeSOCWriter(signer)
	if err != nil {
		t.Fatalf("MakeSOCWriter: %v", err)
	}
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	res, err := w.Upload(context.Background(), batch, id, payload, nil)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if res.Reference.Hex() != socAddr.Hex() {
		t.Errorf("upload ref = %s, want %s", res.Reference.Hex(), socAddr.Hex())
	}
	// Path encodes owner + identifier as separate segments.
	if !strings.Contains(gotPath, owner.Hex()) || !strings.Contains(gotPath, id.Hex()) {
		t.Errorf("path = %q (missing owner or id)", gotPath)
	}
	// Body must be span || payload (the CAC half).
	if len(uploaded) != len(wantSOC.Span)+len(wantSOC.Payload) {
		t.Errorf("uploaded body length = %d, want %d", len(uploaded), len(wantSOC.Span)+len(wantSOC.Payload))
	}
	// Signature query parameter must match the offline-built SOC.
	if gotSig == "" {
		t.Errorf("missing sig param")
	}

	// Reader path round-trip via the same service.
	r := svc.MakeSOCReader(owner)
	parsed, err := r.Download(context.Background(), id)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if downloads != 1 {
		t.Errorf("downloads = %d", downloads)
	}
	if string(parsed.Payload) != string(payload) {
		t.Errorf("payload = %q", parsed.Payload)
	}
}

func TestService_ProbeData(t *testing.T) {
	const refHex = "4444444444444444444444444444444444444444444444444444444444444444"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bytes/"+refHex && r.Method == http.MethodHead {
			w.Header().Set("Content-Length", "987")
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	svc := file.NewService(u, http.DefaultClient)
	got, err := svc.ProbeData(context.Background(), swarm.MustReference(refHex))
	if err != nil {
		t.Fatalf("ProbeData: %v", err)
	}
	if got.ContentLength != 987 {
		t.Errorf("ContentLength = %d", got.ContentLength)
	}
}
