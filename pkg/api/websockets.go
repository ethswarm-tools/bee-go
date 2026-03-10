package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

// Subscription represents a WebSocket subscription.
type Subscription struct {
	conn *websocket.Conn
}

// ReadMessage reads a message from the subscription.
func (s *Subscription) ReadMessage() (int, []byte, error) {
	return s.conn.ReadMessage()
}

// Close closes the subscription.
func (s *Subscription) Close() error {
	return s.conn.Close()
}

// PSSSubscribe subscribes to a PSS topic.
func PSSSubscribe(ctx context.Context, baseURL *url.URL, dialer *websocket.Dialer, topic string) (*Subscription, error) {
	u := baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("pss/subscribe/%s", topic)})
	wsURL := strings.Replace(u.String(), "http", "ws", 1)

	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("pps subscribe failed with status: %d, err: %w", resp.StatusCode, err)
		}
		return nil, err
	}

	return &Subscription{conn: conn}, nil
}

// GSOCSubscribe subscribes to GSOC messages.
func GSOCSubscribe(ctx context.Context, baseURL *url.URL, dialer *websocket.Dialer, address string) (*Subscription, error) {
	u := baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("gsoc/subscribe/%s", address)})
	wsURL := strings.Replace(u.String(), "http", "ws", 1)

	conn, resp, err := dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("gsoc subscribe failed with status: %d, err: %w", resp.StatusCode, err)
		}
		return nil, err
	}

	return &Subscription{conn: conn}, nil
}
