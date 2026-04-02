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

package i18n_adapters

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/i18n/i18n_schema"
	i18n_fb "piko.sh/piko/internal/i18n/i18n_schema/i18n_schema_gen"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// initialBuilderSize is the starting buffer size for the FlatBuffer builder.
	initialBuilderSize = 4096

	// uOffsetTSize is the size in bytes of a FlatBuffer UOffsetT value.
	uOffsetTSize = 4

	// schemaVersion is the version of the i18n FlatBuffer schema format.
	schemaVersion = "1.0.0"

	// fbDirPermission is the file permission mode for FlatBuffer output folders.
	fbDirPermission = 0o750

	// fbFilePermission is the file permission for FlatBuffer output files.
	fbFilePermission = 0o600
)

// FlatBufferEmitter writes translation data to a FlatBuffer binary file.
// All file operations are sandboxed to prevent path traversal attacks.
type FlatBufferEmitter struct {
	// sandbox handles safe file operations for writing output files.
	sandbox safedisk.Sandbox
}

// NewFlatBufferEmitter creates a new FlatBuffer emitter that works within the
// given sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides a safe place for file
// operations.
//
// Returns *FlatBufferEmitter which is set up to emit within the sandbox.
func NewFlatBufferEmitter(sandbox safedisk.Sandbox) *FlatBufferEmitter {
	return &FlatBufferEmitter{sandbox: sandbox}
}

// Emit serialises the translation store to a FlatBuffer file.
// The outputPath should be relative to the sandbox root.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes store (*i18n_domain.Store) which contains the translations to
// serialise.
// Takes defaultLocale (string) which specifies the fallback locale.
// Takes outputPath (string) which is the destination path relative to the
// sandbox root.
//
// Returns error when directory creation, file writing, or atomic rename fails.
func (e *FlatBufferEmitter) Emit(ctx context.Context, store *i18n_domain.Store, defaultLocale, outputPath string) error {
	builder := flatbuffers.NewBuilder(initialBuilderSize)
	rootOffset := packManifest(builder, store, defaultLocale)
	builder.Finish(rootOffset)
	payload := builder.FinishedBytes()

	bytes := make([]byte, fbs.PackedSize(len(payload)))
	i18n_schema.PackInto(bytes, payload)

	relPath := e.sandbox.RelPath(outputPath)

	directory := filepath.Dir(relPath)
	if err := e.sandbox.MkdirAll(directory, fbDirPermission); err != nil {
		return fmt.Errorf("failed to create directory for i18n manifest: %w", err)
	}

	tempPath := relPath + ".tmp"
	if err := e.sandbox.WriteFile(tempPath, bytes, fbFilePermission); err != nil {
		return fmt.Errorf("failed to write i18n FlatBuffer file: %w", err)
	}
	if err := e.sandbox.Rename(tempPath, relPath); err != nil {
		if removeErr := e.sandbox.Remove(tempPath); removeErr != nil {
			_, rl := logger_domain.From(ctx, log)
			rl.Warn("Failed to remove temp file after rename failure",
				logger_domain.String("path", tempPath),
				logger_domain.Error(removeErr))
		}
		return fmt.Errorf("failed to rename i18n FlatBuffer file: %w", err)
	}

	return nil
}

// EmitFromTranslations serialises raw translations to a FlatBuffer file.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes translations (i18n_domain.Translations) which contains the translation
// entries keyed by locale.
// Takes defaultLocale (string) which specifies the fallback locale.
// Takes outputPath (string) which is the file path for the output.
//
// Returns error when serialisation or file writing fails.
func (e *FlatBufferEmitter) EmitFromTranslations(ctx context.Context, translations i18n_domain.Translations, defaultLocale, outputPath string) error {
	store := i18n_domain.NewStore(defaultLocale)
	for locale, entries := range translations {
		store.AddTranslations(locale, entries)
	}
	return e.Emit(ctx, store, defaultLocale, outputPath)
}

// packManifest builds a FlatBuffers i18n manifest from the locale store.
//
// Takes b (*flatbuffers.Builder) which is the builder to write the manifest to.
// Takes store (*i18n_domain.Store) which contains the locale data to pack.
// Takes defaultLocale (string) which specifies the default locale identifier.
//
// Returns flatbuffers.UOffsetT which is the offset of the completed manifest.
func packManifest(b *flatbuffers.Builder, store *i18n_domain.Store, defaultLocale string) flatbuffers.UOffsetT {
	defaultLocaleOff := b.CreateString(defaultLocale)
	versionOff := b.CreateString(schemaVersion)
	localesOff := packLocales(b, store)

	i18n_fb.I18nManifestFBStart(b)
	i18n_fb.I18nManifestFBAddDefaultLocale(b, defaultLocaleOff)
	i18n_fb.I18nManifestFBAddVersion(b, versionOff)
	i18n_fb.I18nManifestFBAddLocales(b, localesOff)
	return i18n_fb.I18nManifestFBEnd(b)
}

// packLocales packs all locale data from the store into a FlatBuffer vector.
//
// When the store contains no locales, returns 0 without writing to the buffer.
//
// Takes b (*flatbuffers.Builder) which builds the FlatBuffer output.
// Takes store (*i18n_domain.Store) which provides the locale data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed locale vector.
func packLocales(b *flatbuffers.Builder, store *i18n_domain.Store) flatbuffers.UOffsetT {
	locales := store.Locales()
	if len(locales) == 0 {
		return 0
	}

	slices.Sort(locales)

	offsets := make([]flatbuffers.UOffsetT, len(locales))
	for i, locale := range locales {
		offsets[i] = packLocaleData(b, store, locale)
	}

	return createVector(b, offsets)
}

// packLocaleData builds a FlatBuffer LocaleDataFB for a single locale.
//
// Takes b (*flatbuffers.Builder) which accumulates the serialised data.
// Takes store (*i18n_domain.Store) which contains all translation entries.
// Takes locale (string) which identifies the locale to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the completed locale
// data within the builder.
func packLocaleData(b *flatbuffers.Builder, store *i18n_domain.Store, locale string) flatbuffers.UOffsetT {
	localeOff := b.CreateString(locale)
	entriesOff := packEntriesForLocale(b, store, locale)

	i18n_fb.LocaleDataFBStart(b)
	i18n_fb.LocaleDataFBAddLocale(b, localeOff)
	i18n_fb.LocaleDataFBAddEntries(b, entriesOff)
	return i18n_fb.LocaleDataFBEnd(b)
}

// packEntriesForLocale packs all translation entries for a locale into a
// FlatBuffer vector.
//
// Takes b (*flatbuffers.Builder) which builds the FlatBuffer output.
// Takes store (*i18n_domain.Store) which provides the translation entries.
// Takes locale (string) which specifies which locale to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed vector, or 0
// if the locale has no entries.
func packEntriesForLocale(b *flatbuffers.Builder, store *i18n_domain.Store, locale string) flatbuffers.UOffsetT {
	entries := store.GetEntriesForLocale(locale)
	if len(entries) == 0 {
		return 0
	}

	keys := slices.Sorted(maps.Keys(entries))

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, key := range keys {
		offsets[i] = packTranslationEntry(b, key, entries[key])
	}

	return createVector(b, offsets)
}

// packTranslationEntry serialises a translation entry into a FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes key (string) which is the translation key.
// Takes entry (*i18n_domain.Entry) which contains the translation data.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised entry.
func packTranslationEntry(b *flatbuffers.Builder, key string, entry *i18n_domain.Entry) flatbuffers.UOffsetT {
	keyOff := b.CreateString(key)
	templateOff := b.CreateString(entry.Template)
	partsOff := packTemplateParts(b, entry.Parts)
	pluralFormsOff := packStringSlice(b, entry.PluralForms)
	placeholderNamesOff := packStringSlice(b, i18n_domain.ExtractExpressions(entry.Template))

	i18n_fb.TranslationEntryFBStart(b)
	i18n_fb.TranslationEntryFBAddKey(b, keyOff)
	i18n_fb.TranslationEntryFBAddTemplate(b, templateOff)
	i18n_fb.TranslationEntryFBAddParts(b, partsOff)
	i18n_fb.TranslationEntryFBAddPluralForms(b, pluralFormsOff)
	i18n_fb.TranslationEntryFBAddHasPlurals(b, entry.HasPlurals)
	i18n_fb.TranslationEntryFBAddPlaceholderNames(b, placeholderNamesOff)
	return i18n_fb.TranslationEntryFBEnd(b)
}

// packTemplateParts serialises template parts into a FlatBuffers vector.
//
// Takes b (*flatbuffers.Builder) which is the builder to write parts into.
// Takes parts ([]i18n_domain.TemplatePart) which contains the template parts
// to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector, or
// zero if parts is empty.
func packTemplateParts(b *flatbuffers.Builder, parts []i18n_domain.TemplatePart) flatbuffers.UOffsetT {
	if len(parts) == 0 {
		return 0
	}

	offsets := make([]flatbuffers.UOffsetT, len(parts))
	for i := len(parts) - 1; i >= 0; i-- {
		part := &parts[i]

		var literalOff, expressionSourceOffset, linkedKeyOff flatbuffers.UOffsetT
		switch part.Kind {
		case i18n_domain.PartLiteral:
			literalOff = b.CreateString(part.Literal)
		case i18n_domain.PartExpression:
			expressionSourceOffset = b.CreateString(part.ExprSource)
		case i18n_domain.PartLinkedMessage:
			linkedKeyOff = b.CreateString(part.LinkedKey)
		}

		i18n_fb.TemplatePartFBStart(b)
		i18n_fb.TemplatePartFBAddKind(b, partKindToSchema(part.Kind))
		if literalOff != 0 {
			i18n_fb.TemplatePartFBAddLiteral(b, literalOff)
		}
		if expressionSourceOffset != 0 {
			i18n_fb.TemplatePartFBAddExpressionSource(b, expressionSourceOffset)
		}
		if linkedKeyOff != 0 {
			i18n_fb.TemplatePartFBAddLinkedKey(b, linkedKeyOff)
		}
		offsets[i] = i18n_fb.TemplatePartFBEnd(b)
	}

	return createVector(b, offsets)
}

// partKindToSchema converts a domain part kind to its schema equivalent.
//
// Takes kind (i18n_domain.PartKind) which is the domain part kind to convert.
//
// Returns i18n_fb.PartKind which is the matching schema part kind.
func partKindToSchema(kind i18n_domain.PartKind) i18n_fb.PartKind {
	switch kind {
	case i18n_domain.PartExpression:
		return i18n_fb.PartKindExpression
	case i18n_domain.PartLinkedMessage:
		return i18n_fb.PartKindLinkedMessage
	default:
		return i18n_fb.PartKindLiteral
	}
}

// packStringSlice packs a slice of strings into a FlatBuffer vector.
//
// Takes b (*flatbuffers.Builder) which builds the FlatBuffer output.
// Takes s ([]string) which contains the strings to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector,
// or 0 if the slice is empty.
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
// Takes offsets ([]flatbuffers.UOffsetT) which contains the element offsets to
// include in the vector.
//
// Returns flatbuffers.UOffsetT which is the offset of the finished vector.
func createVector(b *flatbuffers.Builder, offsets []flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	b.StartVector(uOffsetTSize, len(offsets), uOffsetTSize)
	for i := len(offsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(offsets[i])
	}
	return b.EndVector(len(offsets))
}
