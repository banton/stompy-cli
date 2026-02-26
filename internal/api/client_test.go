package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.example.com", "test-token", "1.2.3", false)

	if c.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.example.com")
	}
	if c.AuthToken != "test-token" {
		t.Errorf("AuthToken = %q, want %q", c.AuthToken, "test-token")
	}
	if c.UserAgent != "stompy-cli/1.2.3" {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, "stompy-cli/1.2.3")
	}
	if c.Verbose {
		t.Error("Verbose should be false")
	}
}

func TestNewClient_DevVersion(t *testing.T) {
	c := NewClient("https://api.example.com", "tok", "dev", false)
	if c.UserAgent != "stompy-cli/dev" {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, "stompy-cli/dev")
	}
}

func TestNewClient_EmptyVersion(t *testing.T) {
	c := NewClient("https://api.example.com", "tok", "", false)
	if c.UserAgent != "stompy-cli/dev" {
		t.Errorf("UserAgent = %q, want %q (fallback for empty version)", c.UserAgent, "stompy-cli/dev")
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://api.example.com/", "tok", "dev", false)
	if c.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, want trailing slash trimmed", c.BaseURL)
	}
}

func TestClient_Do_SetsHeaders(t *testing.T) {
	var gotHeaders http.Header
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeaders = r.Header
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "my-token", "0.2.0", false)
	_, _, err := c.Do(http.MethodPost, "/test", map[string]string{"key": "val"}, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if got := gotHeaders.Get("Authorization"); got != "Bearer my-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer my-token")
	}
	if got := gotHeaders.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := gotHeaders.Get("User-Agent"); got != "stompy-cli/0.2.0" {
		t.Errorf("User-Agent = %q, want %q", got, "stompy-cli/0.2.0")
	}
}

func TestClient_Do_QueryParams(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "dev", false)
	params := url.Values{"foo": {"bar"}, "baz": {"1"}}
	_, _, err := c.Do(http.MethodGet, "/items", nil, params)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}

	parsed, _ := url.Parse(gotPath)
	if parsed.Path != "/items" {
		t.Errorf("path = %q, want /items", parsed.Path)
	}
	if parsed.Query().Get("foo") != "bar" {
		t.Errorf("query foo = %q, want bar", parsed.Query().Get("foo"))
	}
}

func TestClient_Do_NonOKReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "not found",
			"detail":  "project does not exist",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	_, _, err := c.Do(http.MethodGet, "/missing", nil, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
	if apiErr.Message != "not found" {
		t.Errorf("Message = %q, want %q", apiErr.Message, "not found")
	}
}

func TestClient_Do_NonJSONErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "dev", false)
	_, _, err := c.Do(http.MethodGet, "/fail", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr := err.(*APIError)
	if apiErr.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", apiErr.StatusCode)
	}
}

func TestClient_Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]string{"name": "test"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	var result map[string]string
	err := c.Get("/resource", nil, &result)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("name = %q, want %q", result["name"], "test")
	}
}

func TestClient_Post(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotBody)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	var result map[string]string
	err := c.Post("/resource", map[string]string{"name": "new"}, &result)
	if err != nil {
		t.Fatalf("Post() error: %v", err)
	}
	if gotBody["name"] != "new" {
		t.Errorf("request body name = %q, want %q", gotBody["name"], "new")
	}
	if result["id"] != "123" {
		t.Errorf("response id = %q, want %q", result["id"], "123")
	}
}

func TestClient_Put(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	var result map[string]string
	err := c.Put("/resource/1", map[string]string{"name": "updated"}, &result)
	if err != nil {
		t.Fatalf("Put() error: %v", err)
	}
	if result["status"] != "updated" {
		t.Errorf("status = %q, want %q", result["status"], "updated")
	}
}

func TestClient_Delete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	err := c.Delete("/resource/1", nil)
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
	}
}

func TestClient_Do_RetriesOnTimeout(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			// Simulate timeout by not responding before the client gives up.
			// We use a hijack to close the connection abruptly.
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatal("server doesn't support hijacking")
			}
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	// Use a short timeout so the test is fast.
	c.HTTPClient.Timeout = 500 * time.Millisecond

	data, code, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if code != 200 {
		t.Errorf("status = %d, want 200", code)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("body = %q, want {\"ok\":true}", string(data))
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3 (1 initial + 2 retries)", attempts)
	}
}

func TestClient_Do_RetriesOn502(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte("bad gateway"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	data, code, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if code != 200 {
		t.Errorf("status = %d, want 200", code)
	}
	if string(data) != `{"ok":true}` {
		t.Errorf("body = %q", string(data))
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestClient_Do_NoRetryOnPost(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("bad gateway"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	_, _, err := c.Do(http.MethodPost, "/test", map[string]string{"k": "v"}, nil)
	if err == nil {
		t.Fatal("expected error for 502 on POST")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry for POST)", attempts)
	}
}

func TestClient_Do_NoRetryOn4xx(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	_, _, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (no retry for 404)", attempts)
	}
}

func TestClient_Do_ExhaustsRetries(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unavailable"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	_, _, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	// 1 initial + 2 retries = 3
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestIsIdempotent(t *testing.T) {
	tests := []struct {
		method string
		want   bool
	}{
		{http.MethodGet, true},
		{http.MethodHead, true},
		{http.MethodPut, true},
		{http.MethodDelete, true},
		{http.MethodOptions, true},
		{http.MethodPost, false},
		{http.MethodPatch, false},
	}
	for _, tt := range tests {
		if got := isIdempotent(tt.method); got != tt.want {
			t.Errorf("isIdempotent(%q) = %v, want %v", tt.method, got, tt.want)
		}
	}
}

func TestIsRetryableStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{502, true},
		{503, true},
		{504, true},
		{500, false},
		{404, false},
		{200, false},
		{401, false},
	}
	for _, tt := range tests {
		if got := isRetryableStatus(tt.code); got != tt.want {
			t.Errorf("isRetryableStatus(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestClient_Do_ReadsAPIVersionHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Stompy-API-Version", "6.0.0")
		w.Header().Set("X-Stompy-Min-CLI-Version", "0.1.4")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.2.0", false)
	_, _, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if c.APIVersion != "6.0.0" {
		t.Errorf("APIVersion = %q, want %q", c.APIVersion, "6.0.0")
	}
}

func TestClient_Do_CompatWarningPrintedOnce(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Stompy-Min-CLI-Version", "99.0.0")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.2.0", false)

	// First call should set compatWarned
	_, _, _ = c.Do(http.MethodGet, "/test", nil, nil)
	if !c.compatWarned {
		t.Error("compatWarned should be true after outdated version response")
	}

	// Second call should not warn again (flag already set)
	prevWarned := c.compatWarned
	_, _, _ = c.Do(http.MethodGet, "/test", nil, nil)
	if c.compatWarned != prevWarned {
		t.Error("compatWarned should remain true (no double warning)")
	}
}

func TestClient_Do_NoWarningWhenCompatible(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Stompy-Min-CLI-Version", "0.1.0")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "0.2.0", false)
	_, _, _ = c.Do(http.MethodGet, "/test", nil, nil)
	if c.compatWarned {
		t.Error("compatWarned should be false when CLI version is compatible")
	}
}

func TestClient_Do_NoContentType_ForGetRequests(t *testing.T) {
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "", "dev", false)
	_, _, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if gotContentType != "" {
		t.Errorf("Content-Type for GET = %q, want empty", gotContentType)
	}
}

func TestClient_Do_NoCacheSendsCacheControlHeader(t *testing.T) {
	var gotCacheControl string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCacheControl = r.Header.Get("Cache-Control")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	c.NoCache = true
	_, _, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if gotCacheControl != "no-cache" {
		t.Errorf("Cache-Control = %q, want %q", gotCacheControl, "no-cache")
	}
}

func TestClient_Do_NoCacheResetAfterRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	c.NoCache = true
	_, _, _ = c.Do(http.MethodGet, "/test", nil, nil)
	if c.NoCache {
		t.Error("NoCache should be reset to false after Do()")
	}
}

func TestClient_Do_NoCacheNotSetByDefault(t *testing.T) {
	var gotCacheControl string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCacheControl = r.Header.Get("Cache-Control")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "tok", "dev", false)
	_, _, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if gotCacheControl != "" {
		t.Errorf("Cache-Control = %q, want empty (no-cache not set)", gotCacheControl)
	}
}
