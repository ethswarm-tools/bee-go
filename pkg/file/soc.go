package file

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// UploadSOC uploads a Single Owner Chunk. The owner / id / signature triple
// uniquely addresses the chunk on the network.
func (s *Service) UploadSOC(ctx context.Context, batchID swarm.BatchID, owner swarm.EthAddress, id swarm.Identifier, signature swarm.Signature, data []byte, opts *api.UploadOptions) (api.UploadResult, error) {
	path := fmt.Sprintf("soc/%s/%s", owner.Hex(), id.Hex())
	u := s.baseURL.ResolveReference(&url.URL{Path: path})
	q := u.Query()
	q.Set("sig", signature.Hex())
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(data))
	if err != nil {
		return api.UploadResult{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	api.PrepareUploadHeaders(req, batchID, opts)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return api.UploadResult{}, err
	}
	defer resp.Body.Close()

	if err := swarm.CheckResponse(resp); err != nil {
		return api.UploadResult{}, err
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return api.UploadResult{}, err
	}
	return api.ReadUploadResult(res.Reference, resp.Header)
}

// SOCReader downloads SOCs for a known owner. Mirrors bee-js SOCReader.
type SOCReader struct {
	owner   swarm.EthAddress
	service *Service
}

// MakeSOCReader returns a reader for SOCs owned by owner. Convenience
// wrapper around DownloadChunk + the keccak256(identifier || owner)
// addressing rule. Mirrors bee-js Bee.makeSOCReader.
func (s *Service) MakeSOCReader(owner swarm.EthAddress) *SOCReader {
	return &SOCReader{owner: owner, service: s}
}

// Owner returns the address whose SOCs this reader downloads.
func (r *SOCReader) Owner() swarm.EthAddress { return r.owner }

// Download fetches the SOC at keccak256(identifier || owner), parses
// the wire form, and verifies the recovered signer matches the owner.
func (r *SOCReader) Download(ctx context.Context, id swarm.Identifier) (*swarm.SingleOwnerChunk, error) {
	addr, err := swarm.CalculateSingleOwnerChunkAddress(id, r.owner)
	if err != nil {
		return nil, err
	}
	data, err := r.service.DownloadChunk(ctx, addr, nil)
	if err != nil {
		return nil, err
	}
	return swarm.UnmarshalSingleOwnerChunk(data, addr)
}

// SOCWriter is a SOCReader that can also upload signed SOCs. Mirrors
// bee-js SOCWriter.
type SOCWriter struct {
	*SOCReader
	signer *ecdsa.PrivateKey
}

// MakeSOCWriter returns a writer that signs uploads with signer. The
// reader half is keyed off the owner address derived from signer.
// Mirrors bee-js Bee.makeSOCWriter.
func (s *Service) MakeSOCWriter(signer *ecdsa.PrivateKey) (*SOCWriter, error) {
	addr := crypto.PubkeyToAddress(signer.PublicKey)
	owner, err := swarm.NewEthAddress(addr.Bytes())
	if err != nil {
		return nil, err
	}
	return &SOCWriter{SOCReader: s.MakeSOCReader(owner), signer: signer}, nil
}

// Upload signs and uploads a SOC for identifier with the given payload.
// Returns the SOC reference (= keccak256(identifier || owner)).
func (w *SOCWriter) Upload(ctx context.Context, batchID swarm.BatchID, id swarm.Identifier, data []byte, opts *api.UploadOptions) (api.UploadResult, error) {
	soc, err := swarm.MakeSingleOwnerChunk(id, data, w.signer)
	if err != nil {
		return api.UploadResult{}, err
	}
	sig, err := swarm.NewSignature(soc.Signature)
	if err != nil {
		return api.UploadResult{}, err
	}
	full := make([]byte, 0, len(soc.Span)+len(soc.Payload))
	full = append(full, soc.Span...)
	full = append(full, soc.Payload...)
	return w.service.UploadSOC(ctx, batchID, w.owner, id, sig, full, opts)
}
