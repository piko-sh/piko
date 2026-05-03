// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package captcha_provider_turnstile

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/json"
	"piko.sh/piko/wdk/captcha/captcha_provider_turnstile/scripts"
)

const (
	// verifyURL is the Cloudflare Turnstile server-side verification endpoint.
	verifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

	// httpTimeout is the timeout for HTTP requests to the Turnstile API.
	httpTimeout = 10 * time.Second

	// maxResponseBodySize is the maximum number of bytes read from the Turnstile
	// verification response. Prevents unbounded memory allocation from a
	// misbehaving upstream.
	maxResponseBodySize = 64 * 1024
)

// turnstileVerifyResult represents the JSON response from the Cloudflare Turnstile
// siteverify endpoint.
type turnstileVerifyResult struct {
	// ChallengeTimestamp is the ISO 8601 timestamp of when the challenge was solved.
	ChallengeTimestamp string `json:"challenge_ts"`

	// Hostname is the hostname the token was issued for.
	Hostname string `json:"hostname"`

	// Action is the action identifier passed from the client widget, if set.
	Action string `json:"action"`

	// ErrorCodes contains Turnstile-specific error codes when verification fails.
	ErrorCodes []string `json:"error-codes"`

	// Success indicates whether the token verification passed.
	Success bool `json:"success"`
}

// provider implements captcha_domain.CaptchaProvider using Cloudflare Turnstile.
type provider struct {
	// httpClient is the HTTP client used for calls to the Turnstile API.
	httpClient *http.Client

	// config holds the Turnstile site key and secret key.
	config Config
}

var _ captcha_domain.CaptchaProvider = (*provider)(nil)

// NewProvider creates a new Cloudflare Turnstile captcha provider.
//
// Takes config (Config) which specifies the Turnstile site key and secret key.
//
// Returns captcha_domain.CaptchaProvider which provides Turnstile-based captcha
// verification.
// Returns error when the configuration is invalid.
func NewProvider(config Config) (captcha_domain.CaptchaProvider, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	return &provider{
		config: config,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}, nil
}

// Type returns the provider type identifier.
//
// Returns captcha_dto.ProviderType which identifies this as a Turnstile
// provider.
func (*provider) Type() captcha_dto.ProviderType {
	return captcha_dto.ProviderTypeTurnstile
}

// SiteKey returns the public site key for the Turnstile widget.
//
// Returns string which is the Turnstile site key.
func (p *provider) SiteKey() string {
	return p.config.SiteKey
}

// ScriptURL returns the Cloudflare Turnstile JavaScript SDK URL.
//
// Returns string which is the Turnstile script URL.
func (*provider) ScriptURL() string {
	return "https://challenges.cloudflare.com/turnstile/v0/api.js"
}

// Verify verifies a Turnstile token by calling the Cloudflare siteverify
// endpoint.
//
// Takes request (*captcha_dto.VerifyRequest) which contains the token, client
// IP, and action name.
//
// Returns *captcha_dto.VerifyResponse which contains the verification result.
// Returns error when the HTTP request fails or the response cannot be parsed.
func (p *provider) Verify(ctx context.Context, request *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
	ctx, span := tracer.Start(ctx, "captcha_provider_turnstile.Verify")
	defer span.End()

	if request == nil || request.Token == "" {
		return &captcha_dto.VerifyResponse{
			Success:    false,
			ErrorCodes: []string{"missing-input-response"},
		}, nil
	}

	turnstileResult, err := p.callVerifyAPI(ctx, request)
	if err != nil {
		return nil, err
	}

	return &captcha_dto.VerifyResponse{
		Score:      new(normalisedPassFailScore(turnstileResult.Success)),
		Success:    turnstileResult.Success,
		Action:     turnstileResult.Action,
		ErrorCodes: turnstileResult.ErrorCodes,
		Hostname:   turnstileResult.Hostname,
		Timestamp:  parseChallengeTimestamp(turnstileResult.ChallengeTimestamp),
	}, nil
}

// parseChallengeTimestamp parses an RFC 3339 timestamp string, returning the
// zero time if the input is empty or malformed.
//
// Takes raw (string) which is the RFC 3339 timestamp to parse.
//
// Returns time.Time which is the parsed time, or the zero value on failure.
func parseChallengeTimestamp(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, raw)
	return t
}

// RenderRequirements returns the frontend rendering configuration for the
// Cloudflare Turnstile widget.
//
// Returns *captcha_dto.RenderRequirements which describes the script tags, CSP
// domains, container HTML, and init script needed to render the widget.
func (*provider) RenderRequirements() *captcha_dto.RenderRequirements {
	return &captcha_dto.RenderRequirements{
		InitScript:        scripts.InitScript,
		ScriptURLs:        []string{"https://challenges.cloudflare.com/turnstile/v0/api.js?onload=__pikoCaptchaTurnstileReady&render=explicit"},
		CSPScriptDomains:  []string{"https://challenges.cloudflare.com"},
		CSPFrameDomains:   []string{"https://challenges.cloudflare.com"},
		CSPConnectDomains: []string{"https://challenges.cloudflare.com"},
		ProviderType:      "turnstile",
	}
}

// HealthCheck returns nil because Cloudflare Turnstile does not provide a
// test verification endpoint.
//
// Returns error which is always nil for this provider.
func (*provider) HealthCheck(_ context.Context) error {
	return nil
}

// callVerifyAPI sends the verification request to the Turnstile API and parses
// the response.
//
// Takes request (*captcha_dto.VerifyRequest) which contains the token and
// client IP for verification.
//
// Returns *turnstileVerifyResult which is the raw API response.
// Returns error when the HTTP request fails, the response is invalid, or the
// body exceeds size limits.
func (p *provider) callVerifyAPI(ctx context.Context, request *captcha_dto.VerifyRequest) (*turnstileVerifyResult, error) {
	formData := url.Values{
		"secret":   {p.config.SecretKey},
		"response": {request.Token},
	}
	if request.RemoteIP != "" {
		formData.Set("remoteip", request.RemoteIP)
	}

	encodedForm := formData.Encode()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(encodedForm))
	if err != nil {
		return nil, fmt.Errorf("creating turnstile request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResponse, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("sending turnstile verification request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(httpResponse.Body, maxResponseBodySize))
		_ = httpResponse.Body.Close()
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("turnstile verification returned HTTP %d: %w", httpResponse.StatusCode, captcha_dto.ErrProviderUnavailable)
	}

	contentType := httpResponse.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return nil, fmt.Errorf("turnstile returned unexpected content type %q: %w",
			contentType, captcha_dto.ErrProviderUnavailable)
	}

	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading turnstile response body: %w", err)
	}

	if int64(len(body)) >= maxResponseBodySize {
		return nil, fmt.Errorf("turnstile response body exceeded %d byte limit: %w",
			maxResponseBodySize, captcha_dto.ErrProviderUnavailable)
	}

	var result turnstileVerifyResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing turnstile response: %w", err)
	}

	return &result, nil
}

// normalisedPassFailScore returns 1.0 for success and 0.0 for failure,
// providing a consistent score for pass/fail providers.
//
// Takes success (bool) which indicates whether the verification passed.
//
// Returns float64 which is 1.0 for success or 0.0 for failure.
func normalisedPassFailScore(success bool) float64 {
	if success {
		return 1.0
	}
	return 0.0
}
