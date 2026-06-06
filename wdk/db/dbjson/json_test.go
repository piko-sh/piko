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
	"testing"

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

func TestJSONValueRoundTripsNullAndBytes(t *testing.T) {
	t.Parallel()

	nilValue, err := dbjson.JSON(nil).Value()
	require.NoError(t, err)
	assert.Nil(t, nilValue, "a nil JSON must write SQL NULL")

	bytesValue, err := dbjson.JSON(`{"k":"v"}`).Value()
	require.NoError(t, err)
	assert.Equal(t, []byte(`{"k":"v"}`), bytesValue)
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
