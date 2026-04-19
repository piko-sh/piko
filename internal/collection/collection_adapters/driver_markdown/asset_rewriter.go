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

package driver_markdown

import (
	"context"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/assetpath"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/htmllexer"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// tagNameImg is the lowercased tag name for HTML image elements.
	tagNameImg = "img"

	// tagNameAnchor is the lowercased tag name for HTML anchor elements.
	tagNameAnchor = "a"

	// tagNamePikoAnchor is the Piko-namespaced anchor tag. Element-level
	// anchors that we successfully rewrite to an in-collection URL are
	// promoted to this tag so the runtime renderer emits the soft-navigation
	// marker attribute and the frontend intercepts the click.
	tagNamePikoAnchor = "piko:a"

	// attributeSrc is the HTML attribute that carries image references.
	attributeSrc = "src"

	// attributeHref is the HTML attribute that carries hyperlink references.
	attributeHref = "href"

	// markdownExtension is the file suffix that marks an anchor href as a
	// link to another markdown document inside the collection.
	markdownExtension = ".md"

	// maxRawHTMLReplacements caps how many attribute rewrites a single raw
	// HTML block may produce, to stop a pathological document from
	// unbounded registrar or analyser calls.
	maxRawHTMLReplacements = 256

	// initialReplacementCapacity pre-sizes the replacements slice; most raw
	// HTML blocks contain a small handful of rewritten attributes.
	initialReplacementCapacity = 4
)

// srcReplacement identifies a single attribute-value span to replace inside
// a raw HTML blob.
type srcReplacement struct {
	// replacement is the bytes to splice into the source, including the
	// surrounding quote characters when the original was quoted.
	replacement string

	// start is the byte offset in the original source where the replacement
	// span begins (inclusive).
	start int

	// end is the byte offset in the original source where the replacement
	// span ends (exclusive).
	end int
}

// rewriteState carries the per-blob state for the streaming rewrite.
type rewriteState struct {
	// registrar resolves a sandbox-relative asset path to a serve URL and
	// registers the bytes with the artefact registry. When nil, <img src>
	// rewriting is disabled.
	registrar collection_domain.AssetRegistrar

	// analyser maps a relative .md file path to its public URL inside the
	// collection. When nil, <a href> rewriting is disabled.
	analyser *pathAnalyser

	// sandbox provides read access to the source content tree where the
	// relative srcs are resolved.
	sandbox safedisk.Sandbox

	// mdDirectory is the directory of the originating .md file inside the
	// sandbox, used as the anchor when joining relative src or href values.
	mdDirectory string

	// collectionName scopes the artefactID emitted for each registered
	// asset, and prefixes the public URL produced for anchor hrefs.
	collectionName string

	// mdRelativePath is the .md file path used for diagnostic logging only.
	mdRelativePath string

	// replacements accumulates the attribute spans to splice into the
	// rewritten source once lexing completes.
	replacements []srcReplacement

	// currentTag is the lowercased name of the start tag the lexer is
	// currently streaming attribute tokens for, or "" when between tags.
	// Only tagNameImg and tagNameAnchor produce rewrites; other tags set
	// this back to "".
	currentTag string
}

// rewriteRelativeURLs parses a raw HTML blob and rewrites two kinds of
// references in a single pass:
//
//  1. <img src="..."> values that point at relative asset files. Each
//     resolved path is registered via registrar and the src is replaced
//     with the returned serve URL. Skipped when registrar is nil.
//  2. <a href="..."> values that point at sibling .md files in the same
//     collection. Each resolved path is mapped to its public URL via
//     analyser. Skipped when analyser is nil.
//
// Each rewriter is independent. Passing nil for one disables that class
// of rewrite without affecting the other. Absolute URLs, fragment-only
// hrefs (#section), data URIs, and paths that escape the sandbox are
// always left untouched.
//
// Takes sandbox (safedisk.Sandbox) which provides read access to the
// source content tree; used by the registrar to read asset file bytes.
// Takes registrar (collection_domain.AssetRegistrar) which uploads the
// asset bytes and returns a serve URL. Pass nil to disable img rewriting.
// Takes analyser (*pathAnalyser) which converts a collection-relative .md
// path to its public URL. Pass nil to disable anchor href rewriting.
// Takes mdDirectory (string) which is the directory of the originating
// .md file inside the collection; used as the anchor for resolving
// relative src and href values.
// Takes collectionName (string) which scopes the artefactID for assets
// and prefixes the public URL produced for anchor hrefs.
// Takes mdRelativePath (string) which is the .md file path used for
// diagnostic logging only.
// Takes raw (string) which is the raw HTML blob to rewrite.
//
// Returns string which is the rewritten HTML, or the original blob when
// no rewrites were applied.
// Returns bool which is true when at least one attribute was rewritten.
func rewriteRelativeURLs(
	ctx context.Context,
	sandbox safedisk.Sandbox,
	registrar collection_domain.AssetRegistrar,
	analyser *pathAnalyser,
	mdDirectory string,
	collectionName string,
	mdRelativePath string,
	raw string,
) (string, bool) {
	ctx, l := logger_domain.From(ctx, log)

	if (registrar == nil && analyser == nil) || sandbox == nil || raw == "" {
		return raw, false
	}

	source := []byte(raw)
	lexer := htmllexer.NewLexer(source)
	state := &rewriteState{
		registrar:      registrar,
		analyser:       analyser,
		sandbox:        sandbox,
		mdDirectory:    mdDirectory,
		collectionName: collectionName,
		mdRelativePath: mdRelativePath,
		replacements:   make([]srcReplacement, 0, initialReplacementCapacity),
	}

	for {
		if err := ctx.Err(); err != nil {
			l.Warn("Context cancelled while rewriting raw HTML attributes",
				logger_domain.String(keyPath, mdRelativePath),
				logger_domain.Error(err))
			return raw, false
		}

		done, stop := state.handleToken(ctx, lexer)
		if stop {
			return applyReplacements(raw, source, state.replacements), len(state.replacements) > 0
		}
		if done {
			continue
		}
	}
}

// handleToken processes the next lexer token into rewrite state.
//
// Takes lexer (*htmllexer.Lexer) which is advanced one token by this call.
//
// Returns done (bool) which is always true for processed tokens, for
// symmetry with the caller's continue pattern.
// Returns stop (bool) which is true when the caller should exit the loop
// (error token reached or replacement cap exceeded).
func (s *rewriteState) handleToken(ctx context.Context, lexer *htmllexer.Lexer) (done, stop bool) {
	switch lexer.Next() {
	case htmllexer.ErrorToken:
		return true, true

	case htmllexer.StartTagToken:
		s.currentTag = ""
		tag := string(lexer.Text())
		if strings.EqualFold(tag, tagNameImg) {
			s.currentTag = tagNameImg
		} else if strings.EqualFold(tag, tagNameAnchor) {
			s.currentTag = tagNameAnchor
		}

	case htmllexer.StartTagCloseToken, htmllexer.StartTagVoidToken, htmllexer.EndTagToken:
		s.currentTag = ""

	case htmllexer.AttributeToken:
		return s.handleAttribute(ctx, lexer)
	}
	return true, false
}

// handleAttribute processes an attribute token for the current tag and
// dispatches to the appropriate per-tag rewriter. Attributes outside a
// recognised start tag, or for an attribute other than the one carrying
// the rewriteable reference, are ignored.
//
// Takes lexer (*htmllexer.Lexer) which is positioned on the attribute
// token to inspect.
//
// Returns done (bool) which is true after processing.
// Returns stop (bool) which is true when the replacement cap has been
// reached and the loop should exit.
func (s *rewriteState) handleAttribute(ctx context.Context, lexer *htmllexer.Lexer) (done, stop bool) {
	if s.currentTag == "" {
		return true, false
	}
	if len(s.replacements) >= maxRawHTMLReplacements {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Exceeded maximum attribute rewrites in raw HTML block; skipping remainder",
			logger_domain.String(keyPath, s.mdRelativePath),
			logger_domain.Int("max", maxRawHTMLReplacements))
		return true, true
	}

	switch s.currentTag {
	case tagNameImg:
		if s.registrar == nil {
			return true, false
		}
		if !strings.EqualFold(string(lexer.Text()), attributeSrc) {
			return true, false
		}
		rep, ok := buildImgSrcReplacement(ctx, s.sandbox, s.registrar, s.mdDirectory, s.collectionName, s.mdRelativePath, lexer)
		if !ok {
			return true, false
		}
		s.replacements = append(s.replacements, rep)

	case tagNameAnchor:
		if s.analyser == nil {
			return true, false
		}
		if !strings.EqualFold(string(lexer.Text()), attributeHref) {
			return true, false
		}
		rep, ok := buildAnchorHrefReplacement(s.analyser, s.mdDirectory, s.collectionName, lexer)
		if !ok {
			return true, false
		}
		s.replacements = append(s.replacements, rep)
	}
	return true, false
}

// buildImgSrcReplacement inspects the current AttributeToken on lexer, and
// if the value is a relative asset path inside the sandbox, registers the
// asset and prepares a replacement span for splicing back into the source.
//
// Takes sandbox (safedisk.Sandbox) which provides read access to the
// source content tree for the registrar.
// Takes registrar (collection_domain.AssetRegistrar) which uploads the
// bytes and returns a serve URL.
// Takes mdDirectory (string) which is the directory of the originating
// .md file used as the anchor for resolving relative src values.
// Takes collectionName (string) which scopes the artefactID.
// Takes mdRelativePath (string) which is the .md file path used for
// diagnostic logging only.
// Takes lexer (*htmllexer.Lexer) which is positioned on the attribute
// token whose value may be rewritten.
//
// Returns srcReplacement which describes the span to replace and the
// replacement text (with quote characters preserved where present).
// Returns bool which is true when a rewrite should be applied; false when
// the src should be left untouched.
func buildImgSrcReplacement(
	ctx context.Context,
	sandbox safedisk.Sandbox,
	registrar collection_domain.AssetRegistrar,
	mdDirectory string,
	collectionName string,
	mdRelativePath string,
	lexer *htmllexer.Lexer,
) (srcReplacement, bool) {
	ctx, l := logger_domain.From(ctx, log)

	rawValue := lexer.AttrVal()
	if len(rawValue) == 0 {
		return srcReplacement{}, false
	}

	trimmed, opener, closer := splitQuotes(rawValue)
	srcValue := string(trimmed)

	if !assetpath.NeedsTransform(srcValue, assetpath.DefaultServePath) {
		return srcReplacement{}, false
	}

	resolved := filepath.Clean(filepath.Join(mdDirectory, srcValue))
	if resolved == "." || resolved == "" || strings.HasPrefix(resolved, "..") || filepath.IsAbs(resolved) {
		l.Warn("Skipping raw HTML asset that escapes sandbox",
			logger_domain.String(keyPath, mdRelativePath),
			logger_domain.String("src", srcValue))
		return srcReplacement{}, false
	}

	serveURL, err := registrar.RegisterCollectionAsset(ctx, sandbox, resolved, collectionName)
	if err != nil {
		l.Warn("Failed to register raw HTML asset; preserving original src",
			logger_domain.String(keyPath, mdRelativePath),
			logger_domain.String("src", srcValue),
			logger_domain.String("resolved", resolved),
			logger_domain.Error(err))
		return srcReplacement{}, false
	}

	valueStart := lexer.AttrValStart()
	if valueStart < 0 {
		return srcReplacement{}, false
	}

	newValue := serveURL
	if opener != 0 {
		var sb strings.Builder
		sb.Grow(len(newValue) + 2)
		sb.WriteByte(opener)
		sb.WriteString(newValue)
		if closer != 0 {
			sb.WriteByte(closer)
		}
		newValue = sb.String()
	}

	return srcReplacement{
		start:       valueStart,
		end:         valueStart + len(rawValue),
		replacement: newValue,
	}, true
}

// splitQuotes returns the trimmed inner bytes of an attribute value along
// with the opening and closing quote characters when present. An unquoted
// value returns zero quote bytes.
//
// Takes rawValue ([]byte) which is the attribute value as returned by
// htmllexer.Lexer.AttrVal (may include surrounding quotes).
//
// Returns trimmed ([]byte) which is the value without surrounding quotes.
// Returns opener (byte) which is the opening quote character, or zero when
// the value was unquoted.
// Returns closer (byte) which is the closing quote character, or zero when
// the value was unquoted or had no matching closer.
func splitQuotes(rawValue []byte) (trimmed []byte, opener byte, closer byte) {
	if len(rawValue) == 0 {
		return rawValue, 0, 0
	}
	first := rawValue[0]
	if first != '"' && first != '\'' {
		return rawValue, 0, 0
	}
	opener = first
	if len(rawValue) >= 2 && rawValue[len(rawValue)-1] == first {
		closer = first
		return rawValue[1 : len(rawValue)-1], opener, closer
	}
	return rawValue[1:], opener, 0
}

// applyReplacements splices the given replacements into the original
// source, sorted by start offset, and returns the resulting string.
//
// Takes raw (string) which is the original HTML content, returned
// unchanged when there are no replacements.
// Takes source ([]byte) which is the byte view of raw shared with the
// lexer; used to read the spans outside any replacement.
// Takes replacements ([]srcReplacement) which lists the spans to rewrite.
//
// Returns string which is the rewritten content.
func applyReplacements(raw string, source []byte, replacements []srcReplacement) string {
	if len(replacements) == 0 {
		return raw
	}

	sortReplacementsByStart(replacements)

	var sb strings.Builder
	sb.Grow(len(source))
	cursor := 0
	for _, rep := range replacements {
		if rep.start < cursor || rep.end > len(source) || rep.start > rep.end {
			continue
		}
		sb.Write(source[cursor:rep.start])
		sb.WriteString(rep.replacement)
		cursor = rep.end
	}
	if cursor < len(source) {
		sb.Write(source[cursor:])
	}
	return sb.String()
}

// sortReplacementsByStart sorts replacements ascending by start offset.
// Uses an insertion sort because replacement counts are small (bounded by
// maxRawHTMLReplacements) and the lexer emits attribute tokens in source
// order, so the slice is already near-sorted on arrival.
//
// Takes replacements ([]srcReplacement) which is sorted in place.
func sortReplacementsByStart(replacements []srcReplacement) {
	for i := 1; i < len(replacements); i++ {
		current := replacements[i]
		j := i - 1
		for j >= 0 && replacements[j].start > current.start {
			replacements[j+1] = replacements[j]
			j--
		}
		replacements[j+1] = current
	}
}

// resolveAnchorHref maps a relative .md href to the public URL it should
// be served at on the website.
//
// The source markdown is designed to render correctly on GitHub and
// similar static viewers, where internal links carry the .md extension.
// At build time we strip the extension and rebuild the URL through the
// same path analyser used for the canonical URL of each page, so a link
// like "../tutorials/foo.md#section" inside docs/get-started/install.md
// becomes "/docs/tutorials/foo#section".
//
// Hrefs are passed through unchanged when:
//   - empty, schemed (http://, https://, mailto:, tel:, data:),
//     protocol-relative (//host), or absolute (/path);
//   - fragment-only (#section) or query-only (?key=value);
//   - the path component does not end in .md;
//   - resolution against mdDirectory escapes the collection root.
//
// Fragments and query strings are preserved when present.
//
// Takes href (string) which is the raw href value from the markdown source.
// Takes mdDirectory (string) which is the directory of the .md file that
// contains the link, relative to the collection root.
// Takes collectionName (string) which is the collection the link lives in;
// used by analyser to build the public URL prefix.
// Takes analyser (*pathAnalyser) which produces the public URL for a
// resolved .md path.
//
// Returns string which is the rewritten href; empty when no rewrite
// applies.
// Returns bool which is true when the caller should replace href with the
// returned value.
func resolveAnchorHref(href, mdDirectory, collectionName string, analyser *pathAnalyser) (string, bool) {
	if href == "" || analyser == nil {
		return "", false
	}
	if !assetpath.NeedsTransform(href, assetpath.DefaultServePath) {
		return "", false
	}
	if strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
		return "", false
	}
	if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "?") {
		return "", false
	}

	pathPart, suffix := splitURLSuffix(href)
	if pathPart == "" {
		return "", false
	}
	if !strings.HasSuffix(strings.ToLower(pathPart), markdownExtension) {
		return "", false
	}

	resolved := filepath.ToSlash(filepath.Clean(filepath.Join(mdDirectory, pathPart)))
	if resolved == "." || resolved == "" || strings.HasPrefix(resolved, "../") || resolved == ".." || filepath.IsAbs(resolved) {
		return "", false
	}

	info := analyser.Analyse(resolved, collectionName)
	if info == nil || info.url == "" {
		return "", false
	}
	return info.url + suffix, true
}

// splitURLSuffix splits an href into its path component and a trailing
// fragment-or-query suffix. The suffix retains its leading delimiter so
// it can be concatenated back onto the rewritten path verbatim.
//
// Takes href (string) which is the full href value to split.
//
// Returns pathPart (string) which is the href content before any '#' or
// '?' delimiter.
// Returns suffix (string) which is the delimiter and everything after it,
// or "" when neither delimiter is present.
func splitURLSuffix(href string) (pathPart, suffix string) {
	fragment := strings.IndexByte(href, '#')
	query := strings.IndexByte(href, '?')
	cut := -1
	switch {
	case fragment >= 0 && query >= 0:
		if fragment < query {
			cut = fragment
		} else {
			cut = query
		}
	case fragment >= 0:
		cut = fragment
	case query >= 0:
		cut = query
	}
	if cut < 0 {
		return href, ""
	}
	return href[:cut], href[cut:]
}

// buildAnchorHrefReplacement inspects the current AttributeToken on lexer
// and, when the value is a relative .md link inside the collection,
// prepares a replacement span that points at the public URL.
//
// Takes analyser (*pathAnalyser) which maps a resolved .md path to its
// public URL.
// Takes mdDirectory (string) which is the directory of the .md file
// inside the collection used as the anchor for resolving the href.
// Takes collectionName (string) which is the collection the link lives in.
// Takes lexer (*htmllexer.Lexer) which is positioned on the attribute
// token whose value may be rewritten.
//
// Returns srcReplacement which describes the span to replace and the
// replacement text (with quote characters preserved when present).
// Returns bool which is true when a rewrite should be applied.
func buildAnchorHrefReplacement(
	analyser *pathAnalyser,
	mdDirectory string,
	collectionName string,
	lexer *htmllexer.Lexer,
) (srcReplacement, bool) {
	rawValue := lexer.AttrVal()
	if len(rawValue) == 0 {
		return srcReplacement{}, false
	}

	trimmed, opener, closer := splitQuotes(rawValue)
	hrefValue := string(trimmed)

	newHref, ok := resolveAnchorHref(hrefValue, mdDirectory, collectionName, analyser)
	if !ok {
		return srcReplacement{}, false
	}

	valueStart := lexer.AttrValStart()
	if valueStart < 0 {
		return srcReplacement{}, false
	}

	newValue := newHref
	if opener != 0 {
		var sb strings.Builder
		sb.Grow(len(newValue) + 2)
		sb.WriteByte(opener)
		sb.WriteString(newValue)
		if closer != 0 {
			sb.WriteByte(closer)
		}
		newValue = sb.String()
	}

	return srcReplacement{
		start:       valueStart,
		end:         valueStart + len(rawValue),
		replacement: newValue,
	}, true
}
