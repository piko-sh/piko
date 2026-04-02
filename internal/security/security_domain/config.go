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

package security_domain

import "time"

// SecurityConfig holds the settings for security services.
type SecurityConfig struct {
	// HMACSecretKey is the secret key used to sign and verify CSRF tokens.
	// Must not be empty; an empty key causes service creation to fail.
	HMACSecretKey []byte

	// CSRFTokenMaxAge is the safety-net maximum age for CSRF tokens,
	// providing a generous fallback (default: 30 days) to catch truly
	// ancient tokens if a cookie somehow persists indefinitely.
	//
	// With the Double Submit Cookie pattern, the primary expiry mechanism
	// is cookie rotation. 0 means use the default (30 days).
	CSRFTokenMaxAge time.Duration

	// CSRFCookieMaxAge is how long the CSRF cookie lasts, where 0
	// means session cookie (deleted when browser closes).
	//
	// Set to a positive value for persistent cookies (e.g., 30 days).
	CSRFCookieMaxAge time.Duration
}
