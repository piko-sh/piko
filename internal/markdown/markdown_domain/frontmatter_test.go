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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/markdown/markdown_dto"
)

func TestParseFrontmatter_RequiredFields(t *testing.T) {
	t.Run("ValidMinimalFrontmatter", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test Post",
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.NotNil(t, fm)
		assert.Equal(t, "Test Post", fm.Title)
		assert.Equal(t, "", fm.Description)
		assert.False(t, fm.Draft)
		assert.Empty(t, fm.Tags)
		assert.True(t, fm.PublishDate.IsZero())
		assert.NotNil(t, fm.Custom)
		assert.Empty(t, fm.Custom)
	})

	t.Run("MissingTitle", func(t *testing.T) {
		rawData := map[string]any{}

		fm, err := ParseFrontmatter(rawData)
		assert.Error(t, err)
		assert.Nil(t, fm)
		assert.Contains(t, err.Error(), "title")
		assert.Contains(t, err.Error(), "required")
	})

	t.Run("EmptyTitleString", func(t *testing.T) {
		rawData := map[string]any{
			"title": "",
		}

		fm, err := ParseFrontmatter(rawData)
		assert.Error(t, err)
		assert.Nil(t, fm)
		assert.Contains(t, err.Error(), "title")
	})

	t.Run("BothRequiredFieldsMissing", func(t *testing.T) {
		rawData := map[string]any{
			"description": "Some description",
		}

		fm, err := ParseFrontmatter(rawData)
		assert.Error(t, err)
		assert.Nil(t, fm)
	})
}

func TestParseFrontmatter_OptionalFields(t *testing.T) {
	t.Run("AllOptionalFieldsPresent", func(t *testing.T) {
		publishDate := time.Date(2023, 10, 15, 12, 30, 0, 0, time.UTC)
		rawData := map[string]any{
			"title":       "Complete Post",
			"description": "A comprehensive test",
			"draft":       true,
			"date":        publishDate,
			"tags":        []string{"go", "testing"},
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.NotNil(t, fm)
		assert.Equal(t, "Complete Post", fm.Title)
		assert.Equal(t, "A comprehensive test", fm.Description)
		assert.True(t, fm.Draft)
		assert.Equal(t, publishDate, fm.PublishDate)
		assert.Equal(t, []string{"go", "testing"}, fm.Tags)
	})

	t.Run("DescriptionField", func(t *testing.T) {
		rawData := map[string]any{
			"title":       "Test",
			"description": "This is a description",
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.Equal(t, "This is a description", fm.Description)
	})

	t.Run("DraftFieldTrue", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Draft Post",
			"draft": true,
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.True(t, fm.Draft)
	})

	t.Run("DraftFieldFalse", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Published Post",
			"draft": false,
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.False(t, fm.Draft)
	})
}

func TestParseFrontmatter_CustomFields(t *testing.T) {
	t.Run("SingleCustomField", func(t *testing.T) {
		rawData := map[string]any{
			"title":        "Test",
			"custom_field": "custom_value",
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.NotNil(t, fm.Custom)
		assert.Contains(t, fm.Custom, "custom_field")
		assert.Equal(t, "custom_value", fm.Custom["custom_field"])
	})

	t.Run("MultipleCustomFields", func(t *testing.T) {
		rawData := map[string]any{
			"title":    "Test",
			"author":   "John Doe",
			"category": "Technology",
			"views":    1234,
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.Len(t, fm.Custom, 3)
		assert.Equal(t, "John Doe", fm.Custom["author"])
		assert.Equal(t, "Technology", fm.Custom["category"])
		assert.Equal(t, 1234, fm.Custom["views"])
	})

	t.Run("CustomFieldsPreserveCase", func(t *testing.T) {
		rawData := map[string]any{
			"title":       "Test",
			"CustomField": "value",
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.Contains(t, fm.Custom, "CustomField", "Custom field keys should preserve case")
		assert.Equal(t, "value", fm.Custom["CustomField"])
	})

	t.Run("MixedKnownAndCustomFields", func(t *testing.T) {
		rawData := map[string]any{
			"title":        "Test",
			"description":  "Known field",
			"author":       "Custom field",
			"custom_value": 42,
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.Equal(t, "Known field", fm.Description)
		assert.Len(t, fm.Custom, 2)
		assert.Equal(t, "Custom field", fm.Custom["author"])
		assert.Equal(t, 42, fm.Custom["custom_value"])
	})
}

func TestParseFrontmatter_CaseInsensitiveKnownKeys(t *testing.T) {
	t.Run("MixedCaseTitle", func(t *testing.T) {
		rawData := map[string]any{
			"TiTlE": "Test Post",
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.Equal(t, "Test Post", fm.Title)
	})

	t.Run("UppercaseDraft", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test",
			"DRAFT": true,
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.True(t, fm.Draft)
	})
}

func TestGetDate(t *testing.T) {
	t.Run("TimeObject", func(t *testing.T) {
		expected := time.Date(2023, 10, 15, 12, 30, 0, 0, time.UTC)
		result := getDate(expected)
		assert.Equal(t, expected, result)
	})

	t.Run("RFC3339String", func(t *testing.T) {
		dateString := "2023-10-15T12:30:00Z"
		result := getDate(dateString)
		assert.Equal(t, 2023, result.Year())
		assert.Equal(t, time.October, result.Month())
		assert.Equal(t, 15, result.Day())
		assert.Equal(t, 12, result.Hour())
		assert.Equal(t, 30, result.Minute())
	})

	t.Run("DateTimeWithoutZone", func(t *testing.T) {
		dateString := "2023-10-15T12:30:00"
		result := getDate(dateString)
		assert.Equal(t, 2023, result.Year())
		assert.Equal(t, time.October, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("DateOnly", func(t *testing.T) {
		dateString := "2023-10-15"
		result := getDate(dateString)
		assert.Equal(t, 2023, result.Year())
		assert.Equal(t, time.October, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("InvalidDateString", func(t *testing.T) {
		result := getDate("not-a-date")
		assert.True(t, result.IsZero(), "Invalid date string should return zero time")
	})

	t.Run("NonStringNonTime", func(t *testing.T) {
		result := getDate(12345)
		assert.True(t, result.IsZero())
	})

	t.Run("NilValue", func(t *testing.T) {
		result := getDate(nil)
		assert.True(t, result.IsZero())
	})
}

func TestGetStringSlice(t *testing.T) {
	t.Run("StringSlice", func(t *testing.T) {
		input := []string{"tag1", "tag2", "tag3"}
		result := getStringSlice(input)
		assert.Equal(t, input, result)
	})

	t.Run("AnySliceAllStrings", func(t *testing.T) {
		input := []any{"tag1", "tag2", "tag3"}
		result := getStringSlice(input)
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, result)
	})

	t.Run("AnySliceMixedTypes", func(t *testing.T) {
		input := []any{"tag1", 123, "tag2", true, "tag3"}
		result := getStringSlice(input)
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, result, "Should filter out non-string values")
	})

	t.Run("CommaSeparatedString", func(t *testing.T) {
		input := "tag1,tag2,tag3"
		result := getStringSlice(input)
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, result)
	})

	t.Run("CommaSeparatedStringWithWhitespace", func(t *testing.T) {
		input := "tag1, tag2 , tag3"
		result := getStringSlice(input)
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, result, "Should trim whitespace")
	})

	t.Run("CommaSeparatedStringWithEmptyValues", func(t *testing.T) {
		input := "tag1,,tag2, ,tag3"
		result := getStringSlice(input)
		assert.Equal(t, []string{"tag1", "tag2", "tag3"}, result, "Should skip empty values")
	})

	t.Run("SingleStringNoComma", func(t *testing.T) {
		input := "single-tag"
		result := getStringSlice(input)
		assert.Equal(t, []string{"single-tag"}, result)
	})

	t.Run("EmptyString", func(t *testing.T) {
		input := ""
		result := getStringSlice(input)
		assert.Nil(t, result, "Empty string should return nil")
	})

	t.Run("NilValue", func(t *testing.T) {
		result := getStringSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("UnsupportedType", func(t *testing.T) {
		result := getStringSlice(12345)
		assert.Nil(t, result)
	})

	t.Run("EmptySlice", func(t *testing.T) {
		input := []string{}
		result := getStringSlice(input)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})
}

func TestGetString(t *testing.T) {
	t.Run("ValidString", func(t *testing.T) {
		result := getString("test value")
		assert.Equal(t, "test value", result)
	})

	t.Run("NonString", func(t *testing.T) {
		result := getString(12345)
		assert.Equal(t, "", result)
	})

	t.Run("NilValue", func(t *testing.T) {
		result := getString(nil)
		assert.Equal(t, "", result)
	})

	t.Run("BoolValue", func(t *testing.T) {
		result := getString(true)
		assert.Equal(t, "", result)
	})
}

func TestGetBool(t *testing.T) {
	t.Run("TrueValue", func(t *testing.T) {
		result := getBool(true)
		assert.True(t, result)
	})

	t.Run("FalseValue", func(t *testing.T) {
		result := getBool(false)
		assert.False(t, result)
	})

	t.Run("NonBool", func(t *testing.T) {
		result := getBool("true")
		assert.False(t, result, "Non-bool should return false")
	})

	t.Run("NilValue", func(t *testing.T) {
		result := getBool(nil)
		assert.False(t, result)
	})

	t.Run("IntValue", func(t *testing.T) {
		result := getBool(1)
		assert.False(t, result)
	})
}

func TestParseFrontmatter_EmptyInput(t *testing.T) {
	t.Run("EmptyMap", func(t *testing.T) {
		rawData := map[string]any{}

		fm, err := ParseFrontmatter(rawData)
		assert.Error(t, err, "Empty map should fail validation")
		assert.Nil(t, fm)
	})

	t.Run("NilMap", func(t *testing.T) {
		var rawData map[string]any

		fm, err := ParseFrontmatter(rawData)
		assert.Error(t, err)
		assert.Nil(t, fm)
	})
}

func TestParseFrontmatter_Navigation(t *testing.T) {
	t.Run("CompleteNavigationMetadata", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test",
			"nav": map[string]any{
				"sidebar": map[string]any{
					"section":    "get-started",
					"subsection": "basics",
					"order":      10,
					"icon":       "download",
					"hidden":     false,
					"parent":     "",
					"label":      "Install",
				},
			},
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		require.NotNil(t, fm.Navigation)
		require.NotNil(t, fm.Navigation.Groups)
		require.Contains(t, fm.Navigation.Groups, "sidebar")

		sidebar := fm.Navigation.Groups["sidebar"]
		assert.Equal(t, "get-started", sidebar.Section)
		assert.Equal(t, "basics", sidebar.Subsection)
		assert.Equal(t, 10, sidebar.Order)
		assert.Equal(t, "download", sidebar.Icon)
		assert.False(t, sidebar.Hidden)
		assert.Equal(t, "", sidebar.Parent)
		assert.Equal(t, "Install", sidebar.Label)
	})

	t.Run("MultipleNavigationGroups", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test",
			"nav": map[string]any{
				"sidebar": map[string]any{
					"section": "api",
					"order":   5,
				},
				"footer": map[string]any{
					"section": "quick-links",
					"order":   1,
				},
			},
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		require.NotNil(t, fm.Navigation)
		assert.Len(t, fm.Navigation.Groups, 2)
		assert.Contains(t, fm.Navigation.Groups, "sidebar")
		assert.Contains(t, fm.Navigation.Groups, "footer")

		assert.Equal(t, "api", fm.Navigation.Groups["sidebar"].Section)
		assert.Equal(t, "quick-links", fm.Navigation.Groups["footer"].Section)
	})

	t.Run("MinimalNavigationMetadata", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test",
			"nav": map[string]any{
				"sidebar": map[string]any{
					"section": "guides",
				},
			},
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		require.NotNil(t, fm.Navigation)

		sidebar := fm.Navigation.Groups["sidebar"]
		assert.Equal(t, "guides", sidebar.Section)
		assert.Equal(t, "", sidebar.Subsection)
		assert.Equal(t, 0, sidebar.Order)
		assert.Equal(t, "", sidebar.Icon)
		assert.False(t, sidebar.Hidden)
	})

	t.Run("NavigationAlias", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test",
			"navigation": map[string]any{
				"sidebar": map[string]any{
					"section": "guides",
					"order":   5,
				},
			},
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		require.NotNil(t, fm.Navigation)
		assert.Contains(t, fm.Navigation.Groups, "sidebar")
		assert.Equal(t, "guides", fm.Navigation.Groups["sidebar"].Section)
	})

	t.Run("NoNavigationMetadata", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test",
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.Nil(t, fm.Navigation)
	})

	t.Run("EmptyNavigationObject", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test",
			"nav":   map[string]any{},
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.Nil(t, fm.Navigation, "Empty navigation should return nil")
	})

	t.Run("EmptyGroupObjects", func(t *testing.T) {
		rawData := map[string]any{
			"title": "Test",
			"nav": map[string]any{
				"sidebar": map[string]any{},
				"footer":  map[string]any{},
			},
		}

		fm, err := ParseFrontmatter(rawData)
		require.NoError(t, err)
		assert.Nil(t, fm.Navigation, "All empty groups should return nil")
	})
}

func TestGetInt(t *testing.T) {
	t.Run("IntValue", func(t *testing.T) {
		result := getInt(42)
		assert.Equal(t, 42, result)
	})

	t.Run("Int64Value", func(t *testing.T) {
		result := getInt(int64(123))
		assert.Equal(t, 123, result)
	})

	t.Run("Float64Value", func(t *testing.T) {
		result := getInt(float64(99.7))
		assert.Equal(t, 99, result)
	})

	t.Run("StringInteger", func(t *testing.T) {
		result := getInt("42")
		assert.Equal(t, 42, result, "Should parse string integers")
	})

	t.Run("StringNonInteger", func(t *testing.T) {
		result := getInt("not-a-number")
		assert.Equal(t, 0, result)
	})

	t.Run("NilValue", func(t *testing.T) {
		result := getInt(nil)
		assert.Equal(t, 0, result)
	})

	t.Run("BoolValue", func(t *testing.T) {
		result := getInt(true)
		assert.Equal(t, 0, result)
	})

	t.Run("NegativeInt", func(t *testing.T) {
		result := getInt(-10)
		assert.Equal(t, -10, result)
	})

	t.Run("ZeroValue", func(t *testing.T) {
		result := getInt(0)
		assert.Equal(t, 0, result)
	})
}

func TestParseNavigationMetadata(t *testing.T) {
	t.Run("SingleGroup", func(t *testing.T) {
		input := map[string]any{
			"sidebar": map[string]any{
				"section": "guides",
				"order":   10,
			},
		}

		nav := parseNavigationMetadata(input)
		require.NotNil(t, nav)
		assert.Len(t, nav.Groups, 1)
		assert.Equal(t, "guides", nav.Groups["sidebar"].Section)
		assert.Equal(t, 10, nav.Groups["sidebar"].Order)
	})

	t.Run("MultipleGroups", func(t *testing.T) {
		input := map[string]any{
			"sidebar": map[string]any{
				"section": "api",
			},
			"footer": map[string]any{
				"section": "links",
			},
			"breadcrumb": map[string]any{
				"section": "docs",
			},
		}

		nav := parseNavigationMetadata(input)
		require.NotNil(t, nav)
		assert.Len(t, nav.Groups, 3)
		assert.Contains(t, nav.Groups, "sidebar")
		assert.Contains(t, nav.Groups, "footer")
		assert.Contains(t, nav.Groups, "breadcrumb")
	})

	t.Run("InvalidGroupData", func(t *testing.T) {
		input := map[string]any{
			"sidebar": "not-a-map",
		}

		nav := parseNavigationMetadata(input)
		assert.Nil(t, nav, "Invalid group data should result in nil")
	})

	t.Run("NilInput", func(t *testing.T) {
		nav := parseNavigationMetadata(nil)
		assert.Nil(t, nav)
	})

	t.Run("NonMapInput", func(t *testing.T) {
		nav := parseNavigationMetadata("not-a-map")
		assert.Nil(t, nav)
	})

	t.Run("AlreadyNavigationMetadata", func(t *testing.T) {
		existing := &markdown_dto.NavigationMetadata{
			Groups: map[string]*markdown_dto.NavGroupMetadata{
				"sidebar": {Section: "test"},
			},
		}

		nav := parseNavigationMetadata(existing)
		assert.Equal(t, existing, nav)
	})
}
