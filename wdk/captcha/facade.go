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

package captcha

import (
	"context"

	"piko.sh/piko/internal/bootstrap"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
)

// ServicePort is the captcha service interface.
type ServicePort = captcha_domain.CaptchaServicePort

// Provider is the captcha provider interface.
type Provider = captcha_domain.CaptchaProvider

// ProviderType identifies which captcha provider to use.
type ProviderType = captcha_dto.ProviderType

// VerifyRequest contains the data needed to verify a captcha token.
type VerifyRequest = captcha_dto.VerifyRequest

// VerifyResponse contains the result of a captcha verification.
type VerifyResponse = captcha_dto.VerifyResponse

// ServiceConfig holds configuration for the captcha service.
type ServiceConfig = captcha_dto.ServiceConfig

// CaptchaError wraps a captcha error with additional context.
type CaptchaError = captcha_dto.CaptchaError

const (
	// ProviderTypeHMACChallenge identifies the built-in HMAC challenge provider.
	ProviderTypeHMACChallenge = captcha_dto.ProviderTypeHMACChallenge

	// ProviderTypeTurnstile identifies the Cloudflare Turnstile provider.
	ProviderTypeTurnstile = captcha_dto.ProviderTypeTurnstile

	// ProviderTypeRecaptchaV3 identifies the Google reCAPTCHA v3 provider.
	ProviderTypeRecaptchaV3 = captcha_dto.ProviderTypeRecaptchaV3

	// ProviderTypeHCaptcha identifies the hCaptcha provider.
	ProviderTypeHCaptcha = captcha_dto.ProviderTypeHCaptcha
)

var (
	// ErrCaptchaDisabled is returned when captcha verification is attempted but
	// the captcha service is disabled.
	ErrCaptchaDisabled = captcha_dto.ErrCaptchaDisabled

	// ErrVerificationFailed is returned when the captcha token fails provider
	// verification.
	ErrVerificationFailed = captcha_dto.ErrVerificationFailed

	// ErrTokenMissing is returned when no captcha token is provided.
	ErrTokenMissing = captcha_dto.ErrTokenMissing

	// ErrTokenExpired is returned when the captcha token has exceeded its TTL.
	ErrTokenExpired = captcha_dto.ErrTokenExpired

	// ErrProviderUnavailable is returned when the captcha provider cannot be
	// reached or is not configured.
	ErrProviderUnavailable = captcha_dto.ErrProviderUnavailable

	// ErrScoreBelowThreshold is returned when the captcha score is below the
	// configured minimum threshold.
	ErrScoreBelowThreshold = captcha_dto.ErrScoreBelowThreshold
)

// GetDefaultService returns the global captcha service instance.
//
// Returns ServicePort which is the configured captcha service.
// Returns error when the framework is not initialised or the service
// cannot be created.
func GetDefaultService() (ServicePort, error) {
	return bootstrap.GetCaptchaService()
}

// Verify verifies a captcha token using the global default captcha service.
//
// Takes token (string) which is the captcha response token from the client.
// Takes remoteIP (string) which is the client's IP address.
// Takes action (string) which identifies the protected form or flow.
//
// Returns error when verification fails.
func Verify(ctx context.Context, token, remoteIP, action string) error {
	service, err := GetDefaultService()
	if err != nil {
		return err
	}
	return service.Verify(ctx, token, remoteIP, action)
}
