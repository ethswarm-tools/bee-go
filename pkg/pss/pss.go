package pss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// PssSend sends a PSS message.
// topic: 32-byte hex string
// target: 2-byte prefix string (e.g. "1234") or address
// recipient: public key (optional? No, API says `POST /pss/send/:topic/:target`)
// Actually `target` is the routing target.
func (s *Service) PssSend(ctx context.Context, topic string, target string, data io.Reader, recipient string) error {
	path := fmt.Sprintf("pss/send/%s/%s", topic, target)
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	q := u.Query()
	if recipient != "" {
		q.Set("recipient", recipient)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		// PSS send returns 201 Created usually? Or 200? Bee API docs says 200 OK.
		// Let's accept 2xx.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("pss send failed with status: %d", resp.StatusCode)
		}
	}
	return nil
}
