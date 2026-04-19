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

package i18n_domain

import (
	"strings"
	"sync"
)

// defaultResolveMessageBufCapacity is the default buffer capacity for message
// resolution.
const defaultResolveMessageBufCapacity = 256

// resolveMessageBufPool is a package-level buffer pool for message resolution.
// This avoids allocating a new StrBuf on every ResolveMessage call.
var resolveMessageBufPool = NewStrBufPool(defaultResolveMessageBufCapacity)

// Entry represents a single translation entry with pre-parsed template data.
type Entry struct {
	// Template is the raw message string with placeholders.
	Template string

	// Parts holds the parsed template parts for non-plural templates.
	Parts []TemplatePart

	// PluralForms holds the plural form strings split from the template.
	PluralForms []string

	// PluralFormsParts holds pre-parsed plural forms, indexed by form number.
	PluralFormsParts [][]TemplatePart

	// HasPlurals indicates whether this entry has plural forms.
	HasPlurals bool
}

// localeData holds all translations for a single locale.
type localeData struct {
	// entries maps keys to their translated entry values.
	entries map[string]*Entry

	// locale is the language code for translations.
	locale string
}

// Store holds translations for multiple locales with fallback support.
// It implements messageResolver and is safe for concurrent access.
type Store struct {
	// locales maps locale codes to their translation data.
	locales map[string]*localeData

	// fallbackChain maps each locale to its ordered list of fallback locales.
	fallbackChain map[string][]string

	// defaultLocale is the fallback locale used when a key is not found in the
	// requested locale.
	defaultLocale string

	// mu guards all store fields for concurrent access.
	mu sync.RWMutex
}

// NewStore creates a new translation store.
//
// Takes defaultLocale (string) which specifies the fallback locale for
// translations.
//
// Returns *Store which is an empty store ready to have locales added.
func NewStore(defaultLocale string) *Store {
	return &Store{
		locales:       make(map[string]*localeData),
		fallbackChain: make(map[string][]string),
		defaultLocale: defaultLocale,
		mu:            sync.RWMutex{},
	}
}

// NewStoreFromTranslations creates a populated Store from a Translations map.
//
// Takes translations (Translations) which maps locale codes to key-value
// translation pairs.
// Takes defaultLocale (string) which specifies the fallback locale.
//
// Returns *Store which is the populated store ready for use.
func NewStoreFromTranslations(translations Translations, defaultLocale string) *Store {
	store := NewStore(defaultLocale)
	for locale, entries := range translations {
		store.AddTranslations(locale, entries)
	}
	return store
}

// SetDefaultLocale sets the default locale for fallback.
//
// Takes locale (string) which specifies the locale code to use as fallback.
//
// Safe for concurrent use.
func (s *Store) SetDefaultLocale(locale string) {
	s.mu.Lock()
	s.defaultLocale = locale
	s.mu.Unlock()
}

// AddLocale adds or updates translations for a locale.
//
// Takes locale (string) which is the locale identifier to add or update.
// Takes entries (map[string]*Entry) which contains the translation entries.
//
// Safe for concurrent use. Holds a mutex lock while updating the locale data and
// rebuilding the fallback chain.
func (s *Store) AddLocale(locale string, entries map[string]*Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := &localeData{
		locale:  locale,
		entries: entries,
	}
	s.locales[locale] = data

	s.fallbackChain[locale] = buildFallbackChain(locale, s.defaultLocale)
}

// AddTranslations adds raw translation strings for a locale, parsing them
// into entries.
//
// Takes locale (string) which identifies the target locale.
// Takes translations (map[string]string) which provides the raw translation
// strings to parse.
func (s *Store) AddTranslations(locale string, translations map[string]string) {
	entries := make(map[string]*Entry, len(translations))
	for key, template := range translations {
		entries[key] = parseEntry(template)
	}
	s.AddLocale(locale, entries)
}

// Get retrieves a translation entry for the given locale and key.
// It follows the fallback chain if the key is not found in the requested
// locale.
//
// Takes locale (string) which specifies the preferred locale for the entry.
// Takes key (string) which identifies the translation to retrieve.
//
// Returns *Entry which is the found translation entry, or nil if not found.
// Returns bool which indicates whether the entry was found.
//
// Safe for concurrent use.
func (s *Store) Get(locale, key string) (*Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entry := s.getFromLocale(locale, key); entry != nil {
		return entry, true
	}

	chain := s.fallbackChain[locale]
	for _, fallbackLocale := range chain {
		if entry := s.getFromLocale(fallbackLocale, key); entry != nil {
			return entry, true
		}
	}

	if locale != s.defaultLocale {
		if entry := s.getFromLocale(s.defaultLocale, key); entry != nil {
			return entry, true
		}
	}

	return nil, false
}

// HasLocale returns true if the store has translations for the given locale.
//
// Takes locale (string) which specifies the locale code to check.
//
// Returns bool which is true if translations exist for the locale.
//
// Safe for concurrent use.
func (s *Store) HasLocale(locale string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.locales[locale]
	return ok
}

// Locales returns a list of all available locales.
//
// Returns []string which contains the locale identifiers.
//
// Safe for concurrent use.
func (s *Store) Locales() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	locales := make([]string, 0, len(s.locales))
	for locale := range s.locales {
		locales = append(locales, locale)
	}
	return locales
}

// GetEntriesForLocale returns all entries for a specific locale.
// This is used by the FlatBuffer emitter to serialise translations.
//
// Takes locale (string) which specifies the locale to retrieve entries for.
//
// Returns map[string]*Entry which contains all entries for the locale, or nil
// if the locale does not exist.
//
// Safe for concurrent use.
func (s *Store) GetEntriesForLocale(locale string) map[string]*Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, ok := s.locales[locale]
	if !ok {
		return nil
	}
	return data.entries
}

// DefaultLocale returns the default locale.
//
// Returns string which is the configured default locale identifier.
//
// Safe for concurrent use.
func (s *Store) DefaultLocale() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.defaultLocale
}

// ResolveMessage implements the MessageResolver interface for linked message
// resolution.
//
// The key is a dot-separated path like "common.greeting" which maps directly
// to translation keys.
//
// Takes key (string) which specifies the dot-separated path to the message.
// Takes locale (string) which identifies the target language.
// Takes scope (map[string]any) which provides variables for template rendering.
// Takes depth (int) which tracks recursion level for linked messages.
//
// Returns string which contains the resolved and rendered message.
// Returns bool which indicates whether the message was found and resolved.
func (s *Store) ResolveMessage(key, locale string, scope map[string]any, depth int) (string, bool) {
	if depth >= maxLinkedMessageDepth {
		return "", false
	}

	entry, found := s.Get(locale, key)
	if !found {
		return "", false
	}

	buffer := resolveMessageBufPool.Get()
	defer resolveMessageBufPool.Put(buffer)

	var parts []TemplatePart
	if len(entry.Parts) > 0 {
		parts = entry.Parts
	} else {
		parts, _ = ParseTemplate(entry.Template)
	}

	ctx := &renderContext{
		scope:    scope,
		resolver: s,
		locale:   locale,
		count:    nil,
		buffer:   buffer,
		depth:    depth,
	}

	result := renderTemplate(parts, ctx)
	return result, true
}

// getFromLocale retrieves an entry from a specific locale.
//
// Takes locale (string) which identifies the locale to search.
// Takes key (string) which specifies the entry key to look up.
//
// Returns *Entry which is the found entry, or nil if the locale or key does
// not exist.
func (s *Store) getFromLocale(locale, key string) *Entry {
	data, ok := s.locales[locale]
	if !ok {
		return nil
	}
	return data.entries[key]
}

var _ messageResolver = (*Store)(nil)

// parseEntry parses a template string into an Entry.
//
// Takes template (string) which is the translation template to parse.
//
// Returns *Entry which holds the parsed template parts and plural forms.
func parseEntry(template string) *Entry {
	hasPlurals := HasPluralForms(template)

	entry := &Entry{
		Template:         template,
		Parts:            nil,
		PluralForms:      nil,
		PluralFormsParts: nil,
		HasPlurals:       hasPlurals,
	}

	if hasPlurals {
		entry.PluralForms = SplitPluralForms(template)
		entry.PluralFormsParts = make([][]TemplatePart, len(entry.PluralForms))
		for i, form := range entry.PluralForms {
			entry.PluralFormsParts[i], _ = ParseTemplate(form)
			preparseExpressions(entry.PluralFormsParts[i])
		}
		if len(entry.PluralFormsParts) > 0 {
			entry.Parts = entry.PluralFormsParts[0]
		}
	} else {
		entry.Parts, _ = ParseTemplate(template)
		preparseExpressions(entry.Parts)
	}

	return entry
}

// preparseExpressions parses all expression ASTs in the given parts at load
// time rather than at render time.
//
// Takes parts ([]TemplatePart) which contains the template parts to process.
func preparseExpressions(parts []TemplatePart) {
	for i := range parts {
		if parts[i].Kind == PartExpression && parts[i].Expression == nil {
			parts[i].Expression = getOrParseExpression(parts[i].ExprSource)
		}
	}
}

// buildFallbackChain creates a list of base language codes to try in order.
// For example, "en-GB" falls back to "en", then to the default locale.
//
// Takes locale (string) which is the main locale to build fallbacks for.
// Takes defaultLocale (string) which is the fallback when the main locale is
// not found.
//
// Returns []string which contains the base language codes to try in order.
func buildFallbackChain(locale, defaultLocale string) []string {
	var chain []string

	if baseLang, _, found := strings.Cut(locale, "-"); found && baseLang != locale {
		chain = append(chain, baseLang)
	}

	if defaultLocale != "" && defaultLocale != locale {
		if baseLang, _, found := strings.Cut(defaultLocale, "-"); found && baseLang != defaultLocale {
			chain = append(chain, baseLang)
		}
	}

	return chain
}
