package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/refyne/refyne/internal/logger"
	"github.com/refyne/refyne/pkg/fetcher"
)

// ErrFlareSolverrUnavailable indicates FlareSolverr service is not reachable.
var ErrFlareSolverrUnavailable = errors.New("FlareSolverr service unavailable")

// FlareSolverr is a client for the FlareSolverr API.
// FlareSolverr is a proxy server that solves Cloudflare challenges.
type FlareSolverr struct {
	baseURL    string
	httpClient *http.Client
	maxTimeout int // milliseconds
}

// FlareSolverRequest is the request body for FlareSolverr API.
type FlareSolverRequest struct {
	Cmd        string `json:"cmd"`
	URL        string `json:"url,omitempty"`
	Session    string `json:"session,omitempty"`
	MaxTimeout int    `json:"maxTimeout,omitempty"`
}

// FlareSolverResponse is the response from FlareSolverr API.
type FlareSolverResponse struct {
	Status   string                  `json:"status"`
	Message  string                  `json:"message"`
	Solution *FlareSolverSolution    `json:"solution,omitempty"`
	StartTS  float64                 `json:"startTimestamp"`
	EndTS    float64                 `json:"endTimestamp"`
	Version  string                  `json:"version"`
}

// FlareSolverSolution contains the solution data from FlareSolverr.
type FlareSolverSolution struct {
	URL       string                 `json:"url"`
	Status    int                    `json:"status"`
	Headers   map[string]string      `json:"headers"`
	Response  string                 `json:"response"`
	Cookies   []FlareSolverCookie    `json:"cookies"`
	UserAgent string                 `json:"userAgent"`
}

// FlareSolverCookie represents a cookie returned by FlareSolverr.
type FlareSolverCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	Size     int     `json:"size"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	Session  bool    `json:"session"`
	SameSite string  `json:"sameSite"`
}

// FlareSolverSessionResponse is the response for session operations.
type FlareSolverSessionResponse struct {
	Status   string   `json:"status"`
	Message  string   `json:"message"`
	Session  string   `json:"session,omitempty"`
	Sessions []string `json:"sessions,omitempty"`
}

// NewFlareSolverr creates a new FlareSolverr client.
func NewFlareSolverr(baseURL string) *FlareSolverr {
	return &FlareSolverr{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // FlareSolverr can take a while
		},
		maxTimeout: 60000, // 60 seconds default
	}
}

// Solve sends a URL to FlareSolverr and returns the solution.
// If sessionID is provided, it uses an existing session (persistent browser instance).
func (f *FlareSolverr) Solve(ctx context.Context, targetURL string, sessionID string) (*FlareSolverSolution, error) {
	reqBody := FlareSolverRequest{
		Cmd:        "request.get",
		URL:        targetURL,
		Session:    sessionID,
		MaxTimeout: f.maxTimeout,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal FlareSolverr request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create FlareSolverr request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		logger.Warn("FlareSolverr request failed", "url", targetURL, "error", err)
		return nil, fmt.Errorf("%w: %v", ErrFlareSolverrUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read FlareSolverr response: %w", err)
	}

	// Try to parse JSON response even on non-200 status (FlareSolverr returns 500 with JSON body on errors)
	var fsResp FlareSolverResponse
	if err := json.Unmarshal(body, &fsResp); err != nil {
		logger.Warn("FlareSolverr returned invalid response", "status_code", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("failed to parse FlareSolverr response: %w", err)
	}

	if fsResp.Status != "ok" {
		logger.Debug("FlareSolverr returned error status",
			"url", targetURL,
			"status", fsResp.Status,
			"message", fsResp.Message)
		return nil, f.classifyError(targetURL, fsResp.Message)
	}

	if fsResp.Solution == nil {
		logger.Warn("FlareSolverr returned no solution", "url", targetURL)
		return nil, fmt.Errorf("%w: no solution returned", fetcher.ErrAntiBot)
	}

	// Log solution details
	responseSize := len(fsResp.Solution.Response)
	duration := (fsResp.EndTS - fsResp.StartTS) / 1000 // Convert to seconds
	logger.Debug("FlareSolverr solved",
		"url", targetURL,
		"session", sessionID,
		"status_code", fsResp.Solution.Status,
		"cookies", len(fsResp.Solution.Cookies),
		"response_size", responseSize,
		"duration_s", fmt.Sprintf("%.2f", duration))

	return fsResp.Solution, nil
}

// CreateSession creates a new persistent browser session in FlareSolverr.
// Sessions keep the same browser instance running, avoiding repeated challenge solving.
func (f *FlareSolverr) CreateSession(ctx context.Context, sessionID string) error {
	reqBody := FlareSolverRequest{
		Cmd:     "sessions.create",
		Session: sessionID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal session create request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create session request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFlareSolverrUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read session response: %w", err)
	}

	var sessResp FlareSolverSessionResponse
	if err := json.Unmarshal(body, &sessResp); err != nil {
		return fmt.Errorf("failed to parse session response: %w", err)
	}

	if sessResp.Status != "ok" {
		return fmt.Errorf("session create failed: %s", sessResp.Message)
	}

	logger.Debug("FlareSolverr session created", "session", sessionID)
	return nil
}

// DestroySession destroys a persistent browser session in FlareSolverr.
func (f *FlareSolverr) DestroySession(ctx context.Context, sessionID string) error {
	reqBody := FlareSolverRequest{
		Cmd:     "sessions.destroy",
		Session: sessionID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal session destroy request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, f.baseURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create session destroy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		// Don't fail hard on cleanup errors
		logger.Debug("FlareSolverr session destroy failed", "session", sessionID, "error", err)
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Debug("failed to read session destroy response", "session", sessionID, "error", err)
		return nil
	}

	var sessResp FlareSolverSessionResponse
	if err := json.Unmarshal(body, &sessResp); err != nil {
		logger.Debug("failed to parse session destroy response", "session", sessionID, "error", err)
		return nil
	}

	if sessResp.Status == "ok" {
		logger.Debug("FlareSolverr session destroyed", "session", sessionID)
	} else {
		logger.Debug("FlareSolverr session destroy returned error", "session", sessionID, "message", sessResp.Message)
	}
	return nil
}

// classifyError parses FlareSolverr error messages and returns an appropriate typed error.
func (f *FlareSolverr) classifyError(url, message string) error {
	msgLower := strings.ToLower(message)

	// Detect timeout errors (challenge couldn't be solved in time)
	if strings.Contains(msgLower, "timeout") ||
		strings.Contains(msgLower, "timed out") {
		logger.Warn("FlareSolverr timed out", "url", url, "message", message)
		return fmt.Errorf("%w: %s", fetcher.ErrChallengeTimeout, message)
	}

	// Detect unsolvable challenge (captcha that requires human)
	if strings.Contains(msgLower, "could not be solved") ||
		strings.Contains(msgLower, "unable to solve") ||
		strings.Contains(msgLower, "failed to solve") {
		logger.Warn("FlareSolverr could not solve challenge", "url", url, "message", message)
		return fmt.Errorf("%w: %s", fetcher.ErrCaptchaChallenge, message)
	}

	// Detect CAPTCHA-related failures
	if strings.Contains(msgLower, "captcha") ||
		strings.Contains(msgLower, "turnstile") ||
		strings.Contains(msgLower, "hcaptcha") ||
		strings.Contains(msgLower, "recaptcha") {
		logger.Warn("FlareSolverr detected unsolvable CAPTCHA", "url", url, "message", message)
		return fmt.Errorf("%w: %s", fetcher.ErrCaptchaChallenge, message)
	}

	// Detect Cloudflare-specific challenges
	if strings.Contains(msgLower, "cloudflare") ||
		strings.Contains(msgLower, "cf-") ||
		strings.Contains(msgLower, "challenge") {
		logger.Warn("FlareSolverr Cloudflare challenge failed", "url", url, "message", message)
		return fmt.Errorf("%w: %s", fetcher.ErrCaptchaChallenge, message)
	}

	// Detect blocked/access denied
	if strings.Contains(msgLower, "blocked") ||
		strings.Contains(msgLower, "denied") ||
		strings.Contains(msgLower, "forbidden") ||
		strings.Contains(msgLower, "access") ||
		strings.Contains(msgLower, "403") {
		logger.Warn("FlareSolverr blocked by anti-bot", "url", url, "message", message)
		return fmt.Errorf("%w: %s", fetcher.ErrAntiBot, message)
	}

	// Detect browser/internal errors
	if strings.Contains(msgLower, "browser") ||
		strings.Contains(msgLower, "crashed") ||
		strings.Contains(msgLower, "unable to process") {
		logger.Warn("FlareSolverr browser error", "url", url, "message", message)
		return fmt.Errorf("FlareSolverr internal error: %s", message)
	}

	// Generic error - log full message for debugging
	logger.Warn("FlareSolverr failed with unknown error", "url", url, "message", message)
	return fmt.Errorf("%w: %s", fetcher.ErrAntiBot, message)
}

// ToCookies converts FlareSolverr cookies to the fetcher.Cookie type.
func (s *FlareSolverSolution) ToCookies() []fetcher.Cookie {
	cookies := make([]fetcher.Cookie, 0, len(s.Cookies))
	for _, c := range s.Cookies {
		cookies = append(cookies, fetcher.Cookie{
			Name:   c.Name,
			Value:  c.Value,
			Domain: c.Domain,
		})
	}
	return cookies
}
