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

package sfcparser

import "strings"

const (
	// MimeJavaScript is the standard MIME type for JavaScript content.
	MimeJavaScript = "application/javascript"

	// MimeTypeScript is the MIME type for TypeScript content.
	MimeTypeScript = "application/typescript"

	// MimeGo is the MIME type for Go source files.
	MimeGo = "application/x-go"

	// MimeGoShort is the short form MIME type for Go source code.
	MimeGoShort = "application/go"

	// attributeLang is the attribute key for the script language.
	attributeLang = "lang"

	// attributeType is the attribute key for the script MIME type.
	attributeType = "type"
)

// Location represents a position in a source file.
type Location struct {
	// Line is the 1-based line number; 0 means a synthetic location.
	Line int

	// Column is the 1-based column number within the line.
	Column int
}

// Script represents a script block with its content, attributes, and location
// information.
type Script struct {
	// Attributes holds key-value pairs from the script tag, such as "type",
	// "lang", and "name".
	Attributes map[string]string

	// Content is the raw text inside the script block.
	Content string

	// Location specifies the position of the script in the source file.
	Location Location

	// ContentLocation is the position where the script content starts.
	ContentLocation Location
}

// Type returns the MIME type of the script from its type attribute.
// Defaults to MimeJavaScript if no type attribute is present.
//
// Returns string which is the MIME type for this script element.
func (s *Script) Type() string {
	scriptType, ok := s.Attributes["type"]
	if !ok || scriptType == "" {
		return MimeJavaScript
	}
	return scriptType
}

// Style represents a style block with its content, attributes, and location
// information.
type Style struct {
	// Attributes holds key-value pairs from the style tag (e.g. "global",
	// "scoped").
	Attributes map[string]string

	// Content holds the raw CSS text inside the style block.
	Content string

	// Location specifies where this style block starts in the source file.
	Location Location

	// ContentLocation is the line and column where the style content starts.
	ContentLocation Location
}

// I18nBlock represents an internationalisation block with its content,
// attributes, and location information.
type I18nBlock struct {
	// Attributes holds the key-value pairs from the i18n tag, such as "lang".
	Attributes map[string]string

	// Content holds the raw text inside the i18n block.
	Content string

	// Location specifies the position in the source file.
	Location Location

	// ContentLocation specifies where the content begins.
	ContentLocation Location
}

// TimelineBlock represents a piko:timeline block with its content, attributes,
// and location information. Multiple timeline blocks are allowed per component,
// each optionally targeting a different viewport via a media attribute.
type TimelineBlock struct {
	// Attributes holds key-value pairs from the piko:timeline tag.
	Attributes map[string]string

	// Content holds the raw markup inside the timeline block.
	Content string

	// Location specifies the position of the tag in the source file.
	Location Location

	// ContentLocation specifies where the timeline content begins.
	ContentLocation Location
}

// ParseResult holds all the parsed blocks from a single-file component.
type ParseResult struct {
	// Template holds the raw HTML content from the template block.
	Template string

	// TemplateAttributes holds key-value pairs from the template tag.
	TemplateAttributes map[string]string

	// Scripts holds all script elements found in the document.
	Scripts []Script

	// Styles holds the parsed style blocks from the component.
	Styles []Style

	// I18nBlocks holds the i18n translation blocks from the SFC.
	I18nBlocks []I18nBlock

	// Timelines holds the parsed piko:timeline blocks. Multiple blocks are
	// supported, each optionally targeting a different viewport via a media
	// attribute on the tag.
	Timelines []TimelineBlock

	// TemplateLocation is where the template tag starts in the source file.
	TemplateLocation Location

	// TemplateContentLocation is where the template content starts.
	TemplateContentLocation Location
}

// IsJavaScript checks if the script block is JavaScript or TypeScript by
// examining its type or lang attributes. TypeScript is treated as JavaScript
// since it compiles to JavaScript.
//
// Returns bool which is true for:
//   - type="application/javascript", "text/javascript", "module", or
//     "application/typescript"
//   - lang="js", "javascript", "ts", or "typescript"
//   - No type or lang attribute (defaults to JavaScript)
func (s *Script) IsJavaScript() bool {
	if lang, ok := s.Attributes[attributeLang]; ok && lang != "" {
		return strings.EqualFold(lang, "js") ||
			strings.EqualFold(lang, "javascript") ||
			strings.EqualFold(lang, "ts") ||
			strings.EqualFold(lang, "typescript")
	}

	t := s.Type()
	return strings.EqualFold(t, MimeJavaScript) ||
		strings.EqualFold(t, "text/javascript") ||
		strings.EqualFold(t, "module") ||
		strings.EqualFold(t, MimeTypeScript)
}

// IsGo checks if the script block is Go by examining its type or lang
// attributes.
//
// Returns bool which is true if type or lang is "go", "golang", MimeGo, or
// MimeGoShort.
func (s *Script) IsGo() bool {
	if scriptType, ok := s.Attributes[attributeType]; ok && (scriptType == MimeGo || scriptType == MimeGoShort) {
		return true
	}
	if lang, ok := s.Attributes[attributeLang]; ok && (lang == "go" || lang == "golang" || lang == MimeGo || lang == MimeGoShort) {
		return true
	}
	return false
}

// JavaScriptScripts returns all script blocks that are JavaScript.
//
// Returns []Script which contains only the scripts where IsJavaScript is true.
func (pr *ParseResult) JavaScriptScripts() []Script {
	var jsScripts []Script
	for _, s := range pr.Scripts {
		if s.IsJavaScript() {
			jsScripts = append(jsScripts, s)
		}
	}
	return jsScripts
}

// JavaScriptScript returns the first JavaScript script block.
//
// Returns *Script which is the first JavaScript script found.
// Returns bool which indicates whether a JavaScript script was found.
func (pr *ParseResult) JavaScriptScript() (*Script, bool) {
	for i := range pr.Scripts {
		if pr.Scripts[i].IsJavaScript() {
			return &pr.Scripts[i], true
		}
	}
	return nil, false
}

// GoScripts returns all script blocks that are Go scripts.
//
// Returns []Script which contains only the scripts where IsGo returns true.
func (pr *ParseResult) GoScripts() []Script {
	var goScripts []Script
	for _, s := range pr.Scripts {
		if s.IsGo() {
			goScripts = append(goScripts, s)
		}
	}
	return goScripts
}

// GoScript returns the first Go script block from the parse result.
//
// Returns *Script which is the first Go script found, or nil if none exists.
// Returns bool which is true when a Go script was found.
func (pr *ParseResult) GoScript() (*Script, bool) {
	for i := range pr.Scripts {
		if pr.Scripts[i].IsGo() {
			return &pr.Scripts[i], true
		}
	}
	return nil, false
}

// IsTypeScript checks if the script block is TypeScript by examining its
// lang or type attributes.
//
// Returns true if:
//   - lang="ts" or "typescript"
//   - type="application/typescript"
func (s *Script) IsTypeScript() bool {
	if lang, ok := s.Attributes[attributeLang]; ok && (lang == "ts" || lang == "typescript") {
		return true
	}
	if scriptType, ok := s.Attributes[attributeType]; ok && strings.EqualFold(scriptType, MimeTypeScript) {
		return true
	}
	return false
}

// IsClientScript checks if the script block is a client-side script.
//
// Returns true for JavaScript or TypeScript scripts, false for Go scripts.
func (s *Script) IsClientScript() bool {
	return s.IsJavaScript() || s.IsTypeScript()
}

// HasRecognizedScriptType checks if the script block has a recognised language
// or type attribute.
//
// This is used to warn about script blocks with unrecognised, invalid, or
// missing attributes. Script blocks should explicitly declare their language
// to avoid ambiguity.
//
// Valid combinations are:
//   - Go: type="application/x-go", type="application/go", lang="go",
//     lang="golang"
//   - JavaScript: type="application/javascript", type="text/javascript",
//     type="module", lang="js", lang="javascript"
//   - TypeScript: lang="ts", lang="typescript", type="application/typescript"
//
// Returns bool which is true if the script type is explicitly and validly
// declared.
func (s *Script) HasRecognizedScriptType() bool {
	if s.IsGo() || s.IsTypeScript() {
		return true
	}

	if lang, hasLang := s.Attributes[attributeLang]; hasLang && lang != "" {
		return strings.EqualFold(lang, "js") || strings.EqualFold(lang, "javascript")
	}

	if scriptType, hasType := s.Attributes[attributeType]; hasType && scriptType != "" {
		return strings.EqualFold(scriptType, MimeJavaScript) ||
			strings.EqualFold(scriptType, "text/javascript") ||
			strings.EqualFold(scriptType, "module")
	}

	return false
}

// ClientScripts returns all script blocks that are client-side scripts
// (JavaScript or TypeScript).
//
// Returns []Script which contains only the scripts where IsClientScript is
// true.
func (pr *ParseResult) ClientScripts() []Script {
	var clientScripts []Script
	for _, s := range pr.Scripts {
		if s.IsClientScript() {
			clientScripts = append(clientScripts, s)
		}
	}
	return clientScripts
}

// ClientScript returns the first client-side script block, such as JavaScript
// or TypeScript.
//
// Returns *Script which is the first client script found, or nil if none.
// Returns bool which is true if a client script was found.
func (pr *ParseResult) ClientScript() (*Script, bool) {
	for i := range pr.Scripts {
		if pr.Scripts[i].IsClientScript() {
			return &pr.Scripts[i], true
		}
	}
	return nil, false
}

// HasCollectionDirective checks if the template has a p-collection attribute.
//
// Returns bool which is true if the p-collection attribute is present.
func (pr *ParseResult) HasCollectionDirective() bool {
	_, ok := pr.TemplateAttributes["p-collection"]
	return ok
}

// GetCollectionName returns the collection name from the p-collection
// attribute.
//
// Returns string which is the collection name, or empty if p-collection is not
// present.
func (pr *ParseResult) GetCollectionName() string {
	return pr.TemplateAttributes["p-collection"]
}

// GetCollectionProvider returns the provider name from the p-provider
// attribute.
//
// Returns string which is the provider name, or "markdown" if p-provider is not
// specified.
func (pr *ParseResult) GetCollectionProvider() string {
	if provider, ok := pr.TemplateAttributes["p-provider"]; ok && provider != "" {
		return provider
	}
	return "markdown"
}

// GetCollectionParamName returns the URL parameter name from the p-param
// attribute. Returns "slug" as the default if p-param is not set.
//
// This tells collection providers which chi URL parameter to use for content
// lookup. For example, if the route is /products/{id}, set p-param="id" so
// the provider gets the correct parameter value at runtime.
//
// Returns string which is the parameter name for content lookup.
func (pr *ParseResult) GetCollectionParamName() string {
	if param, ok := pr.TemplateAttributes["p-param"]; ok && param != "" {
		return param
	}
	return "slug"
}

// HasCollectionSource checks if the template specifies an external content
// source. When present, the p-collection-source attribute references a Go
// import alias that points to a module containing markdown content.
//
// Returns bool which is true if the p-collection-source attribute is present.
func (pr *ParseResult) HasCollectionSource() bool {
	_, ok := pr.TemplateAttributes["p-collection-source"]
	return ok
}

// GetCollectionSource returns the import alias from the p-collection-source
// attribute. This alias references a Go import that points to a module
// containing markdown content.
//
// Returns string which is the import alias, or empty if p-collection-source
// is not present.
func (pr *ParseResult) GetCollectionSource() string {
	return pr.TemplateAttributes["p-collection-source"]
}

// HasPublicDirective checks if the template has a public attribute, which
// explicitly marks the component as publicly accessible regardless of the
// default visibility for its component type.
//
// Returns bool which is true if the public attribute is present.
func (pr *ParseResult) HasPublicDirective() bool {
	_, ok := pr.TemplateAttributes["public"]
	return ok
}

// HasPrivateDirective checks if the template has a private attribute, which
// explicitly marks the component as private regardless of the default
// visibility for its component type.
//
// Returns bool which is true if the private attribute is present.
func (pr *ParseResult) HasPrivateDirective() bool {
	_, ok := pr.TemplateAttributes["private"]
	return ok
}
