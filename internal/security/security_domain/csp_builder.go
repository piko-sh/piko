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

import (
	"strings"

	"piko.sh/piko/internal/security/security_dto"
)

// defaultDirectiveCapacity is the initial capacity for the order slice in
// CSPBuilder. Most CSP policies use 4-8 directives, so 8 is a reasonable
// starting point.
const defaultDirectiveCapacity = 8

// CSPBuilder provides a fluent API for constructing Content-Security-Policy
// headers. It supports all standard CSP directives, report-only mode, and
// dynamic per-request tokens for strict CSP policies.
type CSPBuilder struct {
	// directives maps each directive to its allowed sources; nil value indicates
	// a boolean directive with no sources.
	directives map[Directive][]Source

	// order maintains the insertion order of directives for consistent output.
	order []Directive

	// reportOnly uses Content-Security-Policy-Report-Only header instead of
	// enforcing.
	reportOnly bool

	// usesTokens indicates whether dynamic request tokens are used in the policy.
	usesTokens bool
}

// NewCSPBuilder creates a new CSP builder with no directives configured.
//
// Returns *CSPBuilder which is ready for directive configuration.
func NewCSPBuilder() *CSPBuilder {
	return &CSPBuilder{
		directives: make(map[Directive][]Source),
		order:      make([]Directive, 0, defaultDirectiveCapacity),
		reportOnly: false,
		usesTokens: false,
	}
}

// ReportOnly configures the builder to produce a
// Content-Security-Policy-Report-Only header instead of an enforcing
// Content-Security-Policy header, allowing policies to be tested before
// enforcement.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ReportOnly() *CSPBuilder {
	b.reportOnly = true
	return b
}

// IsReportOnly returns true if this builder is configured for report-only mode.
//
// Returns bool which indicates whether report-only mode is enabled.
func (b *CSPBuilder) IsReportOnly() bool {
	return b.reportOnly
}

// UsesRequestTokens returns true if this builder uses dynamic per-request
// tokens. When true, the security middleware must replace the
// RequestTokenPlaceholder with a unique cryptographic token for each request.
//
// Returns bool which indicates whether per-request tokens are in use.
func (b *CSPBuilder) UsesRequestTokens() bool {
	return b.usesTokens
}

// Add appends sources to a directive using the generic API.
// Use it when the directive is determined at runtime.
//
// Takes directive (Directive) which specifies the CSP directive to modify.
// Takes sources (...Source) which provides the source values to add.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) Add(directive Directive, sources ...Source) *CSPBuilder {
	if isBooleanDirective(directive) {
		return b.addBoolean(directive)
	}
	return b.addSources(directive, sources...)
}

// DefaultSrc sets the default-src directive, which is the fallback for
// other fetch directives.
//
// Takes sources (...Source) which specifies the allowed content sources.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) DefaultSrc(sources ...Source) *CSPBuilder {
	return b.addSources(DefaultSrc, sources...)
}

// ScriptSrc sets the script-src directive, controlling where scripts can be
// loaded from.
//
// Takes sources (...Source) which specifies the allowed script origins.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ScriptSrc(sources ...Source) *CSPBuilder {
	return b.addSources(ScriptSrc, sources...)
}

// ScriptSrcElem sets the script-src-elem directive for script elements.
//
// Takes sources (...Source) which specifies the allowed sources for script
// elements.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ScriptSrcElem(sources ...Source) *CSPBuilder {
	return b.addSources(ScriptSrcElem, sources...)
}

// ScriptSrcAttr sets the script-src-attr directive for inline event handlers.
//
// Takes sources (...Source) which specifies the allowed sources for inline
// event handlers.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ScriptSrcAttr(sources ...Source) *CSPBuilder {
	return b.addSources(ScriptSrcAttr, sources...)
}

// StyleSrc sets the style-src directive, controlling where styles can be
// loaded from.
//
// Takes sources (...Source) which specifies the allowed style sources.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) StyleSrc(sources ...Source) *CSPBuilder {
	return b.addSources(StyleSrc, sources...)
}

// StyleSrcElem sets the style-src-elem directive for style elements.
//
// Takes sources (...Source) which specifies the allowed sources for style
// elements.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) StyleSrcElem(sources ...Source) *CSPBuilder {
	return b.addSources(StyleSrcElem, sources...)
}

// StyleSrcAttr sets the style-src-attr directive for inline style attributes.
//
// Takes sources (...Source) which specifies the allowed sources for inline
// style attributes.
//
// Returns *CSPBuilder which allows for method chaining.
func (b *CSPBuilder) StyleSrcAttr(sources ...Source) *CSPBuilder {
	return b.addSources(StyleSrcAttr, sources...)
}

// ImgSrc sets the img-src directive, controlling where images can be loaded
// from.
//
// Takes sources (...Source) which specifies the allowed image sources.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ImgSrc(sources ...Source) *CSPBuilder {
	return b.addSources(ImgSrc, sources...)
}

// FontSrc sets the font-src directive, controlling where fonts can be loaded
// from.
//
// Takes sources (...Source) which specifies the allowed font origins.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) FontSrc(sources ...Source) *CSPBuilder {
	return b.addSources(FontSrc, sources...)
}

// ConnectSrc sets the connect-src directive, controlling fetch, XHR, and
// WebSocket URLs.
//
// Takes sources (...Source) which specifies the allowed connection origins.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ConnectSrc(sources ...Source) *CSPBuilder {
	return b.addSources(ConnectSrc, sources...)
}

// MediaSrc sets the media-src directive, controlling where audio and video
// can be loaded from.
//
// Takes sources (...Source) which specifies the allowed origins for media.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) MediaSrc(sources ...Source) *CSPBuilder {
	return b.addSources(MediaSrc, sources...)
}

// ObjectSrc sets the object-src directive, controlling where plugins can be
// loaded from.
//
// Takes sources (...Source) which specifies the allowed plugin origins.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ObjectSrc(sources ...Source) *CSPBuilder {
	return b.addSources(ObjectSrc, sources...)
}

// FrameSrc sets the frame-src directive, controlling where frames can be
// loaded from.
//
// Takes sources (...Source) which specifies the allowed frame origins.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) FrameSrc(sources ...Source) *CSPBuilder {
	return b.addSources(FrameSrc, sources...)
}

// ChildSrc sets the child-src directive, controlling workers and frames.
//
// Takes sources (...Source) which specifies the allowed sources.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ChildSrc(sources ...Source) *CSPBuilder {
	return b.addSources(ChildSrc, sources...)
}

// WorkerSrc sets the worker-src directive, controlling where workers can be
// loaded from.
//
// Takes sources (...Source) which specifies the allowed worker origins.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) WorkerSrc(sources ...Source) *CSPBuilder {
	return b.addSources(WorkerSrc, sources...)
}

// ManifestSrc sets the manifest-src directive for application manifests.
//
// Takes sources (...Source) which specifies the allowed manifest sources.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) ManifestSrc(sources ...Source) *CSPBuilder {
	return b.addSources(ManifestSrc, sources...)
}

// PrefetchSrc sets the prefetch-src directive for prefetched resources.
//
// Takes sources (...Source) which specifies the allowed sources for
// prefetching.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) PrefetchSrc(sources ...Source) *CSPBuilder {
	return b.addSources(PrefetchSrc, sources...)
}

// BaseURI sets the base-uri directive, restricting URLs for the base element.
//
// Takes sources (...Source) which specifies the allowed URLs for base elements.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) BaseURI(sources ...Source) *CSPBuilder {
	return b.addSources(BaseURI, sources...)
}

// FormAction sets the form-action directive, restricting form submission URLs.
//
// Takes sources (...Source) which specifies the allowed form submission URLs.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) FormAction(sources ...Source) *CSPBuilder {
	return b.addSources(FormAction, sources...)
}

// FrameAncestors sets the frame-ancestors directive, controlling who can
// embed this page.
//
// Takes sources (...Source) which specifies the allowed embedding origins.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) FrameAncestors(sources ...Source) *CSPBuilder {
	return b.addSources(FrameAncestors, sources...)
}

// ReportToDirective sets the report-to directive for violation reports
// (CSP Level 3).
//
// Takes groupName (string) which specifies the reporting group name.
//
// Returns *CSPBuilder which allows for method chaining.
func (b *CSPBuilder) ReportToDirective(groupName string) *CSPBuilder {
	return b.addSources(ReportTo, Source(groupName))
}

// UpgradeInsecureRequests adds the upgrade-insecure-requests directive,
// which instructs browsers to treat all insecure URLs as secure.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) UpgradeInsecureRequests() *CSPBuilder {
	return b.addBoolean(UpgradeInsecureRequests)
}

// BlockAllMixedContent adds the block-all-mixed-content directive, which
// prevents loading any mixed (HTTP on HTTPS page) content.
//
// Returns *CSPBuilder which allows for method chaining.
func (b *CSPBuilder) BlockAllMixedContent() *CSPBuilder {
	return b.addBoolean(BlockAllMixedContent)
}

// Sandbox sets the sandbox directive with the specified tokens.
//
// Pass no tokens for maximum restrictions (outputs just "sandbox"). Tokens are
// not quoted in the output (unlike keywords like 'self'). The sandbox directive
// restricts a page to a sandboxed environment where various capabilities are
// disabled. Each token explicitly enables a capability that would otherwise be
// restricted.
//
// Takes tokens (...SandboxToken) which specifies the capabilities to enable.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) Sandbox(tokens ...SandboxToken) *CSPBuilder {
	if len(tokens) == 0 {
		return b.addBoolean(Sandbox)
	}
	sources := make([]Source, len(tokens))
	for i, t := range tokens {
		sources[i] = Source(t)
	}
	return b.addSources(Sandbox, sources...)
}

// RequireTrustedTypesFor enables Trusted Types enforcement at DOM XSS injection
// sinks.
//
// This directive only accepts the 'script' keyword, which is added
// automatically. When enabled, browsers will require the use of Trusted Types
// for dangerous DOM operations.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) RequireTrustedTypesFor() *CSPBuilder {
	return b.addSources(RequireTrustedTypesFor, Script)
}

// TrustedTypes sets the trusted-types directive with the specified policy
// names.
//
// Policy names are NOT quoted in the output. Use PolicyName to create them.
// Pass Wildcard to allow creating policies with any unique names.
// Pass None to explicitly forbid all policy creation.
//
// Takes sources (...Source) which specifies the policy names or special values.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) TrustedTypes(sources ...Source) *CSPBuilder {
	return b.addSources(TrustedTypes, sources...)
}

// TrustedTypesWithDuplicates sets the trusted-types directive with the
// specified policy names and allows duplicate policy names (via
// 'allow-duplicates').
//
// Takes policies (...Source) which specifies the policy names to allow.
//
// Returns *CSPBuilder which allows for method chaining.
func (b *CSPBuilder) TrustedTypesWithDuplicates(policies ...Source) *CSPBuilder {
	sources := make([]Source, 0, len(policies)+1)
	sources = append(sources, policies...)
	sources = append(sources, AllowDuplicates)
	return b.addSources(TrustedTypes, sources...)
}

// TrustedTypesNone sets trusted-types to 'none', explicitly forbidding all
// Trusted Types policy creation.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) TrustedTypesNone() *CSPBuilder {
	return b.addSources(TrustedTypes, None)
}

// WithPikoDefaults configures the builder with Piko's recommended default
// CSP policy. This provides a secure baseline that works with Piko's built-in
// features including font loading, inline styles, and the async font loader
// script.
//
// The default policy is:
//   - default-src 'self'
//   - style-src 'self' 'unsafe-inline' https://fonts.googleapis.com
//   - script-src-attr 'unsafe-hashes' 'sha256-...' (for font loader)
//   - font-src 'self' https://fonts.gstatic.com data:
//   - img-src 'self' data: blob: https:
//   - connect-src 'self'
//
// Returns *CSPBuilder which allows further method chaining.
func (b *CSPBuilder) WithPikoDefaults() *CSPBuilder {
	return b.
		DefaultSrc(Self).
		StyleSrc(Self, UnsafeInline, Host("https://fonts.googleapis.com")).
		ScriptSrcAttr(UnsafeHashes, SHA256("1jAmyYXcRq6zFldLe/GCgIDJBiOONdXjTLgEFMDnDSM=")).
		FontSrc(Self, Host("https://fonts.gstatic.com"), Data).
		ImgSrc(Self, Data, Blob, HTTPS).
		ConnectSrc(Self)
}

// Build produces the final CSP header value string. Keywords are automatically
// single-quoted as required by the CSP specification.
//
// Returns string which contains the formatted header value, or empty if no
// directives were added.
func (b *CSPBuilder) Build() string {
	if len(b.order) == 0 {
		return ""
	}

	parts := make([]string, 0, len(b.order))
	for _, directive := range b.order {
		sources := b.directives[directive]
		if sources == nil {
			parts = append(parts, string(directive))
			continue
		}

		sourceParts := make([]string, 0, len(sources))
		for _, s := range sources {
			sourceParts = append(sourceParts, formatSourceForHeader(s))
		}
		parts = append(parts, string(directive)+" "+strings.Join(sourceParts, " "))
	}
	return strings.Join(parts, "; ")
}

// HeaderName returns the appropriate HTTP header name based on whether
// report-only mode is enabled.
//
// Returns string which is either "Content-Security-Policy-Report-Only" or
// "Content-Security-Policy".
func (b *CSPBuilder) HeaderName() string {
	if b.reportOnly {
		return "Content-Security-Policy-Report-Only"
	}
	return "Content-Security-Policy"
}

// Clone creates a deep copy of the builder, allowing modifications without
// affecting the original.
//
// Returns *CSPBuilder which is an independent copy with its own directives.
func (b *CSPBuilder) Clone() *CSPBuilder {
	clone := &CSPBuilder{
		directives: make(map[Directive][]Source, len(b.directives)),
		order:      make([]Directive, len(b.order)),
		reportOnly: b.reportOnly,
		usesTokens: b.usesTokens,
	}
	copy(clone.order, b.order)
	for k, v := range b.directives {
		if v == nil {
			clone.directives[k] = nil
		} else {
			clone.directives[k] = append([]Source(nil), v...)
		}
	}
	return clone
}

// RuntimeConfig builds and returns an immutable CSPRuntimeConfig from this
// builder. This should be called once at startup and the result passed to
// middleware.
//
// Returns security_dto.CSPRuntimeConfig which contains the built policy and
// configuration flags ready for use by middleware.
func (b *CSPBuilder) RuntimeConfig() security_dto.CSPRuntimeConfig {
	return security_dto.CSPRuntimeConfig{
		Policy:            b.Build(),
		ReportOnly:        b.reportOnly,
		UsesRequestTokens: b.usesTokens,
	}
}

// addSources is the internal method for adding sources to a directive.
//
// Takes directive (Directive) which specifies the CSP directive to modify.
// Takes sources (...Source) which provides the sources to add to the directive.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) addSources(directive Directive, sources ...Source) *CSPBuilder {
	if _, exists := b.directives[directive]; !exists {
		b.order = append(b.order, directive)
	}
	b.directives[directive] = append(b.directives[directive], sources...)

	for _, s := range sources {
		if s == RequestTokenPlaceholder {
			b.usesTokens = true
		}
	}
	return b
}

// addBoolean adds a boolean directive (one that takes no source values).
//
// Takes directive (Directive) which specifies the directive to add.
//
// Returns *CSPBuilder which allows method chaining.
func (b *CSPBuilder) addBoolean(directive Directive) *CSPBuilder {
	if _, exists := b.directives[directive]; !exists {
		b.order = append(b.order, directive)
		b.directives[directive] = nil
	}
	return b
}
