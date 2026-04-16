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

// Package captcha_provider_hmac_challenge provides a built-in HMAC-based
// captcha provider.
//
// This provider generates and verifies challenge tokens locally
// using HMAC-SHA256 signatures without any external service
// dependency. It is suitable for development, testing, and
// integration test scenarios where the full captcha pipeline needs
// to be exercised but external provider credentials are unavailable.
//
// It is not intended for production bot protection. For production
// use, prefer Cloudflare Turnstile, Google reCAPTCHA v3, or hCaptcha
// via their respective provider packages.
//
// # Thread safety
//
// The provider returned by [NewProvider] is safe for concurrent use.
package captcha_provider_hmac_challenge
