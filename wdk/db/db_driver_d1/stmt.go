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
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
)

var (
	_ driver.Stmt = (*d1Stmt)(nil)

	_ driver.StmtExecContext = (*d1Stmt)(nil)

	_ driver.StmtQueryContext = (*d1Stmt)(nil)
)

// d1Stmt implements driver.Stmt, driver.StmtExecContext, and driver.StmtQueryContext.
// Each execution issues an HTTP request to the D1 API unless a transaction is active, in
// which case the statement is queued for batch execution on Commit.
type d1Stmt struct {
	// conn is the parent connection that owns this statement.
	conn *d1Conn

	// query is the SQL statement text.
	query string
}

// Close is a no-op since D1 statements hold no server-side resources.
//
// Returns error which is always nil.
func (*d1Stmt) Close() error {
	return nil
}

// NumInput returns -1 to indicate that the driver does not know the number of
// placeholders. The database/sql package will not validate argument counts.
//
// Returns int which is always -1.
func (*d1Stmt) NumInput() int {
	return -1
}

// Exec executes the statement with the given arguments. It delegates to ExecContext; the
// call is bounded by the configured request timeout.
//
// Takes args ([]driver.Value) which are the positional parameters.
//
// Returns driver.Result which contains last-insert ID and rows-affected counts.
// Returns error when the D1 API call fails.
func (s *d1Stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), namedValues(args))
}

// Query executes the statement and returns rows. It delegates to QueryContext; the call
// is bounded by the configured request timeout.
//
// Takes args ([]driver.Value) which are the positional parameters.
//
// Returns driver.Rows which iterates over the result set.
// Returns error when the D1 API call fails.
func (s *d1Stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), namedValues(args))
}

// ExecContext executes the statement via the D1 HTTP API and returns the result metadata.
// When a transaction is active on the connection, the statement is queued for batch
// execution on Commit instead of executing immediately, and the returned result reports
// ErrResultUnavailableInTransaction because the counters are not known until Commit.
//
// Takes args ([]driver.NamedValue) which are the query parameters.
//
// Returns driver.Result which contains last-insert ID and rows-affected counts.
// Returns error when the D1 query fails or returns a failure status.
func (s *d1Stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if s.conn.activeTx != nil {
		params, err := stringifyNamedParams(args)
		if err != nil {
			return nil, fmt.Errorf("db_driver_d1: exec: %w", err)
		}
		s.conn.activeTx.addStatement(s.query, params)
		return newDeferredResult(), nil
	}

	return s.execDirect(ctx, args)
}

// QueryContext executes the statement via the D1 HTTP API and returns rows. D1 does not
// support queries within transactions since batch execution cannot return intermediate
// row results.
//
// Takes args ([]driver.NamedValue) which are the query parameters.
//
// Returns driver.Rows which iterates over the query results.
// Returns error when the D1 query fails or returns a failure status.
func (s *d1Stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	if s.conn.activeTx != nil {
		return nil, errors.New("db_driver_d1: queries are not supported within D1 transactions; only exec statements can be batched")
	}

	return s.queryDirect(ctx, args)
}

// execDirect executes the statement immediately against the D1 API.
//
// The counters come from the first statement's result, matching what a single-statement
// exec reports.
//
// Takes args ([]driver.NamedValue) which are the query parameters.
//
// Returns driver.Result which contains last-insert ID and rows-affected counts.
// Returns error when the D1 query fails or returns a failure status.
func (s *d1Stmt) execDirect(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	params, err := stringifyNamedParams(args)
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: exec: %w", err)
	}

	results, err := s.conn.client.query(ctx, s.query, params)
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: exec: %w", err)
	}

	if len(results) == 0 {
		return newResult(0, 0), nil
	}

	return newResult(results[0].Meta.LastRowID, results[0].Meta.Changes), nil
}

// queryDirect executes the statement immediately against the D1 API and returns rows.
//
// The rows come from the first statement's result set, keeping the server's column order
// and any duplicate column names.
//
// Takes args ([]driver.NamedValue) which are the query parameters.
//
// Returns driver.Rows which iterates over the query results.
// Returns error when the D1 query fails or returns a failure status.
func (s *d1Stmt) queryDirect(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	params, err := stringifyNamedParams(args)
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: query: %w", err)
	}

	results, err := s.conn.client.query(ctx, s.query, params)
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: query: %w", err)
	}

	if len(results) == 0 {
		return newRows(nil, nil), nil
	}

	return newRows(results[0].Results.Columns, results[0].Results.Rows), nil
}

// namedValues converts positional driver values into ordinal named values.
//
// Takes args ([]driver.Value) which are the positional parameters.
//
// Returns []driver.NamedValue which holds each value with its one-based ordinal.
func namedValues(args []driver.Value) []driver.NamedValue {
	named := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		named[i] = driver.NamedValue{Name: "", Ordinal: i + 1, Value: arg}
	}
	return named
}
