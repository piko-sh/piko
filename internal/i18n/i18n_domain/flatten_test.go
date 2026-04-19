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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlattenTranslations_EmptyMap(t *testing.T) {
	data := map[string]any{}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Empty(t, result)
}

func TestFlattenTranslations_SingleLevel(t *testing.T) {
	data := map[string]any{
		"hello": "world",
		"foo":   "bar",
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Len(t, result, 2)
	assert.Equal(t, "world", result["hello"])
	assert.Equal(t, "bar", result["foo"])
}

func TestFlattenTranslations_NestedMap(t *testing.T) {
	data := map[string]any{
		"user": map[string]any{
			"name":  "Name",
			"email": "Email",
		},
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Len(t, result, 2)
	assert.Equal(t, "Name", result["user.name"])
	assert.Equal(t, "Email", result["user.email"])
}

func TestFlattenTranslations_DeeplyNested(t *testing.T) {
	data := map[string]any{
		"app": map[string]any{
			"settings": map[string]any{
				"theme": map[string]any{
					"dark":  "Dark Mode",
					"light": "Light Mode",
				},
			},
		},
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Len(t, result, 2)
	assert.Equal(t, "Dark Mode", result["app.settings.theme.dark"])
	assert.Equal(t, "Light Mode", result["app.settings.theme.light"])
}

func TestFlattenTranslations_MixedLevels(t *testing.T) {
	data := map[string]any{
		"title": "Welcome",
		"nav": map[string]any{
			"home":    "Home",
			"about":   "About",
			"contact": "Contact",
		},
		"footer": map[string]any{
			"copyright": "2024",
			"links": map[string]any{
				"privacy": "Privacy Policy",
				"terms":   "Terms of Service",
			},
		},
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Len(t, result, 7)
	assert.Equal(t, "Welcome", result["title"])
	assert.Equal(t, "Home", result["nav.home"])
	assert.Equal(t, "About", result["nav.about"])
	assert.Equal(t, "Contact", result["nav.contact"])
	assert.Equal(t, "2024", result["footer.copyright"])
	assert.Equal(t, "Privacy Policy", result["footer.links.privacy"])
	assert.Equal(t, "Terms of Service", result["footer.links.terms"])
}

func TestFlattenTranslations_WithPrefix(t *testing.T) {
	data := map[string]any{
		"hello": "world",
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "en", result))

	assert.Len(t, result, 1)
	assert.Equal(t, "world", result["en.hello"])
}

func TestFlattenTranslations_NonStringValues(t *testing.T) {
	data := map[string]any{
		"count":   42,
		"price":   19.99,
		"enabled": true,
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Len(t, result, 3)
	assert.Equal(t, "42", result["count"])
	assert.Equal(t, "19.99", result["price"])
	assert.Equal(t, "true", result["enabled"])
}

func TestFlattenTranslations_EmptyStrings(t *testing.T) {
	data := map[string]any{
		"empty":    "",
		"nonempty": "value",
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Len(t, result, 2)
	assert.Equal(t, "", result["empty"])
	assert.Equal(t, "value", result["nonempty"])
}

func TestFlattenTranslations_SpecialCharactersInKeys(t *testing.T) {
	data := map[string]any{
		"key-with-dash":       "dash",
		"key_with_underscore": "underscore",
		"key with space":      "space",
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Len(t, result, 3)
	assert.Equal(t, "dash", result["key-with-dash"])
	assert.Equal(t, "underscore", result["key_with_underscore"])
	assert.Equal(t, "space", result["key with space"])
}

func TestFlattenTranslations_UnicodeContent(t *testing.T) {
	data := map[string]any{
		"greeting": "Bonjour",
		"message":  "Willkommen",
		"emoji":    "Hello World",
	}
	result := make(map[string]string)

	require.NoError(t, FlattenTranslations(data, "", result))

	assert.Len(t, result, 3)
	assert.Equal(t, "Bonjour", result["greeting"])
	assert.Equal(t, "Willkommen", result["message"])
	assert.Equal(t, "Hello World", result["emoji"])
}

func TestParseAndFlatten_ValidJSON(t *testing.T) {
	jsonData := []byte(`{
		"greeting": "Hello",
		"user": {
			"name": "Name",
			"email": "Email"
		}
	}`)

	result, err := ParseAndFlatten(jsonData)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "Hello", result["greeting"])
	assert.Equal(t, "Name", result["user.name"])
	assert.Equal(t, "Email", result["user.email"])
}

func TestParseAndFlatten_EmptyJSON(t *testing.T) {
	jsonData := []byte(`{}`)

	result, err := ParseAndFlatten(jsonData)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestParseAndFlatten_InvalidJSON(t *testing.T) {
	jsonData := []byte(`{invalid json}`)

	result, err := ParseAndFlatten(jsonData)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to unmarshal i18n JSON")
}

func TestParseAndFlatten_EmptyInput(t *testing.T) {
	jsonData := []byte(``)

	result, err := ParseAndFlatten(jsonData)

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestParseAndFlatten_NullJSON(t *testing.T) {
	jsonData := []byte(`null`)

	result, err := ParseAndFlatten(jsonData)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestParseAndFlatten_ArrayJSON(t *testing.T) {
	jsonData := []byte(`["item1", "item2"]`)

	result, err := ParseAndFlatten(jsonData)

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestParseAndFlatten_ComplexNesting(t *testing.T) {
	jsonData := []byte(`{
		"common": {
			"buttons": {
				"submit": "Submit",
				"cancel": "Cancel",
				"save": "Save"
			},
			"errors": {
				"required": "This field is required",
				"invalid": "Invalid value"
			}
		},
		"pages": {
			"home": {
				"title": "Welcome",
				"subtitle": "Hello, world!"
			},
			"about": {
				"title": "About Us"
			}
		}
	}`)

	result, err := ParseAndFlatten(jsonData)

	require.NoError(t, err)
	assert.Len(t, result, 8)
	assert.Equal(t, "Submit", result["common.buttons.submit"])
	assert.Equal(t, "Cancel", result["common.buttons.cancel"])
	assert.Equal(t, "Save", result["common.buttons.save"])
	assert.Equal(t, "This field is required", result["common.errors.required"])
	assert.Equal(t, "Invalid value", result["common.errors.invalid"])
	assert.Equal(t, "Welcome", result["pages.home.title"])
	assert.Equal(t, "Hello, world!", result["pages.home.subtitle"])
	assert.Equal(t, "About Us", result["pages.about.title"])
}

func TestParseAndFlatten_NumericValues(t *testing.T) {
	jsonData := []byte(`{
		"count": 100,
		"price": 29.99,
		"label": "Items"
	}`)

	result, err := ParseAndFlatten(jsonData)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "100", result["count"])
	assert.Equal(t, "29.99", result["price"])
	assert.Equal(t, "Items", result["label"])
}

func TestParseAndFlatten_BooleanValues(t *testing.T) {
	jsonData := []byte(`{
		"enabled": true,
		"disabled": false,
		"label": "Status"
	}`)

	result, err := ParseAndFlatten(jsonData)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, "true", result["enabled"])
	assert.Equal(t, "false", result["disabled"])
	assert.Equal(t, "Status", result["label"])
}

func TestFlattenTranslations_RecursionDepthExceeded(t *testing.T) {
	t.Parallel()

	build := func(depth int) map[string]any {
		root := map[string]any{}
		current := root
		for range depth {
			next := map[string]any{}
			current["k"] = next
			current = next
		}
		current["leaf"] = "value"
		return root
	}

	data := build(DefaultFlattenMaxDepth + 1)
	result := make(map[string]string)

	err := FlattenTranslations(data, "", result)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTranslationDepthExceeded)
}

func TestFlattenTranslations_KeyCountExceeded(t *testing.T) {
	t.Parallel()

	data := make(map[string]any, 6)
	for i := range 6 {
		data[fmt.Sprintf("k%d", i)] = "v"
	}
	result := make(map[string]string)

	err := FlattenTranslationsWithOptions(data, "", result, FlattenOptions{MaxKeyCount: 3})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTranslationKeyCountExceeded)
}

func TestParseAndFlatten_RejectsDeeplyNested(t *testing.T) {
	t.Parallel()

	var builder strings.Builder
	depth := DefaultFlattenMaxDepth + 1
	for range depth {
		builder.WriteString(`{"k":`)
	}
	builder.WriteString(`"value"`)
	for range depth {
		builder.WriteByte('}')
	}

	result, err := ParseAndFlatten([]byte(builder.String()))
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTranslationDepthExceeded)
	assert.Nil(t, result)
}

func TestParseAndFlatten_UTF8Content(t *testing.T) {
	jsonData := []byte(`{
		"french": "Bonjour le monde",
		"german": "Hallo Welt",
		"japanese": "Hello World",
		"russian": "Hello, World"
	}`)

	result, err := ParseAndFlatten(jsonData)

	require.NoError(t, err)
	assert.Len(t, result, 4)
	assert.Equal(t, "Bonjour le monde", result["french"])
	assert.Equal(t, "Hallo Welt", result["german"])
	assert.Equal(t, "Hello World", result["japanese"])
	assert.Equal(t, "Hello, World", result["russian"])
}
