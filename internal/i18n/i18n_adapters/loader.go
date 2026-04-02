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

	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/wdk/safedisk"
)

// LoaderMode specifies how translations are loaded.
type LoaderMode string

const (
	// LoaderModeFlatBuffer is the default loading mode using pre-parsed FlatBuffer
	// files with no memory allocation.
	LoaderModeFlatBuffer LoaderMode = "flatbuffer"

	// LoaderModeJSON uses JSON files for debugging and development.
	// Templates are parsed at runtime, which is slower but easier to inspect.
	LoaderModeJSON LoaderMode = "json"
)

// LoaderConfig configures the translation loader.
type LoaderConfig struct {
	// Sandbox provides safe file system access for loading translation files.
	Sandbox safedisk.Sandbox

	// Mode specifies whether to load translations from FlatBuffer or JSON.
	// Default is LoaderModeFlatBuffer.
	Mode LoaderMode

	// FlatBufferPath is the path to the i18n.bin FlatBuffer file, relative to
	// the sandbox root. Required when Mode is LoaderModeFlatBuffer.
	FlatBufferPath string

	// JSONDirectory is the path to the directory with JSON translation files,
	// relative to the sandbox root. Required when Mode is LoaderModeJSON.
	JSONDirectory string

	// DefaultLocale is the fallback locale when a translation is missing.
	DefaultLocale string
}

// Loader provides a unified interface for loading translations from
// either FlatBuffer or JSON sources based on configuration.
// All file operations are sandboxed for security.
type Loader struct {
	// fbProvider holds the FlatBuffer provider; kept alive to retain loaded data.
	fbProvider *flatBufferProvider

	// config holds the loader settings.
	config LoaderConfig
}

// NewLoader creates a new translation loader with the given configuration.
//
// Takes config (LoaderConfig) which specifies the loader settings including
// mode and default locale.
//
// Returns *Loader which is ready for use after loading translations.
func NewLoader(config LoaderConfig) *Loader {
	if config.Mode == "" {
		config.Mode = LoaderModeFlatBuffer
	}
	if config.DefaultLocale == "" {
		config.DefaultLocale = "en-GB"
	}
	return &Loader{
		fbProvider: nil,
		config:     config,
	}
}

// Load reads translations from the configured source and returns a Store.
//
// For FlatBuffer mode: Uses zero-allocation parsing. The Loader must be kept
// alive as long as the Store is in use to prevent the underlying data from
// being garbage collected.
//
// For JSON mode: Parses templates at runtime. Suitable for debugging.
//
// Returns *i18n_domain.Store which contains the loaded translations.
// Returns error when the loader mode is unknown or loading fails.
func (l *Loader) Load() (*i18n_domain.Store, error) {
	switch l.config.Mode {
	case LoaderModeFlatBuffer:
		return l.loadFlatBuffer()
	case LoaderModeJSON:
		return l.loadJSON()
	default:
		return nil, fmt.Errorf("unknown loader mode: %s", l.config.Mode)
	}
}

// Mode returns the current loader mode.
//
// Returns LoaderMode which indicates how the loader processes input.
func (l *Loader) Mode() LoaderMode {
	return l.config.Mode
}

// Config returns the loader configuration.
//
// Returns LoaderConfig which contains the current settings for this loader.
func (l *Loader) Config() LoaderConfig {
	return l.config
}

// loadFlatBuffer loads translations from a FlatBuffer file.
//
// Returns *i18n_domain.Store which contains the loaded translations.
// Returns error when FlatBufferPath is empty, sandbox is nil, or loading fails.
func (l *Loader) loadFlatBuffer() (*i18n_domain.Store, error) {
	if l.config.FlatBufferPath == "" {
		return nil, errors.New("FlatBuffer path is required for FlatBuffer mode")
	}
	if l.config.Sandbox == nil {
		return nil, errors.New("sandbox is required for FlatBuffer mode")
	}

	l.fbProvider = newFlatBufferProvider(l.config.Sandbox, l.config.FlatBufferPath)
	store, err := l.fbProvider.load()
	if err != nil {
		return nil, fmt.Errorf("failed to load FlatBuffer translations: %w", err)
	}

	return store, nil
}

// loadJSON loads translations from JSON files in the configured directory.
//
// Returns *i18n_domain.Store which contains the loaded translations.
// Returns error when the JSON directory or sandbox is not configured, or when
// loading fails.
func (l *Loader) loadJSON() (*i18n_domain.Store, error) {
	if l.config.JSONDirectory == "" {
		return nil, errors.New("JSON directory is required for JSON mode")
	}
	if l.config.Sandbox == nil {
		return nil, errors.New("sandbox is required for JSON mode")
	}

	provider := newJSONProvider(l.config.Sandbox, l.config.JSONDirectory)
	store, err := provider.load(l.config.DefaultLocale)
	if err != nil {
		return nil, fmt.Errorf("failed to load JSON translations: %w", err)
	}

	return store, nil
}
