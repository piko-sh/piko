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

package config_domain

import (
	"cmp"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// summaryEntry holds the details of a single configuration field for summary
// output.
type summaryEntry struct {
	// KeyPath is the full dot-separated path to the field within the config
	// struct (e.g. "Network.Port").
	KeyPath string

	// Value is the final resolved value of the field. It holds strings, ints,
	// bools, and other types.
	Value any

	// Source is the name of the source that provided the final value.
	Source string

	// Tag holds the struct tag for this field. Used to check for metadata
	// like `summary:"hide"` to redact sensitive values.
	Tag reflect.StructTag
}

// Summarise creates a readable string of all configuration fields that were
// set by a source other than 'default'. It groups the fields by their source
// (file, env, flag, etc.) and can hide sensitive values.
//
// A field can be marked to hide its value in the summary by adding the struct
// tag `summary:"hide"`.
//
// Takes ctx (*LoadContext) which provides the configuration target and field
// source mappings to summarise.
//
// Returns string which contains the formatted summary of user-set values.
// Returns error when ctx is nil, ctx.Target is nil, or a field path cannot be
// resolved.
func Summarise(ctx *LoadContext) (string, error) {
	if ctx == nil || ctx.Target == nil {
		return "", errors.New("invalid LoadContext provided")
	}

	entries := make([]summaryEntry, 0, len(ctx.FieldSources))
	targetValue := reflect.Indirect(reflect.ValueOf(ctx.Target))

	for keyPath, source := range ctx.FieldSources {
		if source == sourceDefault {
			continue
		}

		field, value, err := getFieldAndValueByPath(targetValue, keyPath)
		if err != nil {
			return "", fmt.Errorf("could not retrieve value for path %q: %w", keyPath, err)
		}
		if field == nil {
			return "", fmt.Errorf("could not retrieve field for path %q", keyPath)
		}

		entries = append(entries, summaryEntry{
			KeyPath: keyPath,
			Value:   value.Interface(),
			Source:  source,
			Tag:     field.Tag,
		})
	}

	if len(entries) == 0 {
		return "No user-configured values were set. Using all defaults.", nil
	}

	slices.SortFunc(entries, func(a, b summaryEntry) int {
		return cmp.Or(
			cmp.Compare(a.Source, b.Source),
			cmp.Compare(a.KeyPath, b.KeyPath),
		)
	})

	return formatSummary(entries), nil
}

// formatSummary builds the final output string from the sorted entries.
//
// Takes entries ([]summaryEntry) which contains the configuration entries to
// format.
//
// Returns string which is the formatted summary with source groupings and
// sensitive values hidden.
func formatSummary(entries []summaryEntry) string {
	var builder strings.Builder
	builder.WriteString("--- Applied Configuration Summary ---\n")
	currentSource := ""

	for _, entry := range entries {
		if entry.Source != currentSource {
			currentSource = entry.Source
			_, _ = fmt.Fprintf(&builder, "\n[Source: %s]\n", currentSource)
		}

		var displayValue any
		if entry.Tag.Get("summary") == "hide" {
			displayValue = "[REDACTED]"
		} else {
			displayValue = entry.Value
		}

		_, _ = fmt.Fprintf(&builder, "  %-40s = %v\n", entry.KeyPath, displayValue)
	}

	return builder.String()
}

// getFieldAndValueByPath navigates a struct using a dot-separated path and
// returns the StructField and Value for the final field.
//
// Takes v (reflect.Value) which is the struct value to navigate.
// Takes path (string) which is the dot-separated path to the target field.
//
// Returns *reflect.StructField which describes the final field in the path.
// Returns reflect.Value which is the value of the final field.
// Returns error when the path is empty, a path part is not a struct, a field
// is not found, or a pointer in the path is nil.
func getFieldAndValueByPath(v reflect.Value, path string) (*reflect.StructField, reflect.Value, error) {
	if path == "" {
		return nil, reflect.Value{}, errors.New("empty path")
	}
	parts := strings.Split(path, ".")
	currentVal := v
	var currentField *reflect.StructField

	for _, part := range parts {
		if currentVal.Kind() != reflect.Struct {
			return nil, reflect.Value{}, fmt.Errorf("path part %q is not a struct", part)
		}

		fieldStruct, found := currentVal.Type().FieldByName(part)
		if !found {
			return nil, reflect.Value{}, fmt.Errorf("field %q not found in struct", part)
		}
		currentField = &fieldStruct
		currentVal = currentVal.FieldByName(part)

		if currentVal.Kind() == reflect.Pointer {
			if currentVal.IsNil() {
				return nil, reflect.Value{}, fmt.Errorf("path part %q is a nil pointer", part)
			}
			currentVal = currentVal.Elem()
		}
	}

	return currentField, currentVal, nil
}
