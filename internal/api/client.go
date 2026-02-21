package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
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
		fmt.Fprintf(io.Discard, "[DEBUG] %s %s\n", method, u)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response body: %w", err)
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
