package postage_test

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/postage"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

const testBatchHex = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

func TestService_GetPostageBatches(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		wantLen int
		wantErr bool
	}{
		{
			name: "ok",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`{"batches": [{"batchID": "` + testBatchHex + `", "value": "1000", "start": 0, "owner": "abc", "depth": 17, "bucketDepth": 16, "immutable": false, "batchTTL": 86400}]}`))
			},
			wantLen: 1,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(`invalid`))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := httptest.NewServer(tt.handler)
			defer s.Close()

			u, _ := url.Parse(s.URL)
			c := postage.NewService(u, http.DefaultClient)
			got, err := c.GetPostageBatches(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.GetPostageBatches() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Errorf("Service.GetPostageBatches() len = %v, want %v", len(got), tt.wantLen)
			}
			if !tt.wantErr && len(got) > 0 {
				if got[0].Value.Cmp(big.NewInt(1000)) != 0 {
					t.Errorf("Service.GetPostageBatches() value = %v, want 1000", got[0].Value)
				}
			}
		})
	}
}

func TestService_CreatePostageBatch(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/stamps/1000/17" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("label") != "test" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"batchID": "` + testBatchHex + `"}`))
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := postage.NewService(u, http.DefaultClient)

	id, err := c.CreatePostageBatch(context.Background(), big.NewInt(1000), 17, "test")
	if err != nil {
		t.Fatalf("CreatePostageBatch error = %v", err)
	}
	if id.Hex() != testBatchHex {
		t.Errorf("CreatePostageBatch id = %v, want %s", id.Hex(), testBatchHex)
	}
}

func TestService_TopUpBatch(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/stamps/topup/"+testBatchHex+"/100" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := postage.NewService(u, http.DefaultClient)

	err := c.TopUpBatch(context.Background(), swarm.MustBatchID(testBatchHex), big.NewInt(100))
	if err != nil {
		t.Fatalf("TopUpBatch error = %v", err)
	}
}

func TestService_DiluteBatch(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/stamps/dilute/"+testBatchHex+"/18" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := postage.NewService(u, http.DefaultClient)

	err := c.DiluteBatch(context.Background(), swarm.MustBatchID(testBatchHex), 18)
	if err != nil {
		t.Fatalf("DiluteBatch error = %v", err)
	}
}

func TestService_GetPostageBatch(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stamps/"+testBatchHex) && r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"batchID": "` + testBatchHex + `", "value": "200"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := postage.NewService(u, http.DefaultClient)

	batch, err := c.GetPostageBatch(context.Background(), swarm.MustBatchID(testBatchHex))
	if err != nil {
		t.Fatalf("GetPostageBatch error = %v", err)
	}
	if batch.BatchID.Hex() != testBatchHex {
		t.Errorf("BatchID = %v, want %s", batch.BatchID.Hex(), testBatchHex)
	}
	if batch.Value.Cmp(big.NewInt(200)) != 0 {
		t.Errorf("Batch Value = %v, want 200", batch.Value)
	}
}
