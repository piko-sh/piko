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
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/i18n/i18n_adapters"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// I18nEmitterConfig holds the settings needed by the i18n emitter.
type I18nEmitterConfig struct {
	// BaseDir is the project root used to resolve relative paths.
	BaseDir string

	// I18nSourceDir is the directory containing translation JSON files.
	I18nSourceDir string

	// DefaultLocale is the fallback locale when none is specified.
	DefaultLocale string
}

// DrivenI18nEmitter implements I18nEmitterPort.
//
// This adapter orchestrates the generation of i18n FlatBuffer binary:
//  1. Loads translations from JSON files in the configured i18n directory
//  2. Parses and flattens the translations
//  3. Serialises them to a FlatBuffer binary
//  4. Writes the binary to the output path
//
// Architecture:
//   - Uses FlatBufferEmitter from i18n_adapters for serialisation
//   - Reads from the i18n source directory configured in I18nEmitterConfig
//   - Output is designed for zero-copy loading at runtime
//   - All file operations are sandboxed to prevent path traversal attacks
type DrivenI18nEmitter struct {
	// sourceSandbox is the sandbox for reading i18n source files.
	sourceSandbox safedisk.Sandbox

	// outputSandbox is the file system sandbox where output files are written.
	outputSandbox safedisk.Sandbox

	// config holds settings for path resolution and i18n defaults.
	config I18nEmitterConfig
}

var _ generator_domain.I18nEmitterPort = (*DrivenI18nEmitter)(nil)

// NewDrivenI18nEmitter creates a new i18n emitter instance.
//
// Takes config (I18nEmitterConfig) which provides the i18n settings.
// Takes sourceSandbox (safedisk.Sandbox) which gives read access to i18n
// source files in the project root.
// Takes outputSandbox (safedisk.Sandbox) which gives write access to the dist
// folder for output files.
//
// Returns *DrivenI18nEmitter which is ready for use.
func NewDrivenI18nEmitter(
	config I18nEmitterConfig,
	sourceSandbox safedisk.Sandbox,
	outputSandbox safedisk.Sandbox,
) *DrivenI18nEmitter {
	return &DrivenI18nEmitter{
		config: config,

		sourceSandbox: sourceSandbox,
		outputSandbox: outputSandbox,
	}
}

// EmitI18n loads translations and writes them to a FlatBuffer binary file.
//
// Takes outputPath (string) which specifies where to write the binary file.
//
// Returns error when reading or parsing translation files fails, or when
// writing the FlatBuffer output fails.
func (e *DrivenI18nEmitter) EmitI18n(ctx context.Context, outputPath string) error {
	ctx, l := logger_domain.From(ctx, log)
	if e.config.I18nSourceDir == "" {
		l.Internal("i18n source directory not configured, skipping FlatBuffer emission.")
		return nil
	}

	dirPath := e.resolveI18nSourceDir()

	translations, err := e.loadTranslationFiles(ctx, dirPath)
	if err != nil {
		return fmt.Errorf("loading translation files from %q: %w", dirPath, err)
	}

	if len(translations) == 0 {
		l.Internal("No translations found, skipping FlatBuffer emission.")
		return nil
	}

	return e.emitFlatBuffer(ctx, translations, outputPath)
}

// resolveI18nSourceDir builds the full path to the i18n source folder.
//
// Returns string which is the resolved folder path.
func (e *DrivenI18nEmitter) resolveI18nSourceDir() string {
	dirPath := e.config.I18nSourceDir
	if e.config.BaseDir != "" && e.config.BaseDir != "." {
		dirPath = filepath.Join(e.config.BaseDir, e.config.I18nSourceDir)
	}
	return dirPath
}

// loadTranslationFiles reads all JSON translation files from the given
// directory.
//
// Takes ctx (context.Context) which carries logging context.
// Takes dirPath (string) which is the directory to scan for translation files.
//
// Returns i18n_domain.Translations which contains all loaded translations.
// Returns error when reading the directory fails, except for non-existent
// directories which return nil.
func (e *DrivenI18nEmitter) loadTranslationFiles(ctx context.Context, dirPath string) (i18n_domain.Translations, error) {
	ctx, l := logger_domain.From(ctx, log)
	files, err := e.sourceSandbox.ReadDir(dirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			l.Internal("i18n source directory does not exist, skipping FlatBuffer emission.",
				logger_domain.String("path", dirPath))
			return nil, nil
		}
		return nil, fmt.Errorf("reading i18n source directory %q: %w", dirPath, err)
	}

	translations := make(i18n_domain.Translations)
	for _, file := range files {
		if ctx.Err() != nil {
			return translations, ctx.Err()
		}

		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		e.loadSingleTranslationFile(ctx, dirPath, file.Name(), translations)
	}
	return translations, nil
}

// loadSingleTranslationFile loads and parses a single translation JSON file.
//
// Takes ctx (context.Context) which carries logging context.
// Takes dirPath (string) which is the directory containing the file.
// Takes fileName (string) which is the name of the JSON file.
// Takes translations (i18n_domain.Translations) which receives the parsed
// translations.
func (e *DrivenI18nEmitter) loadSingleTranslationFile(
	ctx context.Context,
	dirPath string,
	fileName string,
	translations i18n_domain.Translations,
) {
	locale := strings.TrimSuffix(fileName, ".json")
	filePath := filepath.Join(dirPath, fileName)

	_, l := logger_domain.From(ctx, log)
	content, err := e.sourceSandbox.ReadFile(filePath)
	if err != nil {
		l.Error("Failed to read i18n file",
			logger_domain.String("path", filePath),
			logger_domain.Error(err))
		return
	}

	localeTranslations, err := i18n_domain.ParseAndFlatten(content)
	if err != nil {
		l.Error("Failed to parse and flatten i18n JSON file",
			logger_domain.String("path", filePath),
			logger_domain.Error(err))
		return
	}

	translations[locale] = localeTranslations
	l.Internal("Loaded translations for FlatBuffer emission",
		logger_domain.String("locale", locale),
		logger_domain.Int("keys", len(localeTranslations)))
}

// emitFlatBuffer serialises translations to a FlatBuffer binary file.
//
// Takes translations (i18n_domain.Translations) which contains the translations
// to emit.
// Takes outputPath (string) which specifies where to write the binary.
//
// Returns error when FlatBuffer emission fails.
func (e *DrivenI18nEmitter) emitFlatBuffer(ctx context.Context, translations i18n_domain.Translations, outputPath string) error {
	ctx, l := logger_domain.From(ctx, log)
	defaultLocale := cmp.Or(e.config.DefaultLocale, "en")

	emitter := i18n_adapters.NewFlatBufferEmitter(e.outputSandbox)
	if err := emitter.EmitFromTranslations(ctx, translations, defaultLocale, outputPath); err != nil {
		return fmt.Errorf("emitting i18n FlatBuffer to %q: %w", outputPath, err)
	}

	l.Internal("i18n FlatBuffer emitted successfully",
		logger_domain.String("output_path", outputPath),
		logger_domain.Int("locale_count", len(translations)))

	return nil
}
