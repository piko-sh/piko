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

// Package captcha_provider_turnstile provides a Cloudflare Turnstile
// captcha provider.
//
// This provider delegates verification to Cloudflare's Turnstile
// service (challenges.cloudflare.com), a privacy-friendly, free
// alternative to traditional CAPTCHAs. Turnstile uses non-interactive
// challenges and does not require users to solve puzzles.
//
// Turnstile returns a pass/fail outcome. On success the provider
// reports a normalised score of 1.0; on failure it reports 0.0.
// Tokens are valid for 5 minutes and are single-use. There is no
// official Go SDK, so this provider communicates with the Turnstile
// API via HTTP directly.
//
// Obtain a site key and secret key from the Cloudflare dashboard
// under Turnstile settings. The provider needs only these two
// credentials:
//
//	provider, err := captcha_provider_turnstile.NewProvider(captcha_provider_turnstile.Config{
//	    SiteKey:   "0x4AAAAAAAB...",
//	    SecretKey: "0x4AAAAAAAB...",
//	})
//
// # Thread safety
//
// All methods on the provider returned by [NewProvider] are safe for concurrent use.
package captcha_provider_turnstile
