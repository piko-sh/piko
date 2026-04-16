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

package generator_adapters

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/generator/generator_schema"
	gen_fb "piko.sh/piko/internal/generator/generator_schema/generator_schema_gen"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/safedisk"
)

// FlatBufferManifestProvider is a driven adapter that loads a project manifest
// from a binary FlatBuffers file on disk.
type FlatBufferManifestProvider struct {
	// sandbox provides read access to the manifest file within a safe file system.
	sandbox safedisk.Sandbox

	// factory creates sandboxes with validated paths. When set and sandbox is
	// nil, the factory is used before falling back to NewNoOpSandbox.
	factory safedisk.Factory

	// manifestFileName is the path to the manifest file within the sandbox.
	manifestFileName string
}

var _ generator_domain.ManifestProviderPort = (*FlatBufferManifestProvider)(nil)

// FlatBufferManifestProviderOption sets options for a
// FlatBufferManifestProvider when it is created.
type FlatBufferManifestProviderOption func(*FlatBufferManifestProvider)

// errManifestSchemaVersionMismatch indicates the manifest was serialised with a
// different schema version. This typically occurs when upgrading Piko and
// requires recompilation.
var errManifestSchemaVersionMismatch = fbs.ErrSchemaVersionMismatch

// NewFlatBufferManifestProvider creates a provider that reads from a manifest
// file at the given path.
//
// Takes manifestPath (string) which specifies the path to the manifest file.
// Takes opts (...FlatBufferManifestProviderOption) which provides optional
// configuration such as WithFlatBufferManifestSandbox for testing.
//
// Returns *FlatBufferManifestProvider which is ready for use. If the sandbox
// cannot be set up, the provider is returned with a nil sandbox.
func NewFlatBufferManifestProvider(manifestPath string, opts ...FlatBufferManifestProviderOption) *FlatBufferManifestProvider {
	p := &FlatBufferManifestProvider{
		sandbox:          nil,
		manifestFileName: filepath.Base(manifestPath),
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.sandbox == nil {
		sandbox, err := createManifestSandbox(manifestPath, p.factory, "FlatBuffer manifest provider")
		if err != nil {
			return p
		}
		p.sandbox = sandbox
	}

	return p
}

// Load reads the binary manifest file from disk, performs a zero-copy parse
// using FlatBuffers, and unpacks the data into the Manifest DTO.
//
// Returns *generator_dto.Manifest which contains the parsed manifest data.
// Returns error when the file cannot be read or parsed.
//
// Returns an error wrapping errManifestSchemaVersionMismatch if the manifest
// was
// compiled with a different schema version.
//
// SAFETY: The returned Manifest contains strings that reference the file data
// directly via mem.String. Go's GC keeps the data alive through these string
// references.
func (p *FlatBufferManifestProvider) Load(_ context.Context) (*generator_dto.Manifest, error) {
	if p.manifestFileName == "" {
		return nil, errors.New("FlatBuffers manifest provider requires a valid file path")
	}

	if p.sandbox == nil {
		return nil, errors.New("FlatBuffers manifest provider sandbox not available")
	}

	data, err := p.sandbox.ReadFile(p.manifestFileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("manifest file not found at %s: %w", p.manifestFileName, err)
		}
		return nil, fmt.Errorf("failed to read manifest file %s: %w", p.manifestFileName, err)
	}

	payload, err := generator_schema.Unpack(data)
	if err != nil {
		if errors.Is(err, fbs.ErrSchemaVersionMismatch) {
			return nil, fmt.Errorf("manifest schema version mismatch at %s (recompile required): %w", p.manifestFileName, errManifestSchemaVersionMismatch)
		}
		return nil, fmt.Errorf("failed to unpack versioned manifest at %s: %w", p.manifestFileName, err)
	}

	fbManifest := gen_fb.GetRootAsManifestFB(payload, 0)
	if fbManifest == nil {
		return nil, fmt.Errorf("failed to parse corrupt manifest file at %s", p.manifestFileName)
	}

	return unpackManifest(fbManifest), nil
}

// WithFlatBufferManifestFactory sets the sandbox factory for the FlatBuffer
// manifest provider. When no sandbox is injected, the factory is tried before
// falling back to NewNoOpSandbox.
//
// Takes factory (safedisk.Factory) which creates sandboxes with validated
// paths.
//
// Returns FlatBufferManifestProviderOption which configures the provider with
// the factory.
func WithFlatBufferManifestFactory(factory safedisk.Factory) FlatBufferManifestProviderOption {
	return func(p *FlatBufferManifestProvider) {
		p.factory = factory
	}
}

// WithFlatBufferManifestSandbox sets a custom sandbox for the FlatBuffer
// manifest provider. Inject a mock sandbox to test filesystem operations.
//
// If not provided, a real sandbox is created using safedisk.NewNoOpSandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access for reading
// the manifest file.
//
// Returns FlatBufferManifestProviderOption which configures the provider with
// the given sandbox.
func WithFlatBufferManifestSandbox(sandbox safedisk.Sandbox) FlatBufferManifestProviderOption {
	return func(p *FlatBufferManifestProvider) {
		p.sandbox = sandbox
	}
}

// unpackManifest converts a FlatBuffers manifest into a domain manifest.
//
// Takes fb (*gen_fb.ManifestFB) which is the FlatBuffers data to convert.
//
// Returns *generator_dto.Manifest which contains the unpacked pages, partials,
// and emails.
func unpackManifest(fb *gen_fb.ManifestFB) *generator_dto.Manifest {
	return &generator_dto.Manifest{
		Pages:                    unpackMap(fb.PagesLength(), fb.Pages, unpackPageEntryMapItem),
		Partials:                 unpackMap(fb.PartialsLength(), fb.Partials, unpackPartialEntryMapItem),
		Emails:                   unpackMap(fb.EmailsLength(), fb.Emails, unpackEmailEntryMapItem),
		Pdfs:                     unpackMap(fb.PdfsLength(), fb.Pdfs, unpackPdfEntryMapItem),
		ErrorPages:               unpackMap(fb.ErrorPagesLength(), fb.ErrorPages, unpackErrorPageEntryMapItem),
		CollectionFallbackRoutes: unpackSlice(fb.CollectionFallbackRoutesLength(), fb.CollectionFallbackRoutes, unpackCollectionFallbackRoute),
	}
}

// unpackCollectionFallbackRoute converts a FlatBuffer collection fallback
// route into a domain DTO.
//
// Takes fb (*gen_fb.CollectionFallbackRouteFB) which is the FlatBuffer data.
//
// Returns generator_dto.CollectionFallbackRoute which holds the route patterns.
func unpackCollectionFallbackRoute(fb *gen_fb.CollectionFallbackRouteFB) generator_dto.CollectionFallbackRoute {
	return generator_dto.CollectionFallbackRoute{
		RoutePatterns: unpackRoutePatterns(fb.RoutePatternsLength(), fb.RoutePatterns),
		I18nStrategy:  mem.String(fb.I18nStrategy()),
	}
}

// unpackPageEntryMapItem extracts a key-value pair from a FlatBuffer page
// entry map item.
//
// Takes fb (*gen_fb.PageEntryMapItemFB) which is the FlatBuffer map item to
// unpack.
//
// Returns string which is the page entry key.
// Returns generator_dto.ManifestPageEntry which is the unpacked page entry.
func unpackPageEntryMapItem(fb *gen_fb.PageEntryMapItemFB) (string, generator_dto.ManifestPageEntry) {
	var entry gen_fb.ManifestPageEntryFB
	return mem.String(fb.Key()), unpackPageEntry(fb.Value(&entry))
}

// unpackPartialEntryMapItem extracts a key-value pair from a FlatBuffers map
// item.
//
// Takes fb (*gen_fb.PartialEntryMapItemFB) which is the FlatBuffers map item
// to read from.
//
// Returns string which is the map key.
// Returns generator_dto.ManifestPartialEntry which is the extracted entry.
func unpackPartialEntryMapItem(fb *gen_fb.PartialEntryMapItemFB) (string, generator_dto.ManifestPartialEntry) {
	var entry gen_fb.ManifestPartialEntryFB
	return mem.String(fb.Key()), unpackPartialEntry(fb.Value(&entry))
}

// unpackPageEntry converts a FlatBuffer page entry into a domain object.
//
// Takes fb (*gen_fb.ManifestPageEntryFB) which is the serialised page entry.
//
// Returns generator_dto.ManifestPageEntry which contains the unpacked page
// data.
func unpackPageEntry(fb *gen_fb.ManifestPageEntryFB) generator_dto.ManifestPageEntry {
	return generator_dto.ManifestPageEntry{
		PackagePath:              mem.String(fb.PackagePath()),
		OriginalSourcePath:       mem.String(fb.OriginalSourcePath()),
		RoutePatterns:            unpackRoutePatterns(fb.RoutePatternsLength(), fb.RoutePatterns),
		I18nStrategy:             mem.String(fb.I18nStrategy()),
		StyleBlock:               mem.String(fb.StyleBlock()),
		AssetRefs:                unpackSlice(fb.AssetRefsLength(), fb.AssetRefs, unpackAssetRef),
		CustomTags:               unpackStringSlice(fb.CustomTagsLength(), fb.CustomTags),
		JSArtefactIDs:            unpackStringSlice(fb.JavascriptArtefactIdsLength(), fb.JavascriptArtefactIds),
		HasCachePolicy:           fb.HasCachePolicy(),
		CachePolicyFuncName:      mem.String(fb.CachePolicyFuncName()),
		HasMiddleware:            fb.HasMiddleware(),
		MiddlewareFuncName:       mem.String(fb.MiddlewareFuncName()),
		HasSupportedLocales:      fb.HasSupportedLocales(),
		SupportedLocalesFuncName: mem.String(fb.SupportedLocalesFuncName()),
		LocalTranslations:        unpackLocaleTranslations(fb.LocalTranslationsLength(), fb.LocalTranslations),
		IsE2EOnly:                fb.IsE2eOnly(),
		HasPreview:               fb.HasPreview(),
		UsesCaptcha:              fb.UsesCaptcha(),
	}
}

// unpackPartialEntry converts a FlatBuffer partial entry to a domain object.
//
// Takes fb (*gen_fb.ManifestPartialEntryFB) which is the serialised entry to
// convert.
//
// Returns generator_dto.ManifestPartialEntry which holds the converted data.
func unpackPartialEntry(fb *gen_fb.ManifestPartialEntryFB) generator_dto.ManifestPartialEntry {
	return generator_dto.ManifestPartialEntry{
		PackagePath:        mem.String(fb.PackagePath()),
		OriginalSourcePath: mem.String(fb.OriginalSourcePath()),
		PartialName:        mem.String(fb.PartialName()),
		PartialSrc:         mem.String(fb.PartialSource()),
		RoutePattern:       mem.String(fb.RoutePattern()),
		StyleBlock:         mem.String(fb.StyleBlock()),
		JSArtefactID:       mem.String(fb.JavascriptArtefactId()),
		IsE2EOnly:          fb.IsE2eOnly(),
		HasPreview:         fb.HasPreview(),
	}
}

// unpackEmailEntryMapItem extracts a key-value pair from a FlatBuffer email
// entry map item.
//
// Takes fb (*gen_fb.EmailEntryMapItemFB) which is the FlatBuffer map item to
// unpack.
//
// Returns string which is the email entry key.
// Returns generator_dto.ManifestEmailEntry which is the unpacked email entry.
func unpackEmailEntryMapItem(fb *gen_fb.EmailEntryMapItemFB) (string, generator_dto.ManifestEmailEntry) {
	var entry gen_fb.ManifestEmailEntryFB
	return mem.String(fb.Key()), unpackEmailEntry(fb.Value(&entry))
}

// unpackEmailEntry converts a FlatBuffer email entry into a domain DTO.
//
// Takes fb (*gen_fb.ManifestEmailEntryFB) which is the FlatBuffer data to
// convert.
//
// Returns generator_dto.ManifestEmailEntry which holds the email template
// details.
func unpackEmailEntry(fb *gen_fb.ManifestEmailEntryFB) generator_dto.ManifestEmailEntry {
	return generator_dto.ManifestEmailEntry{
		PackagePath:         mem.String(fb.PackagePath()),
		OriginalSourcePath:  mem.String(fb.OriginalSourcePath()),
		StyleBlock:          mem.String(fb.StyleBlock()),
		HasSupportedLocales: fb.HasSupportedLocales(),
		LocalTranslations:   unpackLocaleTranslations(fb.LocalTranslationsLength(), fb.LocalTranslations),
		HasPreview:          fb.HasPreview(),
	}
}

// unpackPdfEntryMapItem extracts a key-value pair from a FlatBuffer PDF
// entry map item.
//
// Takes fb (*gen_fb.PdfEntryMapItemFB) which is the FlatBuffer map item to
// unpack.
//
// Returns string which is the PDF entry key.
// Returns generator_dto.ManifestPdfEntry which is the unpacked PDF entry.
func unpackPdfEntryMapItem(fb *gen_fb.PdfEntryMapItemFB) (string, generator_dto.ManifestPdfEntry) {
	var entry gen_fb.ManifestPdfEntryFB
	return mem.String(fb.Key()), unpackPdfEntry(fb.Value(&entry))
}

// unpackPdfEntry converts a FlatBuffer PDF entry into a domain DTO.
//
// Takes fb (*gen_fb.ManifestPdfEntryFB) which is the FlatBuffer data to
// convert.
//
// Returns generator_dto.ManifestPdfEntry which holds the PDF template
// details.
func unpackPdfEntry(fb *gen_fb.ManifestPdfEntryFB) generator_dto.ManifestPdfEntry {
	return generator_dto.ManifestPdfEntry{
		PackagePath:         mem.String(fb.PackagePath()),
		OriginalSourcePath:  mem.String(fb.OriginalSourcePath()),
		StyleBlock:          mem.String(fb.StyleBlock()),
		HasSupportedLocales: fb.HasSupportedLocales(),
		LocalTranslations:   unpackLocaleTranslations(fb.LocalTranslationsLength(), fb.LocalTranslations),
		HasPreview:          fb.HasPreview(),
	}
}

// unpackErrorPageEntryMapItem extracts a key-value pair from a FlatBuffer
// error page entry map item.
//
// Takes fb (*gen_fb.ErrorPageEntryMapItemFB) which is the FlatBuffer map item
// to unpack.
//
// Returns string which is the error page entry key.
// Returns generator_dto.ManifestErrorPageEntry which is the unpacked error
// page entry.
func unpackErrorPageEntryMapItem(fb *gen_fb.ErrorPageEntryMapItemFB) (string, generator_dto.ManifestErrorPageEntry) {
	var entry gen_fb.ManifestErrorPageEntryFB
	return mem.String(fb.Key()), unpackErrorPageEntry(fb.Value(&entry))
}

// unpackErrorPageEntry converts a FlatBuffer error page entry into a domain
// DTO.
//
// Takes fb (*gen_fb.ManifestErrorPageEntryFB) which is the FlatBuffer data to
// convert.
//
// Returns generator_dto.ManifestErrorPageEntry which holds the error page
// details.
func unpackErrorPageEntry(fb *gen_fb.ManifestErrorPageEntryFB) generator_dto.ManifestErrorPageEntry {
	return generator_dto.ManifestErrorPageEntry{
		PackagePath:        mem.String(fb.PackagePath()),
		OriginalSourcePath: mem.String(fb.OriginalSourcePath()),
		ScopePath:          mem.String(fb.ScopePath()),
		StyleBlock:         mem.String(fb.StyleBlock()),
		JSArtefactIDs:      unpackStringSlice(fb.JavascriptArtefactIdsLength(), fb.JavascriptArtefactIds),
		CustomTags:         unpackStringSlice(fb.CustomTagsLength(), fb.CustomTags),
		StatusCode:         int(fb.StatusCode()),
		StatusCodeMin:      int(fb.StatusCodeMin()),
		StatusCodeMax:      int(fb.StatusCodeMax()),
		IsCatchAll:         fb.IsCatchAll(),
		IsE2EOnly:          fb.IsE2eOnly(),
	}
}

// unpackAssetRef converts a FlatBuffer asset reference to a DTO.
//
// Takes fb (*gen_fb.AssetRefFB) which is the FlatBuffer asset reference.
//
// Returns templater_dto.AssetRef which holds the kind and path values.
func unpackAssetRef(fb *gen_fb.AssetRefFB) templater_dto.AssetRef {
	return templater_dto.AssetRef{
		Kind: mem.String(fb.Kind()),
		Path: mem.String(fb.Path()),
	}
}

// unpackRoutePatterns unpacks a vector of RoutePatternMapItemFB into a Go map.
//
// Takes length (int) which is the number of items in the vector.
// Takes getItem (func(...)) which gets each item by index from the vector.
//
// Returns map[string]string which maps locale codes to their route patterns,
// or nil when length is zero.
func unpackRoutePatterns(length int, getItem func(*gen_fb.RoutePatternMapItemFB, int) bool) map[string]string {
	if length == 0 {
		return nil
	}
	patterns := make(map[string]string, length)
	var item gen_fb.RoutePatternMapItemFB
	for i := range length {
		if getItem(&item, i) {
			locale := mem.String(item.Locale())
			pattern := mem.String(item.Pattern())
			patterns[locale] = pattern
		}
	}
	return patterns
}

// unpackLocaleTranslations converts a list of LocaleTranslationsFB items into
// a Go map.
//
// Takes length (int) which specifies the number of items in the list.
// Takes getItem (func(...)) which gets each LocaleTranslationsFB by index.
//
// Returns i18n_domain.Translations which maps locale codes to their
// translation key-value pairs, or nil if length is zero.
func unpackLocaleTranslations(length int, getItem func(*gen_fb.LocaleTranslationsFB, int) bool) i18n_domain.Translations {
	if length == 0 {
		return nil
	}
	translations := make(i18n_domain.Translations, length)
	var item gen_fb.LocaleTranslationsFB
	for i := range length {
		if getItem(&item, i) {
			locale := mem.String(item.Locale())
			translations[locale] = unpackTranslationKeyValueMap(item.TranslationsLength(), item.Translations)
		}
	}
	return translations
}

// unpackTranslationKeyValueMap converts a FlatBuffer translation vector into a
// Go map.
//
// Takes length (int) which specifies the number of items in the vector.
// Takes getItem (func(...)) which retrieves each item by index.
//
// Returns map[string]string which contains the key-value pairs, or nil if
// length is zero.
func unpackTranslationKeyValueMap(length int, getItem func(*gen_fb.TranslationKeyValueFB, int) bool) map[string]string {
	if length == 0 {
		return nil
	}
	kvMap := make(map[string]string, length)
	var item gen_fb.TranslationKeyValueFB
	for i := range length {
		if getItem(&item, i) {
			key := mem.String(item.Key())
			value := mem.String(item.Value())
			kvMap[key] = value
		}
	}
	return kvMap
}

// unpackMap builds a map by fetching items and extracting key-value pairs.
//
// Takes length (int) which specifies how many items to process.
// Takes getItem (func(*T, int) bool) which fetches an item at the given index
// into the pointer and returns true on success.
// Takes unpacker (func(*T) (K, V)) which extracts a key and value from an item.
//
// Returns map[K]V which contains the extracted pairs, or nil if length is zero.
func unpackMap[T any, K comparable, V any](length int, getItem func(*T, int) bool, unpacker func(*T) (K, V)) map[K]V {
	if length == 0 {
		return nil
	}
	m := make(map[K]V, length)
	var item T
	for i := range length {
		if getItem(&item, i) {
			k, v := unpacker(&item)
			m[k] = v
		}
	}
	return m
}

// unpackSlice converts a sequence of items into a slice of unpacked values.
//
// Takes length (int) which specifies the number of items to process.
// Takes getItem (func(...)) which retrieves an item by index into the pointer.
// Takes unpacker (func(...)) which converts an item to the desired output type.
//
// Returns []U which contains the unpacked values, or nil if length is zero.
func unpackSlice[T any, U any](length int, getItem func(*T, int) bool, unpacker func(*T) U) []U {
	if length == 0 {
		return nil
	}
	s := make([]U, length)
	var item T
	for i := range length {
		if getItem(&item, i) {
			s[i] = unpacker(&item)
		}
	}
	return s
}

// unpackStringSlice converts a list of byte slices into a slice of strings.
//
// Takes length (int) which specifies how many items to convert.
// Takes getItem (func(int) []byte) which returns each item by its index.
//
// Returns []string which contains the converted strings, or nil if length is
// zero.
func unpackStringSlice(length int, getItem func(int) []byte) []string {
	if length == 0 {
		return nil
	}
	s := make([]string, length)
	for i := range length {
		s[i] = mem.String(getItem(i))
	}
	return s
}
