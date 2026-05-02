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
	"testing"

	"github.com/stretchr/testify/assert"
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
