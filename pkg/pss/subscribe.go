package pss

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Subscription is an active PSS WebSocket subscription. Receive messages
// via the Messages channel; close the subscription via Cancel().
//
// Mirrors bee-js PssSubscription. The Errors channel is closed when the
// subscription ends (Cancel, server close, or fatal error). Messages is
// closed at the same time.
type Subscription struct {
	Topic    swarm.Topic
	Messages <-chan []byte
	Errors   <-chan error

	conn   *websocket.Conn
	cancel func()
	closed bool
}

// Cancel terminates the subscription. Safe to call multiple times.
func (s *Subscription) Cancel() {
	if s.closed {
		return
	}
	s.closed = true
	s.cancel()
	_ = s.conn.Close()
}

// PssSubscribe opens a WebSocket connection to /pss/subscribe/{topic}
// and streams incoming messages on the returned Subscription.Messages
// channel. Mirrors bee-js Bee.pssSubscribe.
//
// The reader goroutine exits when the context is cancelled, the server
// closes the connection, or a read error occurs. Both channels are
// closed on exit.
func (s *Service) PssSubscribe(ctx context.Context, topic swarm.Topic) (*Subscription, error) {
	path := fmt.Sprintf("pss/subscribe/%s", topic.Hex())
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	wsURL := strings.Replace(u.String(), "http", "ws", 1)

	conn, resp, err := s.dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			return nil, swarm.WrapBeeError("pss subscribe", swarm.NewBeeResponseError(http.MethodGet, wsURL, resp))
		}
		return nil, swarm.WrapBeeError("pss subscribe", err)
	}

	subCtx, cancel := context.WithCancel(ctx)
	msgs := make(chan []byte, 16)
	errs := make(chan error, 1)
	sub := &Subscription{
		Topic:    topic,
		Messages: msgs,
		Errors:   errs,
		conn:     conn,
		cancel:   cancel,
	}

	go func() {
		defer close(msgs)
		defer close(errs)
		// Stop reading once the subscription's context is cancelled.
		go func() {
			<-subCtx.Done()
			_ = conn.Close()
		}()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				if subCtx.Err() == nil {
					select {
					case errs <- err:
					default:
					}
				}
				return
			}
			if len(data) == 0 {
				continue
			}
			select {
			case msgs <- data:
			case <-subCtx.Done():
				return
			}
		}
	}()

	return sub, nil
}

// PssReceive waits for one PSS message on the given topic and returns
// it. timeout = 0 means no timeout (block indefinitely until a message
// arrives or ctx is cancelled). Mirrors bee-js Bee.pssReceive.
func (s *Service) PssReceive(ctx context.Context, topic swarm.Topic, timeout time.Duration) ([]byte, error) {
	sub, err := s.PssSubscribe(ctx, topic)
	if err != nil {
		return nil, err
	}
	defer sub.Cancel()

	var timer <-chan time.Time
	if timeout > 0 {
		t := time.NewTimer(timeout)
		defer t.Stop()
		timer = t.C
	}

	select {
	case msg, ok := <-sub.Messages:
		if !ok {
			// channel closed without a message — probably an error.
			select {
			case e := <-sub.Errors:
				return nil, e
			default:
				return nil, swarm.NewBeeError("pss subscription closed without message")
			}
		}
		return msg, nil
	case e := <-sub.Errors:
		return nil, e
	case <-timer:
		return nil, swarm.NewBeeError("pssReceive timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
