package file

import (
	"net/http"
	"net/url"
)

// Service is the data-transfer endpoint group: bytes, files, chunks,
// SOCs, feeds, and tar-packed collections. Get one from the top-level
// Client.File field rather than constructing it directly.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewService wires up a file.Service against a Bee base URL and HTTP
// client. The top-level bee.NewClient calls this for you.
func NewService(baseURL *url.URL, httpClient *http.Client) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient}
}
