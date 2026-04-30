package pss

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ethersphere/bee-go/pkg/swarm"
)

// PssSend sends a PSS message.
//
//   - batchID: postage batch to stamp the PSS chunk with. Required by Bee.
//   - topic: 32-byte topic; use swarm.TopicFromString to derive from a label.
//   - target: routing prefix as hex (e.g. "1234"); not a full address. Bee uses
//     this as a partial XOR target so the network gossip can reach the
//     recipient without revealing the full address.
//   - recipient: optional uncompressed public key for end-to-end encryption.
//     Pass the zero PublicKey to send unencrypted.
func (s *Service) PssSend(ctx context.Context, batchID swarm.BatchID, topic swarm.Topic, target string, data io.Reader, recipient swarm.PublicKey) error {
	path := fmt.Sprintf("pss/send/%s/%s", topic.Hex(), target)
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	if !recipient.IsZero() {
		// Bee accepts the recipient public key as compressed hex.
		compressed, err := recipient.CompressedHex()
		if err != nil {
			return fmt.Errorf("compress recipient: %w", err)
		}
		q := u.Query()
		q.Set("recipient", compressed)
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
	if err != nil {
		return err
	}
	req.Header.Set("Swarm-Postage-Batch-Id", batchID.Hex())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return err
	}
	return nil
}
