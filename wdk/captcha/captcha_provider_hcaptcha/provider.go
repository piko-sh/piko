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

package captcha_provider_hcaptcha

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
	"piko.sh/piko/wdk/captcha/captcha_provider_hcaptcha/scripts"
)

const (
	// verifyURL is the hCaptcha server-side verification endpoint.
	verifyURL = "https://api.hcaptcha.com/siteverify"

	// httpTimeout is the timeout for HTTP requests to the hCaptcha API.
	httpTimeout = 10 * time.Second

	// maxResponseBodySize is the maximum number of bytes read from the hCaptcha
	// verification response. Prevents unbounded memory allocation from a
	// misbehaving upstream.
	maxResponseBodySize = 64 * 1024
)

// hcaptchaVerifyResult represents the JSON response from the hCaptcha siteverify
// API endpoint.
type hcaptchaVerifyResult struct {
	// Score is the Enterprise risk score where 0.0 = no risk and 1.0 =
	// confirmed threat; nil distinguishes "not present" from zero, and
	// the value is inverted compared to the normalised convention.
	Score *float64 `json:"score"`

	// ChallengeTimestamp is the ISO 8601 timestamp of the challenge.
	ChallengeTimestamp string `json:"challenge_ts"`

	// Hostname is the hostname of the site where the challenge was solved.
	Hostname string `json:"hostname"`

	// ErrorCodes contains error codes when verification fails.
	ErrorCodes []string `json:"error-codes"`

	// ScoreReason contains human-readable reasons for the score (Enterprise).
	ScoreReason []string `json:"score_reason"`

	// Success indicates whether the token was valid.
	Success bool `json:"success"`
}

// provider implements captcha_domain.CaptchaProvider using hCaptcha for bot
// detection.
type provider struct {
	// httpClient is the HTTP client used for calls to the hCaptcha API.
	httpClient *http.Client

	// config holds the hCaptcha site key and secret key.
	config Config
}

var _ captcha_domain.CaptchaProvider = (*provider)(nil)

// NewProvider creates a new hCaptcha captcha provider.
//
// Takes config (Config) which specifies the hCaptcha site key and secret key.
//
// Returns captcha_domain.CaptchaProvider which provides hCaptcha-based captcha
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
// Returns captcha_dto.ProviderType which identifies this as an hCaptcha
// provider.
func (*provider) Type() captcha_dto.ProviderType {
	return captcha_dto.ProviderTypeHCaptcha
}

// SiteKey returns the public site key for the hCaptcha widget.
//
// Returns string which is the hCaptcha site key.
func (p *provider) SiteKey() string {
	return p.config.SiteKey
}

// ScriptURL returns the URL of the hCaptcha JavaScript SDK.
//
// Returns string which is the hCaptcha script URL.
func (*provider) ScriptURL() string {
	return "https://js.hcaptcha.com/1/api.js"
}

// Verify verifies an hCaptcha token by calling the hCaptcha siteverify API.
//
// The method POSTs the token, secret key, client IP, and site key to the
// hCaptcha verification endpoint and parses the JSON response.
//
// Takes request (*captcha_dto.VerifyRequest) which contains the captcha token,
// client IP, and action name.
//
// Returns *captcha_dto.VerifyResponse which contains the verification result.
// Returns error when the HTTP request fails or the response cannot be parsed.
func (p *provider) Verify(ctx context.Context, request *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
	ctx, span := tracer.Start(ctx, "captcha_provider_hcaptcha.Verify")
	defer span.End()

	if request == nil || request.Token == "" {
		return &captcha_dto.VerifyResponse{
			Success:    false,
			ErrorCodes: []string{"missing-input-response"},
		}, nil
	}

	hcaptchaResult, err := p.callVerifyAPI(ctx, request)
	if err != nil {
		return nil, err
	}

	return &captcha_dto.VerifyResponse{
		Score:      new(normaliseScore(hcaptchaResult.Success, hcaptchaResult.Score)),
		Success:    hcaptchaResult.Success,
		ErrorCodes: hcaptchaResult.ErrorCodes,
		Hostname:   hcaptchaResult.Hostname,
		Timestamp:  parseChallengeTimestamp(hcaptchaResult.ChallengeTimestamp),
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
// hCaptcha widget.
//
// Returns *captcha_dto.RenderRequirements which describes the script tags, CSP
// domains, container HTML, and init script needed to render the widget.
func (*provider) RenderRequirements() *captcha_dto.RenderRequirements {
	return &captcha_dto.RenderRequirements{
		InitScript:        scripts.InitScript,
		ScriptURLs:        []string{"https://js.hcaptcha.com/1/api.js?onload=__pikoCaptchaHcaptchaReady&render=explicit"},
		CSPScriptDomains:  []string{"https://js.hcaptcha.com"},
		CSPFrameDomains:   []string{"https://hcaptcha.com", "https://*.hcaptcha.com"},
		CSPConnectDomains: []string{"https://hcaptcha.com", "https://*.hcaptcha.com"},
		ProviderType:      "hcaptcha",
	}
}

// HealthCheck returns nil because hCaptcha does not provide a dedicated health
// check endpoint. Connectivity is verified implicitly during token verification.
//
// Returns error which is always nil for this provider.
func (*provider) HealthCheck(_ context.Context) error {
	return nil
}

// callVerifyAPI sends the verification request to the hCaptcha API and parses
// the response.
//
// Takes request (*captcha_dto.VerifyRequest) which contains the token, client
// IP, and site key for verification.
//
// Returns *hcaptchaVerifyResult which is the raw API response.
// Returns error when the HTTP request fails, the response is invalid, or the
// body exceeds size limits.
func (p *provider) callVerifyAPI(ctx context.Context, request *captcha_dto.VerifyRequest) (*hcaptchaVerifyResult, error) {
	formData := url.Values{
		"secret":   {p.config.SecretKey},
		"response": {request.Token},
		"sitekey":  {p.config.SiteKey},
	}
	if request.RemoteIP != "" {
		formData.Set("remoteip", request.RemoteIP)
	}

	encodedForm := formData.Encode()
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, strings.NewReader(encodedForm))
	if err != nil {
		return nil, fmt.Errorf("creating hcaptcha request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResponse, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("sending hcaptcha verification request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(httpResponse.Body, maxResponseBodySize))
		_ = httpResponse.Body.Close()
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hcaptcha verification returned HTTP %d: %w", httpResponse.StatusCode, captcha_dto.ErrProviderUnavailable)
	}

	contentType := httpResponse.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return nil, fmt.Errorf("hcaptcha returned unexpected content type %q: %w",
			contentType, captcha_dto.ErrProviderUnavailable)
	}

	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading hcaptcha response body: %w", err)
	}

	if int64(len(body)) >= maxResponseBodySize {
		return nil, fmt.Errorf("hcaptcha response body exceeded %d byte limit: %w",
			maxResponseBodySize, captcha_dto.ErrProviderUnavailable)
	}

	var result hcaptchaVerifyResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing hcaptcha response: %w", err)
	}

	return &result, nil
}

// normaliseScore converts the hCaptcha score to the normalised convention
// where 0.0 = bot and 1.0 = human, inverting the Enterprise score when
// present (hCaptcha uses 0.0 = safe, 1.0 = threat) and falling back to
// 1.0 for success or 0.0 for failure when absent.
//
// Takes success (bool) which indicates whether the verification passed.
// Takes enterpriseScore (*float64) which is the optional Enterprise risk
// score.
//
// Returns float64 which is the normalised score.
func normaliseScore(success bool, enterpriseScore *float64) float64 {
	if enterpriseScore != nil {
		return max(0.0, min(1.0, 1.0-*enterpriseScore))
	}
	if success {
		return 1.0
	}
	return 0.0
}
