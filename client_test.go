package bee

import (
	"net/http"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid",
			url:     "http://localhost:1633",
			wantErr: false,
		},
		{
			name:    "invalid url",
			url:     "://invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewClient(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("NewClient() returned nil client")
			}
		})
	}
}

func TestWithHTTPClient(t *testing.T) {
	c, _ := NewClient("http://localhost:1633", WithHTTPClient(http.DefaultClient))
	if c.httpClient != http.DefaultClient {
		t.Errorf("WithHTTPClient() failed to set client")
	}
}
