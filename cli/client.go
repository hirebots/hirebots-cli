package main

// client.go — HTTP client for the HireBots.ai API with auto-refresh of expired tokens.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client wraps the HireBots API with authentication.
type Client struct {
	BaseURL      string
	Token        string
	RefreshToken string
	HTTP         *http.Client
}

// newClient creates an API client from flags or config file.
func newClient(baseURL, token string) *Client {
	if token == "" {
		token = loadToken()
	}
	refreshToken := loadRefreshToken()
	return &Client{
		BaseURL:      strings.TrimRight(baseURL, "/"),
		Token:        token,
		RefreshToken: refreshToken,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// configDir returns the hirebots config directory.
//
// Resolution order:
//  1. --config-dir flag (configDirOverride, set in main.go)
//  2. HIREBOTS_CONFIG_DIR environment variable
//  3. ~/.hirebots (default)
func configDir() string {
	if configDirOverride != "" {
		return configDirOverride
	}
	if dir := os.Getenv("HIREBOTS_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".hirebots")
}

// configFilePath returns the path to the config file.
func configFilePath() string {
	return filepath.Join(configDir(), "config.json")
}

// configData represents the on-disk config structure.
type configData struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	APIURL       string `json:"api_url"`
}

// loadConfig reads the full config from the config file (if present).
func loadConfig() configData {
	if t := os.Getenv("HIREBOTS_API_TOKEN"); t != "" {
		return configData{Token: t, APIURL: os.Getenv("HIREBOTS_API_URL")}
	}
	path := configFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return configData{}
	}
	var cfg configData
	if json.Unmarshal(data, &cfg) == nil {
		return cfg
	}
	return configData{}
}

// loadToken reads the API access token from config file or environment.
func loadToken() string {
	return loadConfig().Token
}

// loadRefreshToken reads the refresh token from the config file or environment.
func loadRefreshToken() string {
	return loadConfig().RefreshToken
}

// saveToken persists the API access token to the config file (legacy single-token).
func saveToken(token, apiURL string) error {
	return saveTokens(token, "", apiURL)
}

// saveTokens persists both the access token and refresh token to the config file.
// The existing api_url is preserved if apiURL is empty.
func saveTokens(accessToken, refreshToken, apiURL string) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Preserve existing api_url if not provided.
	if apiURL == "" {
		apiURL = loadConfig().APIURL
	}

	cfg := configData{
		Token:        accessToken,
		RefreshToken: refreshToken,
		APIURL:       apiURL,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	path := configFilePath()
	return os.WriteFile(path, data, 0600)
}

// refreshTokenPair calls POST /auth/refresh, saves the new token pair, and
// updates the client's in-memory tokens.
func (c *Client) refreshTokenPair() error {
	if c.RefreshToken == "" {
		return fmt.Errorf("no refresh token available — please login again")
	}
	body, _, err := c.do("POST", "/auth/refresh", map[string]string{
		"refresh_token": c.RefreshToken,
	})
	if err != nil {
		return fmt.Errorf("refreshing token: %w", err)
	}
	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parsing refresh response: %w", err)
	}
	if resp.AccessToken == "" {
		return fmt.Errorf("no access_token in refresh response")
	}

	// Update in-memory state.
	c.Token = resp.AccessToken
	if resp.RefreshToken != "" {
		c.RefreshToken = resp.RefreshToken
	}

	// Persist to disk.
	if err := saveTokens(resp.AccessToken, c.RefreshToken, c.BaseURL); err != nil {
		// Non-fatal — tokens are valid in-memory for this session.
		fmt.Fprintf(os.Stderr, "warning: could not save refreshed tokens: %v\n", err)
	}
	return nil
}

// do performs an authenticated HTTP request and returns the response body.
// On a 401 response, it automatically attempts a token refresh and retries once.
func (c *Client) do(method, path string, body interface{}) ([]byte, int, error) {
	respBody, statusCode, err := c.doOnce(method, path, body)
	if err == nil {
		return respBody, statusCode, nil
	}

	// If we got a 401 and have a refresh token, try refreshing and retrying once.
	if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == 401 && c.RefreshToken != "" && path != "/auth/refresh" {
		refreshErr := c.refreshTokenPair()
		if refreshErr != nil {
			return respBody, statusCode, fmt.Errorf("auto-refresh failed: %w", refreshErr)
		}
		// Retry the original request with the new token.
		return c.doOnce(method, path, body)
	}

	return respBody, statusCode, err
}

// doOnce performs a single HTTP request without any retry logic.
func (c *Client) doOnce(method, path string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	url := c.BaseURL + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, &APIError{
			StatusCode: resp.StatusCode,
			Body:       respBody,
		}
	}

	return respBody, resp.StatusCode, nil
}

// APIError represents a non-2xx API response.
type APIError struct {
	StatusCode int
	Body       []byte
}

func (e *APIError) Error() string {
	var errResp struct {
		Error  string `json:"error"`
		Detail string `json:"detail"`
	}
	if json.Unmarshal(e.Body, &errResp) == nil && errResp.Detail != "" {
		return fmt.Sprintf("API error %d: %s — %s", e.StatusCode, errResp.Error, errResp.Detail)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, string(e.Body))
}

// get performs a GET request.
func (c *Client) get(path string) ([]byte, error) {
	body, _, err := c.do("GET", path, nil)
	return body, err
}

// post performs a POST request.
func (c *Client) post(path string, payload interface{}) ([]byte, error) {
	body, _, err := c.do("POST", path, payload)
	return body, err
}

// patch performs a PATCH request.
func (c *Client) patch(path string, payload interface{}) ([]byte, error) {
	body, _, err := c.do("PATCH", path, payload)
	return body, err
}

// deleteRequest performs a DELETE request.
func (c *Client) deleteRequest(path string) ([]byte, error) {
	body, _, err := c.do("DELETE", path, nil)
	return body, err
}

// jsonUnmarshal is a thin wrapper to avoid importing json in every file.
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// prettyPrint prints JSON with indentation.
func prettyPrint(data []byte) {
	var buf bytes.Buffer
	if json.Indent(&buf, data, "", "  ") == nil {
		fmt.Println(buf.String())
		return
	}
	// Not JSON — print raw
	fmt.Println(string(data))
}
