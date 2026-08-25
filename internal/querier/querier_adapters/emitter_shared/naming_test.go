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

package emitter_shared

import (
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko/internal/goastutil"
)

func TestSnakeToPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"id", "ID"},
		{"user_id", "UserID"},
		{"name", "Name"},
		{"first_name", "FirstName"},
		{"created_at", "CreatedAt"},
		{"user_api_key", "UserAPIKey"},
		{"http_url", "HTTPURL"},
		{"json_data", "JSONData"},
		{"html_content", "HTMLContent"},
		{"ip_address", "IPAddress"},
		{"uuid", "UUID"},
		{"user_uuid", "UserUUID"},
		{"a_b_c", "ABC"},
		{"", ""},
		{"single", "Single"},
		{"ALREADY_UPPER", "AlreadyUpper"},
		{"user_ids", "UserIDs"},
		{"jobCount", "JobCount"},
		{"job_countValue", "JobCountValue"},

		{"2fa_enabled", "X2faEnabled"},
		{"123", "X123"},
		{"4ever", "X4ever"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := SnakeToPascalCase(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSnakeToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"id", "id"},
		{"user_id", "userID"},
		{"get_user", "getUser"},
		{"get_user_by_id", "getUserByID"},
		{"list_users", "listUsers"},
		{"name", "name"},
		{"created_at", "createdAt"},
		{"", ""},
		{"single", "single"},
		{"get_http_url", "getHTTPURL"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := SnakeToCamelCase(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSnakeToCamelCaseSanitisesLeadingDigit(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2fa_enabled", "_2faEnabled"},
		{"123", "_123"},
		{"4ever", "_4ever"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assert.Equal(t, test.expected, SnakeToCamelCase(test.input))
		})
	}
}

func TestDisambiguateGoFieldNamesSuffixesCollisions(t *testing.T) {

	names := []string{"foo_bar", "foo__bar", "foo_bar"}
	result := DisambiguateGoFieldNames(names)
	assert.Equal(t, []string{"FooBar", "FooBar2", "FooBar3"}, result)
}

func TestDisambiguateGoFieldNamesSuffixAvoidsLiteralCollision(t *testing.T) {

	names := []string{"foo", "foo", "foo2"}
	result := DisambiguateGoFieldNames(names)
	assert.Equal(t, []string{"Foo", "Foo2", "Foo22"}, result)
}

func TestDisambiguateGoFieldNamesSuffixAvoidsEarlierConvertedName(t *testing.T) {

	names := []string{"foo", "foo_2", "foo"}
	result := DisambiguateGoFieldNames(names)
	assert.Equal(t, []string{"Foo", "Foo2", "Foo3"}, result)
}

func TestDisambiguateGoFieldNamesCamelCase(t *testing.T) {
	names := []string{"foo_bar", "foo__bar"}
	result := DisambiguateGoFieldNamesCamelCase(names)
	assert.Equal(t, []string{"fooBar", "fooBar2"}, result)
}

func TestDisambiguateGoFieldNamesLeavesUniqueNamesUntouched(t *testing.T) {
	names := []string{"id", "user_id", "name"}
	result := DisambiguateGoFieldNames(names)
	assert.Equal(t, []string{"ID", "UserID", "Name"}, result)
}

func TestDisambiguateGoFieldNamesPrefixesLeadingDigitThenDisambiguates(t *testing.T) {

	names := []string{"2fa", "2fa"}
	result := DisambiguateGoFieldNames(names)
	assert.Equal(t, []string{"X2fa", "X2fa2"}, result)
}

func TestSnakeToPascalCaseSplitsOnEveryNonIdentifierRune(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-query", "MyQuery"},
		{"user name", "UserName"},
		{"user.name", "UserName"},
		{"COUNT(*)", "Count"},
		{"a--b", "AB"},
		{"total_$", "Total"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assert.Equal(t, test.expected, SnakeToPascalCase(test.input))
		})
	}
}

func TestSnakeToCamelCaseGuardsReservedWords(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"range", "range_"},
		{"select", "select_"},
		{"type", "type_"},
		{"func", "func_"},
		{"string", "string_"},
		{"len", "len_"},
		{"my-query", "myQuery"},
		{"user-id", "userID"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assert.Equal(t, test.expected, SnakeToCamelCase(test.input))
		})
	}
}

func TestSnakeToPascalCaseNeedsNoReservedWordGuard(t *testing.T) {

	assert.Equal(t, "Range", SnakeToPascalCase("range"))
	assert.Equal(t, "String", SnakeToPascalCase("string"))
}

func TestCaseConversionFallsBackWhenNothingSurvivesTheSplit(t *testing.T) {

	tests := []struct {
		input         string
		expectedPasc  string
		expectedCamel string
	}{
		{"$", "Piko", "piko"},
		{"***", "Piko", "___"},
		{"--", "Piko", "__"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			assert.Equal(t, test.expectedPasc, SnakeToPascalCase(test.input))
			assert.Equal(t, test.expectedCamel, SnakeToCamelCase(test.input))
		})
	}
}

var (
	hostileColumnNames = []string{
		"", "_", "__", "-", "--", " ", "   ", ".", "2fa", "2fa_enabled", "123", "4ever",
		"my-query", "my.query", "my query", "my/query", "type", "range", "error", "string",
		"iota", "class", "await", "café", "日本語", "עברית", "ไทย", "$dollar", "😀",
		"a\nb", "select *", "user__name", "COUNT(*)", `he said "hi"`, "a,b", "col`name",
	}
)

func TestSnakeToPascalCaseAlwaysProducesAnExportedIdentifier(t *testing.T) {
	for _, input := range hostileColumnNames {
		result := SnakeToPascalCase(input)
		if input == "" {
			assert.Empty(t, result)
			continue
		}

		assert.True(t, goastutil.IsValidGoIdentifier(result),
			"%q became invalid Go identifier %q", input, result)
		assert.True(t, token.IsExported(result),
			"%q became unexported field name %q, which encoding/json would drop", input, result)
	}
}

func TestSnakeToCamelCaseAlwaysProducesAUsableIdentifier(t *testing.T) {
	for _, input := range hostileColumnNames {
		result := SnakeToCamelCase(input)
		if input == "" {
			assert.Empty(t, result)
			continue
		}

		assert.True(t, goastutil.IsValidGoIdentifier(result),
			"%q became invalid Go identifier %q", input, result)
		assert.False(t, goastutil.IsGoPredeclared(result),
			"%q became predeclared identifier %q", input, result)
	}
}
