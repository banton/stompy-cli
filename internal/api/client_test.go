package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.example.com", "test-token", false)

	if c.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.example.com")
	}
	if c.AuthToken != "test-token" {
		t.Errorf("AuthToken = %q, want %q", c.AuthToken, "test-token")
	}
	if c.UserAgent != "stompy-cli/dev" {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, "stompy-cli/dev")
	}
	if c.Verbose {
		t.Error("Verbose should be false")
	}
}

func TestNewClient_TrimsTrailingSlash(t *testing.T) {
	c := NewClient("https://api.example.com/", "tok", false)
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

	c := NewClient(srv.URL, "my-token", false)
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
	if got := gotHeaders.Get("User-Agent"); got != "stompy-cli/dev" {
		t.Errorf("User-Agent = %q, want %q", got, "stompy-cli/dev")
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

	c := NewClient(srv.URL, "", false)
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

	c := NewClient(srv.URL, "tok", false)
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

	c := NewClient(srv.URL, "", false)
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

	c := NewClient(srv.URL, "tok", false)
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

	c := NewClient(srv.URL, "tok", false)
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

	c := NewClient(srv.URL, "tok", false)
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

	c := NewClient(srv.URL, "tok", false)
	err := c.Delete("/resource/1", nil)
	if err != nil {
		t.Fatalf("Delete() error: %v", err)
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

	c := NewClient(srv.URL, "", false)
	_, _, err := c.Do(http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("Do() error: %v", err)
	}
	if gotContentType != "" {
		t.Errorf("Content-Type for GET = %q, want empty", gotContentType)
	}
}
