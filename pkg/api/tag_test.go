package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/ethersphere/bee-go/pkg/api"
)

func TestService_Tags(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"uid": 123, "name": "tag1", "total": 10}`))
			return
		}
		if r.Method == http.MethodGet {
			if strings.HasSuffix(r.URL.Path, "/123") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"uid": 123, "name": "tag1", "total": 20}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := api.NewService(u, http.DefaultClient)

	tag, err := c.CreateTag(context.Background())
	if err != nil {
		t.Fatalf("CreateTag error = %v", err)
	}
	if tag.UID != 123 {
		t.Errorf("Tag UID = %v, want 123", tag.UID)
	}

	tag2, err := c.GetTag(context.Background(), 123)
	if err != nil {
		t.Fatalf("GetTag error = %v", err)
	}
	if tag2.Total != 20 {
		t.Errorf("Tag Total = %v, want 20", tag2.Total)
	}
}

func TestService_Tags_Extensions(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/tags" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"tags": [{"uid": 123, "name": "tag1"}, {"uid": 456, "name": "tag2"}]}`))
			return
		}
		if r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/123") {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/123") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer s.Close()

	u, _ := url.Parse(s.URL)
	c := api.NewService(u, http.DefaultClient)

	// List
	list, err := c.ListTags(context.Background(), 0, 10)
	if err != nil {
		t.Fatalf("ListTags error = %v", err)
	}
	if len(list) != 2 {
		t.Errorf("ListTags len = %v, want 2", len(list))
	}

	// Delete
	if err := c.DeleteTag(context.Background(), 123); err != nil {
		t.Fatalf("DeleteTag error = %v", err)
	}

	// Update
	if err := c.UpdateTag(context.Background(), 123, api.Tag{Name: "updated"}); err != nil {
		t.Fatalf("UpdateTag error = %v", err)
	}
}
