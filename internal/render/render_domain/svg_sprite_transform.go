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

package render_domain

import (
	"cmp"
	"hash/fnv"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// maxSVGTransformInputBytes caps the inner SVG content the transform will process.
	// Larger content falls back to a verbatim copy to bound work and memory.
	maxSVGTransformInputBytes = 1 << 20

	// maxSVGTransformTokens caps the number of tokens produced while tokenising one asset.
	// Exceeding it triggers a verbatim fallback.
	maxSVGTransformTokens = 100000

	// maxSVGTransformIDs caps the number of distinct element identifiers namespaced per
	// asset. Exceeding it triggers a verbatim fallback.
	maxSVGTransformIDs = 20000

	// urlReferencePrefix is the opening of a CSS url() reference such as url(#gradient).
	urlReferencePrefix = "url("
)

// svgTokenKind classifies a token produced while tokenising SVG fragment content.
type svgTokenKind uint8

const (
	// tokenText is literal character data between tags, emitted unchanged.
	tokenText svgTokenKind = iota

	// tokenStart is an element start tag, which may be self-closing.
	tokenStart

	// tokenEnd is an element end tag.
	tokenEnd

	// tokenVerbatim is a comment, CDATA section, processing instruction, or doctype,
	// emitted unchanged.
	tokenVerbatim

	// tokenStyleText is the raw body of a <style> element, where url(#id) references are
	// rewritten but CSS selectors are left untouched.
	tokenStyleText

	// tokenScriptText is the raw body of a <script> element, emitted unchanged.
	tokenScriptText
)

// svgTagAttribute holds one parsed attribute of a start tag, preserving the original quote
// character so the value can be re-serialised without changing its escaping.
type svgTagAttribute struct {
	// name is the attribute name with its original case (for example xlink:href).
	name string

	// value is the raw, already-escaped attribute value between the quotes.
	value string

	// quote is the original quote byte, either a double or single quote.
	quote byte

	// hasValue reports whether the attribute had a value, distinguishing it from a boolean
	// attribute.
	hasValue bool
}

// svgToken is a single token from the SVG fragment tokeniser.
type svgToken struct {
	// raw holds the literal bytes for text, verbatim, style, and script tokens.
	raw string

	// name is the element name for start and end tokens.
	name string

	// attributes holds the parsed attributes for start tokens.
	attributes []svgTagAttribute

	// kind classifies the token.
	kind svgTokenKind

	// selfClose reports whether a start token closed itself.
	selfClose bool
}

// svgFrame tracks one open element while serialising, recording where its tags and children
// are written and where to resume writing once it closes.
type svgFrame struct {
	// tagDestination is the builder receiving this element's own tags and children.
	tagDestination *strings.Builder

	// parentDestination is the destination to restore when this element closes.
	parentDestination *strings.Builder

	// name is the element name, matched against the corresponding end tag.
	name string

	// emitTags reports whether this element's own start and end tags are written.
	emitTags bool
}

// svgRewriter routes tokens into the symbol body or the hoisted definitions while rewriting
// identifiers and references.
type svgRewriter struct {
	// rename maps each original identifier to its namespaced form.
	rename map[string]string

	// body receives the drawable symbol content.
	body *strings.Builder

	// defs receives hoisted definition elements.
	defs *strings.Builder

	// current is the active destination for the element being processed.
	current *strings.Builder

	// stack holds the open elements, innermost last, each closed by a matching end tag.
	stack []svgFrame
}

var (
	// hoistElements is the set of referenceable definition elements moved out of the symbol.
	// It is the single source of truth for both isHoistElement and containsHoistElement.
	hoistElements = map[string]struct{}{
		"linearGradient": {},
		"radialGradient": {},
		"pattern":        {},
		"filter":         {},
		"clipPath":       {},
		"mask":           {},
		"marker":         {},
	}

	// verbatimMarkers are the opening and closing delimiters of constructs copied unchanged.
	verbatimMarkers = [][2]string{
		{"<!--", "-->"},
		{"<![CDATA[", "]]>"},
		{"<?", "?>"},
	}
)

// ComputeSymbolAndDefs builds the sprite symbol body and the hoisted definitions for an SVG
// asset. It runs once per asset at load time and the results are cached in ParsedSvgData.
//
// Referenceable definitions (gradients, patterns, filters, clip paths, masks, markers, and
// the contents of any <defs>) are moved out of the symbol into the returned definitions
// string so they resolve in document scope. Every identifier defined in the asset is
// namespaced with a per-asset prefix and every reference is rewritten, so two assets
// sharing an identifier such as paint0_linear cannot collide in the shared sprite.
//
// Takes id (string) which specifies the symbol identifier.
// Takes parsedData (*ParsedSvgData) which provides the parsed SVG content.
//
// Returns symbol which is the formatted <symbol> element.
// Returns defs which holds the asset's hoisted definitions, or empty when none exist.
// Returns fellBack which is true when the asset needed the transform but oversized or
// malformed content forced a verbatim copy, so callers can surface the degradation.
// All outputs are zero when parsedData is nil.
func ComputeSymbolAndDefs(id string, parsedData *ParsedSvgData) (symbol string, defs string, fellBack bool) {
	if parsedData == nil {
		return "", "", false
	}

	viewBox := extractViewBox(parsedData.Attributes)
	inner := parsedData.InnerHTML

	if !svgNeedsTransform(inner) {
		return writeSymbol(id, viewBox, inner), "", false
	}

	body, hoisted, ok := transformSVG(svgIDPrefix(id), inner)
	if !ok {
		return writeSymbol(id, viewBox, inner), "", true
	}

	return writeSymbol(id, viewBox, body), hoisted, false
}

// extractViewBox returns the viewBox attribute value from parsed SVG attributes.
//
// Takes attributes ([]ast_domain.HTMLAttribute) which hold the SVG element attributes.
//
// Returns string which is the viewBox value, or empty when absent.
func extractViewBox(attributes []ast_domain.HTMLAttribute) string {
	for i := range attributes {
		if attributes[i].Name == "viewBox" {
			return attributes[i].Value
		}
	}
	return ""
}

// writeSymbol formats a <symbol> element with an escaped identifier and viewBox wrapping the
// given content.
//
// Takes id (string) which specifies the symbol identifier.
// Takes viewBox (string) which is the viewBox attribute value, or empty.
// Takes content (string) which is the inner markup to wrap.
//
// Returns string which is the formatted <symbol> element.
func writeSymbol(id, viewBox, content string) string {
	builder := getBuilder()

	builder.Grow(svgSymbolOverhead + len(id) + len(viewBox) + len(content))

	builder.WriteString(`<symbol id="`)
	builder.WriteString(html.EscapeString(id))
	builder.WriteString(`"`)

	if viewBox != "" {
		builder.WriteString(` viewBox="`)
		builder.WriteString(html.EscapeString(viewBox))
		builder.WriteString(`"`)
	}

	builder.WriteString(`>`)
	builder.WriteString(content)
	builder.WriteString(`</symbol>`)

	result := builder.String()
	putBuilder(builder)
	return result
}

// svgIDPrefix derives a deterministic, collision-resistant prefix for an asset. It
// hashes the artefact identifier because artefact identifiers may contain characters such as
// slashes and dots that are awkward inside identifiers and url() references.
//
// Takes artefactID (string) which identifies the asset.
//
// Returns string which is a stable prefix valid as an identifier fragment.
func svgIDPrefix(artefactID string) string {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(artefactID))
	return "s" + strconv.FormatUint(hasher.Sum64(), 36)
}

// svgNeedsTransform reports whether the inner SVG content contains anything the transform
// must act on. Plain icons with no definitions and no internal references take the verbatim
// fast path.
//
// Takes inner (string) which is the SVG fragment content.
//
// Returns bool which is true when the transform must run.
func svgNeedsTransform(inner string) bool {
	if strings.Contains(inner, "url(#") {
		return true
	}
	if strings.Contains(inner, `href="#`) || strings.Contains(inner, "href='#") {
		return true
	}
	return containsHoistElement(inner)
}

// containsHoistElement reports whether the inner content mentions any element the transform
// hoists into the shared definitions.
//
// Takes inner (string) which is the SVG fragment content.
//
// Returns bool which is true when a hoistable definition element is present.
func containsHoistElement(inner string) bool {
	if strings.Contains(inner, "<defs") {
		return true
	}
	for name := range hoistElements {
		if strings.Contains(inner, "<"+name) {
			return true
		}
	}
	return false
}

// isHoistElement reports whether an element with the given name is itself hoisted into the
// shared definitions. The <defs> wrapper is handled separately because its children
// are flattened into the shared block rather than hoisted as one element.
//
// Takes name (string) which is the element name.
//
// Returns bool which is true when the element is hoisted into the shared definitions.
func isHoistElement(name string) bool {
	_, ok := hoistElements[name]
	return ok
}

// transformSVG tokenises the inner content, namespaces every identifier with the prefix, and
// serialises the result into a symbol body and a hoisted definitions string.
//
// Takes prefix (string) which namespaces every identifier in the asset.
// Takes inner (string) which is the SVG fragment content.
//
// Returns body which is the transformed symbol content.
// Returns defs which holds the hoisted definitions.
// Returns ok which is false when the content is oversized, exceeds the token or identifier
// limits, or is structurally malformed, so the caller falls back to a verbatim copy.
func transformSVG(prefix, inner string) (body string, defs string, ok bool) {
	if len(inner) > maxSVGTransformInputBytes {
		return "", "", false
	}

	tokens, ok := tokeniseSVG(inner)
	if !ok {
		return "", "", false
	}

	renameMap := collectSVGIDs(tokens, prefix)
	if len(renameMap) > maxSVGTransformIDs {
		return "", "", false
	}

	return serialiseSVG(tokens, renameMap)
}

// collectSVGIDs builds the rename map from every identifier defined in the asset to its
// namespaced form.
//
// Takes tokens ([]svgToken) which are the tokenised asset.
// Takes prefix (string) which namespaces each identifier.
//
// Returns map[string]string which maps each original identifier to its namespaced form.
func collectSVGIDs(tokens []svgToken, prefix string) map[string]string {
	renameMap := make(map[string]string)
	for index := range tokens {
		if tokens[index].kind != tokenStart {
			continue
		}
		if collectTokenIDs(tokens[index], prefix, renameMap) {
			return renameMap
		}
	}
	return renameMap
}

// collectTokenIDs adds the namespaced form of each identifier defined on the token to
// renameMap.
//
// Takes token (svgToken) which is the start token to scan.
// Takes prefix (string) which namespaces each identifier.
// Takes renameMap (map[string]string) which receives the mappings.
//
// Returns bool which is true when the identifier cap is exceeded and collection should stop.
func collectTokenIDs(token svgToken, prefix string, renameMap map[string]string) bool {
	for _, attribute := range token.attributes {
		if attribute.name != "id" || !attribute.hasValue || attribute.value == "" {
			continue
		}
		if _, exists := renameMap[attribute.value]; exists {
			continue
		}
		renameMap[attribute.value] = prefix + "-" + attribute.value
		if len(renameMap) > maxSVGTransformIDs {
			return true
		}
	}
	return false
}

// serialiseSVG walks the token stream, routing hoisted definition subtrees into the
// definitions builder and everything else into the symbol body builder.
//
// Takes tokens ([]svgToken) which are the tokenised asset.
// Takes renameMap (map[string]string) which maps identifiers to their namespaced form.
//
// Returns body and defs as built strings, and ok false when the tag structure is unbalanced.
func serialiseSVG(tokens []svgToken, renameMap map[string]string) (body string, defs string, ok bool) {
	bodyBuilder := getBuilder()
	defsBuilder := getBuilder()
	defer putBuilder(bodyBuilder)
	defer putBuilder(defsBuilder)

	rewriter := &svgRewriter{
		rename:  renameMap,
		body:    bodyBuilder,
		defs:    defsBuilder,
		current: bodyBuilder,
		stack:   nil,
	}

	for index := range tokens {
		if !rewriter.writeToken(tokens[index]) {
			return "", "", false
		}
	}

	if len(rewriter.stack) != 0 {
		return "", "", false
	}
	return bodyBuilder.String(), defsBuilder.String(), true
}

// writeToken dispatches one token to the appropriate destination.
//
// Takes token (svgToken) which is the token to write.
//
// Returns false when an end tag does not match the currently open element.
func (rewriter *svgRewriter) writeToken(token svgToken) bool {
	switch token.kind {
	case tokenText, tokenVerbatim, tokenScriptText:
		rewriter.current.WriteString(token.raw)
	case tokenStyleText:
		rewriter.current.WriteString(rewriteURLRefs(token.raw, rewriter.rename))
	case tokenStart:
		rewriter.openElement(token)
	case tokenEnd:
		return rewriter.closeElement(token)
	}
	return true
}

// openElement writes a start tag and, for non-self-closing elements, pushes a frame so its
// children and end tag route to the same destination. Definition elements route to the defs
// builder; a <defs> wrapper is flattened by routing its children without emitting its tags.
//
// Takes token (svgToken) which is the start tag to open.
func (rewriter *svgRewriter) openElement(token svgToken) {
	tagDestination := rewriter.current
	emitTags := true
	switch {
	case token.name == "defs":
		tagDestination = rewriter.defs
		emitTags = false
	case isHoistElement(token.name):
		tagDestination = rewriter.defs
	}

	if emitTags {
		writeStartTag(tagDestination, token, rewriter.rename)
	}

	if !token.selfClose {
		rewriter.stack = append(rewriter.stack, svgFrame{
			tagDestination:    tagDestination,
			parentDestination: rewriter.current,
			name:              token.name,
			emitTags:          emitTags,
		})
		rewriter.current = tagDestination
	}
}

// closeElement pops the open element matching the end tag and restores the parent
// destination.
//
// Takes token (svgToken) which is the end tag to close.
//
// Returns false when the structure is unbalanced.
func (rewriter *svgRewriter) closeElement(token svgToken) bool {
	if len(rewriter.stack) == 0 {
		return false
	}
	top := rewriter.stack[len(rewriter.stack)-1]
	if top.name != token.name {
		return false
	}
	rewriter.stack = rewriter.stack[:len(rewriter.stack)-1]
	if top.emitTags {
		top.tagDestination.WriteString("</")
		top.tagDestination.WriteString(token.name)
		top.tagDestination.WriteString(">")
	}
	rewriter.current = top.parentDestination
	return true
}

// writeStartTag serialises a start tag, rewriting its id and reference attributes.
//
// Takes builder (*strings.Builder) which receives the serialised tag.
// Takes token (svgToken) which is the start tag to serialise.
// Takes renameMap (map[string]string) which maps identifiers to their namespaced form.
func writeStartTag(builder *strings.Builder, token svgToken, renameMap map[string]string) {
	builder.WriteString("<")
	builder.WriteString(token.name)

	for _, attribute := range token.attributes {
		builder.WriteString(" ")
		builder.WriteString(attribute.name)
		if !attribute.hasValue {
			continue
		}
		quote := cmp.Or(attribute.quote, byte('"'))
		builder.WriteString("=")
		builder.WriteByte(quote)
		builder.WriteString(rewriteAttributeValue(attribute.name, attribute.value, renameMap))
		builder.WriteByte(quote)
	}

	if token.selfClose {
		builder.WriteString("/>")
	} else {
		builder.WriteString(">")
	}
}

// rewriteAttributeValue rewrites a single attribute value: identifier definitions, fragment
// references on href attributes, and url(#id) references everywhere else (which also covers
// inline style declarations).
//
// Takes name (string) which is the attribute name.
// Takes value (string) which is the raw attribute value.
// Takes renameMap (map[string]string) which maps identifiers to their namespaced form.
//
// Returns string which is the rewritten value.
func rewriteAttributeValue(name, value string, renameMap map[string]string) string {
	if name == "id" {
		if renamed, ok := renameMap[value]; ok {
			return renamed
		}
		return value
	}
	if isHrefAttribute(name) {
		return rewriteFragmentRef(value, renameMap)
	}
	return rewriteURLRefs(value, renameMap)
}

// isHrefAttribute reports whether the attribute holds a fragment reference, covering
// both href and namespaced forms such as xlink:href.
//
// Takes name (string) which is the attribute name.
//
// Returns bool which is true when the attribute holds a fragment reference.
func isHrefAttribute(name string) bool {
	return name == "href" || name == "xlink:href" || strings.HasSuffix(name, ":href")
}

// rewriteFragmentRef rewrites a bare fragment reference such as #paint0 when its
// identifier is namespaced. External and non-fragment references are left untouched.
//
// Takes value (string) which is the attribute value.
// Takes renameMap (map[string]string) which maps identifiers to their namespaced form.
//
// Returns string which is the rewritten reference, or the original when not namespaced.
func rewriteFragmentRef(value string, renameMap map[string]string) string {
	if len(value) < 2 || value[0] != '#' {
		return value
	}
	if renamed, ok := renameMap[value[1:]]; ok {
		return "#" + renamed
	}
	return value
}

// rewriteURLRefs rewrites every url(#id) reference in a value whose identifier is
// namespaced, preserving surrounding text such as a fallback paint colour.
//
// Takes value (string) which may contain url(#id) references.
// Takes renameMap (map[string]string) which maps identifiers to their namespaced form.
//
// Returns string which is the value with namespaced references rewritten.
func rewriteURLRefs(value string, renameMap map[string]string) string {
	if !strings.Contains(value, urlReferencePrefix) {
		return value
	}

	var builder strings.Builder
	builder.Grow(len(value) + 16)

	rest := value
	for {
		start := strings.Index(rest, urlReferencePrefix)
		if start < 0 {
			builder.WriteString(rest)
			break
		}
		builder.WriteString(rest[:start])
		rest = rest[start:]

		closeIndex := strings.IndexByte(rest, ')')
		if closeIndex < 0 {
			builder.WriteString(rest)
			break
		}

		builder.WriteString(rewriteOneURL(rest[:closeIndex+1], renameMap))
		rest = rest[closeIndex+1:]
	}

	return builder.String()
}

// rewriteOneURL rewrites a single url(...) segment when it is a local fragment
// reference to a namespaced identifier, otherwise it returns the segment unchanged.
//
// Takes segment (string) which is a single url(...) reference.
// Takes renameMap (map[string]string) which maps identifiers to their namespaced form.
//
// Returns string which is the rewritten segment, or the original when not a local match.
func rewriteOneURL(segment string, renameMap map[string]string) string {
	inner := segment[len(urlReferencePrefix) : len(segment)-1]
	trimmed := strings.TrimSpace(inner)

	quote := ""
	if len(trimmed) > 0 && (trimmed[0] == '\'' || trimmed[0] == '"') {
		quote = string(trimmed[0])
		trimmed = strings.TrimSuffix(trimmed[1:], quote)
	}

	if !strings.HasPrefix(trimmed, "#") {
		return segment
	}

	renamed, ok := renameMap[trimmed[1:]]
	if !ok {
		return segment
	}

	return urlReferencePrefix + quote + "#" + renamed + quote + ")"
}

// tokeniseSVG splits inner SVG content into an ordered token stream. The bodies of <style>
// and <script> elements are captured whole so their contents are never mis-parsed as markup.
//
// Takes content (string) which is the SVG fragment to tokenise.
//
// Returns the token stream, and ok false when a construct is unterminated or the token limit
// is exceeded.
func tokeniseSVG(content string) (tokens []svgToken, ok bool) {
	position := 0

	for position < len(content) {
		if len(tokens) > maxSVGTransformTokens {
			return nil, false
		}

		if content[position] != '<' {
			text, consumed := tokeniseText(content, position)
			tokens = append(tokens, text)
			position += consumed
			continue
		}

		emitted, consumed, success := tokeniseMarkup(content, position)
		if !success {
			return nil, false
		}
		tokens = append(tokens, emitted...)
		position += consumed
	}

	return tokens, true
}

// tokeniseText captures literal character data from position up to the next tag.
//
// Takes content (string) which is the SVG fragment.
// Takes position (int) which is the offset where the text begins.
//
// Returns the text token and the number of bytes consumed.
func tokeniseText(content string, position int) (token svgToken, consumed int) {
	next := strings.IndexByte(content[position:], '<')
	if next < 0 {
		return svgToken{kind: tokenText, raw: content[position:]}, len(content) - position
	}
	return svgToken{kind: tokenText, raw: content[position : position+next]}, next
}

// tokeniseMarkup parses the markup construct starting at position (a comment, CDATA section,
// processing instruction, doctype, end tag, or start tag).
//
// Takes content (string) which is the SVG fragment.
// Takes position (int) which is the offset of the markup construct.
//
// Returns the tokens produced, the bytes consumed, and whether parsing succeeded.
func tokeniseMarkup(content string, position int) (tokens []svgToken, consumed int, ok bool) {
	remaining := content[position:]

	for _, marker := range verbatimMarkers {
		if !strings.HasPrefix(remaining, marker[0]) {
			continue
		}
		segment, used, found := sliceUntil(content, position, marker[0], marker[1])
		if !found {
			return nil, 0, false
		}
		return []svgToken{{kind: tokenVerbatim, raw: segment}}, used, true
	}

	if strings.HasPrefix(remaining, "</") {
		end := strings.IndexByte(remaining, '>')
		if end < 0 {
			return nil, 0, false
		}
		return []svgToken{{kind: tokenEnd, name: strings.TrimSpace(remaining[2:end])}}, end + 1, true
	}

	if strings.HasPrefix(remaining, "<!") {
		end := strings.IndexByte(remaining, '>')
		if end < 0 {
			return nil, 0, false
		}
		return []svgToken{{kind: tokenVerbatim, raw: remaining[:end+1]}}, end + 1, true
	}

	return tokeniseStartTag(content, position)
}

// sliceUntil returns the substring from an opening marker through its closing marker.
//
// Takes content (string) which is the SVG fragment.
// Takes position (int) which is the offset of the opening marker.
// Takes opening (string) which is the opening delimiter.
// Takes closing (string) which is the closing delimiter.
//
// Returns the segment, the bytes consumed, and whether the closing marker was found.
func sliceUntil(content string, position int, opening, closing string) (segment string, consumed int, found bool) {
	searchFrom := position + len(opening)
	if searchFrom > len(content) {
		return "", 0, false
	}
	end := strings.Index(content[searchFrom:], closing)
	if end < 0 {
		return "", 0, false
	}
	total := len(opening) + end + len(closing)
	return content[position : position+total], total, true
}

// tokeniseStartTag parses a start tag at position. For <style> and <script> it also captures
// the element body and matching end tag so the body is treated as opaque text.
//
// Takes content (string) which is the SVG fragment.
// Takes position (int) which is the offset of the start tag.
//
// Returns the tokens produced, the bytes consumed, and whether parsing succeeded.
func tokeniseStartTag(content string, position int) (tokens []svgToken, consumed int, ok bool) {
	tagEnd := indexTagEnd(content, position)
	if tagEnd < 0 {
		return nil, 0, false
	}

	inner := content[position+1 : tagEnd]
	selfClose := false
	if strings.HasSuffix(inner, "/") {
		selfClose = true
		inner = inner[:len(inner)-1]
	}

	name, attributeBody := splitTagName(inner)
	if name == "" {
		return nil, 0, false
	}

	startToken := svgToken{
		kind:       tokenStart,
		name:       name,
		attributes: parseTagAttributes(attributeBody),
		selfClose:  selfClose,
		raw:        "",
	}

	bodyStart := tagEnd + 1
	if selfClose || (name != "style" && name != "script") {
		return []svgToken{startToken}, bodyStart - position, true
	}

	return tokeniseRawTextElement(content, position, bodyStart, name, startToken)
}

// tokeniseRawTextElement captures the body and end tag of a <style> or <script> element
// whose content must not be parsed as markup.
//
// Takes content (string) which is the SVG fragment.
// Takes position (int) which is the offset of the start tag.
// Takes bodyStart (int) which is the offset where the element body begins.
// Takes name (string) which is the element name, style or script.
// Takes startToken (svgToken) which is the already-parsed start tag.
//
// Returns the tokens produced, the bytes consumed, and whether the close tag was found.
func tokeniseRawTextElement(content string, position, bodyStart int, name string, startToken svgToken) (tokens []svgToken, consumed int, ok bool) {
	closeStart := indexCloseTag(content, bodyStart, name)
	if closeStart < 0 {
		return nil, 0, false
	}
	closeEnd := strings.IndexByte(content[closeStart:], '>')
	if closeEnd < 0 {
		return nil, 0, false
	}

	bodyKind := tokenScriptText
	if name == "style" {
		bodyKind = tokenStyleText
	}

	produced := []svgToken{
		startToken,
		{kind: bodyKind, raw: content[bodyStart:closeStart]},
		{kind: tokenEnd, name: name},
	}
	return produced, (closeStart + closeEnd + 1) - position, true
}

// indexTagEnd skips over quoted attribute values to find the closing > of a tag.
//
// Takes content (string) which is the SVG fragment.
// Takes position (int) which is the offset of the opening <.
//
// Returns the index of the closing >, or -1 when the tag is unterminated.
func indexTagEnd(content string, position int) int {
	index := position + 1
	for index < len(content) {
		character := content[index]
		if character == '"' || character == '\'' {
			index++
			for index < len(content) && content[index] != character {
				index++
			}
			if index >= len(content) {
				return -1
			}
			index++
			continue
		}
		if character == '>' {
			return index
		}
		index++
	}
	return -1
}

// splitTagName separates the element name from the remaining attribute text inside a start
// tag.
//
// Takes inner (string) which is the tag content between < and >.
//
// Returns the element name and the remaining attribute text.
func splitTagName(inner string) (name string, attributeBody string) {
	inner = strings.TrimLeft(inner, " \t\r\n")
	index := 0
	for index < len(inner) {
		character := inner[index]
		if isWhitespaceASCII(character) || character == '/' {
			break
		}
		index++
	}
	return inner[:index], inner[index:]
}

// parseTagAttributes parses the attributes of a start tag, preserving names, values, and the
// original quote character for each.
//
// Takes body (string) which is the attribute text after the element name.
//
// Returns []svgTagAttribute which holds the parsed attributes.
func parseTagAttributes(body string) []svgTagAttribute {
	var attributes []svgTagAttribute
	position := 0
	for position < len(body) {
		attribute, newPosition, found := parseOneTagAttribute(body, position)
		position = newPosition
		if found && attribute.name != "" {
			attributes = append(attributes, attribute)
		}
	}
	return attributes
}

// parseOneTagAttribute parses a single attribute starting at position.
//
// Takes body (string) which is the attribute text.
// Takes position (int) which is the offset to parse from.
//
// Returns the parsed attribute, the position after it, and whether one was produced.
// newPosition always advances so the caller cannot loop forever.
func parseOneTagAttribute(body string, position int) (attribute svgTagAttribute, newPosition int, found bool) {
	position = skipWhitespaceASCII(body, position)
	if position >= len(body) {
		return svgTagAttribute{}, position, false
	}

	name, afterName := parseAttrName(body, position)
	if name == "" {
		return svgTagAttribute{}, position + 1, false
	}

	position = skipWhitespaceASCII(body, afterName)
	if position >= len(body) || body[position] != '=' {
		return svgTagAttribute{name: name, value: "", quote: 0, hasValue: false}, position, true
	}

	position = skipWhitespaceASCII(body, position+1)
	if position >= len(body) {
		return svgTagAttribute{name: name, value: "", quote: 0, hasValue: false}, position, true
	}

	value, quote, after := parseTagAttributeValue(body, position)
	return svgTagAttribute{name: name, value: value, quote: quote, hasValue: true}, after, true
}

// parseTagAttributeValue reads a quoted or unquoted attribute value starting at position.
//
// Takes body (string) which is the attribute text.
// Takes position (int) which is the offset where the value begins.
//
// Returns the raw value, the quote character (a double quote for unquoted values), and the
// position after the value.
func parseTagAttributeValue(body string, position int) (value string, quote byte, newPosition int) {
	quote = body[position]
	if quote == '"' || quote == '\'' {
		valueStart := position + 1
		valueEnd := valueStart
		for valueEnd < len(body) && body[valueEnd] != quote {
			valueEnd++
		}
		newPosition = valueEnd
		if newPosition < len(body) {
			newPosition++
		}
		return body[valueStart:valueEnd], quote, newPosition
	}

	valueStart := position
	for position < len(body) && !isWhitespaceASCII(body[position]) && body[position] != '>' {
		position++
	}
	return body[valueStart:position], '"', position
}

// indexCloseTag finds the </name> sequence that closes a raw-text element, matching the name
// case-insensitively.
//
// Takes content (string) which is the SVG fragment.
// Takes from (int) which is the offset to search from.
// Takes name (string) which is the element name to match.
//
// Returns the index of the </name> sequence, or -1 when it is not found.
func indexCloseTag(content string, from int, name string) int {
	search := from
	for {
		relative := strings.Index(content[search:], "</")
		if relative < 0 {
			return -1
		}
		nameStart := search + relative + 2
		nameEnd := nameStart + len(name)
		if nameEnd <= len(content) && strings.EqualFold(content[nameStart:nameEnd], name) {
			if nameEnd == len(content) || content[nameEnd] == '>' || isWhitespaceASCII(content[nameEnd]) {
				return search + relative
			}
		}
		search = nameStart
	}
}
