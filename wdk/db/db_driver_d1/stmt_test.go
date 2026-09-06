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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	singleSuccessResult = `{
	"result": [
		{
			"success": true,
			"results": {"columns": ["id", "name"], "rows": [[1, "alice"]]},
			"meta": {"last_row_id": 7, "changes": 3}
		}
	],
	"success": true,
	"errors": [],
	"messages": []
}`
	unorderedDuplicateColumnsResult = `{
	"result": [
		{
			"success": true,
			"results": {
				"columns": ["zeta", "alpha", "name", "name"],
				"rows": [["z1", "a1", "first", "second"], ["z2"]]
			},
			"meta": {"last_row_id": 0, "changes": 0}
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
			"results": {"columns": [], "rows": []},
			"meta": {"last_row_id": 1, "changes": 1}
		},
		{
			"success": false,
			"results": {"columns": [], "rows": []},
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
			"results": {"columns": [], "rows": []},
			"meta": {"last_row_id": 1, "changes": 1}
		},
		{
			"results": {"columns": [], "rows": []},
			"meta": {}
		}
	],
	"success": true,
	"errors": [],
	"messages": []
}`
	emptyResult = `{"result": [], "success": true, "errors": [], "messages": []}`
)

func TestExecContextNilParameterReturnsSentinel(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	stmt := &d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (?)"}

	result, err := stmt.ExecContext(context.Background(), []driver.NamedValue{
		{Ordinal: 1, Value: nil},
	})
	require.ErrorIs(t, err, ErrNullParamUnsupported)
	assert.Nil(t, result)
}

func TestQueryContextNilParameterReturnsSentinel(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	stmt := &d1Stmt{conn: conn, query: "SELECT * FROM t WHERE v = ?"}

	rows, err := stmt.QueryContext(context.Background(), []driver.NamedValue{
		{Ordinal: 1, Value: nil},
	})
	require.ErrorIs(t, err, ErrNullParamUnsupported)
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
	require.ErrorIs(t, err, ErrNullParamUnsupported)
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

func TestStatementFailuresSurfaced(t *testing.T) {
	testCases := []struct {
		name          string
		responseBody  string
		query         string
		expectedError []string
		useQuery      bool
	}{
		{
			name:          "exec later statement failure",
			responseBody:  firstSuccessSecondFailure,
			query:         "UPDATE a SET v = 1; UPDATE b SET v = 2",
			expectedError: []string{"exec", "statement 1", "failure"},
		},
		{
			name:          "exec later statement missing flag",
			responseBody:  firstSuccessSecondMissingFlag,
			query:         "UPDATE a SET v = 1; UPDATE b SET v = 2",
			expectedError: []string{"exec", "statement 1", "success flag"},
		},
		{
			name:          "query later statement failure",
			responseBody:  firstSuccessSecondFailure,
			query:         "SELECT 1; SELECT bad",
			expectedError: []string{"query", "statement 1", "failure"},
			useQuery:      true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			conn := newTestConn(t, testCase.responseBody)
			stmt := &d1Stmt{conn: conn, query: testCase.query}

			var err error
			if testCase.useQuery {
				var rows driver.Rows
				rows, err = stmt.QueryContext(context.Background(), nil)
				assert.Nil(t, rows)
			} else {
				var result driver.Result
				result, err = stmt.ExecContext(context.Background(), nil)
				assert.Nil(t, result)
			}
			require.Error(t, err)
			for _, fragment := range testCase.expectedError {
				assert.Contains(t, err.Error(), fragment)
			}
		})
	}
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

func TestQueryDirectKeepsServerColumnOrderAndDuplicates(t *testing.T) {
	conn := newTestConn(t, unorderedDuplicateColumnsResult)
	stmt := &d1Stmt{conn: conn, query: "SELECT zeta, alpha, a.name, b.name FROM a JOIN b"}

	rows, err := stmt.QueryContext(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })

	assert.Equal(t, []string{"zeta", "alpha", "name", "name"}, rows.Columns())

	dest := make([]driver.Value, len(rows.Columns()))
	require.NoError(t, rows.Next(dest))
	assert.Equal(t, []driver.Value{"z1", "a1", "first", "second"}, dest)

	require.NoError(t, rows.Next(dest))
	assert.Equal(t, []driver.Value{"z2", nil, nil, nil}, dest)
}

func TestEmptyResultsAreNoOps(t *testing.T) {
	conn := newTestConn(t, emptyResult)

	result, err := (&d1Stmt{conn: conn, query: "PRAGMA noop"}).ExecContext(context.Background(), nil)
	require.NoError(t, err)
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Zero(t, rowsAffected)

	rows, err := (&d1Stmt{conn: conn, query: "PRAGMA noop"}).QueryContext(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, rows.Columns())
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

func TestExecWithinTransactionReportsUnavailableCounters(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)
	_, err := conn.begin(context.Background())
	require.NoError(t, err)

	result, err := (&d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (?)"}).ExecContext(
		context.Background(), []driver.NamedValue{{Ordinal: 1, Value: "x"}})
	require.NoError(t, err)

	_, err = result.LastInsertId()
	require.ErrorIs(t, err, ErrResultUnavailableInTransaction)
	_, err = result.RowsAffected()
	require.ErrorIs(t, err, ErrResultUnavailableInTransaction)
}

func TestStatementsWithoutContextUseDetachedContext(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, singleSuccessResult)
	conn := newConn(newTestClient(t, recorder.server.URL), nil)
	stmt := &d1Stmt{conn: conn, query: "SELECT id, name FROM t WHERE v = ?"}

	result, err := stmt.Exec([]driver.Value{"a"})
	require.NoError(t, err)
	rowsAffected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(3), rowsAffected)

	rows, err := stmt.Query([]driver.Value{int64(2)})
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "name"}, rows.Columns())

	requests := recorder.recorded()
	require.Len(t, requests, 2)
	assert.Equal(t, []string{"a"}, requests[0].body.Parameters)
	assert.Equal(t, []string{"2"}, requests[1].body.Parameters)
}

func TestExecIsSentOnceAgainstServerError(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusBadGateway, "<html>bad gateway</html>")
	conn := newConn(newTestClient(t, recorder.server.URL), nil)

	result, err := (&d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (?)"}).ExecContext(
		context.Background(), []driver.NamedValue{{Ordinal: 1, Value: "x"}})
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, int64(1), recorder.count.Load())
}

func TestStatementBasics(t *testing.T) {
	stmt := &d1Stmt{conn: nil, query: "SELECT 1"}

	assert.Equal(t, -1, stmt.NumInput())
	assert.NoError(t, stmt.Close())
}
