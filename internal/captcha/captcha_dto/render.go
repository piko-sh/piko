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

// RenderRequirements describes what a captcha provider needs rendered on the
// page for the frontend widget to function. The renderer uses these to
// assemble the correct HTML, script tags, and CSP directives.
type RenderRequirements struct {
	// InitScript returns the static JavaScript init script content for this
	// provider, using data-attribute selectors to find and initialise all
	// captcha widget instances on the page; nil for providers that generate
	// tokens server-side and need no client JavaScript.
	InitScript func() (string, error)

	// ProviderType is the provider type string used in data-captcha-provider
	// attributes for the init script to match on.
	ProviderType string

	// ScriptURLs lists external provider SDK JavaScript files to load. Each
	// is added as a <script src="..." async defer></script> tag.
	ScriptURLs []string

	// CSPScriptDomains lists origins to allow in the script-src CSP directive.
	CSPScriptDomains []string

	// CSPFrameDomains lists origins to allow in the frame-src CSP directive
	// (for providers that render iframes).
	CSPFrameDomains []string

	// CSPConnectDomains lists origins to allow in the connect-src CSP
	// directive (for providers that make fetch/XHR calls).
	CSPConnectDomains []string

	// ServerSideToken indicates the provider generates tokens at render time
	// without client JavaScript (e.g. the HMAC challenge provider); when
	// true, the renderer calls GenerateChallenge on the provider and
	// pre-populates the hidden input value.
	ServerSideToken bool

	// Invisible indicates the provider has no visible widget. When true, the
	// container div is not rendered and the data attributes are placed on the
	// hidden input instead.
	Invisible bool
}
