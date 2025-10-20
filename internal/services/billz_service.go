package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// Package-level token cache guarded by a mutex to allow safe reuse across requests.
var (
	billzToken       string
	billzTokenExpiry time.Time
	billzTokenMu     sync.RWMutex
	httpClient       = &http.Client{Timeout: 15 * time.Second}
)

const (
	defaultBillzAuthURL = "https://api-admin.billz.ai/v1/auth/login"
	defaultBillzBaseURL = "https://api-admin.billz.ai/v2"
	tokenRefreshLeeway  = 30 * time.Second
)

type billzAuthRequest struct {
	SecretToken string `json:"secret_token"`
}

type billzAuthResponse struct {
	Data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	} `json:"data"`
	Error any `json:"error,omitempty"`
}

// BillzRequestOpts captures inputs for Billz API calls.
type BillzRequestOpts struct {
	Method  string
	Path    string
	Query   map[string]string
	Body    any
	Headers map[string]string
	Token   string
}

// BillzResponse bundles the HTTP response metadata.
type BillzResponse struct {
	Status int
	Body   []byte
	Header http.Header
}

// BillzBaseURL exposes the configured Billz API base URL for other packages.
func BillzBaseURL() string {
	baseURL := strings.TrimSpace(os.Getenv("BILLZ_URL"))
	if baseURL == "" {
		return defaultBillzBaseURL
	}
	return strings.TrimRight(baseURL, "/")
}

// GetBillzToken returns a cached Billz access token, fetching a new one if needed.
func GetBillzToken() (string, error) {
	return getBillzToken(false)
}

// RefreshBillzToken forces retrieval of a fresh Billz access token.
func RefreshBillzToken() (string, error) {
	return getBillzToken(true)
}

func getBillzToken(force bool) (string, error) {
	if !force {
		if token, ok := cachedToken(); ok {
			return token, nil
		}
	}

	billzTokenMu.Lock()
	defer billzTokenMu.Unlock()

	// Check again in case another goroutine refreshed while we waited for the lock.
	if !force {
		if token := currentTokenLocked(); token != "" {
			return token, nil
		}
	}

	authURL := strings.TrimSpace(os.Getenv("BILLZ_AUTH_URL"))
	if authURL == "" {
		authURL = defaultBillzAuthURL
	}
	authURL = strings.TrimRight(authURL, "/")

	secret := strings.TrimSpace(os.Getenv("BILLZ_API_SECRET_KEY"))
	if secret == "" {
		return "", errors.New("BILLZ_API_SECRET_KEY is not configured")
	}

	payload := billzAuthRequest{
		SecretToken: secret,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal Billz auth payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, authURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create Billz auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute Billz auth request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read Billz auth response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("Billz auth request failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var authResp billzAuthResponse
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return "", fmt.Errorf("unmarshal Billz auth response: %w", err)
	}

	if authResp.Data.AccessToken == "" {
		return "", errors.New("Billz auth response missing access_token")
	}

	billzToken = authResp.Data.AccessToken
	if authResp.Data.ExpiresIn > 0 {
		billzTokenExpiry = time.Now().Add(time.Duration(authResp.Data.ExpiresIn) * time.Second)
	} else {
		// Fallback to a short lifetime when expiry is not provided.
		billzTokenExpiry = time.Now().Add(5 * time.Minute)
	}

	return billzToken, nil
}

func cachedToken() (string, bool) {
	billzTokenMu.RLock()
	defer billzTokenMu.RUnlock()

	token := currentTokenLocked()
	if token == "" {
		return "", false
	}
	return token, true
}

func currentTokenLocked() string {
	if billzToken == "" {
		return ""
	}
	if billzTokenExpiry.IsZero() {
		return billzToken
	}
	if time.Now().Add(tokenRefreshLeeway).After(billzTokenExpiry) {
		return ""
	}
	return billzToken
}

// DoBillzRequest performs a generic Billz API request, retrying once on 401.
func DoBillzRequest(opts BillzRequestOpts) (*BillzResponse, error) {
	if opts.Method == "" {
		return nil, errors.New("request method is required")
	}
	path := strings.TrimLeft(opts.Path, "/")
	if path == "" {
		return nil, errors.New("request path is required")
	}

	makeURL := func() (string, error) {
		base := BillzBaseURL()
		u, err := url.Parse(base)
		if err != nil {
			return "", fmt.Errorf("parse Billz base URL: %w", err)
		}

		finalPath := strings.TrimLeft(path, "/")
		versionSeg, remainder, hasVersion := splitVersionSegment(finalPath)

		basePath := strings.TrimRight(u.Path, "/")
		baseSegments := splitPathSegments(basePath)

		if hasVersion {
			if len(baseSegments) > 0 && isVersionSegment(baseSegments[len(baseSegments)-1]) {
				baseSegments = baseSegments[:len(baseSegments)-1]
			}
			if versionSeg != "" {
				baseSegments = append(baseSegments, versionSeg)
			}
			baseSegments = append(baseSegments, splitPathSegments(remainder)...)
		} else {
			baseSegments = append(baseSegments, splitPathSegments(finalPath)...)
		}

		u.Path = "/" + strings.Join(filterEmpty(baseSegments), "/")
		if len(opts.Query) > 0 {
			values := u.Query()
			for k, v := range opts.Query {
				values.Set(k, v)
			}
			u.RawQuery = values.Encode()
		}
		return u.String(), nil
	}

	buildRequest := func(token string) (*http.Request, error) {
		targetURL, err := makeURL()
		if err != nil {
			return nil, err
		}

		var bodyReader io.Reader
		if opts.Body != nil {
			payload, err := json.Marshal(opts.Body)
			if err != nil {
				return nil, fmt.Errorf("marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(payload)
		}

		req, err := http.NewRequest(opts.Method, targetURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		for k, v := range opts.Headers {
			req.Header.Set(k, v)
		}

		if opts.Body != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		return req, nil
	}

	token := opts.Token
	if token == "" {
		var err error
		token, err = GetBillzToken()
		if err != nil {
			return nil, err
		}
	}

	do := func(req *http.Request) (*BillzResponse, error) {
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("execute request: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}

		return &BillzResponse{
			Status: resp.StatusCode,
			Body:   respBody,
			Header: resp.Header.Clone(),
		}, nil
	}

	req, err := buildRequest(token)
	if err != nil {
		return nil, err
	}

	resp, err := do(req)
	if err != nil {
		return nil, err
	}

	if resp.Status != http.StatusUnauthorized || opts.Token != "" {
		return resp, nil
	}

	// Token likely expired; refresh and retry once.
	token, err = RefreshBillzToken()
	if err != nil {
		return nil, err
	}

	req, err = buildRequest(token)
	if err != nil {
		return nil, err
	}

	return do(req)
}

func isVersionSegment(seg string) bool {
	if len(seg) < 2 || seg[0] != 'v' {
		return false
	}
	for _, r := range seg[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func splitVersionSegment(p string) (string, string, bool) {
	p = strings.TrimLeft(p, "/")
	if p == "" {
		return "", "", false
	}

	parts := strings.SplitN(p, "/", 2)
	if isVersionSegment(parts[0]) {
		if len(parts) > 1 {
			return parts[0], parts[1], true
		}
		return parts[0], "", true
	}

	return "", p, false
}

func splitPathSegments(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.FieldsFunc(p, func(r rune) bool { return r == '/' })
}

func filterEmpty(values []string) []string {
	if len(values) == 0 {
		return values
	}
	result := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}
