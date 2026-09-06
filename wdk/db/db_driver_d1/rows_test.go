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

package db_driver_d1

import (
	"database/sql/driver"
	"encoding/json"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertD1Value(t *testing.T) {
	testCases := []struct {
		name          string
		input         any
		expected      driver.Value
		expectedError string
	}{
		{name: "null", input: nil, expected: nil},
		{name: "small integer", input: json.Number("42"), expected: int64(42)},
		{name: "negative integer", input: json.Number("-7"), expected: int64(-7)},
		{name: "integer beyond float64 precision", input: json.Number("9007199254740993"), expected: int64(9007199254740993)},
		{name: "maximum int64", input: json.Number("9223372036854775807"), expected: int64(9223372036854775807)},
		{name: "integer beyond int64", input: json.Number("9223372036854775808"), expected: float64(9223372036854775808)},
		{name: "fraction", input: json.Number("3.5"), expected: 3.5},
		{name: "whole exponent form", input: json.Number("1e3"), expected: int64(1000)},
		{name: "whole decimal form", input: json.Number("2.0"), expected: int64(2)},
		{name: "number out of float range", input: json.Number("1e400"), expectedError: "invalid D1 number"},
		{name: "boolean", input: true, expected: true},
		{name: "text", input: "hello", expected: "hello"},
		{name: "blob", input: []any{json.Number("0"), json.Number("127"), json.Number("255")}, expected: []byte{0, 127, 255}},
		{name: "empty blob", input: []any{}, expected: []byte{}},
		{name: "blob element out of byte range", input: []any{json.Number("256")}, expectedError: "BLOB element 0"},
		{name: "blob element not a number", input: []any{json.Number("1"), "x"}, expectedError: "BLOB element 1 has type string"},
		{name: "object", input: map[string]any{"a": json.Number("1")}, expectedError: "unsupported D1 value"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			value, err := convertD1Value(testCase.input)
			if testCase.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.expectedError)
				assert.Nil(t, value)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, value)
		})
	}
}

func TestRowsNext(t *testing.T) {
	rows := newRows([]string{"id", "data"}, [][]any{
		{json.Number("1"), []any{json.Number("104"), json.Number("105")}},
		{json.Number("2")},
		{json.Number("3"), map[string]any{}},
	})

	dest := make([]driver.Value, 2)
	require.NoError(t, rows.Next(dest))
	assert.Equal(t, []driver.Value{int64(1), []byte("hi")}, dest)

	require.NoError(t, rows.Next(dest))
	assert.Equal(t, []driver.Value{int64(2), nil}, dest)

	err := rows.Next(dest)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "column 1 of row 3")

	assert.ErrorIs(t, rows.Next(dest), io.EOF)
	assert.NoError(t, rows.Close())
}
