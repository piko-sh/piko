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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"piko.sh/piko/internal/json"
	"gopkg.in/yaml.v3"
)

// unmarshalerFunc is a function type that converts bytes into a value.
type unmarshalerFunc func([]byte, any) error

var unmarshalerMap = map[string]unmarshalerFunc{
	".json": json.Unmarshal,
	".yaml": yaml.Unmarshal,
	".yml":  yaml.Unmarshal,
}

// loadFiles loads configuration from each file path in order.
//
// Takes ptr (any) which is the configuration struct to populate.
// Takes ctx (*LoadContext) which tracks the source of each field change.
//
// Returns error when a file cannot be read, has an unsupported format,
// fails strict mode validation, or cannot be unmarshalled.
func (l *Loader) loadFiles(ptr any, ctx *LoadContext) error {
	for _, path := range l.opts.FilePaths {
		data, err := l.fileReader.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("failed to read config file %q: %w", path, err)
		}

		unmarshaler, ext := getUnmarshaler(path)
		if unmarshaler == nil {
			return fmt.Errorf("unsupported config file format: %s", ext)
		}

		if l.opts.StrictFile {
			if err := checkStrict(data, ptr, unmarshaler); err != nil {
				return fmt.Errorf("strict mode check failed for file %q: %w", path, err)
			}
		}

		before := reflect.Indirect(reflect.ValueOf(ptr)).Interface()

		if err := unmarshaler(data, ptr); err != nil {
			return fmt.Errorf("failed to unmarshal config file %q: %w", path, err)
		}

		sourceName := fmt.Sprintf("%s: %s", sourceFile, filepath.Base(path))
		detectChanges(reflect.ValueOf(before), reflect.Indirect(reflect.ValueOf(ptr)), "", sourceName, ctx)
	}
	return nil
}

// getUnmarshaler returns the unmarshaler function for a given file path.
//
// Takes path (string) which is the file path to get the format from.
//
// Returns unmarshalerFunc which is the unmarshaler for the file extension, or
// nil if no unmarshaler is set for that extension.
// Returns string which is the lowercase file extension.
func getUnmarshaler(path string) (unmarshalerFunc, string) {
	ext := strings.ToLower(filepath.Ext(path))
	unmarshaler, ok := unmarshalerMap[ext]
	if !ok {
		return nil, ext
	}
	return unmarshaler, ext
}

// checkStrict checks that data contains no unknown configuration keys.
//
// Takes data ([]byte) which contains the raw configuration data to check.
// Takes ptr (any) which is a pointer to the struct that defines valid keys.
// Takes unmarshaler (unmarshalerFunc) which decodes the data into a map.
//
// Returns error when decoding fails or when unknown keys are found.
func checkStrict(data []byte, ptr any, unmarshaler unmarshalerFunc) error {
	var fileMap map[string]any
	if err := unmarshaler(data, &fileMap); err != nil {
		return fmt.Errorf("cannot unmarshal for strict check: %w", err)
	}
	validKeys := make(map[string]struct{})
	collectTags(reflect.TypeOf(ptr).Elem(), validKeys)

	var unknownKeys []string
	for key := range fileMap {
		if _, ok := validKeys[key]; !ok {
			unknownKeys = append(unknownKeys, key)
		}
	}
	if len(unknownKeys) > 0 {
		return fmt.Errorf("unknown configuration keys found: %s", strings.Join(unknownKeys, ", "))
	}
	return nil
}

// collectTags gathers JSON and YAML struct field tags into a key set.
//
// It looks at each field in the struct and gets the tag name from json or yaml
// tags. If neither tag is present, it uses the lowercase field name. It skips
// fields marked with "-". For nested structs, it calls itself to collect their
// tags as well.
//
// Takes valueType (reflect.Type) which is the struct type to inspect.
// Takes keys (map[string]struct{}) which stores the found tag names.
func collectTags(valueType reflect.Type, keys map[string]struct{}) {
	for field := range valueType.Fields() {
		tag := field.Tag.Get("json")
		if tag == "" {
			tag = field.Tag.Get("yaml")
		}
		if tag == "" {
			tag = strings.ToLower(field.Name)
		}
		tag, _, _ = strings.Cut(tag, ",")
		if tag != "-" {
			keys[tag] = struct{}{}
		}
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType.Kind() == reflect.Struct {
			collectTags(fieldType, keys)
		}
	}
}

// detectChanges compares two struct values and records which fields differ.
//
// The function walks through matching fields of before and after, building a
// dot-separated key path. When field values differ, it records the source in
// the context's FieldSources map. For nested structs, it calls itself to
// compare inner fields.
//
// Takes before (reflect.Value) which is the original struct value.
// Takes after (reflect.Value) which is the struct value that may have changed.
// Takes prefix (string) which is the current field path for nested keys.
// Takes source (string) which identifies where the change came from.
// Takes ctx (*LoadContext) which receives the field source mappings.
func detectChanges(before, after reflect.Value, prefix, source string, ctx *LoadContext) {
	if !before.IsValid() || !after.IsValid() || before.Type() != after.Type() {
		return
	}
	for i := range before.NumField() {
		key := before.Type().Field(i).Name
		if prefix != "" {
			key = prefix + "." + key
		}
		beforeField := before.Field(i)
		afterField := after.Field(i)
		if beforeField.Type().Kind() == reflect.Struct {
			detectChanges(beforeField, afterField, key, source, ctx)
			continue
		}
		if !reflect.DeepEqual(beforeField.Interface(), afterField.Interface()) {
			ctx.FieldSources[key] = source
		}
	}
}
