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

package captcha_dto

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCaptchaError_Error(t *testing.T) {
	underlying := errors.New("connection refused")
	captchaErr := NewCaptchaError("verify", "turnstile", underlying)

	assert.Equal(t, "captcha verify failed with provider turnstile: connection refused", captchaErr.Error())
}

func TestCaptchaError_Unwrap(t *testing.T) {
	underlying := errors.New("connection refused")
	captchaErr := NewCaptchaError("verify", "turnstile", underlying)

	assert.ErrorIs(t, captchaErr, underlying)
}

func TestProviderType_String(t *testing.T) {
	assert.Equal(t, "turnstile", ProviderTypeTurnstile.String())
	assert.Equal(t, "hmac_challenge", ProviderTypeHMACChallenge.String())
}

func TestProviderType_IsValid(t *testing.T) {
	assert.True(t, ProviderTypeTurnstile.IsValid())
	assert.True(t, ProviderTypeRecaptchaV3.IsValid())
	assert.True(t, ProviderTypeHCaptcha.IsValid())
	assert.True(t, ProviderTypeHMACChallenge.IsValid())
	assert.False(t, ProviderType("unknown").IsValid())
}

func TestDefaultServiceConfig(t *testing.T) {
	config := DefaultServiceConfig()
	assert.Equal(t, 0.5, config.DefaultScoreThreshold)
}
