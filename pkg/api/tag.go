package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// Tag represents a Swarm tag.
// Structure from Bee API docs.
type Tag struct {
	Uid       uint32 `json:"uid"`
	Name      string `json:"name"`
	Total     int64  `json:"total"`
	Split     int64  `json:"split"`
	Seen      int64  `json:"seen"`
	Stored    int64  `json:"stored"`
	Sent      int64  `json:"sent"`
	Synced    int64  `json:"synced"`
	Address   string `json:"address"`
	StartedAt string `json:"startedAt"`
}

// CreateTag creates a new tag.
func (s *Service) CreateTag(ctx context.Context) (Tag, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "tags"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	if err != nil {
		return Tag{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return Tag{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return Tag{}, fmt.Errorf("create tag failed with status: %d", resp.StatusCode)
	}

	var t Tag
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return Tag{}, err
	}
	return t, nil
}

// GetTag retrieves a tag by UID.
func (s *Service) GetTag(ctx context.Context, uid uint32) (Tag, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("tags/%d", uid)})
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Tag{}, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return Tag{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Tag{}, fmt.Errorf("get tag failed with status: %d", resp.StatusCode)
	}

	var t Tag
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return Tag{}, err
	}
	return t, nil
}

// ListTags retrieves a list of tags.
func (s *Service) ListTags(ctx context.Context, offset int, limit int) ([]Tag, error) {
	u := s.baseURL.ResolveReference(&url.URL{Path: "tags"})
	q := u.Query()
	if offset > 0 {
		q.Set("offset", fmt.Sprintf("%d", offset))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list tags failed with status: %d", resp.StatusCode)
	}

	var res struct {
		Tags []Tag `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Tags, nil
}

// DeleteTag deletes a tag by UID.
func (s *Service) DeleteTag(ctx context.Context, uid uint32) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("tags/%d", uid)})
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("delete tag failed with status: %d", resp.StatusCode)
	}
	return nil
}

// UpdateTag updates a tag by UID.
func (s *Service) UpdateTag(ctx context.Context, uid uint32, tag Tag) error {
	u := s.baseURL.ResolveReference(&url.URL{Path: fmt.Sprintf("tags/%d", uid)})

	body, err := json.Marshal(tag)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("update tag failed with status: %d", resp.StatusCode)
	}
	return nil
}
