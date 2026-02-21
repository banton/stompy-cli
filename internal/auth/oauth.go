package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	CLIClientID        = "stompy-cli"
	PKCEVerifierLength = 32
	LoginTimeout       = 5 * time.Minute
)

// TokenResponse represents the OAuth token exchange response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// GeneratePKCE creates a code_verifier and code_challenge for OAuth PKCE (RFC 7636).
func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, PKCEVerifierLength)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generating random bytes: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)

	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])

	return verifier, challenge, nil
}

// GenerateState creates a random state parameter for CSRF protection.
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// StartCallbackServer starts a temporary HTTP server to receive the OAuth callback.
// The expectedState parameter is used to verify the CSRF state parameter.
func StartCallbackServer(expectedState string) (port int, codeCh chan string, shutdown func(), err error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, nil, fmt.Errorf("starting callback server: %w", err)
	}

	port = listener.Addr().(*net.TCPAddr).Port
	codeCh = make(chan string, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		if state != expectedState {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			errDesc := r.URL.Query().Get("error_description")
			http.Error(w, fmt.Sprintf("Authentication failed: %s — %s", errMsg, errDesc), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h2>Authentication successful!</h2><p>You can close this window and return to the terminal.</p><script>window.close()</script></body></html>`)

		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener) //nolint:errcheck

	shutdown = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx) //nolint:errcheck
	}

	return port, codeCh, shutdown, nil
}

// OpenBrowser opens the given URL in the user's default browser.
func OpenBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "linux":
		cmd = exec.Command("xdg-open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// ExchangeCode exchanges an authorization code for tokens via POST /oauth/token.
func ExchangeCode(apiURL, code, verifier, redirectURI string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {verifier},
		"redirect_uri":  {redirectURI},
		"client_id":     {CLIClientID},
	}

	tokenURL := strings.TrimSuffix(apiURL, "/api/v1") + "/oauth/token"
	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}

	return &tokenResp, nil
}

// Login performs the full OAuth PKCE login flow:
// generate PKCE pair, start callback server, open browser, wait for code, exchange.
func Login(apiURL string) (*TokenResponse, error) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		return nil, err
	}

	state, err := GenerateState()
	if err != nil {
		return nil, err
	}

	port, codeCh, shutdown, err := StartCallbackServer(state)
	if err != nil {
		return nil, err
	}
	defer shutdown()

	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	baseURL := strings.TrimSuffix(apiURL, "/api/v1")
	authURL := fmt.Sprintf("%s/oauth/authorize?client_id=%s&redirect_uri=%s&code_challenge=%s&code_challenge_method=S256&response_type=code&scope=%s&state=%s",
		baseURL,
		url.QueryEscape(CLIClientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(challenge),
		url.QueryEscape("openid profile email"),
		url.QueryEscape(state),
	)

	fmt.Println("Opening browser to authenticate...")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", authURL)

	if err := OpenBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser: %v\n", err)
	}

	fmt.Print("Waiting for authentication...")
	select {
	case code := <-codeCh:
		fmt.Println(" Done!")
		return ExchangeCode(apiURL, code, verifier, redirectURI)
	case <-time.After(LoginTimeout):
		fmt.Println(" Timed out.")
		return nil, fmt.Errorf("login timed out after %v — please try again", LoginTimeout)
	}
}
