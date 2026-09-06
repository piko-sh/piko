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

package captcha_provider_hcaptcha

import (
	"errors"
	"time"

	"piko.sh/piko/internal/captcha/captcha_adapters/siteverify"
)

var (
	// ErrSiteKeyEmpty is returned when the hCaptcha site key is empty.
	ErrSiteKeyEmpty = errors.New("hcaptcha: site key cannot be empty")

	// ErrSecretKeyEmpty is returned when the hCaptcha secret key is empty.
	ErrSecretKeyEmpty = errors.New("hcaptcha: secret key cannot be empty")
)

// Config holds configuration for the hCaptcha captcha provider.
type Config struct {
	// SiteKey is the public site key from the hCaptcha dashboard. This is embedded in the
	// frontend widget to identify the site.
	SiteKey string

	// SecretKey is the secret key from the hCaptcha dashboard. This is used server-side to
	// verify captcha tokens with the hCaptcha API.
	SecretKey string
}

// Option configures how the provider calls the hCaptcha verification endpoint.
type Option = siteverify.Option

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

// WithVerifyTimeout bounds each call to the hCaptcha verification endpoint, including
// reading the response.
//
// Takes timeout (time.Duration) which is the limit; zero or negative keeps the 10 second
// default.
//
// Returns Option which applies the timeout.
func WithVerifyTimeout(timeout time.Duration) Option {
	return siteverify.WithTimeout(timeout)
}

// WithMaxResponseBytes caps the size of a hCaptcha verification response; a larger
// response is reported as the provider being unavailable rather than truncated.
//
// Takes limit (int64) which is the largest body accepted; zero or negative keeps the 64
// KiB default.
//
// Returns Option which applies the limit.
func WithMaxResponseBytes(limit int64) Option {
	return siteverify.WithMaxResponseBytes(limit)
}
