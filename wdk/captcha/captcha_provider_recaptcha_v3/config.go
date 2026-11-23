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

import (
	"errors"
	"math"
	"net/http"
	"time"

	"piko.sh/piko/internal/captcha/captcha_adapters/siteverify"
)

var (
	// ErrSiteKeyEmpty is returned when the site key is not set.
	ErrSiteKeyEmpty = errors.New("recaptcha: site key cannot be empty")

	// ErrSecretKeyEmpty is returned when the secret key is not set.
	ErrSecretKeyEmpty = errors.New("recaptcha: secret key cannot be empty")

	// ErrScoreMissing is returned when a successful verification carries no score and no
	// test key score is configured.
	ErrScoreMissing = errors.New("recaptcha: verification response has no score")

	// ErrTestKeyScoreOutOfRange is returned when the test key score lies outside 0.0 to 1.0.
	ErrTestKeyScoreOutOfRange = errors.New("recaptcha: test key score must be between 0 and 1")
)

// Config holds configuration for the Google reCAPTCHA v3 captcha provider.
type Config struct {
	// TestKeyScore supplies the score omitted by a successful Google test key response.
	//
	// The fallback applies only to Google's test key hostname. A nil fallback makes a
	// successful response without a score fail verification.
	TestKeyScore *float64

	// SiteKey is the public site key from the Google reCAPTCHA admin console. This is
	// embedded in the frontend widget.
	SiteKey string

	// SecretKey is the secret key from the Google reCAPTCHA admin console. This is used
	// server-side to verify tokens.
	SecretKey string
}

// Option configures how the provider calls the reCAPTCHA verification endpoint.
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
	if c.TestKeyScore != nil && (*c.TestKeyScore < 0 || *c.TestKeyScore > 1 || math.IsNaN(*c.TestKeyScore)) {
		return ErrTestKeyScoreOutOfRange
	}
	return nil
}

// WithVerifyTimeout bounds each call to the reCAPTCHA verification endpoint, including
// reading the response.
//
// Takes timeout (time.Duration) which is the limit; zero or negative keeps the 10 second
// default.
//
// Returns Option which applies the timeout.
func WithVerifyTimeout(timeout time.Duration) Option {
	return siteverify.WithTimeout(timeout)
}

// WithMaxResponseBytes caps the size of a reCAPTCHA verification response; a larger
// response is reported as the provider being unavailable rather than truncated.
//
// Takes limit (int64) which is the largest body accepted; zero or negative keeps the 64
// KiB default.
//
// Returns Option which applies the limit.
func WithMaxResponseBytes(limit int64) Option {
	return siteverify.WithMaxResponseBytes(limit)
}

// WithTransport sends reCAPTCHA verification requests through transport instead of the
// default transport, for example to route through an egress proxy.
//
// Takes transport (http.RoundTripper) which sends requests or is nil to keep the default.
//
// Returns Option which applies the transport.
func WithTransport(transport http.RoundTripper) Option {
	return siteverify.WithTransport(transport)
}
