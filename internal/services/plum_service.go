package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	plumToken       string
	plumTokenExpiry time.Time
	plumTokenMu     sync.RWMutex
	plumHTTPClient  = &http.Client{Timeout: 15 * time.Second}
)

// PlumConfig holds credentials loaded from environment variables.
type PlumConfig struct {
	BaseURL  string
	Username string
	Password string
	Enabled  bool
}

// LoadPlumConfig reads Plum configuration from environment.
func LoadPlumConfig() PlumConfig {
	return PlumConfig{
		BaseURL:  strings.TrimRight(getEnvOrDefault("PLUM_BASE_URL", "https://pay.myuzcard.uz/api"), "/"),
		Username: getEnvOrDefault("PLUM_USERNAME", ""),
		Password: getEnvOrDefault("PLUM_PASSWORD", ""),
		Enabled:  getEnvOrDefault("PLUM_ENABLED", "false") == "true",
	}
}

func getEnvOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

type plumAuthResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expires_in"`
}

// GetPlumToken returns a cached Plum token, fetching a new one if needed.
func GetPlumToken() (string, error) {
	return getPlumToken(false)
}

func getPlumToken(force bool) (string, error) {
	cfg := LoadPlumConfig()
	if !cfg.Enabled {
		return "", errors.New("plum integration is disabled")
	}

	if !force {
		plumTokenMu.RLock()
		if plumToken != "" && time.Now().Before(plumTokenExpiry) {
			t := plumToken
			plumTokenMu.RUnlock()
			return t, nil
		}
		plumTokenMu.RUnlock()
	}

	plumTokenMu.Lock()
	defer plumTokenMu.Unlock()

	// Double-check after acquiring write lock.
	if !force && plumToken != "" && time.Now().Before(plumTokenExpiry) {
		return plumToken, nil
	}

	payload, _ := json.Marshal(map[string]string{
		"username": cfg.Username,
		"password": cfg.Password,
	})

	req, err := http.NewRequest(http.MethodPost, cfg.BaseURL+"/auth/login", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("plum auth request build: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := plumHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("plum auth request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("plum auth failed: status %d, body: %s", resp.StatusCode, string(body))
	}

	var authResp plumAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return "", fmt.Errorf("plum auth unmarshal: %w", err)
	}

	if authResp.Token == "" {
		return "", errors.New("plum auth: empty token")
	}

	plumToken = authResp.Token
	if authResp.ExpiresIn > 0 {
		plumTokenExpiry = time.Now().Add(time.Duration(authResp.ExpiresIn)*time.Second - 30*time.Second)
	} else {
		plumTokenExpiry = time.Now().Add(55 * time.Minute)
	}

	return plumToken, nil
}

// PlumRequestOpts configures a Plum API call.
type PlumRequestOpts struct {
	Method string
	Path   string
	Body   any
}

// PlumResponse wraps the API response.
type PlumResponse struct {
	Status int
	Body   []byte
}

// DoPlumRequest performs a Plum API request with automatic token handling.
func DoPlumRequest(opts PlumRequestOpts) (*PlumResponse, error) {
	cfg := LoadPlumConfig()
	if !cfg.Enabled {
		return nil, errors.New("plum integration is disabled")
	}

	token, err := GetPlumToken()
	if err != nil {
		return nil, err
	}

	url := cfg.BaseURL + "/" + strings.TrimLeft(opts.Path, "/")

	var bodyReader io.Reader
	if opts.Body != nil {
		data, err := json.Marshal(opts.Body)
		if err != nil {
			return nil, fmt.Errorf("plum request marshal: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	method := opts.Method
	if method == "" {
		method = http.MethodPost
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("plum request build: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := plumHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("plum request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	// Retry once on 401.
	if resp.StatusCode == http.StatusUnauthorized {
		token, err = getPlumToken(true)
		if err != nil {
			return nil, err
		}

		if opts.Body != nil {
			data, _ := json.Marshal(opts.Body)
			bodyReader = bytes.NewReader(data)
		}

		req2, err := http.NewRequest(method, url, bodyReader)
		if err != nil {
			return nil, err
		}
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+token)

		resp2, err := plumHTTPClient.Do(req2)
		if err != nil {
			return nil, err
		}
		defer resp2.Body.Close()

		respBody, _ = io.ReadAll(resp2.Body)
		return &PlumResponse{Status: resp2.StatusCode, Body: respBody}, nil
	}

	return &PlumResponse{Status: resp.StatusCode, Body: respBody}, nil
}

// PlumSendSMS sends an SMS verification code via Plum.
func PlumSendSMS(phone, message string) error {
	resp, err := DoPlumRequest(PlumRequestOpts{
		Method: http.MethodPost,
		Path:   "sms/send",
		Body: map[string]string{
			"phone":   phone,
			"message": message,
		},
	})
	if err != nil {
		return fmt.Errorf("plum send sms: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return fmt.Errorf("plum send sms: status %d, body: %s", resp.Status, string(resp.Body))
	}
	return nil
}

// PlumVerifyPhone initiates phone verification via Plum.
func PlumVerifyPhone(phone string) (string, error) {
	resp, err := DoPlumRequest(PlumRequestOpts{
		Method: http.MethodPost,
		Path:   "verification/send",
		Body: map[string]string{
			"phone": phone,
		},
	})
	if err != nil {
		return "", fmt.Errorf("plum verify phone: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return "", fmt.Errorf("plum verify phone: status %d, body: %s", resp.Status, string(resp.Body))
	}

	var result struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", fmt.Errorf("plum verify phone unmarshal: %w", err)
	}
	return result.SessionID, nil
}

// PlumConfirmCode confirms a verification code via Plum.
func PlumConfirmCode(sessionID, code string) (bool, error) {
	resp, err := DoPlumRequest(PlumRequestOpts{
		Method: http.MethodPost,
		Path:   "verification/confirm",
		Body: map[string]string{
			"session_id": sessionID,
			"code":       code,
		},
	})
	if err != nil {
		return false, fmt.Errorf("plum confirm code: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return false, fmt.Errorf("plum confirm code: status %d, body: %s", resp.Status, string(resp.Body))
	}

	var result struct {
		Verified bool `json:"verified"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return false, fmt.Errorf("plum confirm code unmarshal: %w", err)
	}
	return result.Verified, nil
}
