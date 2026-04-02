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
	"slices"

	"github.com/cloudflare/cloudflare-go"
)

// Compile-time interface checks.
var (
	_ driver.Stmt = (*d1Stmt)(nil)

	_ driver.StmtExecContext = (*d1Stmt)(nil)

	_ driver.StmtQueryContext = (*d1Stmt)(nil)
)

// d1Stmt implements driver.Stmt, driver.StmtExecContext, and
// driver.StmtQueryContext. Each execution issues an HTTP request to the D1 API
// unless a transaction is active, in which case the statement is queued for
// batch execution on Commit.
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

// Exec executes the statement with the given arguments. It delegates to
// ExecContext with a background context.
//
// Takes args ([]driver.Value) which are the positional parameters.
//
// Returns driver.Result which contains last-insert ID and rows-affected counts.
// Returns error when the D1 API call fails.
func (s *d1Stmt) Exec(args []driver.Value) (driver.Result, error) {
	named := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: arg}
	}
	return s.ExecContext(context.Background(), named)
}

// Query executes the statement and returns rows. It delegates to QueryContext
// with a background context.
//
// Takes args ([]driver.Value) which are the positional parameters.
//
// Returns driver.Rows which iterates over the result set.
// Returns error when the D1 API call fails.
func (s *d1Stmt) Query(args []driver.Value) (driver.Rows, error) {
	named := make([]driver.NamedValue, len(args))
	for i, arg := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: arg}
	}
	return s.QueryContext(context.Background(), named)
}

// ExecContext executes the statement via the D1 HTTP API and returns the
// result metadata. When a transaction is active on the connection, the
// statement is queued for batch execution on Commit instead of executing
// immediately.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
// Takes args ([]driver.NamedValue) which are the query parameters.
//
// Returns driver.Result which contains last-insert ID and rows-affected counts.
// Returns error when the D1 query fails or returns a failure status.
func (s *d1Stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	if s.conn.activeTx != nil {
		s.conn.activeTx.addStatement(s.query, stringifyNamedParams(args))
		return &d1Result{}, nil
	}

	return s.execDirect(ctx, args)
}

// QueryContext executes the statement via the D1 HTTP API and returns rows.
// D1 does not support queries within transactions since batch execution
// cannot return intermediate row results.
//
// Takes ctx (context.Context) which controls cancellation and timeouts.
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
func (s *d1Stmt) execDirect(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	results, err := s.conn.api.QueryD1Database(ctx, s.conn.rc, cloudflare.QueryD1DatabaseParams{
		DatabaseID: s.conn.databaseID,
		SQL:        s.query,
		Parameters: stringifyNamedParams(args),
	})
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: exec: %w", err)
	}

	if len(results) == 0 {
		return &d1Result{}, nil
	}

	if results[0].Success != nil && !*results[0].Success {
		return nil, errors.New("db_driver_d1: exec: D1 query returned failure")
	}

	return &d1Result{
		lastInsertID: int64(results[0].Meta.LastRowID),
		rowsAffected: int64(results[0].Meta.Changes),
	}, nil
}

// queryDirect executes the statement immediately against the D1 API and
// returns rows.
func (s *d1Stmt) queryDirect(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	results, err := s.conn.api.QueryD1Database(ctx, s.conn.rc, cloudflare.QueryD1DatabaseParams{
		DatabaseID: s.conn.databaseID,
		SQL:        s.query,
		Parameters: stringifyNamedParams(args),
	})
	if err != nil {
		return nil, fmt.Errorf("db_driver_d1: query: %w", err)
	}

	if len(results) == 0 {
		return &d1Rows{}, nil
	}

	if results[0].Success != nil && !*results[0].Success {
		return nil, errors.New("db_driver_d1: query: D1 query returned failure")
	}

	rows := &d1Rows{
		data:  results[0].Results,
		index: 0,
	}

	if len(rows.data) > 0 {
		columns := make([]string, 0, len(rows.data[0]))
		for key := range rows.data[0] {
			columns = append(columns, key)
		}
		slices.Sort(columns)
		rows.columns = columns
	}

	return rows, nil
}
