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

// Directive represents a Content-Security-Policy directive name.
// Using a typed constant prevents typos and allows compile-time checking.
type Directive string

const (
	// DefaultSrc is the fallback directive used when other fetch directives are
	// not set.
	DefaultSrc Directive = "default-src"

	// ScriptSrc is the directive that controls which sources may run scripts.
	ScriptSrc Directive = "script-src"

	// ScriptSrcElem restricts the locations from which script elements may be
	// executed.
	ScriptSrcElem Directive = "script-src-elem"

	// ScriptSrcAttr restricts the locations from which inline script event
	// handlers may be executed.
	ScriptSrcAttr Directive = "script-src-attr"

	// StyleSrc is the directive that controls where stylesheets may load from.
	StyleSrc Directive = "style-src"

	// StyleSrcElem is the directive that controls where style elements may load
	// from.
	StyleSrcElem Directive = "style-src-elem"

	// StyleSrcAttr restricts the locations from which inline style attributes may
	// be applied.
	StyleSrcAttr Directive = "style-src-attr"

	// ImgSrc is the directive that controls where images can be loaded from.
	ImgSrc Directive = "img-src"

	// FontSrc is the directive that controls where fonts may be loaded from.
	FontSrc Directive = "font-src"

	// ConnectSrc is the directive that controls which URLs can be accessed using
	// script interfaces such as fetch, XMLHttpRequest, and WebSocket.
	ConnectSrc Directive = "connect-src"

	// MediaSrc is the directive that controls where video and audio can load from.
	MediaSrc Directive = "media-src"

	// ObjectSrc limits where plugins can be loaded from.
	ObjectSrc Directive = "object-src"

	// FrameSrc is the directive that controls which URLs may be used as sources
	// for frames.
	FrameSrc Directive = "frame-src"

	// ChildSrc is the directive that controls where workers and frames can load
	// from.
	ChildSrc Directive = "child-src"

	// WorkerSrc is the directive that controls where workers may be loaded from.
	WorkerSrc Directive = "worker-src"

	// ManifestSrc restricts the locations from which application manifests may be
	// loaded.
	ManifestSrc Directive = "manifest-src"

	// PrefetchSrc is the directive that limits where resources may be prefetched
	// from.
	PrefetchSrc Directive = "prefetch-src"

	// BaseURI is the CSP directive that controls which URLs can be used in a
	// document's base element.
	BaseURI Directive = "base-uri"

	// Sandbox is the directive name for enabling a sandbox on the requested
	// resource.
	Sandbox Directive = "sandbox"

	// FormAction restricts the URLs which can be used as the target of form
	// submissions.
	FormAction Directive = "form-action"

	// FrameAncestors controls which URLs can embed the page in a frame.
	FrameAncestors Directive = "frame-ancestors"

	// NavigateTo is a directive that limits which URLs a document can navigate to.
	NavigateTo Directive = "navigate-to"

	// ReportTo specifies a group to which violation reports are sent.
	ReportTo Directive = "report-to"

	// UpgradeInsecureRequests instructs user agents to treat all insecure URLs as
	// secure.
	UpgradeInsecureRequests Directive = "upgrade-insecure-requests"

	// BlockAllMixedContent prevents loading any mixed content.
	BlockAllMixedContent Directive = "block-all-mixed-content"

	// RequireTrustedTypesFor enforces Trusted Types at the DOM XSS injection
	// sinks.
	RequireTrustedTypesFor Directive = "require-trusted-types-for"

	// TrustedTypes is the directive that controls which Trusted Types policies
	// may be created.
	TrustedTypes Directive = "trusted-types"
)

// booleanDirectives is the set of directives that take no source values.
var booleanDirectives = map[Directive]bool{
	UpgradeInsecureRequests: true,
	BlockAllMixedContent:    true,
}

// isBooleanDirective returns true if the directive takes no source values.
//
// Takes d (Directive) which specifies the directive to check.
//
// Returns bool which is true when the directive is a boolean directive.
func isBooleanDirective(d Directive) bool {
	return booleanDirectives[d]
}
