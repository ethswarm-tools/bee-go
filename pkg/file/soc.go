package file

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

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
