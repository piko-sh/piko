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
	"fmt"
	"strings"

	qt "github.com/valyala/quicktemplate"
	"piko.sh/piko/internal/ast/ast_domain"
)

const (
	// attributeHref is the HTML href attribute name for link elements.
	attributeHref = "href"

	// attributeLang is the HTML attribute name for language override.
	attributeLang = "lang"
)

// transformHrefForLocale applies locale routing rules to a link target.
//
// Takes href (string) which is the link target to change.
// Takes langOverride (string) which replaces the context locale when set.
// Takes hasLangAttr (bool) which reports whether the link specifies a locale override.
// Takes rctx (*renderContext) which provides the current render context.
//
// Returns string which contains the routed href or the original href when routing is
// skipped.
func transformHrefForLocale(href string, langOverride string, hasLangAttr bool, rctx *renderContext) string {
	if shouldSkipHrefTransform(href) {
		return href
	}

	effectiveLocale, skip := resolveEffectiveLocale(langOverride, hasLangAttr, rctx)
	if skip {
		return href
	}

	return applyLocaleStrategy(href, effectiveLocale, rctx)
}

// shouldSkipHrefTransform checks whether a link should be left unchanged during locale
// transformation.
//
// Takes href (string) which is the link to check.
//
// Returns bool which is true when the href is empty, starts with http://, https://, or #
// (fragment), or uses a special scheme such as mailto or tel.
func shouldSkipHrefTransform(href string) bool {
	if href == "" {
		return true
	}
	switch {
	case strings.HasPrefix(href, "http://"),
		strings.HasPrefix(href, "https://"),
		strings.HasPrefix(href, "#"),
		strings.HasPrefix(href, "mailto:"),
		strings.HasPrefix(href, "tel:"):
		return true
	}
	return false
}

// resolveEffectiveLocale works out which locale to use for URL changes.
//
// Takes langOverride (string) which is a locale from a lang attribute, or empty if there
// is no override.
// Takes hasLangAttr (bool) which shows whether a lang attribute was present.
// Takes rctx (*renderContext) which holds the current locale and i18n settings.
//
// Returns string which is the locale to use for the URL change.
// Returns bool which is true when the URL change should be skipped.
func resolveEffectiveLocale(langOverride string, hasLangAttr bool, rctx *renderContext) (string, bool) {
	effectiveLocale := rctx.currentLocale

	if hasLangAttr {
		if langOverride == "" {
			return "", true
		}
		effectiveLocale = langOverride
	}

	if effectiveLocale == "" || rctx.i18nStrategy == "" || rctx.i18nStrategy == "disabled" {
		return "", true
	}

	return effectiveLocale, false
}

// applyLocaleStrategy changes a link to include locale information based on the routing
// strategy set in the render context.
//
// Takes href (string) which is the link to change.
// Takes effectiveLocale (string) which is the target locale for the link.
// Takes rctx (*renderContext) which provides the routing strategy settings.
//
// Returns string which is the link with locale information added.
func applyLocaleStrategy(href, effectiveLocale string, rctx *renderContext) string {
	switch rctx.i18nStrategy {
	case "prefix":
		return applyPrefixStrategy(href, effectiveLocale)
	case "prefix_except_default":
		return applyPrefixExceptDefaultStrategy(href, effectiveLocale, rctx.defaultLocale)
	case "query-only":
		return applyQueryStrategy(href, effectiveLocale)
	default:
		return href
	}
}

// applyPrefixStrategy adds a locale prefix to paths that do not already have one.
//
// Takes href (string) which is the path to check and possibly prefix.
// Takes locale (string) which is the locale code to use as a prefix.
//
// Returns string which is the path with the locale prefix added, such as /en/about, or
// the original path if it already has the prefix.
func applyPrefixStrategy(href, locale string) string {
	if !strings.HasPrefix(href, "/"+locale+"/") {
		return joinLocalePath(locale, href)
	}
	return href
}

// applyPrefixExceptDefaultStrategy adds a locale prefix to the href for non-default
// locales only.
//
// Takes href (string) which is the path to add a prefix to.
// Takes locale (string) which is the current locale.
// Takes defaultLocale (string) which is the locale that needs no prefix.
//
// Returns string which is the href unchanged if locale matches the default, or the href
// with the locale path added as a prefix.
func applyPrefixExceptDefaultStrategy(href, locale, defaultLocale string) string {
	if locale != defaultLocale && !strings.HasPrefix(href, "/"+locale+"/") {
		return joinLocalePath(locale, href)
	}
	return href
}

// applyQueryStrategy adds a locale query parameter to the given URL.
//
// Takes href (string) which is the URL to modify.
// Takes locale (string) which is the locale value to add.
//
// Returns string which is the URL with the locale parameter added. Appends &locale= if
// the URL already has query parameters, or ?locale= otherwise.
func applyQueryStrategy(href, locale string) string {
	if strings.Contains(href, "?") {
		return href + "&locale=" + locale
	}
	return href + "?locale=" + locale
}

// joinLocalePath adds a locale prefix to a link path.
//
// The root path becomes the locale root without a final slash. Other paths keep their
// trailing slash so directory routes still match.
//
// Takes locale (string) which is the locale prefix to add.
// Takes href (string) which is the path to join after the locale.
//
// Returns string which is the full path with the locale prefix.
func joinLocalePath(locale, href string) string {
	if href == "" || href == "/" {
		return "/" + locale
	}
	if href[0] == '/' {
		return "/" + locale + href
	}
	return "/" + locale + "/" + href
}

// renderPikoA writes a locale-aware anchor without changing the template tree.
//
// Takes ro (*RenderOrchestrator) which provides the rendering context and child rendering
// capabilities.
// Takes node (*ast_domain.TemplateNode) which is the piko:a node to render.
// Takes qw (*qt.Writer) which is the output stream to write to.
// Takes rctx (*renderContext) which contains locale and rendering state.
//
// Returns error when rendering child content fails.
func renderPikoA(
	ro *RenderOrchestrator,
	node *ast_domain.TemplateNode,
	qw *qt.Writer,
	rctx *renderContext,
) error {
	LinkTransformCount.Add(rctx.originalCtx, 1)

	qw.N().Z(openBracket)
	qw.N().S("a")

	writeLinkUserAttributes(node.Attributes, qw)

	href, lang, hasLang := extractLinkAttrs(node.Attributes)
	if override := findAttributeWriter(node.AttributeWriters, "data-locale"); override != nil {
		lang, hasLang = override.StringRaw(), true
	}
	boundHref := findAttributeWriter(node.AttributeWriters, attributeHref)
	if boundHref == nil || boundHref.HasParts() {
		qw.N().Z(space)
		qw.N().S(attributeHref)
		qw.N().Z(equalsQuote)
		if boundHref == nil {
			qw.N().S(transformHrefForLocale(href, lang, hasLang, rctx))
		} else {
			qw.E().S(transformHrefForLocale(boundHref.StringRaw(), lang, hasLang, rctx))
		}
		qw.N().Z(quote)
	}
	writeLinkMarkerOnly(qw)

	ro.writeElementDirectivesExcluding(node, qw, rctx, attributeHref)

	qw.N().Z(closeBracket)

	if err := ro.renderNodeContent(node, qw, rctx); err != nil {
		return fmt.Errorf("rendering content for <piko:a>: %w", err)
	}

	qw.N().Z(closeTagPrefix)
	qw.N().S("a")
	qw.N().Z(closeBracket)

	return nil
}

// hasAttributeWriter checks whether an attribute is bound via an attribute writer.
//
// Takes writers ([]*ast_domain.DirectWriter) which contains the attribute writers to
// search.
// Takes attributeName (string) which specifies the attribute name to find.
//
// Returns bool which is true if a matching writer exists.
func hasAttributeWriter(writers []*ast_domain.DirectWriter, attributeName string) bool {
	return findAttributeWriter(writers, attributeName) != nil
}

// findAttributeWriter finds the first writer with a matching attribute name.
//
// Takes writers ([]*ast_domain.DirectWriter) which contains the writers to search.
// Takes attributeName (string) which is the exact attribute name to match.
//
// Returns *ast_domain.DirectWriter which is the matching non-nil writer, or nil when
// absent.
func findAttributeWriter(writers []*ast_domain.DirectWriter, attributeName string) *ast_domain.DirectWriter {
	for _, w := range writers {
		if w != nil && w.Name == attributeName {
			return w
		}
	}
	return nil
}

// extractLinkAttrs finds a link's href and locale override.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the link attributes to search.
//
// Returns href (string) which is the href attribute value, or empty if not found.
// Returns langOverride (string) which holds data-locale when present or lang otherwise.
// Returns hasLang (bool) which reports whether either locale attribute was present.
func extractLinkAttrs(attrs []ast_domain.HTMLAttribute) (href, langOverride string, hasLang bool) {
	var explicitLocale bool
	for i := range attrs {
		switch attrs[i].Name {
		case attributeHref:
			href = attrs[i].Value
		case attributeLang:
			if !explicitLocale {
				langOverride, hasLang = attrs[i].Value, true
			}
		case "data-locale":
			langOverride, hasLang, explicitLocale = attrs[i].Value, true, true
		}
	}
	return href, langOverride, hasLang
}

// writeLinkUserAttributes writes user-defined attributes to the output, skipping reserved
// attributes such as href, lang, and piko:a.
//
// Takes attrs ([]ast_domain.HTMLAttribute) which contains the attributes to filter and
// write.
// Takes qw (*qt.Writer) which is the output writer for the rendered HTML.
func writeLinkUserAttributes(attrs []ast_domain.HTMLAttribute, qw *qt.Writer) {
	for i := range attrs {
		attr := &attrs[i]
		if attr.Name == attributeHref || attr.Name == attributeLang || attr.Name == tagPikoA {
			continue
		}
		qw.N().Z(space)
		qw.N().S(attr.Name)
		qw.N().Z(equalsQuote)
		qw.N().S(attr.Value)
		qw.N().Z(quote)
	}
}

// writeLinkMarkerOnly writes the marker used for client-side navigation.
//
// Takes qw (*qt.Writer) which receives the marker attribute output.
func writeLinkMarkerOnly(qw *qt.Writer) {
	qw.N().Z(space)
	qw.N().S(tagPikoA)
	qw.N().Z(equalsQuote)
	qw.N().Z(quote)
}
