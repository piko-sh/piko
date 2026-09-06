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
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommitSendsBatchOnce(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, singleSuccessResult)
	conn := newConn(newTestClient(t, recorder.server.URL), nil)

	tx, err := conn.Begin()
	require.NoError(t, err)

	insert := &d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (?) -- first"}
	_, err = insert.ExecContext(context.Background(), []driver.NamedValue{{Ordinal: 1, Value: "a"}})
	require.NoError(t, err)
	_, err = insert.ExecContext(context.Background(), []driver.NamedValue{{Ordinal: 1, Value: "b"}})
	require.NoError(t, err)
	assert.Zero(t, recorder.count.Load())

	require.NoError(t, tx.Commit())
	assert.Nil(t, conn.activeTx)

	requests := recorder.recorded()
	require.Len(t, requests, 1)
	assert.Equal(t,
		"BEGIN;\nINSERT INTO t (v) VALUES (?) -- first\n;\nINSERT INTO t (v) VALUES (?) -- first\n;\nCOMMIT;",
		requests[0].body.SQL)
	assert.Equal(t, []string{"a", "b"}, requests[0].body.Parameters)

	err = tx.Commit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already committed")

	err = tx.Rollback()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot rollback")
}

func TestCommitWithoutStatementsSendsNothing(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, singleSuccessResult)
	conn := newConn(newTestClient(t, recorder.server.URL), nil)

	tx, err := conn.BeginTx(context.Background(), driver.TxOptions{})
	require.NoError(t, err)
	require.NoError(t, tx.Commit())
	assert.Zero(t, recorder.count.Load())
}

func TestCommitFailureIsNotResent(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusBadGateway, "")
	conn := newConn(newTestClient(t, recorder.server.URL), nil)

	tx, err := conn.BeginTx(context.Background(), driver.TxOptions{})
	require.NoError(t, err)
	_, err = (&d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (1)"}).ExecContext(context.Background(), nil)
	require.NoError(t, err)

	err = tx.Commit()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "commit")
	assert.Equal(t, int64(1), recorder.count.Load())
}

func TestCommitHonoursCapturedContext(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, singleSuccessResult)
	conn := newConn(newTestClient(t, recorder.server.URL), nil)

	ctx, cancel := context.WithCancelCause(context.Background())
	tx, err := conn.BeginTx(ctx, driver.TxOptions{})
	require.NoError(t, err)
	_, err = (&d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (1)"}).ExecContext(ctx, nil)
	require.NoError(t, err)

	cancel(errors.New("caller gave up"))
	err = tx.Commit()
	require.Error(t, err)
	assert.Zero(t, recorder.count.Load())
}

func TestRollbackDiscardsStatements(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, singleSuccessResult)
	conn := newConn(newTestClient(t, recorder.server.URL), nil)

	tx, err := conn.Begin()
	require.NoError(t, err)
	_, err = (&d1Stmt{conn: conn, query: "INSERT INTO t (v) VALUES (1)"}).ExecContext(context.Background(), nil)
	require.NoError(t, err)

	require.NoError(t, tx.Rollback())
	assert.Nil(t, conn.activeTx)
	assert.Zero(t, recorder.count.Load())
}

func TestBeginRejectsNestedTransaction(t *testing.T) {
	conn := newTestConn(t, singleSuccessResult)

	_, err := conn.Begin()
	require.NoError(t, err)

	_, err = conn.Begin()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already active")
}

func TestBeginTxOptions(t *testing.T) {
	testCases := []struct {
		name        string
		options     driver.TxOptions
		expectError bool
	}{
		{name: "default", options: driver.TxOptions{}},
		{name: "serialisable", options: driver.TxOptions{Isolation: driver.IsolationLevel(sql.LevelSerializable)}},
		{name: "read committed", options: driver.TxOptions{Isolation: driver.IsolationLevel(sql.LevelReadCommitted)}, expectError: true},
		{name: "read only", options: driver.TxOptions{ReadOnly: true}, expectError: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			conn := newTestConn(t, singleSuccessResult)

			tx, err := conn.BeginTx(context.Background(), testCase.options)
			if testCase.expectError {
				require.ErrorIs(t, err, ErrUnsupportedTransactionOptions)
				assert.Nil(t, tx)
				assert.Nil(t, conn.activeTx)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, tx)
		})
	}
}
