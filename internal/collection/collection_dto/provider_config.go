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

package collection_dto

import "slices"

// ProviderConfig contains configuration for a collection provider.
//
// Passed to providers during initialisation and collection discovery. Provides
// both generic framework configuration and provider-specific custom settings.
//
// Design philosophy:
//   - Generic config: Common settings all providers need (base paths, locales)
//   - Custom config: Opaque map for provider-specific settings
//   - Type-safe access: Helper methods for common patterns
type ProviderConfig struct {
	// Custom holds provider-specific settings as key-value pairs.
	//
	// The framework does not read these values. They are passed directly
	// to the provider from the user's piko.config.yaml file.
	// Use GetCustomString, GetCustomInt, or GetCustomBool to retrieve values.
	Custom map[string]any

	// BasePath is the root folder of the project.
	//
	// Providers use this as the starting point to work out relative paths.
	BasePath string

	// DefaultLocale is the locale used when no locale is specified.
	DefaultLocale string

	// Locales is the list of locales configured for the project.
	//
	// Providers should use this to validate locale-specific content
	// and filter results appropriately.
	Locales []string
}

// GetCustomString retrieves a string value from custom config with a fallback.
//
// Takes key (string) which specifies the custom config key to look up.
// Takes defaultValue (string) which provides the fallback if the key is missing
// or if the value is not a string.
//
// Returns string which is the value from custom config or the default.
func (c *ProviderConfig) GetCustomString(key, defaultValue string) string {
	if value, ok := c.Custom[key]; ok {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

// GetCustomInt retrieves an integer value from custom config with a fallback.
//
// Takes key (string) which specifies the custom config key to look up.
// Takes defaultValue (int) which is returned when the key is not found or
// cannot be converted to int.
//
// Returns int which is the value from custom config, or defaultValue if the
// key is missing or not a numeric type.
func (c *ProviderConfig) GetCustomInt(key string, defaultValue int) int {
	if value, ok := c.Custom[key]; ok {
		switch v := value.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultValue
}

// GetCustomBool retrieves a boolean value from custom config with a fallback.
//
// Takes key (string) which specifies the configuration key to look up.
// Takes defaultValue (bool) which is returned if the key is not found or is
// not a boolean.
//
// Returns bool which is the value from custom config, or the default.
func (c *ProviderConfig) GetCustomBool(key string, defaultValue bool) bool {
	if value, ok := c.Custom[key]; ok {
		if boolValue, ok := value.(bool); ok {
			return boolValue
		}
	}
	return defaultValue
}

// HasCustom checks if a custom config key exists.
//
// Takes key (string) which is the config key to look up.
//
// Returns bool which is true if the key exists in the custom config map.
func (c *ProviderConfig) HasCustom(key string) bool {
	_, ok := c.Custom[key]
	return ok
}

// CollectionInfo describes a collection offered by a provider.
//
// This is returned by the provider's DiscoverCollections() method to inform
// the framework about available collections.
type CollectionInfo struct {
	// Schema maps field names to their types for this collection.
	//
	// The key is the field name and the value is the type as a string
	// (e.g., "string", "int", "bool", "[]string").
	//
	// Checks that user-defined target structs match the available fields.
	Schema map[string]string

	// Metadata holds extra details from the provider for display and debugging.
	// Not used by the framework.
	Metadata map[string]any

	// Name is the unique identifier for this collection.
	//
	// Users reference this value in p-collection:src or GetCollection().
	Name string

	// Path is the source path for this collection.
	//
	// The meaning depends on the provider type:
	//   - File-based providers: directory path (e.g. "content/blog")
	//   - API-based providers: endpoint path (e.g. "/api/v1/collections/blog")
	//   - Database providers: table name (e.g. "public.blog_posts")
	Path string

	// Locales lists the locales this collection supports. Empty when the
	// collection does not use locales.
	Locales []string

	// ItemCount is the number of items in this collection.
	// Set to -1 if the count is not known or does not apply.
	ItemCount int
}

// HasLocale checks if the collection supports a specific locale.
//
// Takes locale (string) which is the locale code to check for.
//
// Returns bool which is true if the locale is supported.
func (c *CollectionInfo) HasLocale(locale string) bool {
	return slices.Contains(c.Locales, locale)
}

// IsMultiLocale reports whether the collection supports multiple locales.
//
// Returns bool which is true when the collection has more than one locale.
func (c *CollectionInfo) IsMultiLocale() bool {
	return len(c.Locales) > 1
}

// HasSchema reports whether schema information is available.
//
// Returns bool which is true when the collection has schema fields defined.
func (c *CollectionInfo) HasSchema() bool {
	return len(c.Schema) > 0
}

// GetFieldType returns the type of a field from the schema.
//
// Takes fieldName (string) which specifies the field to look up.
//
// Returns string which is the field type, or an empty string if the field is
// not in the schema.
func (c *CollectionInfo) GetFieldType(fieldName string) string {
	if c.Schema == nil {
		return ""
	}
	return c.Schema[fieldName]
}
