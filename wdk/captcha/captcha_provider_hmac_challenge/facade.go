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

package captcha_provider_hmac_challenge

import (
	"piko.sh/piko/internal/captcha/captcha_adapters/hmac_challenge"
	"piko.sh/piko/internal/captcha/captcha_domain"
)

// DefaultTTL is the default challenge validity duration (5 minutes).
const DefaultTTL = hmac_challenge.DefaultTTL

var (
	// ErrSecretTooShort is returned when the HMAC secret key is shorter than
	// 16 bytes.
	ErrSecretTooShort = hmac_challenge.ErrSecretTooShort

	// ErrSecretEmpty is returned when the HMAC secret key is empty.
	ErrSecretEmpty = hmac_challenge.ErrSecretEmpty
)

// Config holds configuration for the HMAC challenge captcha provider.
type Config = hmac_challenge.Config

// NewProvider creates a new HMAC challenge captcha provider.
//
// The secret must be at least 16 bytes. The TTL defaults to 5 minutes if zero.
//
// Takes config (Config) which specifies the HMAC secret and token TTL.
//
// Returns captcha_domain.CaptchaProvider which provides HMAC-based captcha
// verification.
// Returns error when the configuration is invalid.
func NewProvider(config Config) (captcha_domain.CaptchaProvider, error) {
	return hmac_challenge.NewProvider(config)
}
