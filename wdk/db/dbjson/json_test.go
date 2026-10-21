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

package dbjson_test

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/db/dbjson"
)

func TestJSONScanAcceptsEveryDriverShape(t *testing.T) {
	t.Parallel()

	t.Run("nil clears the value", func(t *testing.T) {
		t.Parallel()
		value := dbjson.JSON(`{"stale":true}`)
		require.NoError(t, value.Scan(nil))
		assert.Nil(t, value, "a SQL NULL must scan into a nil JSON")
	})

	t.Run("bytes are copied (postgres, mysql)", func(t *testing.T) {
		t.Parallel()
		source := []byte(`{"k":"v"}`)
		var value dbjson.JSON
		require.NoError(t, value.Scan(source))
		assert.JSONEq(t, `{"k":"v"}`, string(value))
		source[0] = 'X'
		assert.JSONEq(t, `{"k":"v"}`, string(value), "Scan must copy, not alias, the driver buffer")
	})

	t.Run("string decodes (sqlite)", func(t *testing.T) {
		t.Parallel()
		var value dbjson.JSON
		require.NoError(t, value.Scan(`{"k":"v"}`))
		assert.JSONEq(t, `{"k":"v"}`, string(value))
	})

	t.Run("decoded map re-encodes (duckdb)", func(t *testing.T) {
		t.Parallel()
		var value dbjson.JSON
		require.NoError(t, value.Scan(map[string]any{"k": "v"}))
		assert.JSONEq(t, `{"k":"v"}`, string(value))
	})
}

func TestJSONValueWritesNullOrText(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		expected driver.Value
		name     string
		value    dbjson.JSON
	}{
		{name: "nil writes SQL NULL", value: nil, expected: nil},
		{name: "a document writes its text", value: dbjson.JSON(`{"k":"v"}`), expected: `{"k":"v"}`},
		{name: "an array writes its text", value: dbjson.JSON(`[1,2]`), expected: `[1,2]`},
		{name: "an empty non-nil value writes empty text", value: dbjson.JSON{}, expected: ""},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			written, err := testCase.value.Value()

			require.NoError(t, err)
			require.Equal(t, testCase.expected, written)
		})
	}
}

func TestScanIntoDecodesDriverShapes(t *testing.T) {
	t.Parallel()

	t.Run("nil leaves the zero value", func(t *testing.T) {
		t.Parallel()
		dest := []string{"stale"}
		require.NoError(t, dbjson.ScanInto(&dest).Scan(nil))
		assert.Nil(t, dest, "a NULL array column must scan to a nil slice")
	})

	t.Run("json bytes decode (postgres to_json)", func(t *testing.T) {
		t.Parallel()
		var dest []string
		require.NoError(t, dbjson.ScanInto(&dest).Scan([]byte(`["a","b"]`)))
		assert.Equal(t, []string{"a", "b"}, dest)
	})

	t.Run("json string decodes (sqlite)", func(t *testing.T) {
		t.Parallel()
		var dest []int64
		require.NoError(t, dbjson.ScanInto(&dest).Scan(`[1,2,3]`))
		assert.Equal(t, []int64{1, 2, 3}, dest)
	})

	t.Run("decoded slice re-encodes (duckdb)", func(t *testing.T) {
		t.Parallel()
		var dest []string
		require.NoError(t, dbjson.ScanInto(&dest).Scan([]any{"a", "b"}))
		assert.Equal(t, []string{"a", "b"}, dest)
	})
}

func TestJSONMarshalJSONMatchesRawMessage(t *testing.T) {
	t.Parallel()

	nullBytes, err := dbjson.JSON(nil).MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "null", string(nullBytes), "a nil JSON must encode as the JSON null literal")

	rawBytes, err := dbjson.JSON(`{"k":"v"}`).MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"k":"v"}`, string(rawBytes))
}

type scannedDocument struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func scanInto[T any](source string) error {
	var destination T
	return dbjson.ScanInto(&destination).Scan([]byte(source))
}

func TestScanIntoReadsANullElementOnlyIntoATypeThatHoldsIt(t *testing.T) {
	t.Parallel()

	var pointers []*int32
	require.NoError(t, dbjson.ScanInto(&pointers).Scan([]byte(`[1,null,3]`)))
	require.Len(t, pointers, 3)
	require.Nil(t, pointers[1])
	require.Equal(t, int32(3), *pointers[2])

	var nested [][]*string
	require.NoError(t, dbjson.ScanInto(&nested).Scan([]byte(`[["a",null],null]`)))
	require.Nil(t, nested[0][1])
	require.Nil(t, nested[1])

	var documents []dbjson.JSON
	require.NoError(t, dbjson.ScanInto(&documents).Scan([]byte(`[{"a":1},null]`)))
	require.Equal(t, []dbjson.JSON{dbjson.JSON(`{"a":1}`), dbjson.JSON(`null`)}, documents)

	var document scannedDocument
	require.NoError(t, dbjson.ScanInto(&document).Scan([]byte(`{"name":null,"tags":null}`)))

	testCases := []struct {
		read   func(source string) error
		name   string
		source string
		want   string
	}{
		{name: "an integer element", source: `[1,null]`, read: scanInto[[]int32], want: "element 1: converting NULL to int32 is unsupported"},
		{name: "a nested text element", source: `[["a"],["b",null]]`, read: scanInto[[][]string], want: "element 1: element 1: converting NULL to string is unsupported"},
		{name: "a map value", source: `{"k":null}`, read: scanInto[map[string]float64], want: "value of key k: converting NULL to float64 is unsupported"},
		{name: "the first map value in key order", source: `{"b":null,"a":null}`, read: scanInto[map[string]bool], want: "value of key a: converting NULL to bool is unsupported"},
		{name: "a fixed-length array element", source: `[null]`, read: scanInto[[1]int64], want: "element 0: converting NULL to int64 is unsupported"},
		{name: "an element of a slice behind a pointer", source: `[1,null]`, read: scanInto[*[]int32], want: "element 1: converting NULL to int32 is unsupported"},
		{name: "an element of a pointer element", source: `[[1,null]]`, read: scanInto[[]*[]int32], want: "element 0: element 1: converting NULL to int32 is unsupported"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.read(testCase.source)
			require.ErrorContains(t, err, "dbjson: decode JSON column: ")
			require.ErrorContains(t, err, testCase.want)
		})
	}
}

func TestScanIntoLeavesTheZeroValueWhenANullElementIsRefused(t *testing.T) {
	t.Parallel()

	destination := []int32{9}
	require.Error(t, dbjson.ScanInto(&destination).Scan([]byte(`[1,null]`)))
	require.Nil(t, destination)
}

func TestScanIntoLeavesTheZeroValueWhenTheColumnCannotBeDecoded(t *testing.T) {
	t.Parallel()

	destination := scannedDocument{Name: "stale", Tags: []string{"old"}}
	err := dbjson.ScanInto(&destination).Scan([]byte(`{"name":`))
	require.ErrorContains(t, err, "dbjson: decode JSON column")
	require.Zero(t, destination)
}

type textDecimal string

func (d *textDecimal) UnmarshalText(text []byte) error {
	*d = textDecimal(text)
	return nil
}

func TestScanIntoReadsTheElementsAnEngineWritesInItsOwnForm(t *testing.T) {
	t.Parallel()

	moment := time.Date(2024, 1, 2, 3, 4, 5, 123456000, time.UTC)
	day := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	var timestamps []time.Time
	require.NoError(t, dbjson.ScanInto(&timestamps).Scan([]byte(`["2024-01-02T03:04:05.123456","2024-01-02T00:00:00"]`)))
	require.Equal(t, []time.Time{moment, day}, timestamps)

	var dates []*time.Time
	require.NoError(t, dbjson.ScanInto(&dates).Scan([]byte(`["2024-01-02",null]`)))
	require.Equal(t, []*time.Time{&day, nil}, dates)

	var zoned []time.Time
	require.NoError(t, dbjson.ScanInto(&zoned).Scan([]byte(`["2024-01-02T03:04:05+02:00"]`)))
	require.True(t, zoned[0].Equal(time.Date(2024, 1, 2, 1, 4, 5, 0, time.UTC)))

	var grid [][]time.Time
	require.NoError(t, dbjson.ScanInto(&grid).Scan([]byte(`[["2024-01-02"]]`)))
	require.Equal(t, [][]time.Time{{day}}, grid)

	var keyed map[string]time.Time
	require.NoError(t, dbjson.ScanInto(&keyed).Scan([]byte(`{"start":"2024-01-02"}`)))
	require.Equal(t, map[string]time.Time{"start": day}, keyed)

	var amounts []textDecimal
	require.NoError(t, dbjson.ScanInto(&amounts).Scan([]byte(`[1.50,2,-0.001]`)))
	require.Equal(t, []textDecimal{"1.50", "2", "-0.001"}, amounts)

	var texts []*string
	require.NoError(t, dbjson.ScanInto(&texts).Scan([]byte(`[1.50,null]`)))
	require.Equal(t, "1.50", *texts[0])
	require.Nil(t, texts[1])

	var optional []*textDecimal
	require.NoError(t, dbjson.ScanInto(&optional).Scan([]byte(`[null,10.00]`)))
	require.Nil(t, optional[0])
	require.Equal(t, textDecimal("10.00"), *optional[1])

	var wide []*textDecimal
	require.NoError(t, dbjson.ScanInto(&wide).Scan([]byte(`[1e400,null]`)))
	require.Equal(t, textDecimal("1e400"), *wide[0])
	require.Nil(t, wide[1])

	var blobs [][]byte
	require.NoError(t, dbjson.ScanInto(&blobs).Scan([]byte(`["\\x0102ff","\\x"]`)))
	require.Equal(t, [][]byte{{0x01, 0x02, 0xff}, {}}, blobs)

	var documents []dbjson.JSON
	require.NoError(t, dbjson.ScanInto(&documents).Scan([]byte(`[{"a":1},"\\x01"]`)))
	require.Equal(t, []dbjson.JSON{dbjson.JSON(`{"a":1}`), dbjson.JSON(`"\\x01"`)}, documents)
}

func TestScanIntoStillReportsAnElementItCannotRead(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		read   func(source string) error
		name   string
		source string
	}{
		{name: "text that is not a timestamp", source: `["yesterday"]`, read: scanInto[[]time.Time]},
		{name: "hex digits that do not decode", source: `["\\xzz"]`, read: scanInto[[][]byte]},
		{name: "a number read into an integer from text", source: `["12"]`, read: scanInto[[]int64]},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorContains(t, testCase.read(testCase.source), "dbjson: decode JSON column")
		})
	}
}

func TestScanIntoRefusesDataAfterTheDocument(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		read   func(source string) error
		name   string
		source string
	}{
		{name: "after a document read as written", source: `[1] [2]`, read: scanInto[[]int64]},
		{name: "after a document an engine wrote in its own form", source: `["2024-01-02"] x`, read: scanInto[[]time.Time]},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.ErrorContains(t, testCase.read(testCase.source), "dbjson: decode JSON column")
		})
	}
}
