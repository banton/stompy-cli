package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	AuthToken  string
	UserAgent  string
	HTTPClient *http.Client
	Verbose    bool
}

func NewClient(baseURL, authToken string, verbose bool) *Client {
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		AuthToken: authToken,
		UserAgent: "stompy-cli/dev",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Verbose: verbose,
	}
}

func (c *Client) Do(method, path string, body any, params url.Values) ([]byte, int, error) {
	u := c.BaseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	var reqBody io.Reader
	var reqBytes []byte
	if body != nil {
		var err error
		reqBytes, err = json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(reqBytes)
	}

	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}
	req.Header.Set("User-Agent", c.UserAgent)

	if body != nil && (method == http.MethodPost || method == http.MethodPut) {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] --> %s %s\n", method, u)
		if len(reqBytes) > 0 {
			preview := string(reqBytes)
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Fprintf(os.Stderr, "[DEBUG]     Body: %s\n", preview)
		}
	}

	start := time.Now()
	resp, err := c.HTTPClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		if c.Verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] <-- ERROR after %s: %v\n", elapsed, err)
		}
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
	}

	if c.Verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] <-- %d %s (%s, %d bytes)\n", resp.StatusCode, http.StatusText(resp.StatusCode), elapsed, len(respBody))
		if len(respBody) > 0 {
			preview := string(respBody)
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			fmt.Fprintf(os.Stderr, "[DEBUG]     Body: %s\n", preview)
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if err := json.Unmarshal(respBody, apiErr); err != nil {
			apiErr.Message = string(respBody)
		}
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return nil, resp.StatusCode, apiErr
	}

	return respBody, resp.StatusCode, nil
}

func (c *Client) Get(path string, params url.Values, result any) error {
	data, _, err := c.Do(http.MethodGet, path, nil, params)
	if err != nil {
		return err
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

func (c *Client) Post(path string, body any, result any) error {
	data, _, err := c.Do(http.MethodPost, path, body, nil)
	if err != nil {
		return err
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

func (c *Client) Put(path string, body any, result any) error {
	data, _, err := c.Do(http.MethodPut, path, body, nil)
	if err != nil {
		return err
	}
	if result != nil {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

func (c *Client) Delete(path string, params url.Values) error {
	_, _, err := c.Do(http.MethodDelete, path, nil, params)
	return err
}

// DeleteWithResult performs a DELETE and decodes the JSON response body.
// Use for endpoints that return data (e.g. context unlock returns ContextDeleteResponse).
func (c *Client) DeleteWithResult(path string, params url.Values, result any) error {
	data, statusCode, err := c.Do(http.MethodDelete, path, nil, params)
	if err != nil {
		return err
	}
	if result != nil && statusCode != http.StatusNoContent && len(data) > 0 {
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}
