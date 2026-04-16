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

// Package hmac_challenge provides a built-in captcha provider using
// HMAC-SHA256 signed challenge tokens.
//
// This adapter generates tokens without any external service dependency.
// It is suitable for development, testing, and integration test
// scenarios where a full captcha pipeline is needed but external
// provider credentials are unavailable.
//
// Each token is base64-encoded and contains a challenge ID, Unix
// timestamp, and action name. The token is signed with HMAC-SHA256.
// Verification checks the signature, validates the timestamp against a
// configurable TTL (default 5 minutes), enforces single-use via an
// in-memory map with automatic eviction, and optionally validates the
// action name against the expected binding.
//
// The provider also exposes an HTTP endpoint for frontend token
// generation, allowing the piko:captcha server-side element to work in test
// scenarios.
//
// Time operations use [clock.Clock] so that TTL and expiry behaviour
// can be controlled deterministically in tests.
//
// For production use, prefer one of the external provider packages
// available via the WDK.
//
// # Thread safety
//
// All [Provider] methods are safe for concurrent use.
package hmac_challenge
