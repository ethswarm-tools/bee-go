package file

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ethswarm-tools/bee-go/pkg/api"
	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

// ReferenceInformation is the result of ProbeData: the size of the data
// at a /bytes reference, learned via HEAD without downloading the body.
type ReferenceInformation struct {
	ContentLength int64
}

// ProbeData fetches the content length for a /bytes reference using a
// HEAD request. Useful for sizing downloads or validating a ref points
// at retrievable data without paying the bandwidth cost.
//
// Mirrors bee-js Bee.probeData.
func (s *Service) ProbeData(ctx context.Context, ref swarm.Reference) (ReferenceInformation, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bytes/" + ref.Hex()})
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
	if err != nil {
		return ReferenceInformation{}, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return ReferenceInformation{}, err
	}
	defer resp.Body.Close()
	if err := swarm.CheckResponse(resp); err != nil {
		return ReferenceInformation{}, err
	}
	cl := resp.Header.Get("Content-Length")
	if cl == "" {
		return ReferenceInformation{ContentLength: resp.ContentLength}, nil
	}
	n, err := strconv.ParseInt(cl, 10, 64)
	if err != nil {
		return ReferenceInformation{}, swarm.NewBeeArgumentError("invalid Content-Length", cl)
	}
	return ReferenceInformation{ContentLength: n}, nil
}

// UploadData uploads raw bytes to Bee. The body is sent as
// application/octet-stream; for typed files use UploadFile.
//
// Returns an UploadResult that exposes the content reference, the optional
// auto-created tag UID, and (when ACT was requested) the history address.
func (s *Service) UploadData(ctx context.Context, batchID swarm.BatchID, data io.Reader, opts *api.RedundantUploadOptions) (api.UploadResult, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bytes"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
	if err != nil {
		return api.UploadResult{}, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	api.PrepareRedundantUploadHeaders(req, batchID, opts)

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

// DownloadData downloads raw bytes from Bee. nil opts means "use Bee
// defaults"; pass DownloadOptions to specify ACT, redundancy strategy or
// chunk-retrieval timeout.
//
// The returned ReadCloser must be closed by the caller.
func (s *Service) DownloadData(ctx context.Context, ref swarm.Reference, opts *api.DownloadOptions) (io.ReadCloser, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bytes/" + ref.Hex()})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	api.PrepareDownloadHeaders(req, opts)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if err := swarm.CheckResponse(resp); err != nil {
		resp.Body.Close()
		return nil, err
	}
	return resp.Body, nil
}
