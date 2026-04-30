package file_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethswarm-tools/bee-go/pkg/file"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

func TestService_Feed(t *testing.T) {
	const (
		manifestRef = "1111111111111111111111111111111111111111111111111111111111111111"
		updateRef   = "2222222222222222222222222222222222222222222222222222222222222222"
	)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/feeds/") {
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"reference": "` + manifestRef + `"}`))
				return
			}
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"reference": "` + updateRef + `"}`))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)

	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	owner, _ := swarm.EthAddressFromHex(strings.Repeat("bb", 20))
	topic := swarm.TopicFromString("test-topic")

	// Create Manifest
	ref, err := c.CreateFeedManifest(context.Background(), batch, owner, topic)
	if err != nil {
		t.Fatalf("CreateFeedManifest error = %v", err)
	}
	if ref.Hex() != manifestRef {
		t.Errorf("CreateFeedManifest ref = %v, want %s", ref.Hex(), manifestRef)
	}

	// Get Lookup
	ref2, err := c.GetFeedLookup(context.Background(), owner, topic)
	if err != nil {
		t.Fatalf("GetFeedLookup error = %v", err)
	}
	if ref2.Hex() != updateRef {
		t.Errorf("GetFeedLookup ref = %v, want %s", ref2.Hex(), updateRef)
	}
}

func TestFeedUpdateChunkReference_Deterministic(t *testing.T) {
	owner, _ := swarm.EthAddressFromHex(strings.Repeat("bb", 20))
	topic := swarm.TopicFromString("test-topic")
	a, err := file.FeedUpdateChunkReference(owner, topic, 0)
	if err != nil {
		t.Fatal(err)
	}
	b, err := file.FeedUpdateChunkReference(owner, topic, 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Hex() != b.Hex() {
		t.Errorf("not deterministic: %s vs %s", a.Hex(), b.Hex())
	}
	c, err := file.FeedUpdateChunkReference(owner, topic, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a.Hex() == c.Hex() {
		t.Errorf("index 0 and 1 collided")
	}
}

func TestIsFeedRetrievable_Latest(t *testing.T) {
	cases := []struct {
		status  int
		want    bool
		wantErr bool
	}{
		{200, true, false},
		{404, false, false},
		{500, false, false},
		{401, false, true},
	}
	for _, tc := range cases {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tc.status == 200 {
				// Bee returns the wrapped chunk payload + feed-index headers.
				w.Header().Set("swarm-feed-index", "0000000000000000")
				w.Header().Set("swarm-feed-index-next", "0000000000000001")
				w.WriteHeader(200)
				w.Write([]byte("payload-bytes"))
				return
			}
			w.WriteHeader(tc.status)
		}))
		u, _ := url.Parse(s.URL)
		c := file.NewService(u, http.DefaultClient)
		owner, _ := swarm.EthAddressFromHex(strings.Repeat("bb", 20))
		topic := swarm.TopicFromString("t")
		got, err := c.IsFeedRetrievable(context.Background(), owner, topic, nil, nil)
		if (err != nil) != tc.wantErr {
			t.Errorf("status=%d err=%v wantErr=%v", tc.status, err, tc.wantErr)
		}
		if got != tc.want {
			t.Errorf("status=%d got=%v want=%v", tc.status, got, tc.want)
		}
		s.Close()
	}
}

func TestAreAllSequentialFeedsUpdateRetrievable(t *testing.T) {
	// Server serves chunks 0..2 OK, chunk 3+ returns 404.
	served := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/chunks/") {
			served++
			if served <= 3 {
				w.WriteHeader(200)
				w.Write([]byte("ok"))
				return
			}
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()
	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)
	owner, _ := swarm.EthAddressFromHex(strings.Repeat("bb", 20))
	topic := swarm.TopicFromString("t")

	// 0..2 inclusive should pass.
	got, err := c.AreAllSequentialFeedsUpdateRetrievable(context.Background(), owner, topic, 2, nil)
	if err != nil || !got {
		t.Errorf("want all retrievable, got=%v err=%v", got, err)
	}
	// 0..3 should fail (chunk 3 returns 404 since served counter > 3).
	got, err = c.AreAllSequentialFeedsUpdateRetrievable(context.Background(), owner, topic, 3, nil)
	if err != nil || got {
		t.Errorf("want at least one missing, got=%v err=%v", got, err)
	}
}

func TestFetchLatestFeedUpdate_AndFindNextIndex(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/feeds/") && r.Method == http.MethodGet {
			w.Header().Set("swarm-feed-index", "0000000000000007")
			w.Header().Set("swarm-feed-index-next", "0000000000000008")
			w.WriteHeader(200)
			w.Write([]byte("payload"))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()
	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)
	owner, _ := swarm.EthAddressFromHex(strings.Repeat("bb", 20))
	topic := swarm.TopicFromString("t")

	upd, err := c.FetchLatestFeedUpdate(context.Background(), owner, topic)
	if err != nil {
		t.Fatal(err)
	}
	if upd.Index != 7 || upd.IndexNext != 8 || string(upd.Payload) != "payload" {
		t.Errorf("got %+v", upd)
	}
	idx, err := c.FindNextIndex(context.Background(), owner, topic)
	if err != nil || idx != 8 {
		t.Errorf("idx=%d err=%v", idx, err)
	}
}

func TestFindNextIndex_NoUpdates(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer s.Close()
	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)
	owner, _ := swarm.EthAddressFromHex(strings.Repeat("bb", 20))
	topic := swarm.TopicFromString("t")
	idx, err := c.FindNextIndex(context.Background(), owner, topic)
	if err != nil || idx != 0 {
		t.Errorf("idx=%d err=%v, want 0/nil", idx, err)
	}
}

func TestFeedWriter_AutoIndexUpload(t *testing.T) {
	const refHex = "abababababababababababababababababababababababababababababababab"
	gotIdxs := []uint64{}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/feeds/") && r.Method == http.MethodGet {
			// First call: index=0 has been written; next should be 1.
			w.Header().Set("swarm-feed-index", "0000000000000000")
			w.Header().Set("swarm-feed-index-next", "0000000000000001")
			w.WriteHeader(200)
			w.Write([]byte{})
			return
		}
		if strings.HasPrefix(r.URL.Path, "/soc/") && r.Method == http.MethodPost {
			gotIdxs = append(gotIdxs, 1) // approximation: count uploads
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference": "` + refHex + `"}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer s.Close()
	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)

	signer, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	topic := swarm.TopicFromString("t")
	w := c.MakeFeedWriter(signer, topic)

	res, err := w.Upload(context.Background(), batch, []byte("hello"))
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if res.Reference.Hex() != refHex {
		t.Errorf("ref = %s", res.Reference.Hex())
	}
	if len(gotIdxs) != 1 {
		t.Errorf("expected 1 SOC upload, got %d", len(gotIdxs))
	}
}

func TestService_Feed_Update(t *testing.T) {
	const refHex = "3333333333333333333333333333333333333333333333333333333333333333"
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/soc/") && r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"reference": "` + refHex + `"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := file.NewService(u, http.DefaultClient)

	signer, _ := swarm.PrivateKeyFromHex(strings.Repeat("11", 32))
	batch := swarm.MustBatchID(strings.Repeat("aa", 32))
	topic := swarm.TopicFromString("test-topic")

	ref, err := c.UpdateFeedWithIndex(context.Background(), batch, signer, topic, 0, []byte("update"))
	if err != nil {
		t.Fatalf("UpdateFeedWithIndex error = %v", err)
	}
	if ref.Reference.Hex() != refHex {
		t.Errorf("UpdateFeedWithIndex ref = %s, want %s", ref.Reference.Hex(), refHex)
	}
}
