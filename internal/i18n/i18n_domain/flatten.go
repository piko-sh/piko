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
	"errors"
	"fmt"

	"piko.sh/piko/internal/json"
)

const (
	// DefaultFlattenMaxDepth caps recursion when walking nested translation
	// payloads. Attacker-uploadable JSON could otherwise trigger stack
	// exhaustion through arbitrary nesting depths.
	DefaultFlattenMaxDepth = 32

	// DefaultFlattenMaxKeyCount caps the total number of keys a flattened
	// translation map may produce. This prevents wide-and-flat exhaustion
	// attacks even when nesting stays shallow.
	DefaultFlattenMaxKeyCount = 100_000
)

// ErrTranslationDepthExceeded is returned when a nested translation map
// exceeds the configured recursion depth. Callers can use errors.Is to
// detect this condition without parsing the message.
var ErrTranslationDepthExceeded = errors.New("translation nesting depth exceeded")

// ErrTranslationKeyCountExceeded is returned when flattening a translation
// payload would produce more entries than permitted. Callers can use
// errors.Is to detect this condition without parsing the message.
var ErrTranslationKeyCountExceeded = errors.New("translation key count exceeded")

// FlattenOptions configures the safety limits used when walking nested
// translation maps. A zero-valued FlattenOptions falls back to the
// package-level defaults.
type FlattenOptions struct {
	// MaxDepth limits how deeply FlattenTranslations recurses through nested
	// maps. A value of zero or below uses DefaultFlattenMaxDepth.
	MaxDepth int

	// MaxKeyCount limits the total number of flattened entries written to the
	// result map. A value of zero or below uses DefaultFlattenMaxKeyCount.
	MaxKeyCount int
}

// resolve returns a FlattenOptions with package-level defaults filled in for
// any unset (zero) fields.
//
// Returns FlattenOptions which is a copy of the receiver with zero-valued
// fields replaced by the package-level defaults.
func (o FlattenOptions) resolve() FlattenOptions {
	resolved := o
	if resolved.MaxDepth <= 0 {
		resolved.MaxDepth = DefaultFlattenMaxDepth
	}
	if resolved.MaxKeyCount <= 0 {
		resolved.MaxKeyCount = DefaultFlattenMaxKeyCount
	}
	return resolved
}

// FlattenTranslations converts nested translation maps into flat key-value
// pairs using dot notation. For example, {"user": {"name": "Name"}} becomes
// {"user.name": "Name"}.
//
// The walk is bounded by the package-level defaults
// (DefaultFlattenMaxDepth and DefaultFlattenMaxKeyCount) to prevent stack
// overflow and memory exhaustion from attacker-uploadable JSON. Use
// FlattenTranslationsWithOptions to override the limits.
//
// Takes data (map[string]any) which contains the nested translation map to
// flatten.
// Takes prefix (string) which specifies the key prefix for nested calls, or an
// empty string for the root call.
// Takes result (map[string]string) which collects the flattened key-value
// pairs.
//
// Returns error which wraps ErrTranslationDepthExceeded when nesting exceeds
// DefaultFlattenMaxDepth or ErrTranslationKeyCountExceeded when the result
// map would grow beyond DefaultFlattenMaxKeyCount.
func FlattenTranslations(data map[string]any, prefix string, result map[string]string) error {
	return FlattenTranslationsWithOptions(data, prefix, result, FlattenOptions{})
}

// FlattenTranslationsWithOptions is the explicit form of FlattenTranslations
// that accepts caller-supplied limits. Zero-valued option fields are filled
// with the package-level defaults.
//
// Takes data (map[string]any) which contains the nested translation map.
// Takes prefix (string) which specifies the key prefix for nested calls.
// Takes result (map[string]string) which collects the flattened entries.
// Takes opts (FlattenOptions) which configures the recursion and key-count
// limits.
//
// Returns error which wraps ErrTranslationDepthExceeded or
// ErrTranslationKeyCountExceeded when the supplied payload exceeds the
// resolved limits.
func FlattenTranslationsWithOptions(data map[string]any, prefix string, result map[string]string, opts FlattenOptions) error {
	resolved := opts.resolve()
	return flattenWithLimits(data, prefix, result, resolved, 0)
}

// flattenWithLimits walks a nested translation map and writes its leaves into
// result using dot-notation keys.
//
// Recursion stops when depth reaches opts.MaxDepth and entry insertion stops
// when result would exceed opts.MaxKeyCount.
//
// Takes data (map[string]any) which contains the nested translation map to
// walk.
// Takes prefix (string) which is the dot-notation key prefix accumulated by
// the caller.
// Takes result (map[string]string) which collects the flattened entries.
// Takes opts (FlattenOptions) which carries the resolved depth and key-count
// limits.
// Takes depth (int) which is the current recursion depth.
//
// Returns error which wraps ErrTranslationDepthExceeded when nesting exceeds
// opts.MaxDepth or ErrTranslationKeyCountExceeded when the result map would
// grow beyond opts.MaxKeyCount.
func flattenWithLimits(data map[string]any, prefix string, result map[string]string, opts FlattenOptions, depth int) error {
	if depth >= opts.MaxDepth {
		return fmt.Errorf("at depth %d (limit %d): %w", depth, opts.MaxDepth, ErrTranslationDepthExceeded)
	}

	for key, value := range data {
		newKey := key
		if prefix != "" {
			newKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]any:
			if err := flattenWithLimits(v, newKey, result, opts, depth+1); err != nil {
				return err
			}
		case string:
			if err := assignFlattened(result, newKey, v, opts.MaxKeyCount); err != nil {
				return err
			}
		default:
			if err := assignFlattened(result, newKey, fmt.Sprintf("%v", v), opts.MaxKeyCount); err != nil {
				return err
			}
		}
	}
	return nil
}

// assignFlattened writes a single flattened key-value pair into result while
// enforcing the configured maximum key count.
//
// Takes result (map[string]string) which receives the new entry.
// Takes key (string) which is the dot-notation key being assigned.
// Takes value (string) which is the translation value to store.
// Takes maxKeyCount (int) which is the upper bound on result's size before
// inserting a new key.
//
// Returns error which wraps ErrTranslationKeyCountExceeded when adding a new
// key would push result beyond maxKeyCount entries.
func assignFlattened(result map[string]string, key, value string, maxKeyCount int) error {
	if _, exists := result[key]; !exists && len(result) >= maxKeyCount {
		return fmt.Errorf("flattened key count would exceed %d: %w", maxKeyCount, ErrTranslationKeyCountExceeded)
	}
	result[key] = value
	return nil
}

// ParseAndFlatten parses JSON translation data and returns a flattened map
// with dot-notation keys. The walk is bounded by the package-level safety
// limits; use ParseAndFlattenWithOptions to override them.
//
// Takes jsonData ([]byte) which contains the raw JSON translation data.
//
// Returns map[string]string which maps dot-notation keys to their translated
// values.
// Returns error when the JSON cannot be parsed or when the payload exceeds
// the configured depth or key-count limits.
func ParseAndFlatten(jsonData []byte) (map[string]string, error) {
	return ParseAndFlattenWithOptions(jsonData, FlattenOptions{})
}

// ParseAndFlattenWithOptions is the explicit form of ParseAndFlatten that
// accepts caller-supplied limits.
//
// Takes jsonData ([]byte) which contains the raw JSON translation data.
// Takes opts (FlattenOptions) which configures the recursion and key-count
// limits.
//
// Returns map[string]string which maps dot-notation keys to their translated
// values.
// Returns error when the JSON cannot be parsed or when the payload exceeds
// the resolved limits.
func ParseAndFlattenWithOptions(jsonData []byte, opts FlattenOptions) (map[string]string, error) {
	var nested map[string]any
	if err := json.Unmarshal(jsonData, &nested); err != nil {
		return nil, fmt.Errorf("failed to unmarshal i18n JSON: %w", err)
	}

	flattened := make(map[string]string)
	if err := FlattenTranslationsWithOptions(nested, "", flattened, opts); err != nil {
		return nil, fmt.Errorf("flattening translations: %w", err)
	}
	return flattened, nil
}
