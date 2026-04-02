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
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultStrBufPoolCapacity is the initial capacity for the string buffer pool.
	defaultStrBufPoolCapacity = 512

	// flatBufferFileName is the name of the pre-compiled FlatBuffer binary.
	flatBufferFileName = "i18n.bin"
)

// fsService implements i18n_domain.Service using the file system.
type fsService struct {
	// store holds the loaded translations; nil if no translations are loaded.
	store *i18n_domain.Store

	// strBufPool provides pooled string buffers for zero-allocation rendering.
	strBufPool *i18n_domain.StrBufPool

	// defaultLocale is the fallback locale used when no other locale matches.
	defaultLocale string
}

// GetStore returns the translation Store for zero-allocation lookups.
//
// Returns *i18n_domain.Store which is the loaded translation store, or nil if
// no translations are loaded.
func (s *fsService) GetStore() *i18n_domain.Store {
	return s.store
}

// GetStrBufPool returns a shared buffer pool for zero-allocation string
// rendering.
//
// Returns *i18n_domain.StrBufPool which is the shared buffer pool, or nil if
// not initialised.
func (s *fsService) GetStrBufPool() *i18n_domain.StrBufPool {
	return s.strBufPool
}

// DefaultLocale returns the default locale for fallback resolution.
//
// Returns string which is the locale code used when no other locale matches.
func (s *fsService) DefaultLocale() string {
	return s.defaultLocale
}

// NewService creates an i18n service using the best available source.
//
// It first checks for a pre-compiled FlatBuffer binary (dist/i18n.bin) for
// optimal performance. If not found, it falls back to loading from JSON files.
// This is the recommended constructor for production use.
//
// Takes sandbox (safedisk.Sandbox) which handles all file operations.
// Takes defaultLocale (string) which specifies the fallback locale.
// Takes i18nSourceDir (string) which specifies the path to i18n JSON files.
//
// Returns i18n_domain.Service which is the configured i18n service.
// Returns error when the service cannot be initialised.
func NewService(ctx context.Context, sandbox safedisk.Sandbox, defaultLocale, i18nSourceDir string) (i18n_domain.Service, error) {
	ctx, l := logger_domain.From(ctx, log)
	flatBufferPath := filepath.Join("dist", flatBufferFileName)
	if _, err := sandbox.Stat(flatBufferPath); err == nil {
		l.Internal("Found pre-compiled i18n FlatBuffer, loading...",
			logger_domain.String("path", flatBufferPath))
		return NewFlatBufferService(ctx, sandbox, flatBufferPath, defaultLocale)
	}

	l.Internal("No pre-compiled i18n FlatBuffer found, loading from JSON files.")
	return NewFSService(ctx, sandbox, defaultLocale, i18nSourceDir)
}

// NewFlatBufferService creates a translation service from a pre-compiled
// FlatBuffer binary file. This gives zero-copy loading for fast startup.
//
// Takes ctx (context.Context) which carries logger and tracing data.
// Takes sandbox (safedisk.Sandbox) which provides filesystem access.
// Takes filePath (string) which is the path relative to the sandbox root.
// Takes defaultLocale (string) which sets the fallback locale; uses "en" if
// empty.
//
// Returns i18n_domain.Service which is the ready-to-use translation service.
// Returns error when the FlatBuffer file cannot be loaded.
func NewFlatBufferService(ctx context.Context, sandbox safedisk.Sandbox, filePath string, defaultLocale string) (i18n_domain.Service, error) {
	_, l := logger_domain.From(ctx, log)
	defaultLocale = cmp.Or(defaultLocale, "en")

	provider := newFlatBufferProvider(sandbox, filePath)
	store, err := provider.load()
	if err != nil {
		return nil, fmt.Errorf("loading i18n FlatBuffer from %q: %w", filePath, err)
	}

	pool := i18n_domain.NewStrBufPool(defaultStrBufPoolCapacity)

	l.Internal("Loaded i18n from FlatBuffer",
		logger_domain.Int("locale_count", len(store.Locales())))

	return &fsService{
		store:         store,
		strBufPool:    pool,
		defaultLocale: defaultLocale,
	}, nil
}

// NewFSService creates a translation service that loads from JSON files.
//
// When the translation folder is not set up or does not exist, this returns
// an empty service.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes defaultLocale (string) which specifies the fallback locale; uses "en"
// if empty.
// Takes i18nSourceDir (string) which specifies the path to i18n JSON files.
//
// Returns i18n_domain.Service which looks up translations.
// Returns error when the service cannot be set up.
func NewFSService(ctx context.Context, sandbox safedisk.Sandbox, defaultLocale, i18nSourceDir string) (i18n_domain.Service, error) {
	defaultLocale = cmp.Or(defaultLocale, "en")

	return newFSServiceFromDir(ctx, sandbox, i18nSourceDir, defaultLocale)
}

// newFSServiceFromDir creates an i18n service from a specific directory path.
//
// Takes ctx (context.Context) which carries logger and tracing data.
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes dirPath (string) which is the path relative to the sandbox root.
// Takes defaultLocale (string) which specifies the fallback locale.
//
// Returns i18n_domain.Service which is the configured translation service.
// Returns error when service creation fails.
func newFSServiceFromDir(ctx context.Context, sandbox safedisk.Sandbox, dirPath, defaultLocale string) (i18n_domain.Service, error) {
	ctx, l := logger_domain.From(ctx, log)
	translations := make(i18n_domain.Translations)

	if dirPath == "" {
		l.Internal("i18n source directory not configured, skipping global translations.")
		return newEmptyService(defaultLocale), nil
	}

	files, err := sandbox.ReadDir(dirPath)
	if err != nil {
		l.Internal("i18n source directory does not exist, skipping global translations.",
			logger_domain.String("path", dirPath))
		return newEmptyService(defaultLocale), nil
	}

	loadJSONFiles(ctx, sandbox, files, dirPath, translations)

	store := i18n_domain.NewStoreFromTranslations(translations, defaultLocale)
	pool := i18n_domain.NewStrBufPool(defaultStrBufPoolCapacity)

	return &fsService{
		store:         store,
		strBufPool:    pool,
		defaultLocale: defaultLocale,
	}, nil
}

// newEmptyService creates an empty i18n service with no translations.
//
// Takes defaultLocale (string) which specifies the fallback locale to use.
//
// Returns *fsService which is an uninitialised service with empty
// translations and nil store.
func newEmptyService(defaultLocale string) *fsService {
	return &fsService{
		store:         nil,
		strBufPool:    nil,
		defaultLocale: defaultLocale,
	}
}

// loadJSONFiles loads all JSON translation files from a directory.
//
// Takes ctx (context.Context) which carries logger and tracing data.
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes files ([]fs.DirEntry) which contains the directory entries to check.
// Takes dirPath (string) which specifies the path to the translation folder.
// Takes translations (i18n_domain.Translations) which stores the loaded
// translations by locale.
func loadJSONFiles(ctx context.Context, sandbox safedisk.Sandbox, files []fs.DirEntry, dirPath string, translations i18n_domain.Translations) {
	_, l := logger_domain.From(ctx, log)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		locale := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(dirPath, file.Name())

		localeTranslations, err := loadJSONFile(sandbox, filePath)
		if err != nil {
			l.Error("Failed to load i18n file",
				logger_domain.String("path", filePath),
				logger_domain.Error(err))
			continue
		}

		translations[locale] = localeTranslations
		l.Internal("Loaded global translations",
			logger_domain.String("locale", locale),
			logger_domain.Int("keys", len(localeTranslations)))
	}
}

// loadJSONFile reads and parses a single JSON translation file.
//
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes filePath (string) which specifies the path to the JSON file.
//
// Returns map[string]string which contains the flattened translation keys and
// values.
// Returns error when the file cannot be read or the JSON is not valid.
func loadJSONFile(sandbox safedisk.Sandbox, filePath string) (map[string]string, error) {
	content, err := sandbox.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading i18n JSON file %q: %w", filePath, err)
	}
	translations, err := i18n_domain.ParseAndFlatten(content)
	if err != nil {
		return nil, fmt.Errorf("parsing and flattening i18n JSON file %q: %w", filePath, err)
	}
	return translations, nil
}
