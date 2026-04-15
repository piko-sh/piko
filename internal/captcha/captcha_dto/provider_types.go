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

// ProviderType identifies which captcha provider to use and implements
// fmt.Stringer.
type ProviderType string

const (
	// ProviderTypeHMACChallenge uses a built-in HMAC-based challenge for
	// testing and development. Not suitable for production use.
	ProviderTypeHMACChallenge ProviderType = "hmac_challenge"

	// ProviderTypeTurnstile uses Cloudflare Turnstile for bot detection.
	// Privacy-friendly, free, and suitable for production.
	ProviderTypeTurnstile ProviderType = "turnstile"

	// ProviderTypeRecaptchaV3 uses Google reCAPTCHA v3 for score-based bot
	// detection. Returns a risk score between 0.0 and 1.0.
	ProviderTypeRecaptchaV3 ProviderType = "recaptcha_v3"

	// ProviderTypeHCaptcha uses hCaptcha for bot detection. Privacy-focused
	// alternative to reCAPTCHA with optional challenge support.
	ProviderTypeHCaptcha ProviderType = "hcaptcha"
)

// String returns the string representation of the provider type.
//
// Returns string which is the provider type as a plain string value.
func (p ProviderType) String() string {
	return string(p)
}

// IsValid reports whether the provider type is a known value.
//
// Returns bool which is true if the provider type matches a known provider.
func (p ProviderType) IsValid() bool {
	switch p {
	case ProviderTypeHMACChallenge,
		ProviderTypeTurnstile,
		ProviderTypeRecaptchaV3,
		ProviderTypeHCaptcha:
		return true
	default:
		return false
	}
}
