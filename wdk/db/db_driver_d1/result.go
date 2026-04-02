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

import "database/sql/driver"

// Compile-time interface check.
var _ driver.Result = (*d1Result)(nil)

// d1Result implements driver.Result, holding the metadata returned by a D1
// exec-style query.
type d1Result struct {
	// lastInsertID is the rowid of the last inserted row.
	lastInsertID int64

	// rowsAffected is the number of rows changed by the statement.
	rowsAffected int64
}

// LastInsertId returns the rowid of the last inserted row.
//
// Returns int64 which is the last insert rowid.
// Returns error which is always nil.
func (r *d1Result) LastInsertId() (int64, error) {
	return r.lastInsertID, nil
}

// RowsAffected returns the number of rows changed by the statement.
//
// Returns int64 which is the count of affected rows.
// Returns error which is always nil.
func (r *d1Result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}
