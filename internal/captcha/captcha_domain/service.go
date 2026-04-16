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
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/sony/gobreaker/v2"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	"piko.sh/piko/wdk/clock"
)

const (
	// circuitBreakerTimeout is how long the circuit stays open before letting a
	// test request through.
	circuitBreakerTimeout = 30 * time.Second

	// circuitBreakerBucketPeriod is the duration of each measurement bucket
	// for tracking failure counts.
	circuitBreakerBucketPeriod = 10 * time.Second

	// circuitBreakerConsecutiveFailures is the number of consecutive failures
	// required to trip the circuit breaker.
	circuitBreakerConsecutiveFailures = 5

	// safeCallCaptchaType is the goroutine.SafeCallValue label for provider
	// type lookups.
	safeCallCaptchaType = "captcha.Type"

	// maxTokenLength is the maximum allowed length of a captcha token at the
	// service level. This provides defence-in-depth above the provider layer.
	maxTokenLength = 8192

	// maxActionLength is the maximum allowed length of an action name at the
	// service level. This provides defence-in-depth above the provider layer.
	maxActionLength = 256
)

var (
	// errProviderNameEmpty is returned when an empty name is passed to RegisterProvider.
	errProviderNameEmpty = errors.New("provider name cannot be empty")

	// errProviderNil is returned when a nil provider is passed to RegisterProvider.
	errProviderNil = errors.New("provider cannot be nil")
)

// captchaService is the concrete implementation of CaptchaServicePort.
type captchaService struct {
	// registry stores and looks up captcha providers by name.
	registry *provider_domain.StandardRegistry[CaptchaProvider]

	// breaker guards against failures in the captcha provider service.
	breaker *gobreaker.CircuitBreaker[*captcha_dto.VerifyResponse]

	// rateLimiter throttles verification and challenge requests per IP. Nil
	// disables rate limiting.
	rateLimiter RateLimiter

	// ipExtractor extracts the real client IP from requests, accounting for
	// trusted proxy headers. Nil falls back to RemoteAddr.
	ipExtractor ClientIPExtractor

	// clock provides the time source for duration measurements.
	clock clock.Clock

	// config stores the service configuration including rate limit settings.
	config *captcha_dto.ServiceConfig

	// defaultScoreThreshold is the minimum score for score-based providers.
	defaultScoreThreshold float64
}

// ServiceOption configures the captcha service.
type ServiceOption func(*captchaService)

// WithDefaultScoreThreshold sets the default score threshold for score-based
// providers.
//
// Takes threshold (float64) which is the minimum required score.
//
// Returns ServiceOption which configures the threshold on the service.
func WithDefaultScoreThreshold(threshold float64) ServiceOption {
	return func(s *captchaService) {
		s.defaultScoreThreshold = threshold
	}
}

// WithRateLimiter sets the rate limiter used to throttle captcha verification
// and challenge requests per IP.
//
// Takes limiter (RateLimiter) which provides the rate limiting check.
//
// Returns ServiceOption which configures the rate limiter on the service.
func WithRateLimiter(limiter RateLimiter) ServiceOption {
	return func(s *captchaService) {
		s.rateLimiter = limiter
	}
}

// WithClientIPExtractor sets the client IP extractor used by the challenge
// rate limiter to determine the real client IP address. Without an extractor
// the service falls back to RemoteAddr, which may be incorrect behind proxies.
//
// Takes extractor (ClientIPExtractor) which extracts the client IP from
// requests using trusted proxy configuration.
//
// Returns ServiceOption which configures the IP extractor on the service.
func WithClientIPExtractor(extractor ClientIPExtractor) ServiceOption {
	return func(s *captchaService) {
		s.ipExtractor = extractor
	}
}

// WithClock sets the time source for the captcha service.
//
// Takes c (clock.Clock) which provides the time source.
//
// Returns ServiceOption which configures the clock on the service.
func WithClock(c clock.Clock) ServiceOption {
	return func(s *captchaService) {
		s.clock = c
	}
}

// NewCaptchaService creates a new captcha service.
//
// Takes config (*captcha_dto.ServiceConfig) which provides service settings.
// Takes options (...ServiceOption) which are optional configuration functions.
//
// Returns CaptchaServicePort which is the configured captcha service.
// Returns error when the service cannot be created.
func NewCaptchaService(config *captcha_dto.ServiceConfig, options ...ServiceOption) (CaptchaServicePort, error) {
	if config == nil {
		config = captcha_dto.DefaultServiceConfig()
	}

	service := &captchaService{
		registry:              provider_domain.NewStandardRegistry[CaptchaProvider]("captcha"),
		breaker:               newCaptchaCircuitBreaker(),
		clock:                 clock.RealClock(),
		config:                config,
		defaultScoreThreshold: config.DefaultScoreThreshold,
	}

	for _, option := range options {
		option(service)
	}

	return service, nil
}

// Verify verifies a captcha token using the default provider and the service's
// default score threshold.
//
// Takes token (string) which is the captcha response token from the client.
// Takes remoteIP (string) which is the client's IP address.
// Takes action (string) which identifies the protected form or flow.
//
// Returns error when verification fails.
func (s *captchaService) Verify(ctx context.Context, token, remoteIP, action string) error {
	_, err := s.VerifyWithScore(ctx, token, remoteIP, action, s.defaultScoreThreshold)
	return err
}

// VerifyWithScore verifies a captcha token and returns the full response.
//
// Takes token (string) which is the captcha response token from the client.
// Takes remoteIP (string) which is the client's IP address.
// Takes action (string) which identifies the protected form or flow.
// Takes scoreThreshold (float64) which is the minimum required score.
//
// Returns *captcha_dto.VerifyResponse which contains the full verification
// result.
// Returns error when verification fails.
func (s *captchaService) VerifyWithScore(ctx context.Context, token, remoteIP, action string, scoreThreshold float64) (*captcha_dto.VerifyResponse, error) {
	if token == "" {
		return nil, captcha_dto.ErrTokenMissing
	}

	if len(token) > maxTokenLength {
		return nil, captcha_dto.ErrTokenTooLong
	}

	if len(action) > maxActionLength {
		return nil, captcha_dto.ErrActionTooLong
	}

	if err := s.checkVerifyRateLimit(ctx, remoteIP); err != nil {
		return nil, err
	}

	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, err
	}

	response, providerType, durationMS, err := s.callProvider(ctx, provider, token, remoteIP, action)
	if err != nil {
		recordMetrics(ctx, providerType, statusError, durationMS)
		return nil, captcha_dto.NewCaptchaError(opVerify, providerType, err)
	}

	return s.evaluateResponse(ctx, response, providerType, durationMS, scoreThreshold, action)
}

// VerifyWithProvider verifies a captcha token using a specific named provider.
//
// Takes providerName (string) which identifies the provider to use.
// Takes token (string) which is the captcha response token from the client.
// Takes remoteIP (string) which is the client's IP address.
// Takes action (string) which identifies the protected form or flow.
// Takes scoreThreshold (float64) which is the minimum required score.
//
// Returns *captcha_dto.VerifyResponse which contains the verification result.
// Returns error when the provider is not found or verification fails.
func (s *captchaService) VerifyWithProvider(ctx context.Context, providerName, token, remoteIP, action string, scoreThreshold float64) (*captcha_dto.VerifyResponse, error) {
	if token == "" {
		return nil, captcha_dto.ErrTokenMissing
	}

	if len(token) > maxTokenLength {
		return nil, captcha_dto.ErrTokenTooLong
	}

	if len(action) > maxActionLength {
		return nil, captcha_dto.ErrActionTooLong
	}

	if err := s.checkVerifyRateLimit(ctx, remoteIP); err != nil {
		return nil, err
	}

	provider, err := s.registry.GetProvider(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("captcha provider %q not found: %w", providerName, err)
	}

	response, providerType, durationMS, err := s.callProvider(ctx, provider, token, remoteIP, action)
	if err != nil {
		recordMetrics(ctx, providerType, statusError, durationMS)
		return nil, captcha_dto.NewCaptchaError(opVerify, providerType, err)
	}

	return s.evaluateResponse(ctx, response, providerType, durationMS, scoreThreshold, action)
}

// GetDefaultProvider returns the default captcha provider.
//
// Returns CaptchaProvider which is the default provider.
// Returns error when no default provider is configured.
func (s *captchaService) GetDefaultProvider(ctx context.Context) (CaptchaProvider, error) {
	return s.getProvider(ctx)
}

// GetProviderByName returns a captcha provider by its registered name.
//
// Takes name (string) which identifies the provider.
//
// Returns CaptchaProvider which is the named provider.
// Returns error when the provider is not found.
func (s *captchaService) GetProviderByName(ctx context.Context, name string) (CaptchaProvider, error) {
	return s.registry.GetProvider(ctx, name)
}

// SiteKey returns the public site key of the default captcha provider.
//
// Returns string which is the site key, or empty if no provider is set.
func (s *captchaService) SiteKey() string {
	provider, err := s.getProvider(context.Background())
	if err != nil {
		return ""
	}
	return provider.SiteKey()
}

// ScriptURL returns the JavaScript SDK URL of the default captcha provider.
//
// Returns string which is the script URL, or empty if not applicable.
func (s *captchaService) ScriptURL() string {
	provider, err := s.getProvider(context.Background())
	if err != nil {
		return ""
	}
	return provider.ScriptURL()
}

// IsEnabled reports whether a captcha provider is configured and active.
//
// Returns bool which is true when a provider is available for verification.
func (s *captchaService) IsEnabled() bool {
	return s.registry.GetDefaultProvider() != ""
}

// RegisterProvider adds a new captcha provider with the given name.
//
// Takes name (string) which identifies the provider.
// Takes provider (CaptchaProvider) which handles verification.
//
// Returns error when the provider cannot be registered.
func (s *captchaService) RegisterProvider(ctx context.Context, name string, provider CaptchaProvider) error {
	if name == "" {
		return errProviderNameEmpty
	}
	if provider == nil {
		return errProviderNil
	}
	return s.registry.RegisterProvider(ctx, name, provider)
}

// SetDefaultProvider sets the provider to use when no specific provider is
// named.
//
// Takes name (string) which is the name of the provider to use as default.
//
// Returns error when the named provider does not exist.
func (s *captchaService) SetDefaultProvider(name string) error {
	return s.registry.SetDefaultProvider(context.Background(), name)
}

// GetProviders returns a sorted list of all registered provider names.
//
// Returns []string which contains the sorted provider names.
func (s *captchaService) GetProviders(ctx context.Context) []string {
	providers := s.registry.ListProviders(ctx)
	names := make([]string, 0, len(providers))
	for _, provider := range providers {
		names = append(names, provider.Name)
	}
	slices.Sort(names)
	return names
}

// HasProvider checks if a provider with the given name has been registered.
//
// Takes name (string) which is the provider name to look up.
//
// Returns bool which is true if the provider exists.
func (s *captchaService) HasProvider(name string) bool {
	return s.registry.HasProvider(name)
}

// ListProviders returns details about all registered providers.
//
// Returns []provider_domain.ProviderInfo which contains information about each
// provider.
func (s *captchaService) ListProviders(ctx context.Context) []provider_domain.ProviderInfo {
	return s.registry.ListProviders(ctx)
}

// HealthCheck verifies the captcha service is operational.
//
// Returns error when the health check fails.
func (s *captchaService) HealthCheck(ctx context.Context) error {
	provider, err := s.getProvider(ctx)
	if err != nil {
		return err
	}
	return provider.HealthCheck(ctx)
}

// ChallengeHandler returns an HTTP handler for generating challenge tokens if
// the default provider supports it, or nil otherwise.
//
// Returns http.Handler which generates challenge tokens, or nil if the provider
// does not support challenge generation.
func (s *captchaService) ChallengeHandler() http.Handler {
	provider, err := s.getProvider(context.Background())
	if err != nil {
		return nil
	}
	if challenger, ok := provider.(interface {
		ChallengeHandler() http.Handler
	}); ok {
		handler := challenger.ChallengeHandler()
		return s.wrapChallengeRateLimit(handler)
	}
	return nil
}

// Close shuts down all providers in a controlled manner.
//
// Returns error when shutdown fails or the context is cancelled.
func (s *captchaService) Close(ctx context.Context) error {
	return s.registry.CloseAll(ctx)
}

// checkVerifyRateLimit checks whether the client IP has exceeded the captcha
// verification rate limit, returning nil if rate limiting is disabled, the
// request is allowed, or on storage errors (fail open).
//
// Takes remoteIP (string) which is the client's IP address used as the rate
// limit key.
//
// Returns error which is ErrRateLimited if the limit is exceeded, or nil
// otherwise.
func (s *captchaService) checkVerifyRateLimit(ctx context.Context, remoteIP string) error {
	if s.rateLimiter == nil || s.config.VerifyRateLimit <= 0 {
		return nil
	}
	allowed, err := s.rateLimiter.IsAllowed(ctx,
		"captcha:verify:"+remoteIP, s.config.VerifyRateLimit, time.Minute)
	if err != nil {
		return nil
	}
	if !allowed {
		return captcha_dto.ErrRateLimited
	}
	return nil
}

// wrapChallengeRateLimit wraps an http.Handler with per-IP rate limiting for
// challenge token generation, returning the handler unchanged if no rate
// limiter is configured or the limit is zero.
//
// Takes next (http.Handler) which is the handler to wrap with rate limiting.
//
// Returns http.Handler which enforces the challenge rate limit before
// delegating to next.
func (s *captchaService) wrapChallengeRateLimit(next http.Handler) http.Handler {
	if s.rateLimiter == nil || s.config.ChallengeRateLimit <= 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		if s.ipExtractor != nil {
			clientIP = s.ipExtractor.ExtractClientIP(r)
		}
		allowed, err := s.rateLimiter.IsAllowed(r.Context(),
			"captcha:challenge:"+clientIP, s.config.ChallengeRateLimit, time.Minute)
		if err == nil && !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// callProvider executes the captcha verification against the provider with
// circuit breaker protection.
//
// Takes provider (CaptchaProvider) which handles the verification.
// Takes token (string) which is the captcha response token.
// Takes remoteIP (string) which is the client's IP address.
// Takes action (string) which identifies the protected flow.
//
// Returns *captcha_dto.VerifyResponse which is the verification result.
// Returns string which is the provider type identifier.
// Returns int64 which is the duration in milliseconds.
// Returns error when verification fails.
func (s *captchaService) callProvider(ctx context.Context, provider CaptchaProvider, token, remoteIP, action string) (*captcha_dto.VerifyResponse, string, int64, error) {
	startTime := s.clock.Now()

	request := &captcha_dto.VerifyRequest{
		Token:    token,
		RemoteIP: remoteIP,
		Action:   action,
	}

	response, verifyErr := goroutine.SafeCall1(ctx, "captcha.Verify", func() (*captcha_dto.VerifyResponse, error) {
		return s.breaker.Execute(func() (*captcha_dto.VerifyResponse, error) {
			return provider.Verify(ctx, request)
		})
	})

	durationMS := s.clock.Now().Sub(startTime).Milliseconds()
	providerType := string(goroutine.SafeCallValue(ctx, safeCallCaptchaType, func() captcha_dto.ProviderType {
		return provider.Type()
	}))

	if verifyErr != nil {
		return nil, providerType, durationMS, verifyErr
	}

	return response, providerType, durationMS, nil
}

// evaluateResponse checks the verification result against success, action
// binding, and score thresholds.
//
// Takes response (*captcha_dto.VerifyResponse) which is the provider's result.
// Takes providerType (string) which identifies the provider for metrics.
// Takes durationMS (int64) which is the verification duration in milliseconds.
// Takes scoreThreshold (float64) which is the minimum required score.
// Takes expectedAction (string) which is the action the token should have been
// generated for; empty skips the check.
//
// Returns *captcha_dto.VerifyResponse which is the original response.
// Returns error when the verification failed, the action mismatches, or the
// score is below threshold.
func (*captchaService) evaluateResponse(
	ctx context.Context,
	response *captcha_dto.VerifyResponse,
	providerType string,
	durationMS int64,
	scoreThreshold float64,
	expectedAction string,
) (*captcha_dto.VerifyResponse, error) {
	ctx, l := logger_domain.From(ctx, log)

	if !response.Success {
		recordMetrics(ctx, providerType, statusError, durationMS)
		return response, captcha_dto.ErrVerificationFailed
	}

	if expectedAction != "" && response.Action != "" && response.Action != expectedAction {
		recordMetrics(ctx, providerType, statusError, durationMS)
		l.Trace("Captcha action mismatch",
			logger_domain.String("expected", expectedAction),
			logger_domain.String("actual", response.Action),
			logger_domain.String(attributeKeyProvider, providerType),
		)
		return response, captcha_dto.ErrVerificationFailed
	}

	if scoreThreshold > 0 && response.Score != nil && *response.Score < scoreThreshold {
		recordMetrics(ctx, providerType, statusError, durationMS)
		l.Trace("Captcha score below threshold",
			logger_domain.Float64("score", *response.Score),
			logger_domain.Float64("threshold", scoreThreshold),
			logger_domain.String(attributeKeyProvider, providerType),
		)
		return response, captcha_dto.ErrScoreBelowThreshold
	}

	recordMetrics(ctx, providerType, statusSuccess, durationMS)

	l.Trace("Captcha verification completed",
		logger_domain.Int64(attributeKeyDurationMS, durationMS),
		logger_domain.String(attributeKeyProvider, providerType),
	)

	return response, nil
}

// getProvider returns the default provider.
//
// Returns CaptchaProvider which is the default provider.
// Returns error when no provider is configured.
func (s *captchaService) getProvider(ctx context.Context) (CaptchaProvider, error) {
	providerName := s.registry.GetDefaultProvider()
	if providerName == "" {
		return nil, captcha_dto.ErrProviderUnavailable
	}
	return s.registry.GetProvider(ctx, providerName)
}

// recordMetrics records OTel metrics for a verification operation.
//
// Takes providerType (string) which identifies the provider for metric labels.
// Takes status (string) which is the verification outcome.
// Takes durationMS (int64) which is the operation duration in milliseconds.
func recordMetrics(ctx context.Context, providerType, status string, durationMS int64) {
	captchaVerifyDuration.Record(ctx, float64(durationMS),
		metricAttributes(attributeKeyOperation, opVerify, attributeKeyProvider, providerType),
	)
	captchaVerifyCount.Add(ctx, 1,
		metricAttributes(attributeKeyOperation, opVerify, attributeKeyProvider, providerType, attributeKeyStatus, status),
	)
}

// newCaptchaCircuitBreaker creates a circuit breaker for the captcha service.
//
// Returns *gobreaker.CircuitBreaker[*captcha_dto.VerifyResponse] which guards
// against cascading failures.
func newCaptchaCircuitBreaker() *gobreaker.CircuitBreaker[*captcha_dto.VerifyResponse] {
	settings := gobreaker.Settings{
		Name:         "captcha-service",
		MaxRequests:  1,
		Interval:     0,
		Timeout:      circuitBreakerTimeout,
		BucketPeriod: circuitBreakerBucketPeriod,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= circuitBreakerConsecutiveFailures
		},
		IsExcluded: func(err error) bool {
			return errors.Is(err, context.Canceled) ||
				errors.Is(err, context.DeadlineExceeded)
		},
	}
	return gobreaker.NewCircuitBreaker[*captcha_dto.VerifyResponse](settings)
}
