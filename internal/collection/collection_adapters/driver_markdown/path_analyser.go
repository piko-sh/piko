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
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// pathSeparator is the forward slash used to join URL path segments.
const pathSeparator = "/"

// asciiControlMaxExclusive is the upper bound (exclusive) of the ASCII C0
// control range. Runes below this are non-printable.
const asciiControlMaxExclusive = 0x20

// asciiDelete is the ASCII DEL character, the only control point above the
// printable range that must also be stripped from slugs.
const asciiDelete = 0x7F

// parentDirSegment matches the path-traversal segment that path_analyser must
// strip from any path component before it becomes part of a slug.
const parentDirSegment = ".."

// indexBasename is the conventional basename for a directory's landing page.
//
// When a file is named "index.md", the slug stored for the item drops the
// trailing /index segment so the item is reachable at the bare directory URL
// (e.g. "tutorials/index.md" stores slug "tutorials" served at /docs/tutorials).
const indexBasename = "index"

// rootIndexSlug is the lookup key used for an index file at the collection
// root. The slug cannot be empty (validateSlugs rejects empty strings) and
// must not collide with any user-authored slug.
const rootIndexSlug = "index"

// maxSlugBytes caps the length of a generated slug. Slugs become FlatBuffer
// keys, log fields and URL captures, so the cap acts as defence-in-depth
// against pathological filenames.
const maxSlugBytes = 1024

// maxSlugDepth caps the number of path segments in a generated slug.
const maxSlugDepth = 32

// pathAnalyser extracts structured data from markdown file paths.
// It finds locale and content type from path patterns.
type pathAnalyser struct {
	// defaultLocale is the fallback locale used when none can be found in the path.
	defaultLocale string

	// locales lists the locale codes to match in URL paths.
	locales []string
}

// pathInfo contains analysed information from a file path.
type pathInfo struct {
	// locale is the detected locale code, for example "en" or "fr".
	locale string

	// slug is the URL-friendly identifier for the item.
	//
	// For flat files such as "my-blog-post.md" the slug is "my-blog-post";
	// for nested files such as "get-started/introduction.md" the slug joins
	// the in-collection directory parts with the basename, producing
	// "get-started/introduction". The runtime lookup keys on this slug.
	slug string

	// translationKey is a locale-independent identifier for linking translations.
	// For example, "blog/my-post" links "en/blog/my-post.md" and "fr/blog/my-post.md".
	translationKey string

	// url is the final URL path for this content, such as "/blog/my-post".
	url string

	// pathSegments contains the directory parts between the collection root and
	// the filename. For example, "blog/2024/post.md" gives ["blog", "2024"].
	pathSegments []string
}

// Analyse extracts structured information from a relative file path.
//
// Takes relativePath (string) which is the path relative to collection root
// (e.g., "en/blog/post.md").
// Takes collectionName (string) which is the name of the collection
// (e.g., "blog").
//
// Returns *pathInfo which contains the analysed path components including
// locale, slug, segments, translation key, and URL.
func (pa *pathAnalyser) Analyse(relativePath, collectionName string) *pathInfo {
	relativePath = sanitiseRelativePath(filepath.ToSlash(relativePath))

	parts := strings.Split(relativePath, "/")
	if len(parts) == 0 {
		return pa.defaultPathInfo(relativePath, collectionName)
	}

	filename := parts[len(parts)-1]
	dirParts := parts[:len(parts)-1]

	locale, cleanDirParts := pa.detectLocale(filename, dirParts)

	var segments []string
	if collectionName != "" {
		segments = append([]string{collectionName}, cleanDirParts...)
	} else {
		segments = cleanDirParts
	}

	basename := pa.generateSlug(filename, locale)

	slug := buildItemSlug(cleanDirParts, basename)

	translationKey := pa.generateTranslationKey(segments, basename)

	url := pa.generateURL(locale, segments, basename)

	return &pathInfo{
		locale:         locale,
		slug:           slug,
		pathSegments:   segments,
		translationKey: translationKey,
		url:            url,
	}
}

// sanitiseRelativePath strips path-traversal segments, NUL bytes and ASCII
// control characters from a slash-separated relative path. Backslashes are
// also normalised to forward slashes so a Windows-style segment like "..\\x"
// does not survive as a single token.
//
// The scanner already operates inside a safedisk sandbox so these segments
// cannot escape the filesystem, but they must not survive into the slug since
// the slug becomes a URL key and the FlatBuffer lookup token.
//
// Takes relativePath (string) which is a slash-separated path relative to a
// collection root.
//
// Returns string which is the path with "..", "." segments, NUL bytes,
// backslashes and control characters removed.
func sanitiseRelativePath(relativePath string) string {
	if relativePath == "" {
		return ""
	}
	relativePath = strings.ReplaceAll(relativePath, "\\", "/")
	relativePath = stripControlChars(relativePath)

	parts := strings.Split(relativePath, "/")
	cleaned := make([]string, 0, len(parts))
	for _, segment := range parts {
		if segment == "" || segment == "." || segment == parentDirSegment {
			continue
		}
		cleaned = append(cleaned, segment)
	}
	return strings.Join(cleaned, "/")
}

// stripControlChars removes ASCII control characters (C0 range and DEL) from
// a string. These characters have no business in a slug and would corrupt log
// output and URL parsing if they reached either.
//
// Takes value (string) which is the source string.
//
// Returns string which is the source with control characters removed.
func stripControlChars(value string) string {
	if !strings.ContainsFunc(value, isControlChar) {
		return value
	}
	var sb strings.Builder
	sb.Grow(len(value))
	for _, r := range value {
		if isControlChar(r) {
			continue
		}
		_, _ = sb.WriteRune(r)
	}
	return sb.String()
}

// isControlChar reports whether r is an ASCII control character.
//
// Takes r (rune) which is the rune to test.
//
// Returns bool which is true for runes in the C0 range or DEL.
func isControlChar(r rune) bool {
	return r < asciiControlMaxExclusive || r == asciiDelete
}

// buildItemSlug forms the canonical slug for a content item.
//
// Joins the in-collection directory parts with the basename. When the
// basename is "index", the slug drops the trailing /index so an index file
// is reachable at its parent directory's URL (e.g. "tutorials/index.md"
// stores slug "tutorials"). For an index file at the collection root,
// rootIndexSlug is used so lookups keyed on an empty path normalise to it.
//
// Takes cleanDirParts ([]string) which are the in-collection directory parts
// after locale stripping.
// Takes basename (string) which is the lowercased filename without
// extension or locale suffix.
//
// Returns string which is the canonical slug, capped at maxSlugDepth
// segments and maxSlugBytes bytes.
func buildItemSlug(cleanDirParts []string, basename string) string {
	if len(cleanDirParts) > maxSlugDepth {
		cleanDirParts = cleanDirParts[:maxSlugDepth]
	}
	if basename == indexBasename {
		if len(cleanDirParts) == 0 {
			return rootIndexSlug
		}
		return capSlugBytes(strings.Join(cleanDirParts, pathSeparator))
	}
	if len(cleanDirParts) == 0 {
		return capSlugBytes(basename)
	}
	parts := make([]string, 0, len(cleanDirParts)+1)
	parts = append(parts, cleanDirParts...)
	parts = append(parts, basename)
	return capSlugBytes(strings.Join(parts, pathSeparator))
}

// capSlugBytes truncates a slug at the last rune boundary that fits within
// maxSlugBytes so the result remains valid UTF-8.
//
// Takes slug (string) which is the source slug.
//
// Returns string which is the slug truncated to a rune boundary not exceeding
// maxSlugBytes.
func capSlugBytes(slug string) string {
	if len(slug) <= maxSlugBytes {
		return slug
	}
	cut := maxSlugBytes
	for cut > 0 && !utf8.RuneStart(slug[cut]) {
		cut--
	}
	return slug[:cut]
}

// detectLocale attempts to detect the locale from the path.
//
// Detection strategies (in order of priority):
//  1. Suffix in filename: "post.fr.md" -> "fr"
//  2. First directory: "fr/blog/post.md" -> "fr"
//  3. Last directory: "blog/fr/post.md" -> "fr"
//  4. Default locale if none found
//
// Takes filename (string) which is the name of the file to check for locale.
// Takes dirParts ([]string) which contains the directory path components.
//
// Returns string which is the detected locale code.
// Returns []string which contains directory parts with the locale removed.
func (pa *pathAnalyser) detectLocale(filename string, dirParts []string) (string, []string) {
	if locale := pa.localeFromFilename(filename); locale != "" {
		return locale, dirParts
	}

	if len(dirParts) > 0 {
		if pa.isLocale(dirParts[0]) {
			return dirParts[0], dirParts[1:]
		}
	}

	if len(dirParts) > 0 {
		lastIndex := len(dirParts) - 1
		if pa.isLocale(dirParts[lastIndex]) {
			locale := dirParts[lastIndex]
			cleanParts := make([]string, lastIndex)
			copy(cleanParts, dirParts[:lastIndex])
			return locale, cleanParts
		}
	}

	return pa.defaultLocale, dirParts
}

// localeFromFilename extracts the locale from a filename suffix.
//
// Takes filename (string) which is the file name to extract the locale from.
//
// Returns string which is the locale code, or empty if no locale is found.
func (pa *pathAnalyser) localeFromFilename(filename string) string {
	name := strings.TrimSuffix(filename, ".md")

	parts := strings.Split(name, ".")
	if len(parts) < 2 {
		return ""
	}

	candidate := parts[len(parts)-1]
	if pa.isLocale(candidate) {
		return candidate
	}

	return ""
}

// isLocale checks if a string matches a configured locale.
//
// Takes s (string) which is the value to check against known locales.
//
// Returns bool which is true if the string matches any configured locale.
func (pa *pathAnalyser) isLocale(s string) bool {
	for _, locale := range pa.locales {
		if strings.EqualFold(s, locale) {
			return true
		}
	}
	return false
}

// generateSlug creates a URL-friendly slug from a filename.
//
// Takes filename (string) which is the source filename to convert.
// Takes detectedLocale (string) which is the locale suffix to remove if present.
//
// Returns string which is the lowercase slug with extension and locale removed.
//
// Process:
//   - Remove .md extension
//   - Remove locale suffix if present
//   - Convert to lowercase
//   - Already assumes kebab-case filename
func (*pathAnalyser) generateSlug(filename, detectedLocale string) string {
	slug := strings.TrimSuffix(filename, ".md")

	if detectedLocale != "" {
		slug = strings.TrimSuffix(slug, "."+detectedLocale)
	}

	slug = strings.ToLower(slug)

	slug = strings.Trim(slug, "-")

	return slug
}

// generateTranslationKey creates a locale-independent identifier.
//
// The translation key links different locale versions of the same content.
//
// Format: "{segments}/{slug}"
//
// Takes segments ([]string) which specifies the path hierarchy.
// Takes slug (string) which provides the content identifier.
//
// Returns string which is the joined translation key.
func (*pathAnalyser) generateTranslationKey(segments []string, slug string) string {
	parts := append([]string{}, segments...)
	parts = append(parts, slug)
	return strings.Join(parts, pathSeparator)
}

// generateURL creates the final public URL for the content.
//
// Takes locale (string) which specifies the content language.
// Takes segments ([]string) which contains the path segments between locale
// and slug.
// Takes slug (string) which is the final path component or "index" for
// directory URLs.
//
// Returns string which is the formatted URL path.
//
// URL Generation Rules:
//   - Default locale: /{segments}/{slug}
//   - Other locales: /{locale}/{segments}/{slug}
//   - Index files: slug="index" produces trailing slash (e.g., /docs/)
func (pa *pathAnalyser) generateURL(locale string, segments []string, slug string) string {
	parts := []string{}

	if locale != pa.defaultLocale {
		parts = append(parts, locale)
	}

	parts = append(parts, segments...)

	isIndex := slug == "index"
	if !isIndex {
		parts = append(parts, slug)
	}

	url := pathSeparator + strings.Join(parts, pathSeparator)

	if isIndex && !strings.HasSuffix(url, pathSeparator) {
		url += pathSeparator
	}

	return url
}

// defaultPathInfo returns a fallback pathInfo when analysis fails.
//
// Takes relativePath (string) which is the file path to derive a basic
// slug from.
//
// Returns *pathInfo which contains default values using the default locale
// and a slug derived from the filename.
func (pa *pathAnalyser) defaultPathInfo(relativePath, _ string) *pathInfo {
	filename := filepath.Base(relativePath)
	slug := strings.TrimSuffix(filename, ".md")
	slug = strings.ToLower(slug)

	return &pathInfo{
		locale:         pa.defaultLocale,
		slug:           slug,
		pathSegments:   []string{},
		translationKey: slug,
		url:            pathSeparator + slug,
	}
}

// newPathAnalyser creates a new path analyser.
//
// Takes locales ([]string) which lists the supported locales (e.g. "en", "fr",
// "de").
// Takes defaultLocale (string) which sets the locale to use when none is found.
//
// Returns *pathAnalyser which is the configured path analyser ready for use.
func newPathAnalyser(locales []string, defaultLocale string) *pathAnalyser {
	return &pathAnalyser{
		locales:       locales,
		defaultLocale: defaultLocale,
	}
}

// slugify converts a string into a URL-friendly slug.
//
// The function makes the text safe for use in URLs by:
//   - Making all letters lowercase
//   - Changing spaces to hyphens
//   - Removing special characters (keeps letters, numbers, and hyphens)
//   - Joining multiple hyphens into one
//   - Removing hyphens from the start and end
//
// Takes s (string) which is the text to convert.
//
// Returns string which is the URL-friendly slug.
func slugify(s string) string {
	s = strings.ToLower(s)

	var result strings.Builder
	lastWasHyphen := false

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			_, _ = result.WriteRune(r)
			lastWasHyphen = false
		} else if unicode.IsSpace(r) || r == '-' || r == '_' {
			if !lastWasHyphen && result.Len() > 0 {
				_, _ = result.WriteRune('-')
				lastWasHyphen = true
			}
		}
	}

	slug := result.String()

	slug = strings.Trim(slug, "-")

	return slug
}
