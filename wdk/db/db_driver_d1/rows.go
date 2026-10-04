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
	"fmt"
	"io"
	"math"
	"strconv"
)

const (
	// maxInt64AsFloat is 2^63, the exclusive upper bound for float64 values that fit in
	// int64. A whole-valued float64 strictly below this and at or above math.MinInt64 can be
	// narrowed to int64 without an out-of-range result.
	maxInt64AsFloat = 9223372036854775808.0
)

var (
	_ driver.Rows = (*d1Rows)(nil)
)

// d1Rows implements driver.Rows over the ordered result set returned by the D1 raw query
// endpoint. Columns keep the server's order, including duplicate names, and each row is
// positionally aligned with them.
type d1Rows struct {
	// columns holds the column names in server order.
	columns []string

	// data holds the raw row arrays from the D1 API.
	data [][]any

	// index is the current cursor position within data.
	index int
}

// Columns returns the names of the columns in the result set.
//
// Returns []string which lists column names in the order the server returned them.
func (r *d1Rows) Columns() []string {
	return r.columns
}

// Close is a no-op since D1 rows are fully materialised in memory.
//
// Returns error which is always nil.
func (*d1Rows) Close() error {
	return nil
}

// Next advances to the next row and populates dest with the row's values, positionally
// matching Columns. A row shorter than the column list leaves the missing trailing values
// nil.
//
// Takes dest ([]driver.Value) which receives the column values for the current row.
//
// Returns error which is io.EOF when no more rows remain, an error when a value has a
// shape D1 does not produce for SQLite values, or nil on success.
func (r *d1Rows) Next(dest []driver.Value) error {
	if r.index >= len(r.data) {
		return io.EOF
	}

	row := r.data[r.index]
	r.index++

	for i := range dest {
		if i >= len(row) {
			dest[i] = nil
			continue
		}
		value, err := convertD1Value(row[i])
		if err != nil {
			return fmt.Errorf("db_driver_d1: column %d of row %d: %w", i, r.index, err)
		}
		dest[i] = value
	}

	return nil
}

// newRows returns rows over an ordered result set.
//
// Takes columns ([]string) which lists the column names in server order.
// Takes data ([][]any) which holds the row arrays aligned with columns.
//
// Returns *d1Rows which iterates over data.
func newRows(columns []string, data [][]any) *d1Rows {
	return &d1Rows{
		columns: columns,
		data:    data,
		index:   0,
	}
}

// convertD1Value converts a value from D1's JSON representation to a driver.Value.
//
// Takes value (any) which is the raw value from the D1 API response.
//
// Returns driver.Value which is the converted value suitable for database/sql.
// Returns error when the value has a shape D1 does not produce for SQLite values.
func convertD1Value(value any) (driver.Value, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case json.Number:
		return convertD1Number(v)
	case bool:
		return v, nil
	case string:
		return v, nil
	case []any:
		return convertD1Blob(v)
	default:
		return nil, fmt.Errorf("unsupported D1 value of type %T", value)
	}
}

// convertD1Number converts a JSON number to int64 when it is a whole value within the
// int64 range, and to float64 otherwise.
//
// Takes number (json.Number) which is the number's literal text.
//
// Returns driver.Value which is an int64 or a float64.
// Returns error when the literal is not a valid number.
func convertD1Number(number json.Number) (driver.Value, error) {
	if integer, err := strconv.ParseInt(number.String(), 10, 64); err == nil {
		return integer, nil
	}
	floatValue, err := strconv.ParseFloat(number.String(), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid D1 number %q: %w", number.String(), err)
	}
	if floatValue == math.Trunc(floatValue) && floatValue >= math.MinInt64 && floatValue < maxInt64AsFloat {
		return int64(floatValue), nil
	}
	return floatValue, nil
}

// convertD1Blob converts the byte-value array D1 uses for BLOB values into []byte.
//
// Takes elements ([]any) which holds one JSON number per byte.
//
// Returns driver.Value which is the blob's bytes.
// Returns error when an element is not an integer between 0 and 255.
func convertD1Blob(elements []any) (driver.Value, error) {
	blob := make([]byte, len(elements))
	for index, element := range elements {
		number, ok := element.(json.Number)
		if !ok {
			return nil, fmt.Errorf("BLOB element %d has type %T, expected a byte value", index, element)
		}
		byteValue, err := strconv.ParseUint(number.String(), 10, 8)
		if err != nil {
			return nil, fmt.Errorf("BLOB element %d: %w", index, err)
		}
		blob[index] = byte(byteValue)
	}
	return blob, nil
}
