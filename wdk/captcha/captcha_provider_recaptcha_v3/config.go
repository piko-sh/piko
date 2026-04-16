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

import "errors"

var (
	// ErrSiteKeyEmpty is returned when the site key is not set.
	ErrSiteKeyEmpty = errors.New("recaptcha: site key cannot be empty")

	// ErrSecretKeyEmpty is returned when the secret key is not set.
	ErrSecretKeyEmpty = errors.New("recaptcha: secret key cannot be empty")
)

// Config holds configuration for the Google reCAPTCHA v3 captcha provider.
type Config struct {
	// SiteKey is the public site key from the Google reCAPTCHA admin console.
	// This is embedded in the frontend widget.
	SiteKey string

	// SecretKey is the secret key from the Google reCAPTCHA admin console.
	// This is used server-side to verify tokens.
	SecretKey string
}

// validate checks that the configuration contains all required fields.
//
// Returns error when validation fails.
func (c *Config) validate() error {
	if c.SiteKey == "" {
		return ErrSiteKeyEmpty
	}
	if c.SecretKey == "" {
		return ErrSecretKeyEmpty
	}
	return nil
}
