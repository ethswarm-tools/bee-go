package file

import (
	"archive/tar"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// UploadFile uploads a single file via POST /bzz. name is the displayed
// filename (sent as the `name` query parameter); contentType becomes the
// stored MIME type and may be overridden by FileUploadOptions.ContentType.
//
// If contentType is empty and opts does not specify one, application/
// octet-stream is used.
func (s *Service) UploadFile(ctx context.Context, batchID swarm.BatchID, data io.Reader, name string, contentType string, opts *api.FileUploadOptions) (api.UploadResult, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bzz"})
	q := u.Query()
	if name != "" {
		q.Set("name", name)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
	if err != nil {
		return api.UploadResult{}, err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	api.PrepareFileUploadHeaders(req, batchID, opts)

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

// DownloadFile downloads a file from Bee. Returns the body reader, the
// parsed file headers (Content-Disposition / Content-Type / Swarm-Tag-Uid)
// and any error. The caller must close the reader.
func (s *Service) DownloadFile(ctx context.Context, ref swarm.Reference, opts *api.DownloadOptions) (io.ReadCloser, api.FileHeaders, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bzz/" + ref.Hex()})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, api.FileHeaders{}, err
	}
	api.PrepareDownloadHeaders(req, opts)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, api.FileHeaders{}, err
	}
	if err := swarm.CheckResponse(resp); err != nil {
		resp.Body.Close()
		return nil, api.FileHeaders{}, err
	}
	return resp.Body, api.ParseFileHeaders(resp.Header), nil
}

// UploadCollection uploads a directory tree as a tar stream via POST /bzz.
// indexFile (e.g. "index.html") is the document served when the collection
// root is requested; opts may further specify an error document.
func (s *Service) UploadCollection(ctx context.Context, batchID swarm.BatchID, dir string, opts *api.CollectionUploadOptions) (api.UploadResult, error) {
	pr, pw := io.Pipe()
	go func() {
		tw := tar.NewWriter(pw)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}
			header.Name = rel
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() { _ = f.Close() }()
			_, err = io.Copy(tw, f)
			return err
		})
		if err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		_ = tw.Close()
		_ = pw.Close()
	}()

	u := s.baseURL.ResolveReference(&url.URL{Path: "bzz"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), pr)
	if err != nil {
		return api.UploadResult{}, err
	}
	req.Header.Set("Content-Type", "application/x-tar")
	req.Header.Set("Swarm-Collection", "true")
	api.PrepareCollectionUploadHeaders(req, batchID, opts)

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
