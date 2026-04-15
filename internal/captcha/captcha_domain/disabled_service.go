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

	"piko.sh/piko/internal/captcha/captcha_dto"
	"piko.sh/piko/internal/provider/provider_domain"
)

// DisabledCaptchaService implements CaptchaServicePort but returns errors for
// verification operations. It is used when no captcha provider is configured,
// allowing the application to start without captcha while giving clear errors
// if verification is attempted.
type DisabledCaptchaService struct{}

var _ CaptchaServicePort = (*DisabledCaptchaService)(nil)

// NewDisabledCaptchaService creates a new disabled captcha service.
//
// Returns *DisabledCaptchaService which is a no-op captcha implementation.
func NewDisabledCaptchaService() *DisabledCaptchaService {
	return &DisabledCaptchaService{}
}

// Verify returns ErrCaptchaDisabled since captcha is not configured.
//
// Returns error which is always ErrCaptchaDisabled.
func (*DisabledCaptchaService) Verify(_ context.Context, _, _, _ string) error {
	return captcha_dto.ErrCaptchaDisabled
}

// VerifyWithScore returns ErrCaptchaDisabled since captcha is not configured.
//
// Returns *captcha_dto.VerifyResponse which is always nil.
// Returns error which is always ErrCaptchaDisabled.
func (*DisabledCaptchaService) VerifyWithScore(_ context.Context, _, _, _ string, _ float64) (*captcha_dto.VerifyResponse, error) {
	return nil, captcha_dto.ErrCaptchaDisabled
}

// VerifyWithProvider returns ErrCaptchaDisabled since captcha is not configured.
//
// Returns *captcha_dto.VerifyResponse which is always nil.
// Returns error which is always ErrCaptchaDisabled.
func (*DisabledCaptchaService) VerifyWithProvider(_ context.Context, _, _, _, _ string, _ float64) (*captcha_dto.VerifyResponse, error) {
	return nil, captcha_dto.ErrCaptchaDisabled
}

// SiteKey returns an empty string since no provider is configured.
//
// Returns string which is always empty.
func (*DisabledCaptchaService) SiteKey() string {
	return ""
}

// ScriptURL returns an empty string since no provider is configured.
//
// Returns string which is always empty.
func (*DisabledCaptchaService) ScriptURL() string {
	return ""
}

// GetDefaultProvider returns ErrCaptchaDisabled since no provider is configured.
//
// Returns CaptchaProvider which is always nil.
// Returns error which is always ErrCaptchaDisabled.
func (*DisabledCaptchaService) GetDefaultProvider(_ context.Context) (CaptchaProvider, error) {
	return nil, captcha_dto.ErrCaptchaDisabled
}

// GetProviderByName returns ErrCaptchaDisabled since no provider is configured.
//
// Returns CaptchaProvider which is always nil.
// Returns error which is always ErrCaptchaDisabled.
func (*DisabledCaptchaService) GetProviderByName(_ context.Context, _ string) (CaptchaProvider, error) {
	return nil, captcha_dto.ErrCaptchaDisabled
}

// IsEnabled returns false since captcha is not configured.
//
// Returns bool which is always false.
func (*DisabledCaptchaService) IsEnabled() bool {
	return false
}

// RegisterProvider returns ErrCaptchaDisabled since provider registration is
// not supported when captcha is disabled.
//
// Returns error which is always ErrCaptchaDisabled.
func (*DisabledCaptchaService) RegisterProvider(_ context.Context, _ string, _ CaptchaProvider) error {
	return captcha_dto.ErrCaptchaDisabled
}

// SetDefaultProvider returns ErrCaptchaDisabled since provider management is
// not supported when captcha is disabled.
//
// Returns error which is always ErrCaptchaDisabled.
func (*DisabledCaptchaService) SetDefaultProvider(_ string) error {
	return captcha_dto.ErrCaptchaDisabled
}

// GetProviders returns an empty list since no providers are registered.
//
// Returns []string which is always empty.
func (*DisabledCaptchaService) GetProviders(_ context.Context) []string {
	return nil
}

// HasProvider returns false since no providers are registered.
//
// Returns bool which is always false.
func (*DisabledCaptchaService) HasProvider(_ string) bool {
	return false
}

// ListProviders returns an empty list since no providers are registered.
//
// Returns []provider_domain.ProviderInfo which is always empty.
func (*DisabledCaptchaService) ListProviders(_ context.Context) []provider_domain.ProviderInfo {
	return nil
}

// HealthCheck returns nil since the disabled service is healthy in the sense
// that it correctly reports its disabled state.
//
// Returns error which is always nil.
func (*DisabledCaptchaService) HealthCheck(_ context.Context) error {
	return nil
}

// Close is a no-op for the disabled service since there are no providers to
// shut down.
//
// Returns error which is always nil.
func (*DisabledCaptchaService) Close(_ context.Context) error {
	return nil
}
