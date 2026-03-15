package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// FileResponse represents an uploaded file/document.
type FileResponse struct {
	ID        int       `json:"id"`
	Filename  string    `json:"filename"`
	Label     string    `json:"label,omitempty"`
	MimeType  string    `json:"mime_type,omitempty"`
	SizeBytes int       `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

// FileListResponse wraps a list of files.
type FileListResponse struct {
	Files []FileResponse `json:"files"`
	Total int            `json:"total"`
}

// ListFiles fetches files for a project with optional search.
func (c *Client) ListFiles(project, search string, limit, offset int) (*FileListResponse, error) {
	params := url.Values{}
	if search != "" {
		params.Set("search", search)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	var resp FileListResponse
	if err := c.Get(fmt.Sprintf("/projects/%s/files", project), params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetFile fetches a single file by ID.
func (c *Client) GetFile(project string, id int) (*FileResponse, error) {
	var resp FileResponse
	if err := c.Get(fmt.Sprintf("/projects/%s/files/%d", project, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UploadFile uploads a file with multipart form data.
func (c *Client) UploadFile(project, filePath, label string) (*FileResponse, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copying file data: %w", err)
	}

	if label != "" {
		if err := writer.WriteField("label", label); err != nil {
			return nil, fmt.Errorf("writing label field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	u := c.BaseURL + fmt.Sprintf("/projects/%s/files", project)

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] --> POST %s (multipart, file: %s)\n", u, filePath)
	}

	req, err := http.NewRequest(http.MethodPost, u, &body)
	if err != nil {
		return nil, fmt.Errorf("creating upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", c.UserAgent)
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	start := time.Now()
	resp, err := c.HTTPClient.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("executing upload request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading upload response: %w", err)
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] <-- %d %s (%s, %d bytes)\n", resp.StatusCode, http.StatusText(resp.StatusCode), elapsed, len(respBody))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(respBody, apiErr); err != nil {
			apiErr.Message = string(respBody)
		}
		return nil, apiErr
	}

	var fileResp FileResponse
	if err := json.Unmarshal(respBody, &fileResp); err != nil {
		return nil, fmt.Errorf("decoding upload response: %w", err)
	}
	return &fileResp, nil
}

// DeleteFile deletes a file by ID.
func (c *Client) DeleteFile(project string, id int) error {
	return c.Delete(fmt.Sprintf("/projects/%s/files/%d", project, id), nil)
}
