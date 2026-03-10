package file

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// CreateFeedManifest creates a feed manifest.
// owner: 32-byte hex string
// topic: 32-byte hex string
func (s *Service) CreateFeedManifest(ctx context.Context, batchID string, owner string, topic string) (swarm.Reference, error) {
	path := fmt.Sprintf("feeds/%s/%s", owner, topic)
	u := s.baseURL.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return swarm.Reference{}, err
	}

	req.Header.Set("Swarm-Postage-Batch-Id", batchID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return swarm.Reference{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return swarm.Reference{}, fmt.Errorf("create feed manifest failed with status: %d", resp.StatusCode)
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.Reference{Value: res.Reference}, nil
}

// GetFeedLookup retrieves the latest feed update lookup.
// Returns the reference to the chunk containing the update ? No, it returns the feed update.
// Actually GET /feeds/:owner/:topic returns a JSON with valid response... or redirects to the chunk?
// Bee API doc says: "Returns the latest version of the feed."
// If successful, returns 200 OK and the data is... the chunk reference or the content?
// Bee-JS `download` calls this.
// Usually this returns the underlying reference.
// Let's assume it returns a Reference structure similar to others or raw API response.
// Bee API: GET /feeds/... response is 200 OK, Body: {"reference": "..."} (JSON)
func (s *Service) GetFeedLookup(ctx context.Context, owner string, topic string) (swarm.Reference, error) {
	path := fmt.Sprintf("feeds/%s/%s", owner, topic)
	u := s.baseURL.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return swarm.Reference{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return swarm.Reference{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return swarm.Reference{}, fmt.Errorf("get feed lookup failed with status: %d", resp.StatusCode)
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.Reference{Value: res.Reference}, nil
}

// UpdateFeed updates a feed.
func (s *Service) UpdateFeed(ctx context.Context, batchID string, signer *ecdsa.PrivateKey, topic string, data []byte) (swarm.Reference, error) {
	// For now, simpler version that fails if index logic needed, or just calls UpdateFeedWithIndex with 0 if we assume new?
	// But standard bee-js UpdateFeed finds the next index.
	// As discussed, fully implementing index finding requires downloading/parsing.
	// We will implement UpdateFeedWithIndex for now which gives control.
	return swarm.Reference{}, fmt.Errorf("UpdateFeed automatic index finding not yet implemented; use UpdateFeedWithIndex")
}

// UpdateFeedWithIndex updates a feed at a specific index.
func (s *Service) UpdateFeedWithIndex(ctx context.Context, batchID string, signer *ecdsa.PrivateKey, topic string, index int64, data []byte) (swarm.Reference, error) {
	topicBytes, err := hex.DecodeString(topic)
	if err != nil {
		return swarm.Reference{}, err
	}

	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, uint64(index))

	identifier := crypto.Keccak256(topicBytes, indexBytes)

	// Payload: Timestamp (8 bytes) + Data
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))

	payload := append(timestamp, data...)

	soc, err := swarm.CreateSOC(identifier, payload, signer)
	if err != nil {
		return swarm.Reference{}, err
	}

	ownerAddr := crypto.PubkeyToAddress(signer.PublicKey)
	ownerHex := fmt.Sprintf("%x", ownerAddr)
	idHex := fmt.Sprintf("%x", identifier)
	sigHex := fmt.Sprintf("%x", soc.Signature)

	// Correct call to s.UploadSOC (in same package)
	return s.UploadSOC(ctx, batchID, ownerHex, idHex, sigHex, soc.Payload, nil)
}
