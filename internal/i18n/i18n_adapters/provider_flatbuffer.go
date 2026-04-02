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
	"errors"
	"fmt"

	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/i18n/i18n_schema"
	i18n_fb "piko.sh/piko/internal/i18n/i18n_schema/i18n_schema_gen"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safedisk"
)

// errI18nSchemaVersionMismatch indicates the i18n manifest was serialised with
// a different schema version. This typically occurs when upgrading Piko and
// requires recompilation.
var errI18nSchemaVersionMismatch = fbs.ErrSchemaVersionMismatch

// flatBufferProvider loads translations from a FlatBuffer binary file using
// zero-allocation parsing via the mem package. All file operations are
// sandboxed for security.
type flatBufferProvider struct {
	// sandbox provides safe file system access for reading translation files.
	sandbox safedisk.Sandbox

	// filePath is the path to the FlatBuffer file, relative to the sandbox root.
	filePath string

	// data holds the raw file bytes. This must be kept alive as long as
	// the Store points to strings within it (via mem.String).
	data []byte
}

// load reads the FlatBuffer file and populates the store with zero-allocation
// parsing.
//
// Returns *i18n_domain.Store which contains the parsed translation data.
// Returns error when the file path is empty, the file cannot be read, or the
// FlatBuffer data is corrupt or has a schema version mismatch.
//
// SAFETY: The returned Store contains strings that reference the file data
// directly via mem.String. The provider keeps the data alive, and Go's GC
// keeps the data alive through these string references.
func (p *flatBufferProvider) load() (*i18n_domain.Store, error) {
	if p.filePath == "" {
		return nil, errors.New("i18n FlatBuffer provider requires a valid file path")
	}

	data, err := p.sandbox.ReadFile(p.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read i18n manifest file %s: %w", p.filePath, err)
	}

	payload, err := i18n_schema.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("i18n schema version mismatch at %s (recompile required): %w", p.filePath, errI18nSchemaVersionMismatch)
	}

	p.data = data

	fbManifest := i18n_fb.GetRootAsI18nManifestFB(payload, 0)
	if fbManifest == nil {
		return nil, fmt.Errorf("failed to parse corrupt i18n manifest file at %s", p.filePath)
	}

	return unpackManifest(fbManifest), nil
}

// rawData returns the raw FlatBuffer data, keeping it alive when the Store
// is passed around independently.
//
// Returns []byte which contains the underlying FlatBuffer bytes.
func (p *flatBufferProvider) rawData() []byte {
	return p.data
}

// newFlatBufferProvider creates a new provider for the given file path.
// The filePath should be relative to the sandbox root.
//
// Takes sandbox (safedisk.Sandbox) which gives access to the file system.
// Takes filePath (string) which sets the path relative to the sandbox root.
//
// Returns *flatBufferProvider which is ready for use after calling Load.
func newFlatBufferProvider(sandbox safedisk.Sandbox, filePath string) *flatBufferProvider {
	return &flatBufferProvider{
		sandbox:  sandbox,
		filePath: filePath,
		data:     nil,
	}
}

// unpackManifest converts a FlatBuffers manifest into a domain store.
//
// Takes fb (*i18n_fb.I18nManifestFB) which is the serialised manifest to
// unpack.
//
// Returns *i18n_domain.Store which holds all locale data from the manifest.
func unpackManifest(fb *i18n_fb.I18nManifestFB) *i18n_domain.Store {
	defaultLocale := mem.String(fb.DefaultLocale())
	store := i18n_domain.NewStore(defaultLocale)

	localesLen := fb.LocalesLength()
	var localeData i18n_fb.LocaleDataFB
	for i := range localesLen {
		if fb.Locales(&localeData, i) {
			unpackLocaleData(&localeData, store)
		}
	}

	return store
}

// unpackLocaleData extracts locale data from a FlatBuffer and adds it to the
// store.
//
// Takes fb (*i18n_fb.LocaleDataFB) which contains the serialised locale data.
// Takes store (*i18n_domain.Store) which receives the unpacked locale entries.
func unpackLocaleData(fb *i18n_fb.LocaleDataFB, store *i18n_domain.Store) {
	locale := mem.String(fb.Locale())
	entries := unpackEntries(fb)
	store.AddLocale(locale, entries)
}

// unpackEntries converts FlatBuffer translation entries to a domain map.
//
// Takes fb (*i18n_fb.LocaleDataFB) which contains the serialised locale data.
//
// Returns map[string]*i18n_domain.Entry which maps keys to their entries, or
// nil when there are no entries.
func unpackEntries(fb *i18n_fb.LocaleDataFB) map[string]*i18n_domain.Entry {
	entriesLen := fb.EntriesLength()
	if entriesLen == 0 {
		return nil
	}

	entries := make(map[string]*i18n_domain.Entry, entriesLen)
	var entryFB i18n_fb.TranslationEntryFB
	for i := range entriesLen {
		if fb.Entries(&entryFB, i) {
			key := mem.String(entryFB.Key())
			entries[key] = unpackEntry(&entryFB)
		}
	}

	return entries
}

// unpackEntry converts a FlatBuffer translation entry to a domain entry.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the serialised entry to
// unpack.
//
// Returns *i18n_domain.Entry which contains the unpacked template, parts,
// and plural forms.
func unpackEntry(fb *i18n_fb.TranslationEntryFB) *i18n_domain.Entry {
	entry := &i18n_domain.Entry{
		Template:    mem.String(fb.Template()),
		Parts:       unpackTemplateParts(fb),
		PluralForms: unpackPluralForms(fb),
		HasPlurals:  fb.HasPlurals(),
	}

	if entry.HasPlurals && len(entry.PluralForms) > 0 {
		entry.PluralFormsParts = make([][]i18n_domain.TemplatePart, len(entry.PluralForms))
		for i, form := range entry.PluralForms {
			entry.PluralFormsParts[i], _ = i18n_domain.ParseTemplate(form)
		}
		if len(entry.PluralFormsParts) > 0 {
			entry.Parts = entry.PluralFormsParts[0]
		}
	}

	return entry
}

// unpackTemplateParts extracts template parts from a FlatBuffer translation
// entry.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the FlatBuffer entry to
// extract parts from.
//
// Returns []i18n_domain.TemplatePart which contains the extracted template
// parts, or nil if the entry has no parts.
func unpackTemplateParts(fb *i18n_fb.TranslationEntryFB) []i18n_domain.TemplatePart {
	partsLen := fb.PartsLength()
	if partsLen == 0 {
		return nil
	}

	parts := make([]i18n_domain.TemplatePart, partsLen)
	var partFB i18n_fb.TemplatePartFB
	for i := range partsLen {
		if fb.Parts(&partFB, i) {
			parts[i] = i18n_domain.TemplatePart{
				Kind:       schemaToPartKind(partFB.Kind()),
				Literal:    mem.String(partFB.Literal()),
				ExprSource: mem.String(partFB.ExpressionSource()),
				LinkedKey:  mem.String(partFB.LinkedKey()),
			}
		}
	}

	return parts
}

// schemaToPartKind converts a schema PartKind to a domain PartKind.
//
// Takes kind (i18n_fb.PartKind) which is the schema part kind to convert.
//
// Returns i18n_domain.PartKind which is the matching domain part kind.
func schemaToPartKind(kind i18n_fb.PartKind) i18n_domain.PartKind {
	switch kind {
	case i18n_fb.PartKindExpression:
		return i18n_domain.PartExpression
	case i18n_fb.PartKindLinkedMessage:
		return i18n_domain.PartLinkedMessage
	default:
		return i18n_domain.PartLiteral
	}
}

// unpackPluralForms extracts plural forms from a FlatBuffer translation entry.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the FlatBuffer entry to
// extract from.
//
// Returns []string which holds the plural forms, or nil if none exist.
func unpackPluralForms(fb *i18n_fb.TranslationEntryFB) []string {
	formsLen := fb.PluralFormsLength()
	if formsLen == 0 {
		return nil
	}

	forms := make([]string, formsLen)
	for i := range formsLen {
		forms[i] = mem.String(fb.PluralForms(i))
	}

	return forms
}
