package file

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/ethersphere/bee-go/pkg/api"
	"github.com/ethersphere/bee-go/pkg/swarm"
)

// UploadFile uploads a file to Swarm.
// contentType is optional, defaults to application/octet-stream.
// name is optional, used for the Swarm-Tag-Name header or similar if supported,
// or just for context. Bee API uses `name` query param for simple uploads sometimes
// but strictly `POST /bzz` expects raw body or multipart.
// Bee-JS uses `POST /bzz` for `uploadFile`.
func (s *Service) UploadFile(ctx context.Context, batchID string, data io.Reader, name string, contentType string, opts *api.UploadOptions) (swarm.Reference, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bzz"})
	q := u.Query()
	if name != "" {
		q.Set("name", name)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), data)
	if err != nil {
		return swarm.Reference{}, err
	}

	req.Header.Set("Swarm-Postage-Batch-Id", batchID)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)

	if opts != nil {
		opts.ApplyToRequest(req)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return swarm.Reference{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return swarm.Reference{}, fmt.Errorf("upload file failed with status: %d", resp.StatusCode)
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.Reference{Value: res.Reference}, nil
}

// DownloadFile downloads a file from Swarm.
// Returns the data reader and the content type.
func (s *Service) DownloadFile(ctx context.Context, ref swarm.Reference) (io.ReadCloser, string, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "bzz/" + ref.Value})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, "", fmt.Errorf("download file failed with status: %d", resp.StatusCode)
	}

	return resp.Body, resp.Header.Get("Content-Type"), nil
}

// UploadCollection uploads a directory to Swarm as a tar stream.
// dir is the local path to the directory.
// indexFile is optional (e.g., "index.html").
func (s *Service) UploadCollection(ctx context.Context, batchID string, dir string, indexFile string, opts *api.UploadOptions) (swarm.Reference, error) {
	// We need to stream the directory as a TAR.
	// Since http.Request body needs to be a Reader, we can use io.Pipe,
	// but we must run the TAR writing in a goroutine.
	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Skip the root directory itself to avoid recursive mess if not handled well,
			// or just standard tar behavior. Standard tar includes it.
			// Bee expects relative paths in the tar.
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			if relPath == "." {
				return nil
			}

			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				return err
			}
			header.Name = relPath

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !info.IsDir() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				if _, err := io.Copy(tw, f); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			pw.CloseWithError(err)
		} else {
			tw.Close()
			pw.Close()
		}
	}()

	u := s.baseURL.ResolveReference(&url.URL{Path: "bzz"})
	q := u.Query()
	if indexFile != "" {
		q.Set("index", indexFile)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), pr)
	if err != nil {
		return swarm.Reference{}, err
	}

	req.Header.Set("Swarm-Postage-Batch-Id", batchID)
	req.Header.Set("Content-Type", "application/x-tar")
	req.Header.Set("Swarm-Collection", "true")

	if opts != nil {
		opts.ApplyToRequest(req)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return swarm.Reference{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return swarm.Reference{}, fmt.Errorf("upload collection failed with status: %d", resp.StatusCode)
	}

	var res struct {
		Reference string `json:"reference"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return swarm.Reference{}, err
	}

	return swarm.Reference{Value: res.Reference}, nil
}
