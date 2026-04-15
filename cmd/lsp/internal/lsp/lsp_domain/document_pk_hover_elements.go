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

package lsp_domain

import (
	"fmt"
	"strings"

	"go.lsp.dev/protocol"
	"piko.sh/piko/wdk/safeconv"
)

// attributeTypeString is the type name used for string-valued element attributes.
const attributeTypeString = "string"

// pikoAttrDoc holds documentation for a single attribute of a Piko element.
type pikoAttrDoc struct {
	// Name is the attribute name.
	Name string

	// Type is the value type of the attribute (e.g., "string", "boolean").
	Type string

	// Description explains what this attribute does.
	Description string
}

// pikoElementDoc holds documentation for a Piko built-in element.
type pikoElementDoc struct {
	// Name is the element tag name (for example, "piko:img").
	Name string

	// Description holds the text that explains what this element does.
	Description string

	// Example holds an HTML code snippet showing how to use this element.
	Example string

	// DocumentsURL is the relative URL path to the documentation page.
	DocumentsURL string

	// RequiredAttrs lists the attributes that must be provided for this element.
	RequiredAttrs []pikoAttrDoc

	// OptionalAttrs lists attributes that may be provided but are not required.
	OptionalAttrs []pikoAttrDoc
}

var (
	// pikoImageRequiredAttrs is shared between piko:img and piko:picture.
	pikoImageRequiredAttrs = []pikoAttrDoc{
		{Name: "src", Type: attributeTypeString, Description: "Image source path (use :src for dynamic binding)"},
	}

	// pikoElementDocuments is the registry of all documented Piko built-in elements.
	pikoElementDocuments = map[string]pikoElementDoc{
		"piko:img": {
			Name:          "piko:img",
			Description:   "Renders optimised, responsive images with automatic srcset generation",
			RequiredAttrs: pikoImageRequiredAttrs,
			OptionalAttrs: newPikoImageOptionalAttrs(
				"Alternative text for accessibility",
				"Comma-separated widths for srcset",
				"Output formats (webp, avif, jpg)",
			),
			Example:      `<piko:img src="/images/hero.jpg" alt="Hero image" sizes="100vw" widths="400,800,1200" />`,
			DocumentsURL: "/docs/api/tags/piko-img",
		},
		"piko:picture": {
			Name:          "piko:picture",
			Description:   "Renders a <picture> element with per-format <source> elements for optimal browser format negotiation",
			RequiredAttrs: pikoImageRequiredAttrs,
			OptionalAttrs: newPikoImageOptionalAttrs(
				"Alternative text for accessibility (goes on fallback <img>)",
				"Comma-separated widths for srcset generation",
				`Output formats in preference order (e.g. "avif, webp"); default: webp`,
			),
			Example:      `<piko:picture src="/images/hero.jpg" alt="Hero" sizes="100vw" widths="640,1280" formats="avif, webp" />`,
			DocumentsURL: "/docs/api/tags/piko-picture",
		},
		"piko:svg": {
			Name:        "piko:svg",
			Description: "Inlines SVG content with support for sprite sheets",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "src", Type: attributeTypeString, Description: "SVG source path (use :src for dynamic binding)"},
			},
			OptionalAttrs: []pikoAttrDoc{
				{Name: "class", Type: attributeTypeString, Description: "CSS class names to apply"},
				{Name: "width", Type: attributeTypeString, Description: "SVG width attribute"},
				{Name: "height", Type: attributeTypeString, Description: "SVG height attribute"},
				{Name: "fill", Type: attributeTypeString, Description: "Fill colour override"},
			},
			Example:      `<piko:svg src="/icons/menu.svg" class="icon" width="24" height="24"></piko:svg>`,
			DocumentsURL: "/docs/api/tags/piko-svg",
		},
		"piko:a": {
			Name:        "piko:a",
			Description: "Locale-aware navigation link with automatic path prefixing",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "href", Type: attributeTypeString, Description: "Link destination (use :href for dynamic binding)"},
			},
			OptionalAttrs: []pikoAttrDoc{
				{Name: "lang", Type: attributeTypeString, Description: "Override the target locale"},
			},
			Example:      `<piko:a href="/about">About Us</piko:a>`,
			DocumentsURL: "/docs/api/tags/piko-a",
		},
		"piko:video": {
			Name:        "piko:video",
			Description: "Video element with HLS streaming support",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "src", Type: attributeTypeString, Description: "Video source path (use :src for dynamic binding)"},
			},
			OptionalAttrs: []pikoAttrDoc{
				{Name: "poster", Type: attributeTypeString, Description: "Poster image URL"},
				{Name: "qualities", Type: attributeTypeString, Description: "Comma-separated quality levels for HLS"},
				{Name: "controls", Type: "boolean", Description: "Show video controls"},
				{Name: "autoplay", Type: "boolean", Description: "Auto-play on load"},
				{Name: "muted", Type: "boolean", Description: "Mute audio by default"},
				{Name: "loop", Type: "boolean", Description: "Loop video playback"},
			},
			Example:      `<piko:video src="/videos/intro.mp4" poster="/images/poster.jpg" controls />`,
			DocumentsURL: "/docs/api/tags/piko-video",
		},
		"piko:slot": {
			Name:          "piko:slot",
			Description:   "Content projection placeholder for component composition",
			RequiredAttrs: []pikoAttrDoc{},
			OptionalAttrs: []pikoAttrDoc{
				{Name: "name", Type: attributeTypeString, Description: "Named slot identifier for targeted projection"},
			},
			Example:      `<piko:slot name="header"></piko:slot>`,
			DocumentsURL: "/docs/api/tags/piko-slot",
		},
		"piko:content": {
			Name:          "piko:content",
			Description:   "Injects markdown content from CMS or data sources",
			RequiredAttrs: []pikoAttrDoc{},
			OptionalAttrs: []pikoAttrDoc{},
			Example:       `<piko:content />`,
			DocumentsURL:  "/docs/api/tags/piko-content",
		},
		"piko:partial": {
			Name:        "piko:partial",
			Description: "Invokes a partial component, rendering its template with the provided props",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "is", Type: attributeTypeString, Description: "The import alias of the partial to render (required, cannot be dynamic)"},
			},
			OptionalAttrs: []pikoAttrDoc{},
			Example:       `<piko:partial is="card" :title="state.Title" />`,
			DocumentsURL:  "/docs/api/tags/piko-partial",
		},
		"piko:captcha": {
			Name: "piko:captcha",
			Description: "Renders a captcha widget using the configured provider. Replaced at render time with " +
				"provider-specific HTML including the widget container, script tags, and a hidden input for the verification token.",
			RequiredAttrs: []pikoAttrDoc{},
			OptionalAttrs: []pikoAttrDoc{
				{Name: "provider", Type: attributeTypeString, Description: "Registered provider name. Uses the default provider when omitted."},
				{Name: "name", Type: attributeTypeString, Description: "Hidden input field name for the captcha token. Defaults to \"_captcha_token\"."},
				{Name: "theme", Type: attributeTypeString, Description: "Widget theme: \"light\" (default) or \"dark\"."},
				{Name: "size", Type: attributeTypeString, Description: "Widget size: \"normal\" (default) or \"compact\"."},
				{Name: "action", Type: attributeTypeString, Description: "Action name bound to the token for server-side providers. Prevents cross-form token reuse."},
			},
			Example:      "<piko:captcha />\n<piko:captcha provider=\"turnstile\" theme=\"dark\" />",
			DocumentsURL: "/docs/api/tags/piko-captcha",
		},
	}

	// pikoTimelineElementDocuments contains hover documentation for timeline
	// elements. These are only valid inside <piko:timeline> blocks in PKC files.
	pikoTimelineElementDocuments = map[string]pikoElementDoc{
		"piko:timeline": {
			Name: "piko:timeline",
			Description: "Declares a timeline animation block for a PKC component. " +
				"Contains `<piko:at>` elements that group actions at specific time points. " +
				"Requires `enable=\"animation\"` on the component's template tag.",
			RequiredAttrs: []pikoAttrDoc{},
			OptionalAttrs: []pikoAttrDoc{},
			Example: `<piko:timeline>
  <piko:at time="1s">
    <piko:show ref="title" />
  </piko:at>
  <piko:at time="2s">
    <piko:type ref="code" speed="30" />
  </piko:at>
</piko:timeline>`,
		},
		"piko:at": {
			Name:        "piko:at",
			Description: "Groups one or more timeline actions at a specific time point. All child actions trigger at this time.",
			RequiredAttrs: []pikoAttrDoc{
				{
					Name: "time", Type: attributeTypeString,
					Description: "Time point when actions trigger. Supports seconds (\"1.5s\"), " +
						"milliseconds (\"1500ms\"), or plain numbers treated as seconds (\"1.5\")",
				},
			},
			OptionalAttrs: []pikoAttrDoc{},
			Example:       `<piko:at time="2.5s"><piko:show ref="title" /></piko:at>`,
		},
		"piko:show": {
			Name:        "piko:show",
			Description: "Removes the `p-timeline-hidden` attribute from the target element at the specified time, making it visible. Before the time point, the attribute is re-added (seekable).",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "ref", Type: attributeTypeString, Description: "The p-ref name of the target element"},
			},
			OptionalAttrs: []pikoAttrDoc{},
			Example:       `<piko:show ref="title" />`,
		},
		"piko:hide": {
			Name:        "piko:hide",
			Description: "Adds the `p-timeline-hidden` attribute to the target element at the specified time, hiding it. Before the time point, the attribute is removed (seekable).",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "ref", Type: attributeTypeString, Description: "The p-ref name of the target element"},
			},
			OptionalAttrs: []pikoAttrDoc{},
			Example:       `<piko:hide ref="overlay" />`,
		},
		"piko:type": {
			Name: "piko:type",
			Description: "Reveals the element's text content character by character, creating a typewriter effect. " +
				"The element's initial textContent is captured and cleared; characters appear at a rate of " +
				"one per `speed` milliseconds. Pure function of time (seekable).",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "ref", Type: attributeTypeString, Description: "The p-ref name of the target element"},
			},
			OptionalAttrs: []pikoAttrDoc{
				{Name: "speed", Type: "number", Description: "Milliseconds per character (default: 50)"},
			},
			Example: `<piko:type ref="command" speed="30" />`,
		},
		"piko:typehtml": {
			Name: "piko:typehtml",
			Description: "Reveals the element's innerHTML character by character while keeping HTML tags intact. " +
				"Only visible characters (not tag markup or entity references) count against the typing speed. " +
				"Suitable for syntax-highlighted code wrapped in spans. Pure function of time (seekable).",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "ref", Type: attributeTypeString, Description: "The p-ref name of the target element"},
			},
			OptionalAttrs: []pikoAttrDoc{
				{Name: "speed", Type: "number", Description: "Milliseconds per visible character (default: 25)"},
			},
			Example: `<piko:typehtml ref="codeBlock" speed="20" />`,
		},
		"piko:addclass": {
			Name: "piko:addclass",
			Description: "Adds a CSS class to the target element at the specified time. " +
				"Before the time point, the class is removed (seekable). Useful for triggering CSS transitions " +
				"or toggling visual states without the dual-element show/hide pattern.",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "ref", Type: attributeTypeString, Description: "The p-ref name of the target element"},
				{Name: "class", Type: attributeTypeString, Description: "The CSS class name to add"},
			},
			OptionalAttrs: []pikoAttrDoc{},
			Example:       `<piko:addclass ref="tab1" class="active" />`,
		},
		"piko:removeclass": {
			Name:        "piko:removeclass",
			Description: "Removes a CSS class from the target element at the specified time. Before the time point, the class is added back (seekable). The inverse of addclass.",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "ref", Type: attributeTypeString, Description: "The p-ref name of the target element"},
				{Name: "class", Type: attributeTypeString, Description: "The CSS class name to remove"},
			},
			OptionalAttrs: []pikoAttrDoc{},
			Example:       `<piko:removeclass ref="highlight" class="glow" />`,
		},
		"piko:tooltip": {
			Name: "piko:tooltip",
			Description: "Sets the `title` attribute on the target element at the specified time. " +
				"Multiple tooltip actions on the same ref at different times create time-dependent hover text; " +
				"the latest matching action wins. An empty value clears the tooltip. Pure function of time (seekable).",
			RequiredAttrs: []pikoAttrDoc{
				{Name: "ref", Type: attributeTypeString, Description: "The p-ref name of the target element"},
			},
			OptionalAttrs: []pikoAttrDoc{
				{Name: "value", Type: attributeTypeString, Description: "The tooltip text to display on hover (empty to clear)"},
			},
			Example: `<piko:tooltip ref="editor" value="This is the code editor" />`,
		},
	}
)

// checkPikoElementHoverContext checks if the cursor is on a piko:* element tag
// name.
//
// Takes line (string) which is the current line text.
// Takes cursor (int) which is the cursor position within the line.
// Takes position (protocol.Position) which is the LSP position in the document.
//
// Returns *PKHoverContext which provides hover context when the cursor is on
// a piko element tag name, or nil when no match is found.
func (d *document) checkPikoElementHoverContext(line string, cursor int, position protocol.Position) *PKHoverContext {
	tagStart := findPikoTagStart(line, cursor)
	if tagStart == -1 {
		return nil
	}

	tagName, tagEnd := extractPikoTagName(line, tagStart)
	if tagName == "" {
		return nil
	}

	if cursor < tagStart || cursor > tagEnd {
		return nil
	}

	if tagName == "piko:partial" {
		return nil
	}

	_, inGeneral := pikoElementDocuments[tagName]
	_, inTimeline := pikoTimelineElementDocuments[tagName]

	if !inGeneral && !inTimeline {
		return nil
	}

	if inTimeline && !inGeneral && !d.isPKCFile() {
		return nil
	}

	return &PKHoverContext{
		Kind:     PKDefPikoElement,
		Name:     tagName,
		Position: position,
		Range: protocol.Range{
			Start: protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(tagStart)},
			End:   protocol.Position{Line: position.Line, Character: safeconv.IntToUint32(tagEnd)},
		},
	}
}

// getPikoElementHover returns hover information for a piko:* element.
//
// Takes ctx (*PKHoverContext) which provides the hover request context.
//
// Returns *protocol.Hover which contains the hover information to display.
// Returns error which is always nil for this function.
func (*document) getPikoElementHover(ctx *PKHoverContext) (*protocol.Hover, error) {
	elementDoc, exists := pikoElementDocuments[ctx.Name]
	if !exists {
		elementDoc, exists = pikoTimelineElementDocuments[ctx.Name]
		if !exists {
			return nil, nil
		}
	}

	content := formatPikoElementDocumentation(elementDoc)

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.Markdown,
			Value: content,
		},
		Range: &ctx.Range,
	}, nil
}

// newPikoImageOptionalAttrs builds the optional attribute list for an
// image-like element. The altDesc and formatsDesc parameters customise the
// descriptions that differ between piko:img and piko:picture.
//
// Takes altDesc (string) which is the description text for the alt attribute.
// Takes widthsDesc (string) which is the description text for the widths
// attribute.
// Takes formatsDesc (string) which is the description text for the formats
// attribute.
//
// Returns []pikoAttrDoc which contains the optional attribute definitions for
// the image-like element.
func newPikoImageOptionalAttrs(altDesc, widthsDesc, formatsDesc string) []pikoAttrDoc {
	return []pikoAttrDoc{
		{Name: "alt", Type: attributeTypeString, Description: altDesc},
		{Name: "sizes", Type: attributeTypeString, Description: "CSS sizes for responsive images"},
		{Name: "widths", Type: attributeTypeString, Description: widthsDesc},
		{Name: "densities", Type: attributeTypeString, Description: "Comma-separated pixel densities"},
		{Name: "formats", Type: attributeTypeString, Description: formatsDesc},
		{Name: "variant", Type: attributeTypeString, Description: "CMS variant name to select"},
		{Name: "cms-media", Type: "boolean", Description: "Enable CMS media handling"},
	}
}

// findPikoTagStart finds the start position of a piko:* tag near the cursor.
//
// Takes line (string) which contains the text to search.
// Takes cursor (int) which is the position within the line.
//
// Returns int which is the position where the tag name starts, or -1 if no
// tag is found.
func findPikoTagStart(line string, cursor int) int {
	if len(line) == 0 {
		return -1
	}

	searchEnd := min(cursor+20, len(line))
	searchStart := max(0, cursor-20)

	if searchStart >= searchEnd {
		return -1
	}

	searchArea := line[searchStart:searchEnd]

	for _, pattern := range []string{"<piko:", "</piko:"} {
		index := strings.LastIndex(searchArea, pattern)
		if index != -1 {
			absoluteIndex := searchStart + index
			tagNameStart := absoluteIndex + len(pattern) - len("piko:")
			if tagNameStart <= cursor {
				return tagNameStart
			}
		}
	}

	return -1
}

// extractPikoTagName extracts the tag name starting at the given position.
//
// Takes line (string) which contains the text to search.
// Takes startPosition (int) which is the position where the tag name starts.
//
// Returns tagName (string) which is the extracted tag name (e.g. "piko:img").
// Returns endPosition (int) which is the position after the tag name.
func extractPikoTagName(line string, startPosition int) (tagName string, endPosition int) {
	endPosition = startPosition
	for i := startPosition; i < len(line); i++ {
		character := line[i]
		if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') || character == '_' || character == ':' {
			endPosition = i + 1
			continue
		}
		break
	}

	if endPosition > startPosition {
		tagName = line[startPosition:endPosition]
	}

	return tagName, endPosition
}

// formatPikoElementDocumentation formats element documentation as markdown.
//
// Takes elementDoc (pikoElementDoc) which contains the element documentation.
//
// Returns string which is the formatted markdown content.
func formatPikoElementDocumentation(elementDoc pikoElementDoc) string {
	var b strings.Builder

	_, _ = fmt.Fprintf(&b, "## `<%s>`\n\n", elementDoc.Name)

	b.WriteString(elementDoc.Description)
	b.WriteString("\n\n")

	if len(elementDoc.RequiredAttrs) > 0 {
		b.WriteString("**Required:**\n")
		for _, attr := range elementDoc.RequiredAttrs {
			_, _ = fmt.Fprintf(&b, "- `%s` (%s) - %s\n", attr.Name, attr.Type, attr.Description)
		}
		b.WriteString("\n")
	}

	if len(elementDoc.OptionalAttrs) > 0 {
		b.WriteString("**Optional:**\n")
		for _, attr := range elementDoc.OptionalAttrs {
			_, _ = fmt.Fprintf(&b, "- `%s` (%s) - %s\n", attr.Name, attr.Type, attr.Description)
		}
		b.WriteString("\n")
	}

	if elementDoc.Example != "" {
		b.WriteString("**Example:**\n")
		_, _ = fmt.Fprintf(&b, "```html\n%s\n```\n\n", elementDoc.Example)
	}

	if elementDoc.DocumentsURL != "" {
		b.WriteString("---\n\n")
		_, _ = fmt.Fprintf(&b, "[Documentation](%s)", elementDoc.DocumentsURL)
	}

	return b.String()
}
