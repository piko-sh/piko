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

package premailer

// Options holds settings for CSS transformation during email rendering.
type Options struct {
	// LinkQueryParams specifies query parameters to add to all links.
	LinkQueryParams map[string]string

	// Theme maps CSS variable names to their styling values.
	Theme map[string]string

	// ExternalCSS is additional CSS to include before processing; empty means none.
	ExternalCSS string

	// KeepBangImportant keeps CSS rules that have !important declarations.
	KeepBangImportant bool

	// RemoveClasses removes class attributes from all elements when true.
	RemoveClasses bool

	// RemoveIDs controls whether id attributes are stripped from all elements.
	RemoveIDs bool

	// MakeLeftoverImportant adds !important to leftover CSS rules that cannot be
	// inlined into elements.
	MakeLeftoverImportant bool

	// ExpandShorthands enables expansion of CSS shorthand properties into their
	// individual parts.
	ExpandShorthands bool

	// ResolvePseudoElements controls whether ::before and ::after pseudo-element
	// rules are collected separately instead of being discarded. When true,
	// pseudo-element rules are placed in RuleSet.PseudoElementRules rather than
	// LeftoverRules.
	ResolvePseudoElements bool

	// SkipEmailValidation disables email-specific HTML tag and CSS property
	// validation. Use this when processing templates for non-email output such
	// as PDF layout.
	SkipEmailValidation bool

	// SkipHTMLAttributeMapping disables the conversion of CSS properties back
	// to HTML attributes (e.g. width, bgcolor).
	//
	// Email clients need this mapping, but layout engines do not.
	SkipHTMLAttributeMapping bool

	// SkipStyleExtraction disables the removal of <style> tags from the AST
	// during processing. Use this when the AST should not be modified.
	SkipStyleExtraction bool
}

// Option is a functional option for configuring the Premailer.
type Option func(*Options)

// ToFunctionalOptions converts an Options struct into a slice of functional
// Option values, so a settings struct can be passed to a function that accepts
// functional options.
//
// Returns []Option which contains only options with non-default values.
func (o *Options) ToFunctionalOptions() []Option {
	var opts []Option

	if o.KeepBangImportant {
		opts = append(opts, WithKeepBangImportant(true))
	}
	if o.RemoveClasses {
		opts = append(opts, WithRemoveClasses(true))
	}
	if o.RemoveIDs {
		opts = append(opts, WithRemoveIDs(true))
	}
	if o.MakeLeftoverImportant {
		opts = append(opts, WithMakeLeftoverImportant(true))
	}
	if !o.ExpandShorthands {
		opts = append(opts, WithExpandShorthands(false))
	}
	if o.ResolvePseudoElements {
		opts = append(opts, WithResolvePseudoElements(true))
	}
	if o.SkipEmailValidation {
		opts = append(opts, WithSkipEmailValidation(true))
	}
	if o.SkipHTMLAttributeMapping {
		opts = append(opts, WithSkipHTMLAttributeMapping(true))
	}
	if o.SkipStyleExtraction {
		opts = append(opts, WithSkipStyleExtraction(true))
	}
	if o.LinkQueryParams != nil {
		opts = append(opts, WithLinkQueryParams(o.LinkQueryParams))
	}
	if o.Theme != nil {
		opts = append(opts, WithTheme(o.Theme))
	}
	if o.ExternalCSS != "" {
		opts = append(opts, WithExternalCSS(o.ExternalCSS))
	}

	return opts
}

// WithKeepBangImportant returns an Option that sets whether to keep
// !important declarations in both inline styles and the style block.
//
// When enabled, CSS rules with !important declarations are both inlined and
// kept in the <style> block. This creates dual-path styling that works well
// with webmail clients. The inline styles (without !important) provide basic
// rendering, while the <style> block rules (with !important) can override
// styles added by the webmail client.
//
// Takes keep (bool) which enables or disables !important preservation.
//
// Returns Option which sets the KeepBangImportant setting.
func WithKeepBangImportant(keep bool) Option {
	return func(o *Options) {
		o.KeepBangImportant = keep
	}
}

// WithRemoveClasses returns an Option that sets the RemoveClasses setting.
//
// When enabled, class attributes are removed from elements after styles are
// applied. This can reduce HTML size and prevent styling conflicts in email
// clients.
//
// Takes remove (bool) which specifies whether to remove class attributes.
//
// Returns Option which configures the RemoveClasses setting.
func WithRemoveClasses(remove bool) Option {
	return func(o *Options) {
		o.RemoveClasses = remove
	}
}

// WithRemoveIDs returns an Option that sets whether to remove ID attributes
// from elements after styles are applied.
//
// When enabled, ID attributes are removed from elements. This helps with email
// compatibility because webmail clients can have ID conflicts when emails are
// added to pages with elements that share the same IDs. Removing IDs also makes
// the HTML smaller.
//
// Takes remove (bool) which enables or disables ID removal.
//
// Returns Option which sets the RemoveIDs setting.
func WithRemoveIDs(remove bool) Option {
	return func(o *Options) {
		o.RemoveIDs = remove
	}
}

// WithMakeLeftoverImportant returns an Option that controls whether leftover
// CSS properties are marked as !important.
//
// When enabled, all CSS properties in leftover rules (non-inlineable styles
// like :hover, @media queries) are automatically marked as !important. This
// is needed for email clients like Gmail, which often only respect styles in
// <style> tags if they are marked !important.
//
// Takes makeImportant (bool) which enables or disables the !important marking.
//
// Returns Option which configures the MakeLeftoverImportant setting.
func WithMakeLeftoverImportant(makeImportant bool) Option {
	return func(o *Options) {
		o.MakeLeftoverImportant = makeImportant
	}
}

// WithExpandShorthands returns an Option that sets ExpandShorthands.
//
// When enabled, CSS shorthand properties are expanded into their longhand
// equivalents. This is critical for email compatibility, especially with
// Outlook, which has poor support for CSS shorthands like margin, padding,
// and border.
//
// Takes expand (bool) which enables or disables shorthand expansion.
//
// Returns Option which configures the ExpandShorthands setting.
func WithExpandShorthands(expand bool) Option {
	return func(o *Options) {
		o.ExpandShorthands = expand
	}
}

// WithLinkQueryParams returns an Option that sets query parameters to append
// to all HTTP/HTTPS links.
//
// Use it for email marketing analytics and campaign tracking such as UTM
// parameters.
//
// The function intelligently merges parameters:
//   - If the link already has query parameters, it appends with "&"
//   - If the link has no query parameters, it appends with "?"
//
// Non-HTTP links are automatically skipped:
//   - mailto: links
//   - tel: links
//   - JavaScript pseudo-protocols (javascript:)
//   - Anchor-only links (#section)
//
// Takes params (map[string]string) which specifies the query parameters to
// append to each link.
//
// Returns Option which configures the link query parameters on the Options.
func WithLinkQueryParams(params map[string]string) Option {
	return func(o *Options) {
		o.LinkQueryParams = params
	}
}

// WithTheme returns an Option that sets the CSS variable theme map.
//
// This map resolves var(--variable-name) functions into static values before
// inlining. This is critical for email compatibility as email clients cannot
// evaluate CSS variables at runtime.
//
// The keys should be variable names without the leading "--" prefix.
// The values can be static values or contain nested var() references.
//
// The resolver handles:
//   - Nested variables (e.g., var(--border-colour) -> var(--gray-200) -> #CAD1D8)
//   - Fallback values (e.g., var(--undefined, #FFF))
//   - Circular reference detection
//
// Takes theme (map[string]string) which maps CSS variable names to their
// values for resolution.
//
// Returns Option which configures the theme map on the premailer options.
func WithTheme(theme map[string]string) Option {
	return func(o *Options) {
		o.Theme = theme
	}
}

// WithResolvePseudoElements returns an Option that controls whether
// ::before and ::after pseudo-element rules are collected into
// RuleSet.PseudoElementRules instead of being discarded as leftover rules.
//
// Takes resolve (bool) which enables or disables pseudo-element collection.
//
// Returns Option which configures the ResolvePseudoElements setting.
func WithResolvePseudoElements(resolve bool) Option {
	return func(o *Options) {
		o.ResolvePseudoElements = resolve
	}
}

// WithSkipEmailValidation returns an Option that disables email-specific
// HTML tag and CSS property validation. Use this when processing templates
// for non-email output such as PDF layout.
//
// Takes skip (bool) which enables or disables the validation skip.
//
// Returns Option which configures the SkipEmailValidation setting.
func WithSkipEmailValidation(skip bool) Option {
	return func(o *Options) {
		o.SkipEmailValidation = skip
	}
}

// WithSkipHTMLAttributeMapping returns an Option that disables the conversion
// of CSS properties back to HTML attributes. Email clients need this mapping
// for properties like width and bgcolor, but layout engines do not.
//
// Takes skip (bool) which enables or disables the attribute mapping skip.
//
// Returns Option which configures the SkipHTMLAttributeMapping setting.
func WithSkipHTMLAttributeMapping(skip bool) Option {
	return func(o *Options) {
		o.SkipHTMLAttributeMapping = skip
	}
}

// WithSkipStyleExtraction returns an Option that prevents the removal of
// <style> tags from the AST during processing. Use this when the AST
// should not be modified, such as during layout resolution.
//
// Takes skip (bool) which enables or disables the style extraction skip.
//
// Returns Option which configures the SkipStyleExtraction setting.
func WithSkipStyleExtraction(skip bool) Option {
	return func(o *Options) {
		o.SkipStyleExtraction = skip
	}
}

// WithExternalCSS returns an Option that provides CSS from external sources.
//
// This CSS will be processed along with any CSS found in <style> tags within
// the AST. Use it when the template and styles are kept separate, such as
// in .pk SFC files where <style> and <template> sections are parsed on
// their own.
//
// Takes css (string) which contains the CSS rules to process.
//
// Returns Option which sets up the premailer to use the provided CSS.
func WithExternalCSS(css string) Option {
	return func(o *Options) {
		o.ExternalCSS = css
	}
}

// defaultOptions returns a new Options value with standard settings.
//
// Returns *Options which contains the default premailer settings.
func defaultOptions() *Options {
	return &Options{
		KeepBangImportant:        false,
		RemoveClasses:            false,
		RemoveIDs:                false,
		MakeLeftoverImportant:    false,
		ExpandShorthands:         true,
		ResolvePseudoElements:    false,
		SkipEmailValidation:      false,
		SkipHTMLAttributeMapping: false,
		SkipStyleExtraction:      false,
		LinkQueryParams:          nil,
		Theme:                    nil,
		ExternalCSS:              "",
	}
}

// applyOptions applies a list of option functions to an Options struct.
//
// Takes opts (...Option) which are functions that modify the default options.
//
// Returns *Options which contains the configured options after all functions
// have been applied.
func applyOptions(opts ...Option) *Options {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}
