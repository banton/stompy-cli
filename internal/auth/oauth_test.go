package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	// Verifier must be at least 43 characters (32 bytes base64url = 43 chars)
	if len(verifier) < 43 {
		t.Errorf("verifier length = %d, want >= 43", len(verifier))
	}

	// Challenge must differ from verifier (SHA256 hash)
	if challenge == verifier {
		t.Error("challenge should differ from verifier")
	}

	// Both must be valid base64url (no padding)
	if strings.ContainsAny(verifier, "+/=") {
		t.Error("verifier contains non-base64url characters")
	}
	if strings.ContainsAny(challenge, "+/=") {
		t.Error("challenge contains non-base64url characters")
	}

	// Both must decode without error
	if _, err := base64.RawURLEncoding.DecodeString(verifier); err != nil {
		t.Errorf("verifier is not valid base64url: %v", err)
	}
	if _, err := base64.RawURLEncoding.DecodeString(challenge); err != nil {
		t.Errorf("challenge is not valid base64url: %v", err)
	}
}

func TestGeneratePKCE_Uniqueness(t *testing.T) {
	v1, _, _ := GeneratePKCE()
	v2, _, _ := GeneratePKCE()
	if v1 == v2 {
		t.Error("two PKCE calls produced identical verifiers")
	}
}

func TestStartCallbackServer(t *testing.T) {
	state := "fixed-test-state-abc123"
	port, codeCh, shutdown, err := StartCallbackServer(state)
	if err != nil {
		t.Fatalf("StartCallbackServer() error: %v", err)
	}
	defer shutdown()

	if port <= 0 {
		t.Fatalf("expected positive port, got %d", port)
	}

	// Simulate OAuth callback with correct state and code
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=test-auth-code&state=%s", port, state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("callback status = %d, want 200", resp.StatusCode)
	}

	// Code should arrive on channel
	select {
	case code := <-codeCh:
		if code != "test-auth-code" {
			t.Errorf("got code %q, want %q", code, "test-auth-code")
		}
	default:
		t.Error("expected code on channel, got nothing")
	}
}

func TestStartCallbackServer_InvalidState(t *testing.T) {
	state := "correct-state"
	port, _, shutdown, err := StartCallbackServer(state)
	if err != nil {
		t.Fatalf("StartCallbackServer() error: %v", err)
	}
	defer shutdown()

	// Send callback with wrong state
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=test-code&state=wrong-state", port)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("callback with wrong state: status = %d, want 400", resp.StatusCode)
	}
}

func TestStartCallbackServer_MissingCode(t *testing.T) {
	state := "test-state"
	port, _, shutdown, err := StartCallbackServer(state)
	if err != nil {
		t.Fatalf("StartCallbackServer() error: %v", err)
	}
	defer shutdown()

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?error=access_denied&error_description=user+cancelled&state=%s", port, state)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("callback request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("callback with error: status = %d, want 400", resp.StatusCode)
	}
}

func TestExchangeCode(t *testing.T) {
	wantToken := TokenResponse{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth/token" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("ParseForm error: %v", err)
		}

		// Verify expected form values
		if got := r.FormValue("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q, want %q", got, "authorization_code")
		}
		if got := r.FormValue("code"); got != "test-code" {
			t.Errorf("code = %q, want %q", got, "test-code")
		}
		if got := r.FormValue("code_verifier"); got != "test-verifier" {
			t.Errorf("code_verifier = %q, want %q", got, "test-verifier")
		}
		if got := r.FormValue("redirect_uri"); got != "http://localhost:9999/callback" {
			t.Errorf("redirect_uri = %q, want %q", got, "http://localhost:9999/callback")
		}
		if got := r.FormValue("client_id"); got != CLIClientID {
			t.Errorf("client_id = %q, want %q", got, CLIClientID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wantToken)
	}))
	defer server.Close()

	// ExchangeCode strips /api/v1 and appends /oauth/token
	apiURL := server.URL + "/api/v1"
	got, err := ExchangeCode(apiURL, "test-code", "test-verifier", "http://localhost:9999/callback")
	if err != nil {
		t.Fatalf("ExchangeCode() error: %v", err)
	}

	if got.AccessToken != wantToken.AccessToken {
		t.Errorf("AccessToken = %q, want %q", got.AccessToken, wantToken.AccessToken)
	}
	if got.RefreshToken != wantToken.RefreshToken {
		t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, wantToken.RefreshToken)
	}
	if got.ExpiresIn != wantToken.ExpiresIn {
		t.Errorf("ExpiresIn = %d, want %d", got.ExpiresIn, wantToken.ExpiresIn)
	}
	if got.TokenType != wantToken.TokenType {
		t.Errorf("TokenType = %q, want %q", got.TokenType, wantToken.TokenType)
	}
}

func TestExchangeCode_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := ExchangeCode(server.URL+"/api/v1", "code", "verifier", "http://localhost:9999/callback")
	if err == nil {
		t.Error("ExchangeCode() expected error for 500 response, got nil")
	}
}

func TestGenerateState(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}
	if state == "" {
		t.Error("GenerateState() returned empty string")
	}

	// Must be valid base64url
	if _, err := base64.RawURLEncoding.DecodeString(state); err != nil {
		t.Errorf("state is not valid base64url: %v", err)
	}
}

func TestGenerateState_Uniqueness(t *testing.T) {
	s1, _ := GenerateState()
	s2, _ := GenerateState()
	if s1 == s2 {
		t.Error("two GenerateState calls produced identical values")
	}
}
