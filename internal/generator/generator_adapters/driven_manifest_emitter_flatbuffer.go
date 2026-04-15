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
	"fmt"
	"maps"
	"slices"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/generator/generator_schema"
	gen_fb "piko.sh/piko/internal/generator/generator_schema/generator_schema_gen"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// initialBuilderSize is the starting buffer size for the FlatBuffer builder.
	initialBuilderSize = 1024

	// uOffsetTSize is the size in bytes of a FlatBuffer UOffsetT (unsigned 32-bit
	// offset).
	uOffsetTSize = 4
)

// FlatBufferManifestEmitter implements ManifestEmitterPort to write a binary
// FlatBuffers manifest. It uses atomic writes and sandboxes all file
// operations to prevent path traversal attacks.
type FlatBufferManifestEmitter struct {
	// sandbox restricts file operations to a safe directory.
	sandbox safedisk.Sandbox
}

var _ generator_domain.ManifestEmitterPort = (*FlatBufferManifestEmitter)(nil)

// NewFlatBufferManifestEmitter creates a manifest emitter that writes output
// in FlatBuffers format.
//
// Takes sandbox (safedisk.Sandbox) which provides the output folder.
//
// Returns *FlatBufferManifestEmitter which is ready to write manifests.
func NewFlatBufferManifestEmitter(sandbox safedisk.Sandbox) *FlatBufferManifestEmitter {
	return &FlatBufferManifestEmitter{sandbox: sandbox}
}

// EmitCode generates the final manifest.bin file by serialising the given
// Manifest to FlatBuffers format and writing it atomically. The output includes
// a 32-byte schema hash prefix for automatic cache invalidation.
//
// Takes manifest (*generator_dto.Manifest) which contains the data to
// serialise.
// Takes outputPath (string) which specifies where to write the output file.
//
// Returns error when the atomic write to the filesystem fails.
func (e *FlatBufferManifestEmitter) EmitCode(
	ctx context.Context,
	manifest *generator_dto.Manifest,
	outputPath string,
) error {
	builder := flatbuffers.NewBuilder(initialBuilderSize)
	rootOffset := packManifest(builder, manifest)
	builder.Finish(rootOffset)
	payload := builder.FinishedBytes()

	versionedBytes := make([]byte, fbs.PackedSize(len(payload)))
	generator_schema.PackInto(versionedBytes, payload)

	relPath := e.sandbox.RelPath(outputPath)

	if err := generator_domain.AtomicWriteFile(ctx, e.sandbox, relPath, versionedBytes, generator_domain.FilePermission); err != nil {
		return fmt.Errorf("failed to write FlatBuffers manifest file atomically: %w", err)
	}

	return nil
}

// packManifest serialises a manifest into a FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the builder to write into.
// Takes manifest (*generator_dto.Manifest) which holds the pages, partials,
// and emails to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed manifest.
func packManifest(b *flatbuffers.Builder, manifest *generator_dto.Manifest) flatbuffers.UOffsetT {
	pagesOffset := packMap(b, manifest.Pages, packPageEntryMapItem)
	partialsOffset := packMap(b, manifest.Partials, packPartialEntryMapItem)
	emailsOffset := packMap(b, manifest.Emails, packEmailEntryMapItem)
	pdfsOffset := packMap(b, manifest.Pdfs, packPdfEntryMapItem)
	errorPagesOffset := packMap(b, manifest.ErrorPages, packErrorPageEntryMapItem)
	fallbackRoutesOffset := packSlice(b, manifest.CollectionFallbackRoutes, packCollectionFallbackRoute)

	gen_fb.ManifestFBStart(b)
	gen_fb.ManifestFBAddPages(b, pagesOffset)
	gen_fb.ManifestFBAddPartials(b, partialsOffset)
	gen_fb.ManifestFBAddEmails(b, emailsOffset)
	gen_fb.ManifestFBAddPdfs(b, pdfsOffset)
	gen_fb.ManifestFBAddErrorPages(b, errorPagesOffset)
	gen_fb.ManifestFBAddCollectionFallbackRoutes(b, fallbackRoutesOffset)
	return gen_fb.ManifestFBEnd(b)
}

// packCollectionFallbackRoute writes a collection fallback route to a
// FlatBuffer table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes route (generator_dto.CollectionFallbackRoute) which holds the route
// data.
//
// Returns flatbuffers.UOffsetT which is the offset of the finished table.
func packCollectionFallbackRoute(b *flatbuffers.Builder, route generator_dto.CollectionFallbackRoute) flatbuffers.UOffsetT {
	routePatternsOff := packRoutePatterns(b, route.RoutePatterns)
	i18nStrategyOff := b.CreateString(route.I18nStrategy)

	gen_fb.CollectionFallbackRouteFBStart(b)
	gen_fb.CollectionFallbackRouteFBAddRoutePatterns(b, routePatternsOff)
	gen_fb.CollectionFallbackRouteFBAddI18nStrategy(b, i18nStrategyOff)
	return gen_fb.CollectionFallbackRouteFBEnd(b)
}

// packPageEntryMapItem packs a key-value pair into a FlatBuffers map item.
//
// Takes b (*flatbuffers.Builder) which builds the binary output.
// Takes key (string) which is the name for this page entry in the map.
// Takes value (generator_dto.ManifestPageEntry) which holds the page data.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed map item.
func packPageEntryMapItem(b *flatbuffers.Builder, key string, value generator_dto.ManifestPageEntry) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packPageEntry(b, &value)
	gen_fb.PageEntryMapItemFBStart(b)
	gen_fb.PageEntryMapItemFBAddKey(b, keyOffset)
	gen_fb.PageEntryMapItemFBAddValue(b, valueOffset)
	return gen_fb.PageEntryMapItemFBEnd(b)
}

// packPartialEntryMapItem writes a key-value pair to a FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which builds the binary output.
// Takes key (string) which is the map entry key.
// Takes value (generator_dto.ManifestPartialEntry) which holds the entry data.
//
// Returns flatbuffers.UOffsetT which is the offset of the written item.
func packPartialEntryMapItem(b *flatbuffers.Builder, key string, value generator_dto.ManifestPartialEntry) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packPartialEntry(b, &value)
	gen_fb.PartialEntryMapItemFBStart(b)
	gen_fb.PartialEntryMapItemFBAddKey(b, keyOffset)
	gen_fb.PartialEntryMapItemFBAddValue(b, valueOffset)
	return gen_fb.PartialEntryMapItemFBEnd(b)
}

// packPageEntry writes a manifest page entry to the FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the builder for the FlatBuffer.
// Takes entry (*generator_dto.ManifestPageEntry) which contains the page data.
//
// Returns flatbuffers.UOffsetT which is the offset of the written entry.
func packPageEntry(b *flatbuffers.Builder, entry *generator_dto.ManifestPageEntry) flatbuffers.UOffsetT {
	pkgPathOff := b.CreateString(entry.PackagePath)
	srcPathOff := b.CreateString(entry.OriginalSourcePath)
	i18nStrategyOff := b.CreateString(entry.I18nStrategy)
	styleBlockOff := b.CreateString(entry.StyleBlock)
	assetRefsOff := packSlice(b, entry.AssetRefs, packAssetRef)
	customTagsOff := packStringSlice(b, entry.CustomTags)
	jsArtefactIDsOff := packStringSlice(b, entry.JSArtefactIDs)
	routePatternsOff := packRoutePatterns(b, entry.RoutePatterns)
	cachePolicyFuncNameOff := b.CreateString(entry.CachePolicyFuncName)
	middlewareFuncNameOff := b.CreateString(entry.MiddlewareFuncName)
	supportedLocalesFuncNameOff := b.CreateString(entry.SupportedLocalesFuncName)
	localTranslationsOff := packLocaleTranslations(b, entry.LocalTranslations)

	gen_fb.ManifestPageEntryFBStart(b)
	gen_fb.ManifestPageEntryFBAddPackagePath(b, pkgPathOff)
	gen_fb.ManifestPageEntryFBAddOriginalSourcePath(b, srcPathOff)
	gen_fb.ManifestPageEntryFBAddRoutePatterns(b, routePatternsOff)
	gen_fb.ManifestPageEntryFBAddI18nStrategy(b, i18nStrategyOff)
	gen_fb.ManifestPageEntryFBAddStyleBlock(b, styleBlockOff)
	gen_fb.ManifestPageEntryFBAddAssetRefs(b, assetRefsOff)
	gen_fb.ManifestPageEntryFBAddCustomTags(b, customTagsOff)
	gen_fb.ManifestPageEntryFBAddJavascriptArtefactIds(b, jsArtefactIDsOff)
	gen_fb.ManifestPageEntryFBAddHasCachePolicy(b, entry.HasCachePolicy)
	gen_fb.ManifestPageEntryFBAddCachePolicyFuncName(b, cachePolicyFuncNameOff)
	gen_fb.ManifestPageEntryFBAddHasMiddleware(b, entry.HasMiddleware)
	gen_fb.ManifestPageEntryFBAddMiddlewareFuncName(b, middlewareFuncNameOff)
	gen_fb.ManifestPageEntryFBAddHasSupportedLocales(b, entry.HasSupportedLocales)
	gen_fb.ManifestPageEntryFBAddSupportedLocalesFuncName(b, supportedLocalesFuncNameOff)
	gen_fb.ManifestPageEntryFBAddLocalTranslations(b, localTranslationsOff)
	gen_fb.ManifestPageEntryFBAddIsE2eOnly(b, entry.IsE2EOnly)
	gen_fb.ManifestPageEntryFBAddHasPreview(b, entry.HasPreview)
	gen_fb.ManifestPageEntryFBAddUsesCaptcha(b, entry.UsesCaptcha)
	return gen_fb.ManifestPageEntryFBEnd(b)
}

// packPartialEntry writes a manifest partial entry to FlatBuffers format.
//
// Takes b (*flatbuffers.Builder) which is the builder to write the entry to.
// Takes entry (*generator_dto.ManifestPartialEntry) which holds the partial
// entry data to write.
//
// Returns flatbuffers.UOffsetT which is the offset of the written entry.
func packPartialEntry(b *flatbuffers.Builder, entry *generator_dto.ManifestPartialEntry) flatbuffers.UOffsetT {
	pkgPathOff := b.CreateString(entry.PackagePath)
	srcPathOff := b.CreateString(entry.OriginalSourcePath)
	partialNameOff := b.CreateString(entry.PartialName)
	partialSrcOff := b.CreateString(entry.PartialSrc)
	routePatternOff := b.CreateString(entry.RoutePattern)
	styleBlockOff := b.CreateString(entry.StyleBlock)
	jsArtefactIDOff := b.CreateString(entry.JSArtefactID)

	gen_fb.ManifestPartialEntryFBStart(b)

	gen_fb.ManifestPartialEntryFBAddPackagePath(b, pkgPathOff)
	gen_fb.ManifestPartialEntryFBAddOriginalSourcePath(b, srcPathOff)
	gen_fb.ManifestPartialEntryFBAddPartialName(b, partialNameOff)
	gen_fb.ManifestPartialEntryFBAddPartialSource(b, partialSrcOff)
	gen_fb.ManifestPartialEntryFBAddRoutePattern(b, routePatternOff)
	gen_fb.ManifestPartialEntryFBAddStyleBlock(b, styleBlockOff)
	gen_fb.ManifestPartialEntryFBAddJavascriptArtefactId(b, jsArtefactIDOff)
	gen_fb.ManifestPartialEntryFBAddIsE2eOnly(b, entry.IsE2EOnly)
	gen_fb.ManifestPartialEntryFBAddHasPreview(b, entry.HasPreview)

	return gen_fb.ManifestPartialEntryFBEnd(b)
}

// packEmailEntryMapItem serialises a key-value pair into a FlatBuffer email
// entry map item.
//
// Takes b (*flatbuffers.Builder) which is the builder to write the item to.
// Takes key (string) which is the map key for this entry.
// Takes value (generator_dto.ManifestEmailEntry) which is the email entry data.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised item.
func packEmailEntryMapItem(b *flatbuffers.Builder, key string, value generator_dto.ManifestEmailEntry) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packEmailEntry(b, &value)
	gen_fb.EmailEntryMapItemFBStart(b)
	gen_fb.EmailEntryMapItemFBAddKey(b, keyOffset)
	gen_fb.EmailEntryMapItemFBAddValue(b, valueOffset)
	return gen_fb.EmailEntryMapItemFBEnd(b)
}

// localisableEntryFields holds pre-created string offsets shared by email and
// PDF FlatBuffer entries, both of which have identical field layouts.
type localisableEntryFields struct {
	// pkgPathOff holds the FlatBuffer offset for the package path string.
	pkgPathOff flatbuffers.UOffsetT

	// srcPathOff holds the FlatBuffer offset for the source path string.
	srcPathOff flatbuffers.UOffsetT

	// styleBlockOff holds the FlatBuffer offset for the style block string.
	styleBlockOff flatbuffers.UOffsetT

	// localTranslationsOff holds the FlatBuffer offset for the locale translations vector.
	localTranslationsOff flatbuffers.UOffsetT

	// hasSupportedLocales indicates whether this entry defines a supported locales function.
	hasSupportedLocales bool
}

// prepareLocalisableEntry creates the string offsets common to email and PDF
// entries.
//
// Takes b (*flatbuffers.Builder) which is the builder to write strings into.
// Takes pkgPath (string) which is the Go package path.
// Takes srcPath (string) which is the source file path.
// Takes styleBlock (string) which is the scoped CSS content.
// Takes hasSupportedLocales (bool) which indicates whether a supported locales
// function exists.
// Takes translations (i18n_domain.Translations) which holds locale translation
// key-value pairs.
//
// Returns localisableEntryFields which contains the pre-created offsets.
func prepareLocalisableEntry(
	b *flatbuffers.Builder,
	pkgPath, srcPath, styleBlock string,
	hasSupportedLocales bool,
	translations i18n_domain.Translations,
) localisableEntryFields {
	return localisableEntryFields{
		pkgPathOff:           b.CreateString(pkgPath),
		srcPathOff:           b.CreateString(srcPath),
		styleBlockOff:        b.CreateString(styleBlock),
		localTranslationsOff: packLocaleTranslations(b, translations),
		hasSupportedLocales:  hasSupportedLocales,
	}
}

// localisableEntryPacker holds the FlatBuffer Start/Add/End callbacks for a
// single table type, allowing email and PDF entries (which share an identical
// field layout) to be packed by the same generic helper.
type localisableEntryPacker struct {
	// start begins a new FlatBuffer table for this entry type.
	start func(*flatbuffers.Builder)

	// addPackagePath writes the Go package path field to the table.
	addPackagePath func(*flatbuffers.Builder, flatbuffers.UOffsetT)

	// addSourcePath writes the original source file path field to the table.
	addSourcePath func(*flatbuffers.Builder, flatbuffers.UOffsetT)

	// addStyleBlock writes the scoped CSS content field to the table.
	addStyleBlock func(*flatbuffers.Builder, flatbuffers.UOffsetT)

	// addHasSupportedLocales writes the supported-locales flag to the table.
	addHasSupportedLocales func(*flatbuffers.Builder, bool)

	// addLocalTranslations writes the locale translations vector to the table.
	addLocalTranslations func(*flatbuffers.Builder, flatbuffers.UOffsetT)

	// addHasPreview writes the preview-available flag to the table.
	addHasPreview func(*flatbuffers.Builder, bool)

	// end finishes the FlatBuffer table and returns its offset.
	end func(*flatbuffers.Builder) flatbuffers.UOffsetT
}

// emailEntryPacker holds the FlatBuffer callbacks for email entry tables.
var emailEntryPacker = localisableEntryPacker{
	start:                  gen_fb.ManifestEmailEntryFBStart,
	addPackagePath:         gen_fb.ManifestEmailEntryFBAddPackagePath,
	addSourcePath:          gen_fb.ManifestEmailEntryFBAddOriginalSourcePath,
	addStyleBlock:          gen_fb.ManifestEmailEntryFBAddStyleBlock,
	addHasSupportedLocales: gen_fb.ManifestEmailEntryFBAddHasSupportedLocales,
	addLocalTranslations:   gen_fb.ManifestEmailEntryFBAddLocalTranslations,
	addHasPreview:          gen_fb.ManifestEmailEntryFBAddHasPreview,
	end:                    gen_fb.ManifestEmailEntryFBEnd,
}

// pdfEntryPacker holds the FlatBuffer callbacks for PDF entry tables.
var pdfEntryPacker = localisableEntryPacker{
	start:                  gen_fb.ManifestPdfEntryFBStart,
	addPackagePath:         gen_fb.ManifestPdfEntryFBAddPackagePath,
	addSourcePath:          gen_fb.ManifestPdfEntryFBAddOriginalSourcePath,
	addStyleBlock:          gen_fb.ManifestPdfEntryFBAddStyleBlock,
	addHasSupportedLocales: gen_fb.ManifestPdfEntryFBAddHasSupportedLocales,
	addLocalTranslations:   gen_fb.ManifestPdfEntryFBAddLocalTranslations,
	addHasPreview:          gen_fb.ManifestPdfEntryFBAddHasPreview,
	end:                    gen_fb.ManifestPdfEntryFBEnd,
}

// packLocalisableEntry serialises an email or PDF entry into a FlatBuffer
// table using the callbacks in p to target the correct generated type.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes f (localisableEntryFields) which holds the pre-created offsets.
// Takes hasPreview (bool) which indicates whether a preview is available.
// Takes p (localisableEntryPacker) which provides the type-specific FlatBuffer
// callbacks.
//
// Returns flatbuffers.UOffsetT which is the offset of the finished table.
func packLocalisableEntry(
	b *flatbuffers.Builder,
	f localisableEntryFields,
	hasPreview bool,
	p localisableEntryPacker,
) flatbuffers.UOffsetT {
	p.start(b)
	p.addPackagePath(b, f.pkgPathOff)
	p.addSourcePath(b, f.srcPathOff)
	p.addStyleBlock(b, f.styleBlockOff)
	p.addHasSupportedLocales(b, f.hasSupportedLocales)
	p.addLocalTranslations(b, f.localTranslationsOff)
	p.addHasPreview(b, hasPreview)
	return p.end(b)
}

// packEmailEntry writes an email entry to a FlatBuffer table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes entry (*generator_dto.ManifestEmailEntry) which holds the email data
// to store.
//
// Returns flatbuffers.UOffsetT which is the offset of the finished table.
func packEmailEntry(b *flatbuffers.Builder, entry *generator_dto.ManifestEmailEntry) flatbuffers.UOffsetT {
	f := prepareLocalisableEntry(b, entry.PackagePath, entry.OriginalSourcePath, entry.StyleBlock, entry.HasSupportedLocales, entry.LocalTranslations)
	return packLocalisableEntry(b, f, entry.HasPreview, emailEntryPacker)
}

// packPdfEntryMapItem serialises a key-value pair into a FlatBuffer PDF
// entry map item.
//
// Takes b (*flatbuffers.Builder) which is the builder to write the item to.
// Takes key (string) which is the map key for this entry.
// Takes value (generator_dto.ManifestPdfEntry) which is the PDF entry data.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised item.
func packPdfEntryMapItem(b *flatbuffers.Builder, key string, value generator_dto.ManifestPdfEntry) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packPdfEntry(b, &value)
	gen_fb.PdfEntryMapItemFBStart(b)
	gen_fb.PdfEntryMapItemFBAddKey(b, keyOffset)
	gen_fb.PdfEntryMapItemFBAddValue(b, valueOffset)
	return gen_fb.PdfEntryMapItemFBEnd(b)
}

// packPdfEntry writes a PDF entry to a FlatBuffer table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes entry (*generator_dto.ManifestPdfEntry) which holds the PDF data
// to store.
//
// Returns flatbuffers.UOffsetT which is the offset of the finished table.
func packPdfEntry(b *flatbuffers.Builder, entry *generator_dto.ManifestPdfEntry) flatbuffers.UOffsetT {
	f := prepareLocalisableEntry(b, entry.PackagePath, entry.OriginalSourcePath, entry.StyleBlock, entry.HasSupportedLocales, entry.LocalTranslations)
	return packLocalisableEntry(b, f, entry.HasPreview, pdfEntryPacker)
}

// packErrorPageEntryMapItem serialises a key-value pair into a FlatBuffer
// error page entry map item.
//
// Takes b (*flatbuffers.Builder) which is the builder to write the item to.
// Takes key (string) which is the map key for this entry.
// Takes value (generator_dto.ManifestErrorPageEntry) which is the error page
// entry data.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised item.
func packErrorPageEntryMapItem(b *flatbuffers.Builder, key string, value generator_dto.ManifestErrorPageEntry) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packErrorPageEntry(b, &value)
	gen_fb.ErrorPageEntryMapItemFBStart(b)
	gen_fb.ErrorPageEntryMapItemFBAddKey(b, keyOffset)
	gen_fb.ErrorPageEntryMapItemFBAddValue(b, valueOffset)
	return gen_fb.ErrorPageEntryMapItemFBEnd(b)
}

// packErrorPageEntry writes an error page entry to a FlatBuffer table.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes entry (*generator_dto.ManifestErrorPageEntry) which holds the error
// page data to store.
//
// Returns flatbuffers.UOffsetT which is the offset of the finished table.
func packErrorPageEntry(b *flatbuffers.Builder, entry *generator_dto.ManifestErrorPageEntry) flatbuffers.UOffsetT {
	pkgPathOff := b.CreateString(entry.PackagePath)
	srcPathOff := b.CreateString(entry.OriginalSourcePath)
	scopePathOff := b.CreateString(entry.ScopePath)
	styleBlockOff := b.CreateString(entry.StyleBlock)
	jsArtefactIDsOff := packStringSlice(b, entry.JSArtefactIDs)
	customTagsOff := packStringSlice(b, entry.CustomTags)

	gen_fb.ManifestErrorPageEntryFBStart(b)

	gen_fb.ManifestErrorPageEntryFBAddPackagePath(b, pkgPathOff)
	gen_fb.ManifestErrorPageEntryFBAddOriginalSourcePath(b, srcPathOff)
	gen_fb.ManifestErrorPageEntryFBAddScopePath(b, scopePathOff)
	gen_fb.ManifestErrorPageEntryFBAddStyleBlock(b, styleBlockOff)
	gen_fb.ManifestErrorPageEntryFBAddJavascriptArtefactIds(b, jsArtefactIDsOff)
	gen_fb.ManifestErrorPageEntryFBAddCustomTags(b, customTagsOff)
	gen_fb.ManifestErrorPageEntryFBAddStatusCode(b, safeconv.IntToInt32(entry.StatusCode))
	gen_fb.ManifestErrorPageEntryFBAddStatusCodeMin(b, safeconv.IntToInt32(entry.StatusCodeMin))
	gen_fb.ManifestErrorPageEntryFBAddStatusCodeMax(b, safeconv.IntToInt32(entry.StatusCodeMax))
	gen_fb.ManifestErrorPageEntryFBAddIsCatchAll(b, entry.IsCatchAll)
	gen_fb.ManifestErrorPageEntryFBAddIsE2eOnly(b, entry.IsE2EOnly)

	return gen_fb.ManifestErrorPageEntryFBEnd(b)
}

// packAssetRef packs an asset reference into a FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes ref (templater_dto.AssetRef) which contains the kind and path to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed asset
// reference.
func packAssetRef(b *flatbuffers.Builder, ref templater_dto.AssetRef) flatbuffers.UOffsetT {
	kindOff := b.CreateString(ref.Kind)
	pathOff := b.CreateString(ref.Path)
	gen_fb.AssetRefFBStart(b)
	gen_fb.AssetRefFBAddKind(b, kindOff)
	gen_fb.AssetRefFBAddPath(b, pathOff)
	return gen_fb.AssetRefFBEnd(b)
}

// packRoutePatterns packs a map of locale codes to route patterns into a
// FlatBuffers vector of RoutePatternMapItemFB entries.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes patterns (map[string]string) which maps locale codes to their route
// patterns.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed vector, or 0
// if patterns is empty.
func packRoutePatterns(b *flatbuffers.Builder, patterns map[string]string) flatbuffers.UOffsetT {
	if len(patterns) == 0 {
		return 0
	}

	locales := slices.Sorted(maps.Keys(patterns))

	offsets := make([]flatbuffers.UOffsetT, len(locales))
	for i, locale := range locales {
		pattern := patterns[locale]
		localeOff := b.CreateString(locale)
		patternOff := b.CreateString(pattern)

		gen_fb.RoutePatternMapItemFBStart(b)
		gen_fb.RoutePatternMapItemFBAddLocale(b, localeOff)
		gen_fb.RoutePatternMapItemFBAddPattern(b, patternOff)
		offsets[i] = gen_fb.RoutePatternMapItemFBEnd(b)
	}

	gen_fb.ManifestPageEntryFBStartRoutePatternsVector(b, len(offsets))
	for i := len(offsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(offsets[i])
	}
	return b.EndVector(len(offsets))
}

// packLocaleTranslations writes locale translations into a FlatBuffers vector.
//
// Locales are sorted in alphabetical order to ensure the same output each time.
//
// When translations is empty, returns 0.
//
// Takes b (*flatbuffers.Builder) which is the buffer to write to.
// Takes translations (i18n_domain.Translations) which maps locales to their
// translation key-value pairs.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed vector.
func packLocaleTranslations(b *flatbuffers.Builder, translations i18n_domain.Translations) flatbuffers.UOffsetT {
	if len(translations) == 0 {
		return 0
	}

	locales := slices.Sorted(maps.Keys(translations))

	offsets := make([]flatbuffers.UOffsetT, len(locales))
	for i, locale := range locales {
		localeOffset := b.CreateString(locale)
		translationsOffset := packTranslationKeyValueMap(b, translations[locale])

		gen_fb.LocaleTranslationsFBStart(b)
		gen_fb.LocaleTranslationsFBAddLocale(b, localeOffset)
		gen_fb.LocaleTranslationsFBAddTranslations(b, translationsOffset)
		offsets[i] = gen_fb.LocaleTranslationsFBEnd(b)
	}

	return createVector(b, offsets)
}

// packTranslationKeyValueMap converts a map of translation key-value pairs
// into a FlatBuffers vector of TranslationKeyValueFB entries.
//
// Takes b (*flatbuffers.Builder) which is the builder used to write data.
// Takes kvMap (map[string]string) which holds the translation pairs to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector, or
// zero if the map is empty.
func packTranslationKeyValueMap(b *flatbuffers.Builder, kvMap map[string]string) flatbuffers.UOffsetT {
	if len(kvMap) == 0 {
		return 0
	}

	keys := slices.Sorted(maps.Keys(kvMap))

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, key := range keys {
		keyOffset := b.CreateString(key)
		valueOffset := b.CreateString(kvMap[key])

		gen_fb.TranslationKeyValueFBStart(b)
		gen_fb.TranslationKeyValueFBAddKey(b, keyOffset)
		gen_fb.TranslationKeyValueFBAddValue(b, valueOffset)
		offsets[i] = gen_fb.TranslationKeyValueFBEnd(b)
	}

	return createVector(b, offsets)
}

// packMap converts a map to a FlatBuffers vector with consistent ordering.
//
// When the map is empty, returns 0 without writing anything.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes m (map[K]V) which is the map to convert.
// Takes packer (func(...)) which converts each key-value pair to an offset.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector.
func packMap[K comparable, V any](b *flatbuffers.Builder, m map[K]V, packer func(*flatbuffers.Builder, K, V) flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if len(m) == 0 {
		return 0
	}
	keys := make([]string, 0, len(m))
	keyMap := make(map[string]K, len(m))
	for k := range m {
		kString := fmt.Sprintf("%v", k)
		keys = append(keys, kString)
		keyMap[kString] = k
	}
	slices.Sort(keys)

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, kString := range keys {
		originalKey := keyMap[kString]
		offsets[i] = packer(b, originalKey, m[originalKey])
	}
	return createVector(b, offsets)
}

// packSlice packs a slice of items into a FlatBuffers vector.
//
// Takes b (*flatbuffers.Builder) which builds the buffer.
// Takes s ([]T) which holds the items to pack.
// Takes packer (func(...)) which converts each item into a FlatBuffers offset.
//
// Returns flatbuffers.UOffsetT which is the offset of the new vector, or 0 if
// the slice is empty.
func packSlice[T any](b *flatbuffers.Builder, s []T, packer func(*flatbuffers.Builder, T) flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if len(s) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(s))
	for i := len(s) - 1; i >= 0; i-- {
		offsets[i] = packer(b, s[i])
	}
	return createVector(b, offsets)
}

// packStringSlice converts a slice of strings into a FlatBuffer vector.
//
// When the slice is empty, returns 0 without creating any buffer entries.
//
// Takes b (*flatbuffers.Builder) which is the buffer to write strings into.
// Takes s ([]string) which contains the strings to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector.
func packStringSlice(b *flatbuffers.Builder, s []string) flatbuffers.UOffsetT {
	if len(s) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(s))
	for i := len(s) - 1; i >= 0; i-- {
		offsets[i] = b.CreateString(s[i])
	}
	return createVector(b, offsets)
}

// createVector builds a FlatBuffers vector from a slice of offsets.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes offsets ([]flatbuffers.UOffsetT) which are the element offsets to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the finished vector.
func createVector(b *flatbuffers.Builder, offsets []flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	b.StartVector(uOffsetTSize, len(offsets), uOffsetTSize)
	for i := len(offsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(offsets[i])
	}
	return b.EndVector(len(offsets))
}
