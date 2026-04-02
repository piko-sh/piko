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

package markdown_domain

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"piko.sh/piko/internal/markdown/markdown_dto"
)

// Frontmatter represents the structured metadata extracted from the YAML front
// matter of a markdown file.
type Frontmatter struct {
	// PublishDate is when the content should be published.
	PublishDate time.Time

	// Custom holds user-defined frontmatter fields not recognised by the parser.
	Custom map[string]any

	// Navigation holds page navigation settings; nil means no navigation data.
	Navigation *markdown_dto.NavigationMetadata

	// Title is the document title; required in frontmatter.
	Title string

	// Description is a short summary of the page content.
	Description string

	// Tags contains labels used to group and filter content.
	Tags []string

	// Draft indicates whether this content is a draft that should not be
	// published.
	Draft bool
}

// ParseFrontmatter parses raw frontmatter data into a structured Frontmatter
// object and checks that required fields are present.
//
// Takes rawData (map[string]any) which contains the raw frontmatter key-value
// pairs to parse.
//
// Returns *Frontmatter which is the parsed and checked frontmatter data.
// Returns error when required field 'title' is missing or empty.
func ParseFrontmatter(rawData map[string]any) (*Frontmatter, error) {
	knownKeys := map[string]bool{
		"title":       true,
		"description": true,
		"draft":       true,
		"date":        true,
		"tags":        true,
		"nav":         true,
		"navigation":  true,
	}

	fm := &Frontmatter{
		PublishDate: time.Time{},
		Custom:      make(map[string]any),
		Title:       "",
		Description: "",
		Tags:        nil,
		Draft:       false,
		Navigation:  nil,
	}

	for key, value := range rawData {
		lowerKey := strings.ToLower(key)

		if !knownKeys[lowerKey] {
			fm.Custom[key] = value
			continue
		}

		switch lowerKey {
		case "title":
			fm.Title = getString(value)
		case "description":
			fm.Description = getString(value)
		case "draft":
			fm.Draft = getBool(value)
		case "date":
			fm.PublishDate = getDate(value)
			fm.Custom[key] = value
		case "tags":
			fm.Tags = getStringSlice(value)
		case "nav", "navigation":
			fm.Navigation = parseNavigationMetadata(value)
		}
	}

	if fm.Title == "" {
		return nil, errors.New("frontmatter validation failed: required key 'title' is missing or empty")
	}

	return fm, nil
}

// getString extracts a string from an interface value.
//
// Takes v (any) which is the value to extract.
//
// Returns string which is the extracted value, or an empty string if v is not
// a string.
func getString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// getBool extracts a boolean value from an interface value.
//
// Takes v (any) which is the value to extract a boolean from.
//
// Returns bool which is the extracted value, or false if v is not a bool.
func getBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// getDate converts a value to a time.Time.
//
// When v is already a time.Time, returns it directly. When v is a string,
// tries to parse it using RFC3339, datetime, and date-only formats.
//
// Takes v (any) which is the value to convert.
//
// Returns time.Time which is the parsed time, or zero time if parsing fails.
func getDate(v any) time.Time {
	if t, ok := v.(time.Time); ok {
		return t
	}
	if s, ok := v.(string); ok {
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			if t, err := time.Parse(format, s); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// getStringSlice converts a value to a string slice.
//
// When v is nil or an unsupported type, returns nil.
//
// Takes v (any) which is the value to convert.
//
// Returns []string which holds the extracted strings.
func getStringSlice(v any) []string {
	if v == nil {
		return nil
	}

	switch item := v.(type) {
	case []string:
		return item
	case []any:
		return extractStringsFromSlice(item)
	case string:
		return parseStringAsSlice(item)
	default:
		return nil
	}
}

// extractStringsFromSlice picks out string values from a slice of mixed types.
//
// Takes items ([]any) which is the slice to filter.
//
// Returns []string which holds only the string values from the input.
func extractStringsFromSlice(items []any) []string {
	slice := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			slice = append(slice, s)
		}
	}
	return slice
}

// parseStringAsSlice splits a string into a slice using commas as separators.
//
// Takes value (string) which is the input string to split.
//
// Returns []string which holds the trimmed parts, or nil if value is empty.
func parseStringAsSlice(value string) []string {
	if value == "" {
		return nil
	}
	if !strings.Contains(value, ",") {
		return []string{value}
	}
	parts := strings.Split(value, ",")
	slice := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			slice = append(slice, trimmed)
		}
	}
	return slice
}

// parseNavigationMetadata extracts navigation metadata from frontmatter.
//
// Takes v (any) which is the frontmatter value to parse.
//
// Returns *markdown_dto.NavigationMetadata which contains the parsed navigation
// groups, or nil if parsing fails or the value is empty.
//
// Supports both map[string]any (from YAML parsing) and NavigationMetadata
// structs.
//
// Expected YAML structure:
// nav:
//
//	sidebar:
//	  section: "get-started"
//	  order: 10
//	footer:
//	  section: "quick-links"
func parseNavigationMetadata(v any) *markdown_dto.NavigationMetadata {
	if v == nil {
		return nil
	}
	if nav, ok := v.(*markdown_dto.NavigationMetadata); ok {
		return nav
	}

	navMap := toStringKeyMap(v)
	if navMap == nil {
		return nil
	}

	groups := make(map[string]*markdown_dto.NavGroupMetadata)
	hasNonEmptyGroup := false

	for groupName, groupData := range navMap {
		group := parseNavGroup(groupData)
		if group == nil {
			continue
		}
		if isNonEmptyNavGroup(group) {
			hasNonEmptyGroup = true
		}
		groups[groupName] = group
	}

	if !hasNonEmptyGroup {
		return nil
	}
	return &markdown_dto.NavigationMetadata{Groups: groups}
}

// toStringKeyMap converts a value to a map with string keys.
//
// It handles both map[string]any and map[any]any input types.
//
// Takes v (any) which is the value to convert.
//
// Returns map[string]any which is the converted map, or nil if the input
// is not a supported map type.
func toStringKeyMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if m, ok := v.(map[any]any); ok {
		return convertInterfaceMap(m)
	}
	return nil
}

// parseNavGroup parses a single navigation group from map data.
//
// Takes groupData (any) which contains the raw map data to parse.
//
// Returns *markdown_dto.NavGroupMetadata which contains the parsed navigation
// group, or nil if groupData cannot be converted to a string-keyed map.
func parseNavGroup(groupData any) *markdown_dto.NavGroupMetadata {
	groupMap := toStringKeyMap(groupData)
	if groupMap == nil {
		return nil
	}
	return &markdown_dto.NavGroupMetadata{
		Section:    getString(groupMap["section"]),
		Subsection: getString(groupMap["subsection"]),
		Order:      getInt(groupMap["order"]),
		Icon:       getString(groupMap["icon"]),
		Hidden:     getBool(groupMap["hidden"]),
		Parent:     getString(groupMap["parent"]),
		Label:      getString(groupMap["label"]),
	}
}

// isNonEmptyNavGroup reports whether the group has any non-zero values.
//
// Takes g (*markdown_dto.NavGroupMetadata) which is the navigation group to
// check.
//
// Returns bool which is true when any field in the group has a non-zero value.
func isNonEmptyNavGroup(g *markdown_dto.NavGroupMetadata) bool {
	return g.Section != "" || g.Subsection != "" || g.Order != 0 ||
		g.Icon != "" || g.Hidden || g.Parent != "" || g.Label != ""
}

// convertInterfaceMap converts map[any]any to map[string]any.
// This is needed because YAML parsers return any keys by default.
//
// Takes m (map[any]any) which is the map with any-typed keys to convert.
//
// Returns map[string]any which contains only string keys from the input.
func convertInterfaceMap(m map[any]any) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		strKey, ok := k.(string)
		if !ok {
			continue
		}

		if nestedMap, ok := v.(map[any]any); ok {
			result[strKey] = convertInterfaceMap(nestedMap)
		} else {
			result[strKey] = v
		}
	}
	return result
}

// getInt extracts an integer from an any value with fallback to 0.
//
// Supports type coercion from int, int64, float64, and string.
//
// Takes v (any) which is the value to extract an integer from.
//
// Returns int which is the extracted integer, or 0 for invalid or nil values.
func getInt(v any) int {
	if v == nil {
		return 0
	}

	switch item := v.(type) {
	case int:
		return item
	case int64:
		return int(item)
	case float64:
		return int(item)
	case string:
		if i, err := strconv.Atoi(item); err == nil {
			return i
		}
	}

	return 0
}
