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
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/captcha/captcha_dto"
)

func TestDisabledService_Verify(t *testing.T) {
	service := NewDisabledCaptchaService()
	err := service.Verify(t.Context(), "token", "127.0.0.1", "submit")
	assert.ErrorIs(t, err, captcha_dto.ErrCaptchaDisabled)
}

func TestDisabledService_VerifyWithScore(t *testing.T) {
	service := NewDisabledCaptchaService()
	response, err := service.VerifyWithScore(t.Context(), "token", "127.0.0.1", "submit", 0.5)
	assert.ErrorIs(t, err, captcha_dto.ErrCaptchaDisabled)
	assert.Nil(t, response)
}

func TestDisabledService_IsEnabled(t *testing.T) {
	service := NewDisabledCaptchaService()
	assert.False(t, service.IsEnabled())
}

func TestDisabledService_SiteKey(t *testing.T) {
	service := NewDisabledCaptchaService()
	assert.Empty(t, service.SiteKey())
}

func TestDisabledService_ScriptURL(t *testing.T) {
	service := NewDisabledCaptchaService()
	assert.Empty(t, service.ScriptURL())
}

func TestDisabledService_HealthCheck(t *testing.T) {
	service := NewDisabledCaptchaService()
	assert.NoError(t, service.HealthCheck(t.Context()))
}

func TestDisabledService_ProviderManagement(t *testing.T) {
	service := NewDisabledCaptchaService()
	ctx := t.Context()

	assert.ErrorIs(t, service.RegisterProvider(ctx, "test", nil), captcha_dto.ErrCaptchaDisabled)
	assert.ErrorIs(t, service.SetDefaultProvider("test"), captcha_dto.ErrCaptchaDisabled)
	assert.Empty(t, service.GetProviders(ctx))
	assert.False(t, service.HasProvider("test"))
	assert.Empty(t, service.ListProviders(ctx))
}

func TestDisabledService_Close(t *testing.T) {
	service := NewDisabledCaptchaService()
	assert.NoError(t, service.Close(t.Context()))
}

func TestDisabledService_VerifyWithProvider(t *testing.T) {
	service := NewDisabledCaptchaService()
	_, err := service.VerifyWithProvider(t.Context(), "any", "token", "127.0.0.1", "submit", 0)
	assert.ErrorIs(t, err, captcha_dto.ErrCaptchaDisabled)
}

func TestDisabledService_GetDefaultProvider(t *testing.T) {
	service := NewDisabledCaptchaService()
	_, err := service.GetDefaultProvider(t.Context())
	assert.ErrorIs(t, err, captcha_dto.ErrCaptchaDisabled)
}

func TestDisabledService_GetProviderByName(t *testing.T) {
	service := NewDisabledCaptchaService()
	_, err := service.GetProviderByName(t.Context(), "any")
	assert.ErrorIs(t, err, captcha_dto.ErrCaptchaDisabled)
}
