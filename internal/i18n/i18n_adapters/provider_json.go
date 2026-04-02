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
	"io/fs"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// dirPermission is the permission mode for created directories (owner rwx,
	// group rx).
	dirPermission fs.FileMode = 0o750

	// filePermission is the file mode for created files (owner read and write
	// only).
	filePermission fs.FileMode = 0o600
)

// jsonProvider loads translations from JSON files. This provider is intended
// for debugging and development, as it parses templates at runtime rather than
// using pre-parsed FlatBuffer data.
type jsonProvider struct {
	// sandbox provides safe file system access for reading translation files.
	sandbox safedisk.Sandbox

	// directory is the path to the folder with JSON translation files,
	// relative to the sandbox root.
	directory string
}

// load reads all JSON translation files from the directory and populates the
// store.
//
// Takes defaultLocale (string) which specifies the fallback locale for missing
// translations.
//
// Returns *i18n_domain.Store which contains all loaded translations.
// Returns error when the directory path is empty, unreadable, or a file fails
// to parse.
func (p *jsonProvider) load(defaultLocale string) (*i18n_domain.Store, error) {
	if p.directory == "" {
		return nil, errors.New("JSON provider requires a valid directory path")
	}

	store := i18n_domain.NewStore(defaultLocale)

	entries, err := p.sandbox.ReadDir(p.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read i18n directory %s: %w", p.directory, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		locale := strings.TrimSuffix(name, ".json")

		filePath := filepath.Join(p.directory, name)
		if err := p.parseAndLoadJSONFile(store, locale, filePath); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", filePath, err)
		}
	}

	return store, nil
}

// parseAndLoadJSONFile reads, parses, and loads translations from a single
// JSON file.
//
// Takes store (*i18n_domain.Store) which receives the parsed translations.
// Takes locale (string) which identifies the language for the translations.
// Takes filePath (string) which specifies the JSON file to read.
//
// Returns error when the file cannot be read or the JSON is invalid.
func (p *jsonProvider) parseAndLoadJSONFile(store *i18n_domain.Store, locale, filePath string) error {
	data, err := p.sandbox.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	translations, err := i18n_domain.ParseAndFlatten(data)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	store.AddTranslations(locale, translations)
	return nil
}

// jsonEmitter writes translations to JSON files for debugging purposes.
// All file operations use a sandbox for security.
type jsonEmitter struct {
	// sandbox provides file system operations for writing output files.
	sandbox safedisk.Sandbox
}

// emit writes the store's translations to JSON files in the specified
// directory. Each locale gets its own file (e.g., en-GB.json, fr-FR.json).
//
// Takes store (*i18n_domain.Store) which contains the translations to emit.
// Takes outputDir (string) which specifies the output directory relative to
// the sandbox root.
//
// Returns error when the output directory cannot be created, marshalling
// fails, or a file cannot be written.
func (e *jsonEmitter) emit(store *i18n_domain.Store, outputDir string) error {
	if err := e.sandbox.MkdirAll(outputDir, dirPermission); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, locale := range store.Locales() {
		entries := store.GetEntriesForLocale(locale)
		if entries == nil {
			continue
		}

		translations := make(map[string]string, len(entries))
		for key, entry := range entries {
			translations[key] = entry.Template
		}

		data, err := json.ConfigStd.MarshalIndent(translations, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal translations for %s: %w", locale, err)
		}

		filePath := filepath.Join(outputDir, locale+".json")
		if err := e.sandbox.WriteFile(filePath, data, filePermission); err != nil {
			return fmt.Errorf("failed to write %s: %w", filePath, err)
		}
	}

	return nil
}

// emitSingle writes all translations to a single JSON file.
// The structure is: { "locale": { "key": "value" } }.
//
// Takes store (*i18n_domain.Store) which provides the translations to emit.
// Takes outputPath (string) which is the file path relative to the sandbox
// root.
//
// Returns error when marshalling fails or the file cannot be written.
func (e *jsonEmitter) emitSingle(store *i18n_domain.Store, outputPath string) error {
	allTranslations := make(map[string]map[string]string)

	for _, locale := range store.Locales() {
		entries := store.GetEntriesForLocale(locale)
		if entries == nil {
			continue
		}

		translations := make(map[string]string, len(entries))
		for key, entry := range entries {
			translations[key] = entry.Template
		}
		allTranslations[locale] = translations
	}

	data, err := json.ConfigStd.MarshalIndent(allTranslations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal translations: %w", err)
	}

	directory := filepath.Dir(outputPath)
	if err := e.sandbox.MkdirAll(directory, dirPermission); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := e.sandbox.WriteFile(outputPath, data, filePermission); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// newJSONProvider creates a new JSON provider for the given directory.
// The directory should contain files named {locale}.json (e.g., en-GB.json,
// fr-FR.json) and should be relative to the sandbox root.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes directory (string) which specifies the path to locale files.
//
// Returns *jsonProvider which is the configured provider ready for use.
func newJSONProvider(sandbox safedisk.Sandbox, directory string) *jsonProvider {
	return &jsonProvider{
		sandbox:   sandbox,
		directory: directory,
	}
}

// newJSONEmitter creates a new JSON emitter with the given sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
//
// Returns *jsonEmitter which is the configured emitter ready for use.
func newJSONEmitter(sandbox safedisk.Sandbox) *jsonEmitter {
	return &jsonEmitter{sandbox: sandbox}
}
