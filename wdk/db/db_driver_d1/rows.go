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
	"fmt"
	"io"
	"math"
)

// Compile-time interface check.
var _ driver.Rows = (*d1Rows)(nil)

// d1Rows implements driver.Rows over the slice of map results returned by the
// D1 HTTP API. Column order is determined once at construction time and kept
// consistent across all rows.
type d1Rows struct {
	// columns holds the column names in sorted order.
	columns []string

	// data holds the raw result rows from the D1 API.
	data []map[string]any

	// index is the current cursor position within data.
	index int
}

// Columns returns the names of the columns in the result set.
//
// Returns []string which lists column names in sorted order.
func (r *d1Rows) Columns() []string {
	return r.columns
}

// Close is a no-op since D1 rows are fully materialised in memory.
//
// Returns error which is always nil.
func (*d1Rows) Close() error {
	return nil
}

// Next advances to the next row and populates dest with the row's values. The
// values are ordered to match Columns.
//
// D1 returns JSON, so the type mapping is:
//   - nil    -> nil
//   - float64 -> int64 if the value is a whole number, otherwise float64
//   - bool   -> bool
//   - string -> string
//   - other  -> fmt.Sprintf("%v", value)
//
// Takes dest ([]driver.Value) which receives the column values for the current
// row.
//
// Returns error which is io.EOF when no more rows remain, or nil on success.
func (r *d1Rows) Next(dest []driver.Value) error {
	if r.index >= len(r.data) {
		return io.EOF
	}

	row := r.data[r.index]
	r.index++

	for i, column := range r.columns {
		value, ok := row[column]
		if !ok {
			dest[i] = nil
			continue
		}
		dest[i] = convertD1Value(value)
	}

	return nil
}

// convertD1Value converts a value from D1's JSON representation to a
// driver.Value.
//
// Takes value (any) which is the raw value from the D1 API response.
//
// Returns driver.Value which is the converted value suitable for database/sql.
func convertD1Value(value any) driver.Value {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case float64:

		if v == math.Trunc(v) && !math.IsInf(v, 0) && !math.IsNaN(v) {
			return int64(v)
		}
		return v
	case bool:
		return v
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}
