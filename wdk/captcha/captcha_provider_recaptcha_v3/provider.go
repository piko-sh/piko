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

package captcha_provider_recaptcha_v3

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
	"piko.sh/piko/wdk/captcha/captcha_provider_recaptcha_v3/scripts"
)

const (
	// verifyURL is the Google reCAPTCHA server-side verification endpoint.
	verifyURL = "https://www.google.com/recaptcha/api/siteverify"

	// httpTimeout is the timeout for HTTP requests to the reCAPTCHA API.
	httpTimeout = 10 * time.Second

	// maxResponseBodySize is the maximum number of bytes read from the reCAPTCHA
	// verification response. Prevents unbounded memory allocation from a
	// misbehaving upstream.
	maxResponseBodySize = 64 * 1024
)

// recaptchaVerifyResult represents the JSON response from the Google reCAPTCHA v3
// siteverify endpoint.
type recaptchaVerifyResult struct {
	// Action is the action name that was passed when the token was generated.
	Action string `json:"action"`

	// ChallengeTimestamp is the ISO 8601 timestamp of when the challenge was solved.
	ChallengeTimestamp string `json:"challenge_ts"`

	// Hostname is the hostname of the site where the challenge was solved.
	Hostname string `json:"hostname"`

	// ErrorCodes contains reCAPTCHA-specific error codes when verification fails.
	ErrorCodes []string `json:"error-codes"`

	// Score is the risk score between 0.0 (likely bot) and 1.0 (likely human).
	Score float64 `json:"score"`

	// Success indicates whether the token verification passed.
	Success bool `json:"success"`
}

// provider implements captcha_domain.CaptchaProvider using Google reCAPTCHA v3.
type provider struct {
	// httpClient is the HTTP client used for calls to the reCAPTCHA API.
	httpClient *http.Client

	// config holds the validated reCAPTCHA v3 configuration.
	config Config
}

var _ captcha_domain.CaptchaProvider = (*provider)(nil)

// NewProvider creates a new Google reCAPTCHA v3 captcha provider.
//
// Takes config (Config) which specifies the reCAPTCHA v3 site key and secret
// key from the Google reCAPTCHA admin console.
//
// Returns captcha_domain.CaptchaProvider which provides reCAPTCHA v3
// score-based captcha verification.
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
// Returns captcha_dto.ProviderType which identifies this as a reCAPTCHA v3
// provider.
func (*provider) Type() captcha_dto.ProviderType {
	return captcha_dto.ProviderTypeRecaptchaV3
}

// SiteKey returns the public site key for the reCAPTCHA v3 widget.
//
// Returns string which is the reCAPTCHA v3 site key.
func (p *provider) SiteKey() string {
	return p.config.SiteKey
}

// ScriptURL returns the Google reCAPTCHA v3 JavaScript SDK URL with the site
// key appended as the render parameter.
//
// Returns string which is the reCAPTCHA v3 script URL.
func (p *provider) ScriptURL() string {
	return "https://www.google.com/recaptcha/api.js?render=" + p.config.SiteKey
}

// Verify verifies a reCAPTCHA v3 token by calling the Google siteverify
// endpoint.
//
// The method POSTs the token, secret key, and client IP to the reCAPTCHA
// verification endpoint and parses the JSON response. The response includes
// a risk score between 0.0 (likely bot) and 1.0 (likely human) which is
// populated in VerifyResponse.Score.
//
// Takes request (*captcha_dto.VerifyRequest) which contains the captcha token,
// client IP, and action name.
//
// Returns *captcha_dto.VerifyResponse which contains the verification result
// including the risk score.
// Returns error when the HTTP request fails or the response cannot be parsed.
func (p *provider) Verify(ctx context.Context, request *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
	ctx, span := tracer.Start(ctx, "captcha_provider_recaptcha_v3.Verify")
	defer span.End()

	if request == nil || request.Token == "" {
		return &captcha_dto.VerifyResponse{
			Success:    false,
			ErrorCodes: []string{"missing-input-response"},
		}, nil
	}

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
		return nil, fmt.Errorf("creating recaptcha request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpResponse, err := p.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("sending recaptcha verification request: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, io.LimitReader(httpResponse.Body, maxResponseBodySize))
		_ = httpResponse.Body.Close()
	}()

	if httpResponse.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recaptcha verification returned HTTP %d: %w", httpResponse.StatusCode, captcha_dto.ErrProviderUnavailable)
	}

	contentType := httpResponse.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return nil, fmt.Errorf("recaptcha returned unexpected content type %q: %w",
			contentType, captcha_dto.ErrProviderUnavailable)
	}

	body, err := io.ReadAll(io.LimitReader(httpResponse.Body, maxResponseBodySize))
	if err != nil {
		return nil, fmt.Errorf("reading recaptcha response body: %w", err)
	}

	if int64(len(body)) >= maxResponseBodySize {
		return nil, fmt.Errorf("recaptcha response body exceeded %d byte limit: %w",
			maxResponseBodySize, captcha_dto.ErrProviderUnavailable)
	}

	var recaptchaResult recaptchaVerifyResult
	if err := json.Unmarshal(body, &recaptchaResult); err != nil {
		return nil, fmt.Errorf("parsing recaptcha response: %w", err)
	}

	return buildVerifyResponse(recaptchaResult), nil
}

// buildVerifyResponse converts the provider-specific response into the
// standard VerifyResponse.
//
// Takes result (recaptchaVerifyResult) which is the raw reCAPTCHA API response.
//
// Returns *captcha_dto.VerifyResponse which contains the normalised result.
func buildVerifyResponse(result recaptchaVerifyResult) *captcha_dto.VerifyResponse {
	response := &captcha_dto.VerifyResponse{
		Success:    result.Success,
		Score:      new(normaliseScore(result.Success, result.Score)),
		Action:     result.Action,
		ErrorCodes: result.ErrorCodes,
		Hostname:   result.Hostname,
	}

	if result.ChallengeTimestamp != "" {
		challengeTime, parseErr := time.Parse(time.RFC3339, result.ChallengeTimestamp)
		if parseErr == nil {
			response.Timestamp = challengeTime
		}
	}

	return response
}

// normaliseScore ensures the score follows the convention where 0.0 = bot and
// 1.0 = human, returning 1.0 when verification succeeded but the score is
// zero (which happens with Google's test keys) since a successful verification
// cannot be a confirmed bot.
//
// Takes success (bool) which indicates whether the verification passed.
// Takes score (float64) which is the raw reCAPTCHA risk score.
//
// Returns float64 which is the normalised score.
func normaliseScore(success bool, score float64) float64 {
	if success && score == 0 {
		return 1.0
	}
	return max(0.0, min(1.0, score))
}

// RenderRequirements returns the frontend rendering configuration for the
// Google reCAPTCHA v3 widget. The widget is invisible and runs in the
// background without a visible container.
//
// Returns *captcha_dto.RenderRequirements which describes the script tags, CSP
// domains, and init script needed to render the invisible widget.
func (p *provider) RenderRequirements() *captcha_dto.RenderRequirements {
	return &captcha_dto.RenderRequirements{
		InitScript:        scripts.InitScript,
		ScriptURLs:        []string{"https://www.google.com/recaptcha/api.js?render=" + p.config.SiteKey + "&onload=__pikoCaptchaRecaptchaReady"},
		CSPScriptDomains:  []string{"https://www.google.com", "https://www.gstatic.com"},
		CSPFrameDomains:   []string{"https://www.google.com", "https://www.recaptcha.net"},
		CSPConnectDomains: []string{"https://www.google.com"},
		ProviderType:      "recaptcha_v3",
		Invisible:         true,
	}
}

// HealthCheck returns nil because Google reCAPTCHA does not provide a
// dedicated health check endpoint. Connectivity is verified implicitly
// during token verification.
//
// Returns error which is always nil for this provider.
func (*provider) HealthCheck(_ context.Context) error {
	return nil
}
