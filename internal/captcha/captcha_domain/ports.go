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

package captcha_domain

import (
	"context"
	"net/http"
	"time"

	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/provider/provider_domain"
)

// RateLimiter checks whether a keyed rate limit has been exceeded, used by
// the captcha service to throttle verification and challenge requests per IP.
// Implementations are provided by the security hexagon via bootstrap wiring.
type RateLimiter interface {
	// IsAllowed checks whether the given key is within its rate limit.
	//
	// Takes key (string) which identifies the resource being limited (e.g.
	// "captcha:verify:192.168.1.1").
	// Takes limit (int) which is the maximum allowed requests.
	// Takes window (time.Duration) which is the time period for the limit.
	//
	// Returns bool which is true if the request is allowed.
	// Returns error when the storage operation fails.
	IsAllowed(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// ClientIPExtractor extracts the real client IP address from an HTTP request,
// accounting for trusted proxy headers. Implementations are provided by the
// security hexagon via bootstrap wiring.
type ClientIPExtractor interface {
	// ExtractClientIP returns the client IP address from the request.
	//
	// Takes request (*http.Request) which provides headers and remote address.
	//
	// Returns string which is the extracted client IP.
	ExtractClientIP(request *http.Request) string
}

// CaptchaProvider defines the interface for captcha verification adapters.
type CaptchaProvider interface {
	// Type returns the provider type for identification and logging.
	Type() captcha_dto.ProviderType

	// Verify verifies a captcha token and returns the verification result.
	//
	// Takes request (*captcha_dto.VerifyRequest) which contains the captcha
	// token, client IP, and action name.
	//
	// Returns *captcha_dto.VerifyResponse which contains the verification
	// result including success status and optional score.
	// Returns error when the verification request fails.
	Verify(ctx context.Context, request *captcha_dto.VerifyRequest) (*captcha_dto.VerifyResponse, error)

	// SiteKey returns the public site key for the captcha provider. This is
	// used by the frontend widget to initialise the captcha challenge.
	//
	// Returns string which is the public site key.
	SiteKey() string

	// ScriptURL returns the URL of the captcha provider's JavaScript SDK,
	// injected into the page as a script tag, or an empty string for providers
	// that do not require an external script (e.g. the built-in HMAC challenge
	// provider).
	//
	// Returns string which is the script URL, or empty.
	ScriptURL() string

	// RenderRequirements returns the frontend rendering configuration for this
	// provider, describing the HTML, scripts, and CSP domains needed to
	// display the captcha widget on a page.
	//
	// Returns *captcha_dto.RenderRequirements which contains the rendering
	// configuration.
	RenderRequirements() *captcha_dto.RenderRequirements

	// HealthCheck verifies provider connectivity and configuration.
	//
	// Returns error when the health check fails.
	HealthCheck(ctx context.Context) error
}

// CaptchaServicePort defines the public interface for captcha verification.
// This is what other hexagons and the action handler middleware depend on.
//
// The service layer abstracts provider-specific details and provides:
//   - Simplified API for common use cases
//   - Score threshold enforcement for score-based providers
//   - Provider management (registration, selection)
//   - Observability integration (metrics, tracing, logging)
type CaptchaServicePort interface {
	// Verify verifies a captcha token using the default provider and the
	// service's default score threshold for score-based providers.
	//
	// Takes token (string) which is the captcha response token from the client.
	// Takes remoteIP (string) which is the client's IP address.
	// Takes action (string) which identifies the protected form or flow.
	//
	// Returns error when verification fails, the token is missing, the score
	// is below the default threshold, or the provider is unavailable.
	Verify(ctx context.Context, token, remoteIP, action string) error

	// VerifyWithScore verifies a captcha token and returns the full response,
	// allowing callers to inspect the score and other provider-specific data.
	//
	// Takes token (string) which is the captcha response token from the client.
	// Takes remoteIP (string) which is the client's IP address.
	// Takes action (string) which identifies the protected form or flow.
	// Takes scoreThreshold (float64) which is the minimum required score; use
	// 0 to accept any score.
	//
	// Returns *captcha_dto.VerifyResponse which contains the full verification
	// result.
	// Returns error when verification fails.
	VerifyWithScore(ctx context.Context, token, remoteIP, action string, scoreThreshold float64) (*captcha_dto.VerifyResponse, error)

	// VerifyWithProvider verifies a captcha token using a specific named
	// provider instead of the default. The provider must have been registered
	// via WithCaptchaProvider.
	//
	// Takes providerName (string) which identifies the provider to use.
	// Takes token (string) which is the captcha response token from the client.
	// Takes remoteIP (string) which is the client's IP address.
	// Takes action (string) which identifies the protected form or flow.
	// Takes scoreThreshold (float64) which is the minimum required score.
	//
	// Returns *captcha_dto.VerifyResponse which contains the verification
	// result.
	// Returns error when the provider is not found or verification fails.
	VerifyWithProvider(ctx context.Context, providerName, token, remoteIP, action string, scoreThreshold float64) (*captcha_dto.VerifyResponse, error)

	// SiteKey returns the public site key of the default captcha provider.
	//
	// Returns string which is the site key, or empty if no provider is set.
	SiteKey() string

	// ScriptURL returns the JavaScript SDK URL of the default captcha provider.
	//
	// Returns string which is the script URL, or empty if not applicable.
	ScriptURL() string

	// IsEnabled reports whether a captcha provider is configured and active.
	//
	// Returns bool which is true when a provider is available for verification.
	IsEnabled() bool

	// GetDefaultProvider returns the default captcha provider.
	//
	// Returns CaptchaProvider which is the default provider.
	// Returns error when no default provider is configured.
	GetDefaultProvider(ctx context.Context) (CaptchaProvider, error)

	// GetProviderByName returns a captcha provider by its registered name.
	//
	// Takes name (string) which identifies the provider.
	//
	// Returns CaptchaProvider which is the named provider.
	// Returns error when the provider is not found.
	GetProviderByName(ctx context.Context, name string) (CaptchaProvider, error)

	// RegisterProvider adds a new captcha provider with the given name.
	//
	// Takes name (string) which identifies the provider.
	// Takes provider (CaptchaProvider) which handles verification.
	//
	// Returns error when the provider cannot be registered.
	RegisterProvider(ctx context.Context, name string, provider CaptchaProvider) error

	// SetDefaultProvider sets the provider to use when no specific provider
	// is named.
	//
	// Takes name (string) which is the name of the provider to use as default.
	//
	// Returns error when the named provider does not exist.
	SetDefaultProvider(name string) error

	// GetProviders returns a sorted list of all registered provider names.
	GetProviders(ctx context.Context) []string

	// HasProvider checks if a provider with the given name has been registered.
	//
	// Returns bool which is true if the provider exists.
	HasProvider(name string) bool

	// ListProviders returns details about all registered providers.
	//
	// Returns []provider_domain.ProviderInfo which contains information about
	// each provider.
	ListProviders(ctx context.Context) []provider_domain.ProviderInfo

	// HealthCheck verifies the captcha service is operational by checking the
	// default provider's connectivity.
	//
	// Returns error when the health check fails.
	HealthCheck(ctx context.Context) error

	// Close shuts down all providers in a controlled manner.
	//
	// Returns error when shutdown fails or the context is cancelled.
	Close(ctx context.Context) error
}
