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

package bootstrap

import (
	"context"
	"fmt"
	"time"

	"net/http"

	"piko.sh/piko/internal/captcha/captcha_adapters/hmac_challenge"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
)

// rateLimitServiceAdapter bridges security_domain.RateLimitService to the
// captcha_domain.RateLimiter interface, avoiding a cross-hexagon import.
type rateLimitServiceAdapter struct {
	// service is the underlying rate limit service from the security hexagon.
	service security_domain.RateLimitService
}

// clientIPExtractorFunc adapts a function to the ClientIPExtractor interface.
type clientIPExtractorFunc func(*http.Request) string

var (
	_ captcha_domain.RateLimiter = rateLimitServiceAdapter{}

	_ captcha_domain.ClientIPExtractor = clientIPExtractorFunc(nil)
)

// AddCaptchaProvider registers a named captcha provider for verification
// operations.
//
// If the provider implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes name (string) which identifies the provider for later retrieval.
// Takes provider (CaptchaProvider) which handles captcha verification.
func (c *Container) AddCaptchaProvider(name string, provider captcha_domain.CaptchaProvider) {
	if c.captchaProviders == nil {
		c.captchaProviders = make(map[string]captcha_domain.CaptchaProvider)
	}
	c.captchaProviders[name] = provider
	registerCloseableForShutdown(c.GetAppContext(), "CaptchaProvider-"+name, provider)
}

// SetCaptchaDefaultProvider sets the default captcha provider to use when
// none is specified.
//
// Takes name (string) which identifies the provider to use as the default.
func (c *Container) SetCaptchaDefaultProvider(name string) {
	c.captchaDefaultProvider = name
}

// GetCaptchaService returns the captcha service, initialising a default one if
// none was provided.
//
// Returns captcha_domain.CaptchaServicePort which provides captcha verification
// operations.
// Returns error when the default captcha service fails to initialise.
func (c *Container) GetCaptchaService() (captcha_domain.CaptchaServicePort, error) {
	c.captchaOnce.Do(func() {
		c.createDefaultCaptchaService()
	})
	return c.captchaService, c.captchaErr
}

// createDefaultCaptchaService sets up the captcha service using default
// settings.
func (c *Container) createDefaultCaptchaService() {
	ctx := c.GetAppContext()
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Creating default CaptchaService...")

	providerName, baseProvider, err := c.selectCaptchaProvider(ctx)
	if err != nil {
		c.captchaErr = err
		l.Error("Failed to create captcha provider", logger_domain.Error(c.captchaErr))
		return
	}

	if providerName == "disabled" {
		l.Internal("Captcha not configured; captcha service disabled.")
		c.captchaService = captcha_domain.NewDisabledCaptchaService()
		return
	}

	captchaService, err := c.buildCaptchaService(ctx, providerName, baseProvider)
	if err != nil {
		c.captchaErr = err
		l.Error("Failed to create captcha service", logger_domain.Error(c.captchaErr))
		return
	}
	c.captchaService = captchaService

	l.Internal("Captcha service created",
		logger_domain.String("provider", providerName))
}

// selectCaptchaProvider selects the appropriate captcha provider based on
// options or config.
//
// Returns string which is the selected provider name.
// Returns captcha_domain.CaptchaProvider which provides verification.
// Returns error when the provider cannot be selected or created.
func (c *Container) selectCaptchaProvider(ctx context.Context) (providerName string, provider captcha_domain.CaptchaProvider, err error) {
	ctx, l := logger_domain.From(ctx, log)
	if len(c.captchaProviders) > 0 {
		l.Internal("Using captcha provider registered via options")

		if c.captchaDefaultProvider != "" {
			providerName = c.captchaDefaultProvider
			provider = c.captchaProviders[providerName]
			if provider == nil {
				return "", nil, fmt.Errorf("captcha default provider %q not registered", providerName)
			}
		} else {
			if len(c.captchaProviders) > 1 {
				l.Warn("Multiple captcha providers registered without an explicit default; selection is non-deterministic. Use piko.WithDefaultCaptchaProvider() to set one.")
			}
			for name, captchaProvider := range c.captchaProviders {
				providerName, provider = name, captchaProvider
				break
			}
		}

		l.Internal("Captcha provider selected from options",
			logger_domain.String("provider", providerName))
		return providerName, provider, nil
	}

	l.Internal("No captcha providers registered via options; creating from config")
	return c.createCaptchaProviderFromConfig(ctx)
}

// createCaptchaProviderFromConfig creates a captcha provider based on config
// settings. This fallback only supports the hmac_challenge provider for
// simple config-based initialisation.
//
// Returns string which is the selected provider name.
// Returns captcha_domain.CaptchaProvider which provides verification.
// Returns error when the provider cannot be created.
func (c *Container) createCaptchaProviderFromConfig(ctx context.Context) (providerName string, provider captcha_domain.CaptchaProvider, err error) {
	_, l := logger_domain.From(ctx, log)

	providerType := deref(c.serverConfig.Security.CaptchaProvider, "")
	if providerType == "" {
		return "disabled", nil, nil
	}

	l.Internal("Creating captcha provider", logger_domain.String("provider", providerType))

	switch providerType {
	case "hmac_challenge":
		return c.createHMACChallengeProvider()

	case "turnstile", "recaptcha_v3", "hcaptcha":
		return "", nil, c.captchaCloudProviderConfigError(providerType)

	default:
		return "", nil, fmt.Errorf(
			"unknown captcha provider '%s'.\n\n"+
				"Supported config provider: hmac_challenge\n"+
				"For cloud providers (turnstile, recaptcha_v3, hcaptcha), use the option-based approach.\n\n"+
				"See: captcha/README.md for examples",
			providerType,
		)
	}
}

// createHMACChallengeProvider creates the built-in HMAC challenge provider.
// The secret resolution follows the same pattern as CSRF secrets: config >
// persisted temp file > generated ephemeral.
//
// Returns string which is the provider name.
// Returns captcha_domain.CaptchaProvider which provides HMAC-based verification.
// Returns error when the provider cannot be created.
func (c *Container) createHMACChallengeProvider() (providerName string, provider captcha_domain.CaptchaProvider, err error) {
	secret := resolveCaptchaSecret(deref(c.serverConfig.Security.CaptchaSecretKey, ""))

	provider, err = hmac_challenge.NewProvider(hmac_challenge.Config{
		Secret: secret,
	})
	if err != nil {
		return "", nil, fmt.Errorf("creating HMAC challenge provider: %w", err)
	}

	return "hmac_challenge", provider, nil
}

// captchaCloudProviderConfigError returns an error for cloud captcha providers
// that cannot be set up via a config file.
//
// Takes providerType (string) which identifies the unsupported provider.
//
// Returns error describing how to use the option-based approach instead.
func (*Container) captchaCloudProviderConfigError(providerType string) error {
	return fmt.Errorf(
		"captcha provider '%s' cannot be configured via config file.\n\n"+
			"Please use the option-based approach:\n\n"+
			"  import (\n"+
			"      \"piko.sh/piko\"\n"+
			"      \"piko.sh/piko/wdk/captcha/captcha_provider_%s\"\n"+
			"  )\n\n"+
			"  provider, _ := captcha_provider_%s.NewProvider(ctx, config)\n"+
			"  server := piko.New(\n"+
			"      piko.WithCaptchaProvider(\"%s\", provider),\n"+
			"      piko.WithDefaultCaptchaProvider(\"%s\"),\n"+
			"  )",
		providerType, providerType, providerType, providerType, providerType,
	)
}

// buildCaptchaService creates the captcha service with the selected provider.
//
// Takes providerName (string) which identifies the provider to register.
// Takes baseProvider (captcha_domain.CaptchaProvider) which handles
// verification.
//
// Returns captcha_domain.CaptchaServicePort which is the configured service.
// Returns error when the service cannot be built.
func (c *Container) buildCaptchaService(
	ctx context.Context,
	providerName string,
	baseProvider captcha_domain.CaptchaProvider,
) (captcha_domain.CaptchaServicePort, error) {
	serviceConfig := captcha_dto.DefaultServiceConfig()

	if threshold := deref(c.serverConfig.Security.CaptchaScoreThreshold, 0); threshold > 0 {
		serviceConfig.DefaultScoreThreshold = threshold
	}

	ctx, l := logger_domain.From(ctx, log)

	var serviceOptions []captcha_domain.ServiceOption
	if rateLimitSvc, rateLimitErr := c.GetRateLimitService(); rateLimitErr == nil {
		serviceOptions = append(serviceOptions,
			captcha_domain.WithRateLimiter(rateLimitServiceAdapter{rateLimitSvc}))
	} else {
		l.Warn("Rate limit service unavailable for captcha, operating without rate limiting")
	}

	serviceOptions = append(serviceOptions,
		captcha_domain.WithClientIPExtractor(clientIPExtractorFunc(security_dto.ClientIPFromRequest)))

	service, err := captcha_domain.NewCaptchaService(serviceConfig, serviceOptions...)
	if err != nil {
		return nil, fmt.Errorf("creating captcha service: %w", err)
	}

	if len(c.captchaProviders) > 0 {
		for name, provider := range c.captchaProviders {
			if err := service.RegisterProvider(ctx, name, provider); err != nil {
				return nil, fmt.Errorf("registering captcha provider %q: %w", name, err)
			}
		}
	} else {
		if err := service.RegisterProvider(ctx, providerName, baseProvider); err != nil {
			return nil, fmt.Errorf("registering captcha provider: %w", err)
		}
	}

	if err := service.SetDefaultProvider(providerName); err != nil {
		return nil, fmt.Errorf("setting default captcha provider: %w", err)
	}

	return service, nil
}

// SetCaptchaService sets a pre-configured captcha service on the container.
//
// If the service implements a shutdown interface (Close, Shutdown, or Stop),
// it will be automatically registered for graceful shutdown.
//
// Takes service (captcha_domain.CaptchaServicePort) which is the captcha
// service to use.
func (c *Container) SetCaptchaService(service captcha_domain.CaptchaServicePort) {
	c.captchaOnce.Do(func() {})
	c.captchaService = service
	registerCloseableForShutdown(c.GetAppContext(), "CaptchaService", service)
}

// IsAllowed checks whether the given key is within its rate limit.
//
// Takes ctx (context.Context) which propagates cancellation into the
// underlying rate limit service.
// Takes key (string) which identifies the resource being limited.
// Takes limit (int) which is the maximum allowed requests.
// Takes window (time.Duration) which is the time period for the limit.
//
// Returns bool which is true if the request is allowed.
// Returns error when the underlying rate limit check fails.
func (a rateLimitServiceAdapter) IsAllowed(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	result, err := a.service.CheckLimit(ctx, key, limit, window)
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// ExtractClientIP returns the client IP address from the request.
//
// Takes request (*http.Request) which provides headers and remote address.
//
// Returns string which is the extracted client IP.
func (f clientIPExtractorFunc) ExtractClientIP(request *http.Request) string {
	return f(request)
}
