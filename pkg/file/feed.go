package file

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// CreateFeedManifest creates a feed manifest for the (owner, topic) pair.
func (s *Service) CreateFeedManifest(ctx context.Context, batchID swarm.BatchID, owner swarm.EthAddress, topic swarm.Topic) (swarm.Reference, error) {
	path := fmt.Sprintf("feeds/%s/%s", owner.Hex(), topic.Hex())
	u := s.baseURL.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return swarm.Reference{}, err
	}

	req.Header.Set("Swarm-Postage-Batch-Id", batchID.Hex())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return swarm.Reference{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return swarm.Reference{}, err
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.ReferenceFromHex(res.Reference)
}

// GetFeedLookup retrieves the latest feed update lookup for the (owner, topic)
// pair. Bee returns 200 OK with body {"reference": "..."}.
func (s *Service) GetFeedLookup(ctx context.Context, owner swarm.EthAddress, topic swarm.Topic) (swarm.Reference, error) {
	path := fmt.Sprintf("feeds/%s/%s", owner.Hex(), topic.Hex())
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

	if err := swarm.CheckResponse(resp); err != nil {
		return swarm.Reference{}, err
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.ReferenceFromHex(res.Reference)
}

// FeedUpdate is the result of a feed lookup. Payload is the raw chunk
// payload (timestamp + data, when written via UpdateFeedWithIndex).
// Index is the feed index of the returned update; IndexNext is the index
// where the *next* update should be written (Index + 1 for sequential
// feeds).
//
// Mirrors bee-js FeedPayloadResult.
type FeedUpdate struct {
	Payload   []byte
	Index     uint64
	IndexNext uint64
}

// FetchLatestFeedUpdate downloads the most recent update for (owner,
// topic) by hitting GET /feeds. The response body is the wrapped chunk
// payload; swarm-feed-index / swarm-feed-index-next headers carry the
// indexes (BE-uint64 hex). Mirrors bee-js fetchLatestFeedUpdate.
func (s *Service) FetchLatestFeedUpdate(ctx context.Context, owner swarm.EthAddress, topic swarm.Topic) (FeedUpdate, error) {
	path := fmt.Sprintf("feeds/%s/%s", owner.Hex(), topic.Hex())
	u := s.baseURL.ResolveReference(&url.URL{Path: path})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return FeedUpdate{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return FeedUpdate{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return FeedUpdate{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FeedUpdate{}, err
	}
	idx, err := decodeFeedIndexHeader(resp.Header.Get("swarm-feed-index"))
	if err != nil {
		return FeedUpdate{}, swarm.WrapBeeError("swarm-feed-index", err)
	}
	idxNext, err := decodeFeedIndexHeader(resp.Header.Get("swarm-feed-index-next"))
	if err != nil {
		return FeedUpdate{}, swarm.WrapBeeError("swarm-feed-index-next", err)
	}
	return FeedUpdate{Payload: body, Index: idx, IndexNext: idxNext}, nil
}

// FindNextIndex returns the index where the next feed update should be
// written. Returns 0 if the feed has no updates yet (Bee responds 404).
// Mirrors bee-js findNextIndex.
func (s *Service) FindNextIndex(ctx context.Context, owner swarm.EthAddress, topic swarm.Topic) (uint64, error) {
	upd, err := s.FetchLatestFeedUpdate(ctx, owner, topic)
	if err == nil {
		return upd.IndexNext, nil
	}
	if rerr, ok := swarm.IsBeeResponseError(err); ok && (rerr.Status == 404 || rerr.Status == 500) {
		return 0, nil
	}
	return 0, err
}

// UpdateFeed updates a feed at the next available index by first calling
// FindNextIndex. data is wrapped as `BE-uint64(timestamp) || data` in the
// SOC payload, exactly like UpdateFeedWithIndex. Mirrors bee-js
// updateFeedWithPayload.
func (s *Service) UpdateFeed(ctx context.Context, batchID swarm.BatchID, signer swarm.PrivateKey, topic swarm.Topic, data []byte) (api.UploadResult, error) {
	owner := signer.PublicKey().Address()
	idx, err := s.FindNextIndex(ctx, owner, topic)
	if err != nil {
		return api.UploadResult{}, err
	}
	return s.UpdateFeedWithIndex(ctx, batchID, signer, topic, idx, data)
}

// UpdateFeedWithReference updates a feed to point at the given Swarm
// reference. The chunk payload is `timestamp(8) || reference(32)` —
// matches bee-js updateFeedWithReference. If index is nil, FindNextIndex
// is called.
func (s *Service) UpdateFeedWithReference(ctx context.Context, batchID swarm.BatchID, signer swarm.PrivateKey, topic swarm.Topic, ref swarm.Reference, index *uint64) (api.UploadResult, error) {
	owner := signer.PublicKey().Address()
	idx := uint64(0)
	if index != nil {
		idx = *index
	} else {
		next, err := s.FindNextIndex(ctx, owner, topic)
		if err != nil {
			return api.UploadResult{}, err
		}
		idx = next
	}
	return s.UpdateFeedWithIndex(ctx, batchID, signer, topic, idx, ref.Raw())
}

func decodeFeedIndexHeader(s string) (uint64, error) {
	if s == "" {
		return 0, fmt.Errorf("missing header")
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return 0, err
	}
	if len(b) != 8 {
		return 0, fmt.Errorf("expected 8 bytes, got %d", len(b))
	}
	return binary.BigEndian.Uint64(b), nil
}

// UpdateFeedWithIndex updates a feed at the specified index.
//
// The feed identifier is keccak256(topic || BE-uint64(index)). The chunk
// payload is BE-uint64(timestamp) || data. The chunk is signed via SOC and
// uploaded.
func (s *Service) UpdateFeedWithIndex(ctx context.Context, batchID swarm.BatchID, signer swarm.PrivateKey, topic swarm.Topic, index uint64, data []byte) (api.UploadResult, error) {
	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, index)

	identifierBytes := keccak256(topic.Raw(), indexBytes)
	identifier, err := swarm.NewIdentifier(identifierBytes)
	if err != nil {
		return api.UploadResult{}, err
	}

	timestamp := make([]byte, 8)
	//nolint:gosec // unix epoch fits in uint64 for any plausible date.
	binary.BigEndian.PutUint64(timestamp, uint64(time.Now().Unix()))
	payload := make([]byte, 0, len(timestamp)+len(data))
	payload = append(payload, timestamp...)
	payload = append(payload, data...)

	ecdsaSigner, err := signer.ToECDSA()
	if err != nil {
		return api.UploadResult{}, err
	}
	soc, err := swarm.CreateSOC(identifierBytes, payload, ecdsaSigner)
	if err != nil {
		return api.UploadResult{}, err
	}
	signature, err := swarm.NewSignature(soc.Signature)
	if err != nil {
		return api.UploadResult{}, err
	}

	full := make([]byte, 0, len(soc.Span)+len(soc.Payload))
	full = append(full, soc.Span...)
	full = append(full, soc.Payload...)
	owner := signer.PublicKey().Address()
	return s.UploadSOC(ctx, batchID, owner, identifier, signature, full, nil)
}

// keccak256 wrapper local to this file to avoid pulling go-ethereum into the
// import set just for the hash. swarm.Keccak256 already exists.
func keccak256(parts ...[]byte) []byte {
	return swarm.Keccak256(parts...)
}

// MakeFeedIdentifier returns the identifier for a feed update at the given
// topic + index: keccak256(topic || BE-uint64(index)). Mirrors bee-js
// makeFeedIdentifier.
func MakeFeedIdentifier(topic swarm.Topic, index uint64) (swarm.Identifier, error) {
	indexBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(indexBytes, index)
	return swarm.NewIdentifier(keccak256(topic.Raw(), indexBytes))
}

// FeedUpdateChunkReference returns the SOC chunk address for the feed
// update at (owner, topic, index): keccak256(identifier || owner). Use
// this with DownloadChunk to verify retrievability of past updates.
// Mirrors bee-js getFeedUpdateChunkReference.
func FeedUpdateChunkReference(owner swarm.EthAddress, topic swarm.Topic, index uint64) (swarm.Reference, error) {
	id, err := MakeFeedIdentifier(topic, index)
	if err != nil {
		return swarm.Reference{}, err
	}
	return swarm.NewReference(keccak256(id.Raw(), owner.Raw()))
}

// IsFeedRetrievable reports whether the feed at (owner, topic) currently
// resolves on the network.
//
// If index is nil, only the latest feed update is checked (a weaker
// guarantee, since "latest" is observer-dependent). If index is non-nil,
// every sequential chunk up to and including that index is checked — see
// AreAllSequentialFeedsUpdateRetrievable.
//
// Mirrors bee-js Bee.isFeedRetrievable: 404 / 500 from Bee become
// (false, nil); other errors propagate.
func (s *Service) IsFeedRetrievable(ctx context.Context, owner swarm.EthAddress, topic swarm.Topic, index *uint64, opts *api.DownloadOptions) (bool, error) {
	if index == nil {
		_, err := s.FetchLatestFeedUpdate(ctx, owner, topic)
		if err == nil {
			return true, nil
		}
		if rerr, ok := swarm.IsBeeResponseError(err); ok && (rerr.Status == 404 || rerr.Status == 500) {
			return false, nil
		}
		return false, err
	}
	return s.AreAllSequentialFeedsUpdateRetrievable(ctx, owner, topic, *index, opts)
}

// AreAllSequentialFeedsUpdateRetrievable verifies that every feed-update
// chunk from index 0 through `index` (inclusive) is currently retrievable
// from the network. Returns true only if all are present.
//
// Used to validate that a feed can be replayed from its origin. Mirrors
// bee-js areAllSequentialFeedsUpdateRetrievable.
func (s *Service) AreAllSequentialFeedsUpdateRetrievable(ctx context.Context, owner swarm.EthAddress, topic swarm.Topic, index uint64, opts *api.DownloadOptions) (bool, error) {
	for i := uint64(0); i <= index; i++ {
		ref, err := FeedUpdateChunkReference(owner, topic, i)
		if err != nil {
			return false, err
		}
		if _, err := s.DownloadChunk(ctx, ref, opts); err != nil {
			if rerr, ok := swarm.IsBeeResponseError(err); ok && (rerr.Status == 404 || rerr.Status == 500) {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}
