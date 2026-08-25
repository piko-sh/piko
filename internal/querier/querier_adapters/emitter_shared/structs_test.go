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
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripIndexedQuestionMarkPlaceholdersBasic(t *testing.T) {
	input := "SELECT id FROM users WHERE id = ?1 AND name = ?2"
	expected := "SELECT id FROM users WHERE id = ? AND name = ?"

	result := stripIndexedQuestionMarkPlaceholders(input)
	assert.Equal(t, expected, result)
}

func TestStripIndexedQuestionMarkPlaceholdersPreservesStringLiterals(t *testing.T) {
	input := "SELECT '?1 inside literal' FROM users WHERE id = ?1"
	expected := "SELECT '?1 inside literal' FROM users WHERE id = ?"

	result := stripIndexedQuestionMarkPlaceholders(input)
	assert.Equal(t, expected, result)
}

func TestStripIndexedQuestionMarkPlaceholdersPreservesBlockComments(t *testing.T) {
	input := "SELECT id /* ?1 */ FROM users WHERE id = ?1"
	expected := "SELECT id /* ?1 */ FROM users WHERE id = ?"

	result := stripIndexedQuestionMarkPlaceholders(input)
	assert.Equal(t, expected, result)
}

func TestStripIndexedQuestionMarkPlaceholdersPreservesLineComments(t *testing.T) {
	input := "SELECT id -- ?1 in line comment\nFROM users WHERE id = ?1"
	expected := "SELECT id -- ?1 in line comment\nFROM users WHERE id = ?"

	result := stripIndexedQuestionMarkPlaceholders(input)
	assert.Equal(t, expected, result)
}

func TestSQLStringLiteralUsesRawStringForPlainSQL(t *testing.T) {
	literal := sqlStringLiteral("SELECT 1")
	assert.Equal(t, "`SELECT 1`", literal.Value)
}

func TestSQLStringLiteralQuotesWhenSQLContainsBacktick(t *testing.T) {

	literal := sqlStringLiteral("SELECT `col` FROM t")
	assert.Equal(t, `"SELECT `+"`col`"+` FROM t"`, literal.Value)
}

func TestSQLStringLiteralQuotesWhenSQLContainsCarriageReturn(t *testing.T) {

	literal := sqlStringLiteral("SELECT '\r' FROM t")
	assert.Equal(t, `"SELECT '\r' FROM t"`, literal.Value)
	assert.Contains(t, literal.Value, `\r`, "the carriage return must survive as an escape, not be dropped")
}

func TestJSONStructTagSanitisesColumnNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldName string
		expected  string
	}{
		{name: "plain name is untouched", fieldName: "user_id", expected: "`json:\"user_id\"`"},
		{name: "tolerated punctuation is untouched", fieldName: "COUNT(*)", expected: "`json:\"COUNT(*)\"`"},
		{name: "backtick cannot close the raw literal", fieldName: "we`ird", expected: "`json:\"we_ird\"`"},
		{name: "double quote cannot close the tag", fieldName: `we"ird`, expected: "`json:\"we_ird\"`"},
		{name: "comma cannot fake a tag option", fieldName: "id,omitempty", expected: "`json:\"id_omitempty\"`"},
		{name: "empty name falls back", fieldName: "", expected: "`json:\"field\"`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tag := jsonStructTag(tt.fieldName)

			require.NotNil(t, tag)
			assert.Equal(t, tt.expected, tag.Value)

			fset := token.NewFileSet()
			source := "package p\n\ntype T struct {\n\tField string " + tag.Value + "\n}\n"
			_, err := parser.ParseFile(fset, "structs.go", source, parser.AllErrors)
			require.NoError(t, err, "a struct carrying the tag must still parse")
		})
	}
}
