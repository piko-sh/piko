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
	"flag"
	"fmt"
	"reflect"
	"strings"
)

// stringSliceValue wraps a string slice pointer for use as a
// flag value, implementing fmt.Stringer, where slice points to
// the target slice that stores the parsed flag values.
type stringSliceValue struct {
	// slice points to the target slice that stores parsed values.
	slice *[]string
}

// String returns the slice as a string with elements joined by a separator.
//
// Returns string which is the joined elements, or empty if the slice is nil.
func (s *stringSliceValue) String() string {
	if s.slice == nil {
		return ""
	}
	return strings.Join(*s.slice, defaultDelimiter)
}

// Set parses a delimiter-separated string into the slice.
//
// Takes value (string) which is the string to parse. An empty string clears
// the slice.
//
// Returns error which is always nil.
func (s *stringSliceValue) Set(value string) error {
	if value == "" {
		*s.slice = []string{}
	} else {
		*s.slice = strings.Split(value, defaultDelimiter)
	}
	return nil
}

// stringMapValue holds the state for parsing string-to-string map flags.
// It implements fmt.Stringer.
type stringMapValue struct {
	// sMap points to the map that holds parsed key-value pairs.
	sMap *map[string]string

	// tags holds struct field tags for parsing options like delimiter and separator.
	tags reflect.StructTag
}

// String returns the map as a formatted string for flag display.
//
// Returns string which is the formatted key-value pairs using configured
// delimiters, or an empty string if the map is nil or empty.
func (m *stringMapValue) String() string {
	if m.sMap == nil || len(*m.sMap) == 0 {
		return ""
	}
	delimiter := m.tags.Get("delimiter")
	if delimiter == "" {
		delimiter = defaultDelimiter
	}
	separator := m.tags.Get("separator")
	if separator == "" {
		separator = defaultSeparator
	}

	var builder strings.Builder
	builder.Grow(len(*m.sMap) * 16)

	i := 0
	for key, value := range *m.sMap {
		if i > 0 {
			builder.WriteString(delimiter)
		}
		builder.WriteString(key)
		builder.WriteString(separator)
		builder.WriteString(value)
		i++
	}
	return builder.String()
}

// Set parses a formatted string into the map.
//
// Takes value (string) which contains the key=value pairs to parse.
//
// Returns error when the string format is invalid or parsing fails.
func (m *stringMapValue) Set(value string) error {
	tempMap := make(map[string]string)
	if err := setMapField(reflect.ValueOf(&tempMap).Elem(), value, m.tags); err != nil {
		return fmt.Errorf("setting map field from flag value %q: %w", value, err)
	}
	*m.sMap = tempMap
	return nil
}

// parseFlags parses command-line flags and records which flags were set on
// the struct fields.
//
// Takes ptr (any) which is the struct pointer to receive flag values.
// Takes ctx (*LoadContext) which provides the loading context with prefix
// information.
//
// Returns error when flag parsing fails.
func (l *Loader) parseFlags(ptr any, ctx *LoadContext) error {
	coordinator := l.flagCoordinator

	if err := coordinator.Parse(); err != nil {
		return fmt.Errorf("error parsing flags: %w", err)
	}

	l.attributeVisitedFlagsFromCoordinator(ptr, ctx, coordinator)
	return nil
}

// attributeVisitedFlagsFromCoordinator assigns visited flags from the
// coordinator to the configuration structure.
//
// Takes ptr (any) which is a pointer to the configuration structure.
// Takes ctx (*LoadContext) which provides the loading context.
// Takes coordinator (*FlagCoordinator) which tracks visited flags.
func (l *Loader) attributeVisitedFlagsFromCoordinator(ptr any, ctx *LoadContext, coordinator *FlagCoordinator) {
	visitedFlags := coordinator.GetVisitedFlags(l.opts.FlagPrefix)

	for _, visitedFlag := range visitedFlags {
		processor := makeFlagAttributionProcessor(ctx, visitedFlag)
		state := &walkState{
			processor: processor,
			ctx:       ctx,
			keyPrefix: "",
			source:    "",
		}
		_ = l.walk(reflect.ValueOf(ptr), state)
	}
}

// makeFlagAttributionProcessor creates a processor that records where a
// configuration field value came from when it was set by a command-line flag.
//
// Takes ctx (*LoadContext) which holds the map of field source attributions.
// Takes visitedFlag (*flag.Flag) which is the flag being recorded.
//
// Returns processorFunc which updates the context with the flag source.
func makeFlagAttributionProcessor(ctx *LoadContext, visitedFlag *flag.Flag) processorFunc {
	sourceName := fmt.Sprintf("%s: -%s", sourceFlag, visitedFlag.Name)
	return func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		if flagName, ok := field.Tag.Lookup("flag"); ok && flagName == visitedFlag.Name {
			ctx.FieldSources[keyPath] = sourceName
		}
		return nil
	}
}

// buildFlagUsage builds a usage string from struct field tags.
//
// Takes tags (reflect.StructTag) which holds the field metadata.
//
// Returns string which is the usage text with environment variable name and
// default value added if present.
func buildFlagUsage(tags reflect.StructTag) string {
	usage := tags.Get("usage")
	if envName := tags.Get("env"); envName != "" {
		usage = fmt.Sprintf("%s (env: %s)", usage, envName)
	}
	if defaultVal, ok := tags.Lookup("default"); ok {
		usage = fmt.Sprintf("%s [default: %s]", usage, defaultVal)
	}
	return usage
}

// filterTestFlags removes Go test flags and common test utility flags from
// the given argument list.
//
// Takes arguments ([]string) which contains the command line arguments to filter.
//
// Returns []string which contains only arguments that are not test flags.
func filterTestFlags(arguments []string) []string {
	filtered := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if strings.HasPrefix(argument, "-test.") || argument == "-update" {
			continue
		}
		filtered = append(filtered, argument)
	}
	return filtered
}
