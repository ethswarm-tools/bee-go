// Package gsoc implements Swarm Generic Single-Owner Chunk messaging.
//
// GSOC piggy-backs on Swarm's Single-Owner Chunk primitive to deliver a
// targeted message to a node whose overlay is "near" the SOC's address.
// A signer is "mined" (see swarm.GSOCMine) so the resulting SOC address
// (= keccak256(identifier || owner)) lands inside the target's
// neighbourhood; subscribing to /gsoc/subscribe/{socAddress} on that
// target node delivers any chunks that hash to that address.
//
// Mirrors bee-js Bee.gsocSend / gsocSubscribe.
package gsoc

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/file"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// Service handles GSOC operations. Construct via NewService.
type Service struct {
	baseURL    *url.URL
	httpClient *http.Client
	dialer     *websocket.Dialer
	file       *file.Service
}

// NewService creates a new GSOC service. The file service is reused for
// the underlying SOC upload.
func NewService(baseURL *url.URL, httpClient *http.Client, dialer *websocket.Dialer, fileService *file.Service) *Service {
	return &Service{baseURL: baseURL, httpClient: httpClient, dialer: dialer, file: fileService}
}

// SOCAddress returns keccak256(identifier || ownerAddress) — the SOC
// address that GSOC subscribers listen on.
func SOCAddress(identifier swarm.Identifier, owner swarm.EthAddress) (swarm.Reference, error) {
	return swarm.NewReference(swarm.Keccak256(identifier.Raw(), owner.Raw()))
}

// Send creates a SOC at (identifier, signer) carrying `data` as the
// payload and uploads it. Use swarm.GSOCMine to derive `signer` so the
// SOC lands in the target's neighbourhood. Mirrors bee-js gsoc.send.
//
// Returns the upload result (reference is the SOC address, since GSOC
// SOCs always go to /chunks via the SOC endpoint).
func (s *Service) Send(ctx context.Context, batchID swarm.BatchID, signer swarm.PrivateKey, identifier swarm.Identifier, data []byte, opts *api.UploadOptions) (api.UploadResult, error) {
	ecdsaSigner, err := signer.ToECDSA()
	if err != nil {
		return api.UploadResult{}, err
	}
	soc, err := swarm.CreateSOC(identifier.Raw(), data, ecdsaSigner)
	if err != nil {
		return api.UploadResult{}, err
	}
	owner := signer.PublicKey().Address()
	signature, err := swarm.NewSignature(soc.Signature)
	if err != nil {
		return api.UploadResult{}, err
	}
	return s.file.UploadSOC(ctx, batchID, owner, identifier, signature, soc.Payload, opts)
}

// Subscription is an active GSOC subscription. Mirrors bee-js
// GsocSubscription with a Go-shaped channel API.
type Subscription struct {
	Address  swarm.Reference
	Messages <-chan []byte
	Errors   <-chan error

	conn   *websocket.Conn
	cancel func()
	closed bool
}

// Cancel terminates the subscription.
func (s *Subscription) Cancel() {
	if s.closed {
		return
	}
	s.closed = true
	s.cancel()
	_ = s.conn.Close()
}

// Subscribe opens a WebSocket to /gsoc/subscribe/{socAddress} where
// socAddress = keccak256(identifier || owner). Messages produced via
// Send with the same (signer, identifier) pair stream into the
// returned Subscription.Messages channel.
//
// Mirrors bee-js Bee.gsocSubscribe.
func (s *Service) Subscribe(ctx context.Context, owner swarm.EthAddress, identifier swarm.Identifier) (*Subscription, error) {
	socAddr, err := SOCAddress(identifier, owner)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("gsoc/subscribe/%s", socAddr.Hex())
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	wsURL := strings.Replace(u.String(), "http", "ws", 1)

	conn, resp, err := s.dialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		if resp != nil {
			return nil, swarm.WrapBeeError("gsoc subscribe", swarm.NewBeeResponseError(http.MethodGet, wsURL, resp))
		}
		return nil, swarm.WrapBeeError("gsoc subscribe", err)
	}

	subCtx, cancel := context.WithCancel(ctx)
	msgs := make(chan []byte, 16)
	errs := make(chan error, 1)
	sub := &Subscription{
		Address:  socAddr,
		Messages: msgs,
		Errors:   errs,
		conn:     conn,
		cancel:   cancel,
	}

	go func() {
		defer close(msgs)
		defer close(errs)
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
