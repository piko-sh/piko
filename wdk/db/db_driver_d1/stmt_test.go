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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConn(t *testing.T, responseBody string) *d1Conn {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	api, err := cloudflare.NewWithAPIToken("test-token",
		cloudflare.BaseURL(server.URL),
		cloudflare.HTTPClient(server.Client()),
	)
	require.NoError(t, err)

	return &d1Conn{
		api:        api,
		rc:         cloudflare.AccountIdentifier("test-account"),
		databaseID: "test-database",
	}
}

const (
	singleSuccessResult = `{
	"result": [
		{
			"success": true,
			"results": [{"id": 1, "name": "alice"}],
			"meta": {"last_row_id": 7, "changes": 3}
		}
	],
	"success": true,
	"errors": [],
	"messages": []
}`
	firstSuccessSecondFailure = `{
	"result": [
		{
			"success": true,
			"results": [],
			"meta": {"last_row_id": 1, "changes": 1}
		},
		{
			"success": false,
			"results": [],
			"meta": {}
		}
	],
	"success": true,
	"errors": [],
	"messages": []
}`
	firstSuccessSecondMissingFlag = `{
	"result": [
		{
			"success": true,
			"results": [],
			"meta": {"last_row_id": 1, "changes": 1}
		},
		{
			"results": [],
			"meta": {}
		}
	],
	"success": true,
	"errors": [],
	"messages": []
}`
)

func TestExecContextNilParameterReturnsSentinel(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	stmt := &d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (?)"}

	result, err := stmt.ExecContext(context.Background(), []driver.NamedValue{
		{Ordinal: 1, Value: nil},
	})
	require.ErrorIs(t, err, errNullParamUnsupported)
	assert.Nil(t, result)
}

func TestQueryContextNilParameterReturnsSentinel(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	stmt := &d1Stmt{conn: conn, query: "SELECT * FROM t WHERE v = ?"}

	rows, err := stmt.QueryContext(context.Background(), []driver.NamedValue{
		{Ordinal: 1, Value: nil},
	})
	require.ErrorIs(t, err, errNullParamUnsupported)
	assert.Nil(t, rows)
}

func TestExecContextTransactionNilParameterReturnsSentinel(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	tx, err := conn.begin(context.Background())
	require.NoError(t, err)

	stmt := &d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (?)"}
	result, err := stmt.ExecContext(context.Background(), []driver.NamedValue{
		{Ordinal: 1, Value: nil},
	})
	require.ErrorIs(t, err, errNullParamUnsupported)
	assert.Nil(t, result)

	d1Transaction, ok := tx.(*d1Tx)
	require.True(t, ok)
	assert.Empty(t, d1Transaction.statements)
}

func TestExecDirectSuccess(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	stmt := &d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (?)"}

	result, err := stmt.ExecContext(context.Background(), []driver.NamedValue{
		{Ordinal: 1, Value: "x"},
	})
	require.NoError(t, err)

	lastInsertID, err := result.LastInsertId()
	require.NoError(t, err)
	assert.Equal(t, int64(7), lastInsertID)

	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(3), rowsAffected)
}

func TestExecDirectLaterStatementFailureSurfaced(t *testing.T) {
	conn := newTestConn(t, firstSuccessSecondFailure)
	stmt := &d1Stmt{conn: conn, query: "UPDATE a SET v = 1; UPDATE b SET v = 2"}

	result, err := stmt.ExecContext(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "statement 1")
	assert.Contains(t, err.Error(), "failure")
}

func TestExecDirectLaterStatementMissingFlagSurfaced(t *testing.T) {
	conn := newTestConn(t, firstSuccessSecondMissingFlag)
	stmt := &d1Stmt{conn: conn, query: "UPDATE a SET v = 1; UPDATE b SET v = 2"}

	result, err := stmt.ExecContext(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "statement 1")
	assert.Contains(t, err.Error(), "success flag")
}

func TestQueryDirectSuccess(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	stmt := &d1Stmt{conn: conn, query: "SELECT id, name FROM t"}

	rows, err := stmt.QueryContext(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })

	assert.Equal(t, []string{"id", "name"}, rows.Columns())

	dest := make([]driver.Value, len(rows.Columns()))
	require.NoError(t, rows.Next(dest))
	assert.Equal(t, int64(1), dest[0])
	assert.Equal(t, "alice", dest[1])
}

func TestQueryDirectLaterStatementFailureSurfaced(t *testing.T) {
	conn := newTestConn(t, firstSuccessSecondFailure)
	stmt := &d1Stmt{conn: conn, query: "SELECT 1; SELECT bad"}

	rows, err := stmt.QueryContext(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, rows)
	assert.Contains(t, err.Error(), "statement 1")
	assert.Contains(t, err.Error(), "failure")
}

func TestExecDirectEmptyResultsIsNoOp(t *testing.T) {
	conn := newTestConn(t, `{"result": [], "success": true, "errors": [], "messages": []}`)
	stmt := &d1Stmt{conn: conn, query: "PRAGMA noop"}

	result, err := stmt.ExecContext(context.Background(), nil)
	require.NoError(t, err)

	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Zero(t, rowsAffected)
}

func TestQueryContextWithinTransactionRejected(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	_, err := conn.begin(context.Background())
	require.NoError(t, err)

	stmt := &d1Stmt{conn: conn, query: "SELECT 1"}
	rows, err := stmt.QueryContext(context.Background(), nil)
	require.Error(t, err)
	assert.Nil(t, rows)
	assert.Contains(t, err.Error(), "not supported within D1 transactions")
}
