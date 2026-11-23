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
	"net/url"
	"time"

	"piko.sh/piko/internal/captcha/captcha_adapters/siteverify"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/wdk/captcha/captcha_provider_recaptcha_v3/scripts"
	"piko.sh/piko/wdk/logger"
)

const (
	// verifyURL is the Google reCAPTCHA server-side verification endpoint.
	verifyURL = "https://www.google.com/recaptcha/api/siteverify"

	// providerName names reCAPTCHA in verification errors.
	providerName = "recaptcha"

	// testKeyHostname is the hostname Google reports for tokens issued by its test keys.
	testKeyHostname = "testkey.google.com"
)

// recaptchaVerifyResult represents the JSON response from the Google reCAPTCHA v3
// siteverify endpoint.
type recaptchaVerifyResult struct {
	// Score is the risk score between 0.0 (likely bot) and 1.0 (likely human), or nil when
	// the response carries no score.
	Score *float64 `json:"score"`

	// Action is the action name that was passed when the token was generated.
	Action string `json:"action"`

	// ChallengeTimestamp is the ISO 8601 timestamp of when the challenge was solved.
	ChallengeTimestamp string `json:"challenge_ts"`

	// Hostname is the hostname of the site where the challenge was solved.
	Hostname string `json:"hostname"`

	// ErrorCodes contains reCAPTCHA-specific error codes when verification fails.
	ErrorCodes []string `json:"error-codes"`

	// Success indicates whether the token verification passed.
	Success bool `json:"success"`
}

// provider implements captcha_domain.CaptchaProvider using Google reCAPTCHA v3.
type provider struct {
	// client posts verification requests to the reCAPTCHA API.
	client *siteverify.Client

	// config holds the validated reCAPTCHA v3 configuration.
	config Config
}

var (
	_ captcha_domain.CaptchaProvider = (*provider)(nil)
)

// NewProvider creates a new Google reCAPTCHA v3 captcha provider.
//
// Takes config (Config) which specifies the reCAPTCHA v3 site key and secret key from the
// Google reCAPTCHA admin console.
//
// Takes options (...Option) which override the verification timeout and response size
// limit.
//
// Returns captcha_domain.CaptchaProvider which provides reCAPTCHA v3 score-based captcha
// verification.
// Returns error when the configuration is invalid.
func NewProvider(config Config, options ...Option) (captcha_domain.CaptchaProvider, error) {
	return newProvider(config, verifyURL, options...)
}

// newProvider creates a reCAPTCHA v3 provider that verifies tokens against endpoint.
//
// Takes config (Config) which specifies the reCAPTCHA v3 site key and secret key.
// Takes endpoint (string) which is the siteverify URL.
// Takes options (...Option) which override the verification timeout and response size
// limit.
//
// Returns *provider which is ready for use.
// Returns error when the configuration is invalid.
func newProvider(config Config, endpoint string, options ...Option) (*provider, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	if config.TestKeyScore != nil {
		log.Warn("reCAPTCHA test key score is set; never use it with production keys",
			logger.Float64("test_key_score", *config.TestKeyScore),
		)
	}

	return &provider{
		config: config,
		client: siteverify.NewClient(providerName, endpoint, options...),
	}, nil
}

// Type returns the provider type identifier.
//
// Returns captcha_dto.ProviderType which identifies this as a reCAPTCHA v3 provider.
func (*provider) Type() captcha_dto.ProviderType {
	return captcha_dto.ProviderTypeRecaptchaV3
}

// SiteKey returns the public site key for the reCAPTCHA v3 widget.
//
// Returns string which is the reCAPTCHA v3 site key.
func (p *provider) SiteKey() string {
	return p.config.SiteKey
}

// ScriptURL returns the Google reCAPTCHA v3 JavaScript SDK URL with the site key appended
// as the render parameter.
//
// Returns string which is the reCAPTCHA v3 script URL.
func (p *provider) ScriptURL() string {
	return "https://www.google.com/recaptcha/api.js?render=" + p.config.SiteKey
}

// Verify verifies a reCAPTCHA v3 token by calling the Google siteverify endpoint.
//
// The method POSTs the token, secret key, and client IP to the reCAPTCHA verification
// endpoint and parses the JSON response. The response includes a risk score between 0.0
// (likely bot) and 1.0 (likely human) which is populated in VerifyResponse.Score.
//
// Takes request (*captcha_dto.VerifyRequest) which contains the captcha token, client IP,
// and action name.
//
// Returns *captcha_dto.VerifyResponse which contains the verification result including
// the risk score.
// Returns error when the HTTP request fails or the response cannot be parsed, or one
// wrapping ErrScoreMissing when a successful response carries no score and no test key
// score applies.
func (p *provider) Verify(ctx context.Context, request *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error) {
	ctx, span := tracer.Start(ctx, "captcha_provider_recaptcha_v3.Verify")
	defer span.End()

	if request == nil || request.Token == "" {
		response := captcha_dto.VerifyResponse{}
		response.ErrorCodes = []string{"missing-input-response"}
		return &response, nil
	}

	formData := url.Values{
		"secret":   {p.config.SecretKey},
		"response": {request.Token},
	}
	if request.RemoteIP != "" {
		formData.Set("remoteip", request.RemoteIP)
	}

	recaptchaResult, err := siteverify.Verify[recaptchaVerifyResult](ctx, p.client, formData)
	if err != nil {
		return nil, err
	}

	return buildVerifyResponse(recaptchaResult, p.config.TestKeyScore, request.Action)
}

// buildVerifyResponse converts the provider-specific response into the standard
// VerifyResponse.
//
// Takes result (recaptchaVerifyResult) which is the raw reCAPTCHA API response.
// Takes testKeyScore (*float64) which is the score to report for a successful test key
// response without a score, or nil to reject such a response.
// Takes requestedAction (string) which supplies the action omitted by a response verified
// with the test key fallback.
//
// Returns *captcha_dto.VerifyResponse which contains the normalised result.
// Returns error which wraps ErrScoreMissing when a successful response has no score and
// no test key score applies.
func buildVerifyResponse(result recaptchaVerifyResult, testKeyScore *float64, requestedAction string) (*captcha_dto.VerifyResponse, error) {
	score, usedTestKey, err := resolveScore(result, testKeyScore)
	if err != nil {
		return nil, err
	}

	action := result.Action
	if usedTestKey {
		action = requestedAction
	}

	response := &captcha_dto.VerifyResponse{
		Success:    result.Success,
		Score:      new(score),
		Action:     action,
		ErrorCodes: result.ErrorCodes,
		Hostname:   result.Hostname,
		Timestamp:  time.Time{},
	}

	if result.ChallengeTimestamp != "" {
		challengeTime, parseErr := time.Parse(time.RFC3339, result.ChallengeTimestamp)
		if parseErr == nil {
			response.Timestamp = challengeTime
		}
	}

	return response, nil
}

// resolveScore returns the score to report for a verification response, where 0.0 means a
// bot and 1.0 means a human.
//
// A successful response without a score is rejected, because treating a missing score as
// confident would let any token that verifies pass every threshold. Google's test keys
// omit the score, so testKeyScore supplies one, but only for a response from the test key
// hostname. A response using a v2 secret without a score still fails.
//
// Takes result (recaptchaVerifyResult) which is the raw reCAPTCHA API response.
// Takes testKeyScore (*float64) which is the score for a successful test key response
// without a score, or nil to reject such a response.
//
// Returns float64 which is the score clamped to the range 0.0 to 1.0.
// Returns bool which is true when the score came from testKeyScore.
// Returns error which wraps ErrScoreMissing and captcha_dto.ErrProviderMisconfigured when
// a successful response has no score and no test key score applies.
func resolveScore(result recaptchaVerifyResult, testKeyScore *float64) (float64, bool, error) {
	switch {
	case result.Score != nil:
		return clampScore(*result.Score), false, nil
	case !result.Success:
		return 0, false, nil
	case testKeyScore != nil && result.Hostname == testKeyHostname:
		return clampScore(*testKeyScore), true, nil
	default:
		return 0, false, fmt.Errorf("%s verification succeeded without a score: %w: %w",
			providerName, ErrScoreMissing, captcha_dto.ErrProviderMisconfigured)
	}
}

// clampScore limits a score to the range 0.0 to 1.0.
//
// Takes score (float64) which is the raw score.
//
// Returns float64 which is the clamped score.
func clampScore(score float64) float64 {
	return max(0.0, min(1.0, score))
}

// RenderRequirements returns the frontend rendering configuration for the Google
// reCAPTCHA v3 widget. The widget is invisible and runs in the background without a
// visible container.
//
// Returns *captcha_dto.RenderRequirements which describes the script tags, CSP domains,
// and init script needed to render the invisible widget.
func (p *provider) RenderRequirements() *captcha_dto.RenderRequirements {
	return &captcha_dto.RenderRequirements{
		InitScript:        scripts.InitScript,
		ScriptURLs:        []string{"https://www.google.com/recaptcha/api.js?render=" + p.config.SiteKey + "&onload=__pikoCaptchaRecaptchaReady"},
		CSPScriptDomains:  []string{"https://www.google.com", "https://www.gstatic.com"},
		CSPFrameDomains:   []string{"https://www.google.com", "https://www.recaptcha.net"},
		CSPConnectDomains: []string{"https://www.google.com"},
		ProviderType:      "recaptcha_v3",
		Invisible:         true,
		ServerSideToken:   false,
	}
}

// HealthCheck returns nil because Google reCAPTCHA does not provide a dedicated health
// check endpoint. Connectivity is verified implicitly during token verification.
//
// Returns error which is always nil for this provider.
func (*provider) HealthCheck(_ context.Context) error {
	return nil
}
