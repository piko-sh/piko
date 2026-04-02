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

package i18n

import (
	"errors"

	i18n_fb "piko.sh/piko/internal/i18n/i18n_schema/i18n_schema_gen"
)

// errFlatBufferParseFailed is returned when a FlatBuffer payload cannot be
// decoded.
var errFlatBufferParseFailed = errors.New("failed to parse FlatBuffer payload")

// I18nManifest is a JSON-serialisable representation of a compiled i18n manifest.
type I18nManifest struct {
	// Locales maps locale codes to their translation data.
	Locales map[string]LocaleData `json:"locales"`

	// DefaultLocale is the fallback locale code.
	DefaultLocale string `json:"default_locale"`

	// Version is the schema version string.
	Version string `json:"version"`
}

// LocaleData holds all translation entries for a single locale.
type LocaleData struct {
	// Entries maps translation keys to their entries.
	Entries map[string]TranslationEntry `json:"entries"`
}

// TranslationEntry is a single translation with its pre-parsed template.
type TranslationEntry struct {
	// Template is the raw template string.
	Template string `json:"template"`

	// Parts holds the pre-parsed template segments.
	Parts []TemplatePart `json:"parts,omitempty"`

	// PluralForms holds the plural variants if any.
	PluralForms []string `json:"plural_forms,omitempty"`

	// PlaceholderNames lists the expression names found in the template.
	PlaceholderNames []string `json:"placeholder_names,omitempty"`

	// HasPlurals indicates whether the template uses plural forms.
	HasPlurals bool `json:"has_plurals,omitempty"`
}

// TemplatePart is a parsed segment of a translation template.
type TemplatePart struct {
	// Kind is the part type: "literal", "expression", or "linked_msg".
	Kind string `json:"kind"`

	// Literal is the text for literal parts.
	Literal string `json:"literal,omitempty"`

	// ExprSource is the expression source code.
	ExprSource string `json:"expr_source,omitempty"`

	// LinkedKey is the key path for linked message references.
	LinkedKey string `json:"linked_key,omitempty"`
}

// partKindNames maps FlatBuffer PartKind enum values to display strings.
var partKindNames = [...]string{
	i18n_fb.PartKindLiteral:       "literal",
	i18n_fb.PartKindExpression:    "expression",
	i18n_fb.PartKindLinkedMessage: "linked_msg",
}

// ConvertI18n parses a raw FlatBuffer i18n payload into a JSON-serialisable
// struct.
//
// Takes payload ([]byte) which is the raw FlatBuffer data after stripping the
// version header (use Unpack first).
//
// Returns *I18nManifest which contains all locale translation data.
// Returns error when the payload cannot be parsed.
func ConvertI18n(payload []byte) (*I18nManifest, error) {
	fb := i18n_fb.GetRootAsI18nManifestFB(payload, 0)
	if fb == nil {
		return nil, errFlatBufferParseFailed
	}

	locales := convertLocales(fb)

	return &I18nManifest{
		DefaultLocale: string(fb.DefaultLocale()),
		Version:       string(fb.Version()),
		Locales:       locales,
	}, nil
}

// convertLocales extracts all locale data from the FlatBuffer.
//
// Takes fb (*i18n_fb.I18nManifestFB) which provides the source manifest.
//
// Returns map[string]LocaleData which maps locale codes to their data, or nil
// if no locales exist.
func convertLocales(fb *i18n_fb.I18nManifestFB) map[string]LocaleData {
	length := fb.LocalesLength()
	if length == 0 {
		return nil
	}
	locales := make(map[string]LocaleData, length)
	var item i18n_fb.LocaleDataFB
	for i := range length {
		if fb.Locales(&item, i) {
			locale := string(item.Locale())
			locales[locale] = convertLocaleData(&item)
		}
	}
	return locales
}

// convertLocaleData extracts entries from a single locale.
//
// Takes fb (*i18n_fb.LocaleDataFB) which contains the FlatBuffer locale data.
//
// Returns LocaleData which contains the converted translation entries.
func convertLocaleData(fb *i18n_fb.LocaleDataFB) LocaleData {
	length := fb.EntriesLength()
	if length == 0 {
		return LocaleData{}
	}
	entries := make(map[string]TranslationEntry, length)
	var item i18n_fb.TranslationEntryFB
	for i := range length {
		if fb.Entries(&item, i) {
			key := string(item.Key())
			entries[key] = convertEntry(&item)
		}
	}
	return LocaleData{Entries: entries}
}

// convertEntry converts a single FlatBuffer translation entry.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the FlatBuffer entry to
// convert.
//
// Returns TranslationEntry which contains the parsed template, parts, plural
// forms, and placeholder names.
func convertEntry(fb *i18n_fb.TranslationEntryFB) TranslationEntry {
	return TranslationEntry{
		Template:         string(fb.Template()),
		Parts:            convertParts(fb),
		PluralForms:      convertPluralForms(fb),
		HasPlurals:       fb.HasPlurals(),
		PlaceholderNames: convertPlaceholderNames(fb),
	}
}

// convertParts extracts template parts from a translation entry.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the FlatBuffers translation
// entry to extract parts from.
//
// Returns []TemplatePart which contains the extracted template parts, or nil
// if the entry has no parts.
func convertParts(fb *i18n_fb.TranslationEntryFB) []TemplatePart {
	length := fb.PartsLength()
	if length == 0 {
		return nil
	}
	parts := make([]TemplatePart, length)
	var item i18n_fb.TemplatePartFB
	for i := range length {
		if fb.Parts(&item, i) {
			kind := "literal"
			kindIndex := int(item.Kind())
			if kindIndex < len(partKindNames) {
				kind = partKindNames[kindIndex]
			}
			parts[i] = TemplatePart{
				Kind:       kind,
				Literal:    string(item.Literal()),
				ExprSource: string(item.ExpressionSource()),
				LinkedKey:  string(item.LinkedKey()),
			}
		}
	}
	return parts
}

// convertPluralForms extracts plural form strings from a FlatBuffer entry.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the FlatBuffer translation
// entry to extract plural forms from.
//
// Returns []string which contains the plural form strings, or nil if no plural
// forms exist.
func convertPluralForms(fb *i18n_fb.TranslationEntryFB) []string {
	length := fb.PluralFormsLength()
	if length == 0 {
		return nil
	}
	forms := make([]string, length)
	for i := range length {
		forms[i] = string(fb.PluralForms(i))
	}
	return forms
}

// convertPlaceholderNames extracts placeholder name strings from a FlatBuffer.
//
// Takes fb (*i18n_fb.TranslationEntryFB) which is the translation entry to
// extract names from.
//
// Returns []string which contains the placeholder names, or nil if none exist.
func convertPlaceholderNames(fb *i18n_fb.TranslationEntryFB) []string {
	length := fb.PlaceholderNamesLength()
	if length == 0 {
		return nil
	}
	names := make([]string, length)
	for i := range length {
		names[i] = string(fb.PlaceholderNames(i))
	}
	return names
}
