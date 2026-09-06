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
	"errors"
	"fmt"
	"slices"

	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/i18n/i18n_schema"
	i18n_fb "piko.sh/piko/internal/i18n/i18n_schema/i18n_schema_gen"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safedisk"
)

var (
	// errI18nSchemaVersionMismatch indicates the i18n manifest was serialised with a
	// different schema version. This typically occurs when upgrading Piko and requires
	// recompilation.
	errI18nSchemaVersionMismatch = fbs.ErrSchemaVersionMismatch
)

// flatBufferProvider loads translations from a FlatBuffer binary file using
// zero-allocation parsing via the mem package. All file operations are sandboxed for
// security.
type flatBufferProvider struct {
	// sandbox provides safe file system access for reading translation files.
	sandbox safedisk.Sandbox

	// filePath is the path to the FlatBuffer file, relative to the sandbox root.
	filePath string

	// data holds the raw file bytes. This must be kept alive as long as the Store points to
	// strings within it (via mem.String).
	data []byte
}

// load reads the FlatBuffer file and populates the store with zero-allocation parsing.
// Plural forms whose templates fail to parse render as their literal text, the entry
// keeps its stored parts, and the failures are reported once as a warning through the
// logger carried by ctx.
//
// Returns *i18n_domain.Store which contains the parsed translation data.
// Returns error when the file path is empty, the file cannot be read, or the FlatBuffer
// data is corrupt or has a schema version mismatch.
//
// SAFETY: The returned Store contains strings that reference the file data directly via
// mem.String. The provider keeps the data alive, and Go's GC keeps the data alive through
// these string references.
func (p *flatBufferProvider) load(ctx context.Context) (*i18n_domain.Store, error) {
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

	store, problems := unpackManifest(fbManifest)
	i18n_domain.ReportTemplateProblems(ctx, p.filePath, problems)
	return store, nil
}

// rawData returns the raw FlatBuffer data, keeping it alive when the Store is passed
// around independently.
//
// Returns []byte which contains the underlying FlatBuffer bytes.
func (p *flatBufferProvider) rawData() []byte {
	return p.data
}

// newFlatBufferProvider creates a new provider for the given file path. The filePath
// should be relative to the sandbox root.
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
// Takes fb (*i18n_fb.I18nManifestFB) which is the serialised manifest to unpack.
//
// Returns *i18n_domain.Store which holds all locale data from the manifest.
// Returns []i18n_domain.TemplateProblem which lists the plural forms that failed to
// parse.
func unpackManifest(fb *i18n_fb.I18nManifestFB) (*i18n_domain.Store, []i18n_domain.TemplateProblem) {
	defaultLocale := mem.String(fb.DefaultLocale())
	store := i18n_domain.NewStore(defaultLocale)

	var problems []i18n_domain.TemplateProblem
	localesLen := fb.LocalesLength()
	var localeData i18n_fb.LocaleDataFB
	for i := range localesLen {
		if fb.Locales(&localeData, i) {
			problems = append(problems, unpackLocaleData(&localeData, store)...)
		}
	}

	return store, problems
}

// unpackLocaleData extracts locale data from a FlatBuffer and adds it to the store.
//
// Takes fb (*i18n_fb.LocaleDataFB) which contains the serialised locale data.
// Takes store (*i18n_domain.Store) which receives the unpacked locale entries.
//
// Returns []i18n_domain.TemplateProblem which lists the plural forms in this locale that
// failed to parse.
func unpackLocaleData(fb *i18n_fb.LocaleDataFB, store *i18n_domain.Store) []i18n_domain.TemplateProblem {
	locale := mem.String(fb.Locale())
	entries, problems := unpackEntries(fb)
	store.AddLocale(locale, entries)
	for i := range problems {
		problems[i].Locale = locale
	}
	return problems
}

// unpackEntries converts FlatBuffer translation entries to a domain map.
//
// Takes fb (*i18n_fb.LocaleDataFB) which contains the serialised locale data.
//
// Returns map[string]*i18n_domain.Entry which maps keys to their entries, or nil when
// there are no entries.
// Returns []i18n_domain.TemplateProblem which lists the plural forms that failed to
// parse, with the locale left for the caller to fill in.
func unpackEntries(fb *i18n_fb.LocaleDataFB) (map[string]*i18n_domain.Entry, []i18n_domain.TemplateProblem) {
	entriesLen := fb.EntriesLength()
	if entriesLen == 0 {
		return nil, nil
	}

	entries := make(map[string]*i18n_domain.Entry, entriesLen)
	var problems []i18n_domain.TemplateProblem
	var entryFB i18n_fb.TranslationEntryFB
	for i := range entriesLen {
		if !fb.Entries(&entryFB, i) {
			continue
		}
		key := mem.String(entryFB.Key())
		entry, entryProblems := unpackEntry(&entryFB)
		entries[key] = entry
		for _, problem := range entryProblems {
			problem.Key = key
			problems = append(problems, problem)
		}
	}

	return entries, problems
}

// unpackEntry converts a FlatBuffer translation entry to a domain entry.
//
// Plural forms are parsed into their own parts. A form that fails to parse renders as its
// literal text; when the first form fails the entry keeps the parts stored in the
// manifest instead of losing them.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the serialised entry to unpack.
//
// Returns *i18n_domain.Entry which contains the unpacked template, parts, and plural
// forms.
// Returns []i18n_domain.TemplateProblem which lists the forms that failed to parse, with
// the locale and key left for the caller to fill in.
func unpackEntry(fb *i18n_fb.TranslationEntryFB) (*i18n_domain.Entry, []i18n_domain.TemplateProblem) {
	entry := &i18n_domain.Entry{
		Template:         mem.String(fb.Template()),
		Parts:            unpackTemplateParts(fb),
		PluralForms:      unpackPluralForms(fb),
		HasPlurals:       fb.HasPlurals(),
		PluralFormsParts: nil,
	}

	if !entry.HasPlurals || len(entry.PluralForms) == 0 {
		return entry, nil
	}

	formsParts, problems := i18n_domain.ParsePluralForms(entry.PluralForms)
	entry.PluralFormsParts = formsParts
	firstFormFailed := slices.ContainsFunc(problems, func(problem i18n_domain.TemplateProblem) bool {
		return problem.FormIndex == 0
	})
	if !firstFormFailed {
		entry.Parts = formsParts[0]
	}

	return entry, problems
}

// unpackTemplateParts extracts template parts from a FlatBuffer translation entry.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the FlatBuffer entry to extract parts
// from.
//
// Returns []i18n_domain.TemplatePart which contains the extracted template parts, or nil
// if the entry has no parts.
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
				Expression: nil,
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
// Takes fb (*i18n_fb.TranslationEntryFB) which is the FlatBuffer entry to extract from.
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
