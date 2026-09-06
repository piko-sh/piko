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
	"errors"
)

var (
	_ driver.Result = (*d1Result)(nil)
)

var (
	// ErrResultUnavailableInTransaction reports that the last-insert ID or rows-affected
	// count of a statement executed inside a transaction was requested.
	//
	// D1 transactions are batched and sent on Commit, so the counters do not exist when the
	// statement is queued. Callers can match it with errors.Is.
	ErrResultUnavailableInTransaction = errors.New(
		"db_driver_d1: result counters are unavailable for statements batched in a transaction",
	)
)

// d1Result implements driver.Result, holding the metadata returned by a D1 exec-style
// query.
type d1Result struct {
	// lastInsertID is the rowid of the last inserted row.
	lastInsertID int64

	// rowsAffected is the number of rows changed by the statement.
	rowsAffected int64

	// deferred is true for a statement queued in a transaction batch, whose counters are not
	// known until Commit.
	deferred bool
}

// LastInsertId returns the rowid of the last inserted row.
//
// Returns int64 which is the last insert rowid.
// Returns error which is ErrResultUnavailableInTransaction for a statement batched in a
// transaction, nil otherwise.
func (r *d1Result) LastInsertId() (int64, error) {
	if r.deferred {
		return 0, ErrResultUnavailableInTransaction
	}
	return r.lastInsertID, nil
}

// RowsAffected returns the number of rows changed by the statement.
//
// Returns int64 which is the count of affected rows.
// Returns error which is ErrResultUnavailableInTransaction for a statement batched in a
// transaction, nil otherwise.
func (r *d1Result) RowsAffected() (int64, error) {
	if r.deferred {
		return 0, ErrResultUnavailableInTransaction
	}
	return r.rowsAffected, nil
}

// newResult returns the result of a statement executed immediately.
//
// Takes lastInsertID (int64) which is the rowid of the last inserted row.
// Takes rowsAffected (int64) which is the number of rows changed.
//
// Returns *d1Result which reports the counters.
func newResult(lastInsertID int64, rowsAffected int64) *d1Result {
	return &d1Result{
		lastInsertID: lastInsertID,
		rowsAffected: rowsAffected,
		deferred:     false,
	}
}

// newDeferredResult returns the result of a statement queued in a transaction batch.
//
// Returns *d1Result which reports ErrResultUnavailableInTransaction for both counters.
func newDeferredResult() *d1Result {
	return &d1Result{
		lastInsertID: 0,
		rowsAffected: 0,
		deferred:     true,
	}
}
