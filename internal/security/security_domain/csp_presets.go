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

// WithStrictPolicy configures a strict CSP policy following Google's
// recommendations, using per-request tokens and 'strict-dynamic' for strong
// XSS protection while allowing dynamically loaded scripts.
//
// The policy includes:
//   - default-src 'self'
//   - script-src 'strict-dynamic' {{REQUEST_TOKEN}} (token-based)
//   - style-src 'self' {{REQUEST_TOKEN}} (token-based)
//   - object-src 'none' (blocks plugins like Flash)
//   - base-uri 'self' (prevents base tag hijacking)
//   - frame-ancestors 'self' (clickjacking protection)
//   - upgrade-insecure-requests
//
// Returns *CSPBuilder which allows method chaining for further configuration.
//
// IMPORTANT: This policy uses request tokens. Templates must use
// {{ .CSPTokenAttr }} on inline script and style elements:
// <script {{ .CSPTokenAttr }}>console.log("safe");</script>
// <style {{ .CSPTokenAttr }}>.my-class { color: red; }</style>
func (b *CSPBuilder) WithStrictPolicy() *CSPBuilder {
	return b.
		DefaultSrc(Self).
		ScriptSrc(StrictDynamic, RequestTokenPlaceholder).
		StyleSrc(Self, RequestTokenPlaceholder).
		ObjectSrc(None).
		BaseURI(Self).
		FrameAncestors(Self).
		UpgradeInsecureRequests()
}

// WithRelaxedPolicy configures a permissive CSP policy for legacy applications.
//
// Returns *CSPBuilder which allows method chaining.
//
// WARNING: This policy allows 'unsafe-inline' and 'unsafe-eval', which
// significantly reduces XSS protection. Use only when migrating legacy code
// that cannot be updated to use token-based CSP or content hashes.
//
// The policy includes:
//   - default-src 'self'
//   - script-src 'self' 'unsafe-inline' 'unsafe-eval'
//   - style-src 'self' 'unsafe-inline'
//   - img-src 'self' data: https:
//   - font-src 'self' data:
//   - connect-src 'self'
//   - frame-ancestors 'self' (clickjacking protection)
func (b *CSPBuilder) WithRelaxedPolicy() *CSPBuilder {
	return b.
		DefaultSrc(Self).
		ScriptSrc(Self, UnsafeInline, UnsafeEval).
		StyleSrc(Self, UnsafeInline).
		ImgSrc(Self, Data, HTTPS).
		FontSrc(Self, Data).
		ConnectSrc(Self).
		FrameAncestors(Self)
}

// WithAPIPolicy configures a minimal CSP policy for JSON API servers.
// This policy blocks all resource types, which protects against cases where
// the API accidentally serves HTML content (e.g., error pages with user input).
//
// The policy includes:
//   - default-src 'none' (blocks everything by default)
//   - frame-ancestors 'none' (cannot be embedded in frames)
//   - base-uri 'none' (no base element allowed)
//   - form-action 'none' (no form submissions)
//
// This is ideal for API endpoints that should only return JSON data.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) WithAPIPolicy() *CSPBuilder {
	return b.
		DefaultSrc(None).
		FrameAncestors(None).
		BaseURI(None).
		FormAction(None)
}
