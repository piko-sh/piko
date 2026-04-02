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

// Package security_domain defines the core security abstractions, services,
// and policies for the Piko framework.
//
// It handles CSRF protection using the Double Submit Cookie pattern,
// request rate limiting, and a fluent Content-Security-Policy builder.
// Port interfaces define the contracts that adapters must satisfy.
//
// # CSRF protection
//
// The CSRF implementation uses the Double Submit Cookie pattern where
// tokens are bound to a browser session cookie rather than
// timestamp-based expiry. This provides strong protection while
// allowing framework users to integrate with their own session
// systems via [CSRFCookieSourceAdapter].
//
//	service, err := security_domain.NewCSRFTokenService(
//	    config, binder, cookieSource,
//	)
//	pair, err := service.GenerateCSRFPair(w, r, buffer)
//	valid, err := service.ValidateCSRFPair(r, ephemeral, actionToken)
//
// # Content-Security-Policy
//
// The [CSPBuilder] provides a fluent API for constructing CSP headers
// with type-safe [Directive] and [Source] constants. Preset policies
// are available for common use cases:
//
//	policy := security_domain.NewCSPBuilder().
//	    WithPikoDefaults().
//	    Build()
//
//	strict := security_domain.NewCSPBuilder().
//	    WithStrictPolicy().
//	    Build()
//
// # Thread safety
//
// All service methods are safe for concurrent use. The CSRF service
// uses pooled HMAC instances and builders internally to minimise
// allocations under high concurrency. [CSPBuilder] instances are not
// safe for concurrent use; call [CSPBuilder.Clone] or build separate
// instances per goroutine.
package security_domain
