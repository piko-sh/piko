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

package manifest

import (
	"errors"

	gen_fb "piko.sh/piko/internal/generator/generator_schema/generator_schema_gen"
)

// errFlatBufferParseFailed is returned when a FlatBuffer payload cannot be
// decoded.
var errFlatBufferParseFailed = errors.New("failed to parse FlatBuffer payload")

// Manifest is a JSON-serialisable representation of a compiled Piko manifest.
type Manifest struct {
	// Pages maps source paths to page entries.
	Pages map[string]PageEntry `json:"pages"`

	// Partials maps source paths to partial entries.
	Partials map[string]PartialEntry `json:"partials"`

	// Emails maps source paths to email entries.
	Emails map[string]EmailEntry `json:"emails"`

	// Pdfs maps source paths to PDF entries.
	Pdfs map[string]PdfEntry `json:"pdfs"`
}

// PageEntry contains the compiled metadata for a single routable page.
type PageEntry struct {
	// RoutePatterns maps route identifiers to their URL patterns.
	RoutePatterns map[string]string `json:"route_patterns,omitempty"`

	// LocalTranslations holds page-specific translation pairs keyed by locale code.
	LocalTranslations map[string]map[string]string `json:"local_translations,omitempty"`

	// CachePolicyFuncName is the function name that determines the cache policy.
	CachePolicyFuncName string `json:"cache_policy_func_name,omitempty"`

	// I18nStrategy specifies how internationalisation is handled for this page.
	I18nStrategy string `json:"i18n_strategy,omitempty"`

	// StyleBlock contains the CSS style block for this page entry.
	StyleBlock string `json:"style_block,omitempty"`

	// PackagePath is the Go import path used to look up registered functions.
	PackagePath string `json:"package_path"`

	// MiddlewareFuncName is the name of the middleware function for this page.
	MiddlewareFuncName string `json:"middleware_func_name,omitempty"`

	// SupportedLocalesFuncName is the function name that returns supported locales.
	SupportedLocalesFuncName string `json:"supported_locales_func_name,omitempty"`

	// OriginalSourcePath is the path to the original source file for error reporting.
	OriginalSourcePath string `json:"original_source_path"`

	// CustomTags lists the allowed custom tag names for this page.
	CustomTags []string `json:"custom_tags,omitempty"`

	// JSArtefactIDs lists the IDs of JavaScript artefacts required by this page.
	JSArtefactIDs []string `json:"js_artefact_ids,omitempty"`

	// AssetRefs contains external asset references used by this page.
	AssetRefs []AssetRef `json:"asset_refs,omitempty"`

	// HasCachePolicy indicates whether the page has a cache policy defined.
	HasCachePolicy bool `json:"has_cache_policy,omitempty"`

	// HasMiddleware indicates whether the page uses middleware.
	HasMiddleware bool `json:"has_middleware,omitempty"`

	// HasSupportedLocales indicates whether the page has supported locales defined.
	HasSupportedLocales bool `json:"has_supported_locales,omitempty"`

	// HasPreview indicates whether the page defines a Preview function.
	HasPreview bool `json:"has_preview,omitempty"`
}

// PartialEntry contains the compiled metadata for a reusable partial.
type PartialEntry struct {
	// PackagePath is the import path of the package containing the entry.
	PackagePath string `json:"package_path"`

	// OriginalSourcePath is the file path where the entry was first found.
	OriginalSourcePath string `json:"original_source_path"`

	// PartialName is the matched portion of the symbol name.
	PartialName string `json:"partial_name"`

	// PartialSrc is the source code fragment being documented.
	PartialSrc string `json:"partial_src"`

	// RoutePattern is the URL pattern used to match this route.
	RoutePattern string `json:"route_pattern"`

	// StyleBlock is the style formatting for this partial entry.
	StyleBlock string `json:"style_block,omitempty"`

	// JSArtefactID is the identifier for the associated JavaScript artefact.
	JSArtefactID string `json:"js_artefact_id,omitempty"`

	// HasPreview indicates whether the partial defines a Preview function.
	HasPreview bool `json:"has_preview,omitempty"`
}

// EmailEntry contains the compiled metadata for an email template.
type EmailEntry struct {
	// LocalTranslations maps locale codes to key-value translation pairs.
	LocalTranslations map[string]map[string]string `json:"local_translations,omitempty"`

	// PackagePath is the full import path of the package containing this email.
	PackagePath string `json:"package_path"`

	// OriginalSourcePath is the file path where the email was originally found.
	OriginalSourcePath string `json:"original_source_path"`

	// StyleBlock contains custom CSS styles to apply to the email.
	StyleBlock string `json:"style_block,omitempty"`

	// HasSupportedLocales indicates whether the email has supported locale files.
	HasSupportedLocales bool `json:"has_supported_locales,omitempty"`

	// HasPreview indicates whether the email defines a Preview function.
	HasPreview bool `json:"has_preview,omitempty"`
}

// PdfEntry contains the compiled metadata for a PDF template.
type PdfEntry struct {
	// LocalTranslations maps locale codes to key-value translation pairs.
	LocalTranslations map[string]map[string]string `json:"local_translations,omitempty"`

	// PackagePath is the full import path of the package containing this PDF.
	PackagePath string `json:"package_path"`

	// OriginalSourcePath is the file path where the PDF was originally found.
	OriginalSourcePath string `json:"original_source_path"`

	// StyleBlock contains custom CSS styles to apply to the PDF.
	StyleBlock string `json:"style_block,omitempty"`

	// HasSupportedLocales indicates whether the PDF has supported locale files.
	HasSupportedLocales bool `json:"has_supported_locales,omitempty"`

	// HasPreview indicates whether the PDF defines a Preview function.
	HasPreview bool `json:"has_preview,omitempty"`
}

// AssetRef describes a static asset referenced by a page.
type AssetRef struct {
	// Kind specifies the asset type, such as "svg".
	Kind string `json:"kind"`

	// Path string `json:"path"` // Path is the location of the asset file.
	Path string `json:"path"`
}

// ConvertManifest parses a raw FlatBuffer manifest payload into a
// JSON-serialisable Manifest struct.
//
// Takes payload ([]byte) which is the raw FlatBuffer data after stripping the
// version header (use Unpack first).
//
// Returns *Manifest which contains all pages, partials, and emails.
// Returns error when the payload cannot be parsed.
func ConvertManifest(payload []byte) (*Manifest, error) {
	fb := gen_fb.GetRootAsManifestFB(payload, 0)
	if fb == nil {
		return nil, errFlatBufferParseFailed
	}

	return &Manifest{
		Pages:    convertPageMap(fb),
		Partials: convertPartialMap(fb),
		Emails:   convertEmailMap(fb),
		Pdfs:     convertPdfMap(fb),
	}, nil
}

// convertPageMap extracts all page entries from the FlatBuffer manifest.
//
// Takes fb (*gen_fb.ManifestFB) which is the FlatBuffer manifest to extract
// pages from.
//
// Returns map[string]PageEntry which maps page keys to their entries, or nil
// if the manifest contains no pages.
func convertPageMap(fb *gen_fb.ManifestFB) map[string]PageEntry {
	length := fb.PagesLength()
	if length == 0 {
		return nil
	}
	pages := make(map[string]PageEntry, length)
	var item gen_fb.PageEntryMapItemFB
	for i := range length {
		if fb.Pages(&item, i) {
			var entry gen_fb.ManifestPageEntryFB
			key := string(item.Key())
			pages[key] = convertPageEntry(item.Value(&entry))
		}
	}
	return pages
}

// convertPageEntry converts a single FlatBuffer page entry.
//
// Takes fb (*gen_fb.ManifestPageEntryFB) which is the FlatBuffer page entry to
// convert.
//
// Returns PageEntry which contains the converted page entry data.
func convertPageEntry(fb *gen_fb.ManifestPageEntryFB) PageEntry {
	return PageEntry{
		PackagePath:              string(fb.PackagePath()),
		OriginalSourcePath:       string(fb.OriginalSourcePath()),
		RoutePatterns:            convertRoutePatterns(fb),
		I18nStrategy:             string(fb.I18nStrategy()),
		StyleBlock:               string(fb.StyleBlock()),
		HasCachePolicy:           fb.HasCachePolicy(),
		CachePolicyFuncName:      string(fb.CachePolicyFuncName()),
		HasMiddleware:            fb.HasMiddleware(),
		MiddlewareFuncName:       string(fb.MiddlewareFuncName()),
		HasSupportedLocales:      fb.HasSupportedLocales(),
		SupportedLocalesFuncName: string(fb.SupportedLocalesFuncName()),
		CustomTags:               convertStringSlice(fb.CustomTagsLength(), fb.CustomTags),
		JSArtefactIDs:            convertStringSlice(fb.JavascriptArtefactIdsLength(), fb.JavascriptArtefactIds),
		AssetRefs:                convertAssetRefs(fb),
		LocalTranslations:        convertLocaleTranslations(fb.LocalTranslationsLength(), fb.LocalTranslations),
		HasPreview:               fb.HasPreview(),
	}
}

// convertRoutePatterns extracts locale-to-pattern mappings from the FlatBuffer.
//
// Takes fb (*gen_fb.ManifestPageEntryFB) which provides the page entry data.
//
// Returns map[string]string which maps locale codes to their route patterns,
// or nil when no patterns exist.
func convertRoutePatterns(fb *gen_fb.ManifestPageEntryFB) map[string]string {
	length := fb.RoutePatternsLength()
	if length == 0 {
		return nil
	}
	patterns := make(map[string]string, length)
	var item gen_fb.RoutePatternMapItemFB
	for i := range length {
		if fb.RoutePatterns(&item, i) {
			patterns[string(item.Locale())] = string(item.Pattern())
		}
	}
	return patterns
}

// convertAssetRefs extracts asset references from the FlatBuffer.
//
// Takes fb (*gen_fb.ManifestPageEntryFB) which is the FlatBuffer page entry
// to extract asset references from.
//
// Returns []AssetRef which contains the extracted asset references, or nil if
// the entry has no asset references.
func convertAssetRefs(fb *gen_fb.ManifestPageEntryFB) []AssetRef {
	length := fb.AssetRefsLength()
	if length == 0 {
		return nil
	}
	refs := make([]AssetRef, length)
	var item gen_fb.AssetRefFB
	for i := range length {
		if fb.AssetRefs(&item, i) {
			refs[i] = AssetRef{
				Kind: string(item.Kind()),
				Path: string(item.Path()),
			}
		}
	}
	return refs
}

// convertPartialMap extracts all partial entries from the FlatBuffer manifest.
//
// Takes fb (*gen_fb.ManifestFB) which is the FlatBuffer manifest to extract
// partials from.
//
// Returns map[string]PartialEntry which maps partial keys to their entries,
// or nil when the manifest contains no partials.
func convertPartialMap(fb *gen_fb.ManifestFB) map[string]PartialEntry {
	length := fb.PartialsLength()
	if length == 0 {
		return nil
	}
	partials := make(map[string]PartialEntry, length)
	var item gen_fb.PartialEntryMapItemFB
	for i := range length {
		if fb.Partials(&item, i) {
			var entry gen_fb.ManifestPartialEntryFB
			key := string(item.Key())
			partials[key] = convertPartialEntry(item.Value(&entry))
		}
	}
	return partials
}

// convertPartialEntry converts a single FlatBuffer partial entry.
//
// Takes fb (*gen_fb.ManifestPartialEntryFB) which is the FlatBuffer partial
// entry to convert.
//
// Returns PartialEntry which contains the converted partial entry data.
func convertPartialEntry(fb *gen_fb.ManifestPartialEntryFB) PartialEntry {
	return PartialEntry{
		PackagePath:        string(fb.PackagePath()),
		OriginalSourcePath: string(fb.OriginalSourcePath()),
		PartialName:        string(fb.PartialName()),
		PartialSrc:         string(fb.PartialSource()),
		RoutePattern:       string(fb.RoutePattern()),
		StyleBlock:         string(fb.StyleBlock()),
		JSArtefactID:       string(fb.JavascriptArtefactId()),
		HasPreview:         fb.HasPreview(),
	}
}

// convertEmailMap extracts all email entries from the FlatBuffer manifest.
//
// Takes fb (*gen_fb.ManifestFB) which provides the FlatBuffer manifest to read
// from.
//
// Returns map[string]EmailEntry which maps email keys to their entries, or nil
// if no emails exist.
func convertEmailMap(fb *gen_fb.ManifestFB) map[string]EmailEntry {
	length := fb.EmailsLength()
	if length == 0 {
		return nil
	}
	emails := make(map[string]EmailEntry, length)
	var item gen_fb.EmailEntryMapItemFB
	for i := range length {
		if fb.Emails(&item, i) {
			var entry gen_fb.ManifestEmailEntryFB
			key := string(item.Key())
			emails[key] = convertEmailEntry(item.Value(&entry))
		}
	}
	return emails
}

// convertEmailEntry converts a single FlatBuffer email entry.
//
// Takes fb (*gen_fb.ManifestEmailEntryFB) which is the FlatBuffer email entry
// to convert.
//
// Returns EmailEntry which contains the converted email entry data.
func convertEmailEntry(fb *gen_fb.ManifestEmailEntryFB) EmailEntry {
	return EmailEntry{
		PackagePath:         string(fb.PackagePath()),
		OriginalSourcePath:  string(fb.OriginalSourcePath()),
		StyleBlock:          string(fb.StyleBlock()),
		HasSupportedLocales: fb.HasSupportedLocales(),
		LocalTranslations:   convertLocaleTranslations(fb.LocalTranslationsLength(), fb.LocalTranslations),
		HasPreview:          fb.HasPreview(),
	}
}

// convertPdfMap extracts all PDF entries from the FlatBuffer manifest.
//
// Takes fb (*gen_fb.ManifestFB) which provides the FlatBuffer manifest to read
// from.
//
// Returns map[string]PdfEntry which maps PDF keys to their entries, or nil
// if no PDFs exist.
func convertPdfMap(fb *gen_fb.ManifestFB) map[string]PdfEntry {
	length := fb.PdfsLength()
	if length == 0 {
		return nil
	}
	pdfs := make(map[string]PdfEntry, length)
	var item gen_fb.PdfEntryMapItemFB
	for i := range length {
		if fb.Pdfs(&item, i) {
			var entry gen_fb.ManifestPdfEntryFB
			key := string(item.Key())
			pdfs[key] = convertPdfEntry(item.Value(&entry))
		}
	}
	return pdfs
}

// convertPdfEntry converts a single FlatBuffer PDF entry.
//
// Takes fb (*gen_fb.ManifestPdfEntryFB) which is the FlatBuffer PDF entry
// to convert.
//
// Returns PdfEntry which contains the converted PDF entry data.
func convertPdfEntry(fb *gen_fb.ManifestPdfEntryFB) PdfEntry {
	return PdfEntry{
		PackagePath:         string(fb.PackagePath()),
		OriginalSourcePath:  string(fb.OriginalSourcePath()),
		StyleBlock:          string(fb.StyleBlock()),
		HasSupportedLocales: fb.HasSupportedLocales(),
		LocalTranslations:   convertLocaleTranslations(fb.LocalTranslationsLength(), fb.LocalTranslations),
		HasPreview:          fb.HasPreview(),
	}
}

// convertLocaleTranslations extracts locale translation maps from the
// FlatBuffer.
//
// Takes length (int) which specifies the number of locale entries to process.
// Takes getItem (func(...)) which retrieves each locale translation item.
//
// Returns map[string]map[string]string which maps locale codes to their
// translation key-value pairs, or nil when length is zero.
func convertLocaleTranslations(length int, getItem func(*gen_fb.LocaleTranslationsFB, int) bool) map[string]map[string]string {
	if length == 0 {
		return nil
	}
	translations := make(map[string]map[string]string, length)
	var item gen_fb.LocaleTranslationsFB
	for i := range length {
		if getItem(&item, i) {
			locale := string(item.Locale())
			translations[locale] = convertTranslationKVMap(item.TranslationsLength(), item.Translations)
		}
	}
	return translations
}

// convertTranslationKVMap extracts a key-value translation map from the
// FlatBuffer.
//
// Takes length (int) which specifies the number of items to process.
// Takes getItem (func(...)) which retrieves each translation key-value pair.
//
// Returns map[string]string which contains the extracted translations, or nil
// when length is zero.
func convertTranslationKVMap(length int, getItem func(*gen_fb.TranslationKeyValueFB, int) bool) map[string]string {
	if length == 0 {
		return nil
	}
	kvMap := make(map[string]string, length)
	var item gen_fb.TranslationKeyValueFB
	for i := range length {
		if getItem(&item, i) {
			kvMap[string(item.Key())] = string(item.Value())
		}
	}
	return kvMap
}

// convertStringSlice extracts a string slice from a FlatBuffer vector.
//
// Takes length (int) which specifies the number of items in the vector.
// Takes getItem (func(int) []byte) which retrieves each item by index.
//
// Returns []string which contains the converted string values, or nil if
// length is zero.
func convertStringSlice(length int, getItem func(int) []byte) []string {
	if length == 0 {
		return nil
	}
	s := make([]string, length)
	for i := range length {
		s[i] = string(getItem(i))
	}
	return s
}
