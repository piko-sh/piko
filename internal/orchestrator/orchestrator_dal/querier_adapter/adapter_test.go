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

package querier_adapter

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_dal"
	orchestrator_db "piko.sh/piko/internal/orchestrator/orchestrator_dal/querier_sqlite/db"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

type txDoneConnector struct {
	rollbackCalls atomic.Int32
}

func (c *txDoneConnector) Connect(_ context.Context) (driver.Conn, error) {
	return &txDoneConn{connector: c}, nil
}

func (*txDoneConnector) Driver() driver.Driver { return &txDoneDriver{} }

type txDoneDriver struct {
	connector *txDoneConnector
}

func (d *txDoneDriver) Open(_ string) (driver.Conn, error) {
	return &txDoneConn{connector: d.connector}, nil
}

type txDoneConn struct {
	connector *txDoneConnector
}

func (*txDoneConn) Prepare(_ string) (driver.Stmt, error) { return nil, errors.New("not supported") }
func (*txDoneConn) Close() error                          { return nil }
func (c *txDoneConn) Begin() (driver.Tx, error) {
	return &txDoneTx{conn: c, committed: false}, nil
}
func (c *txDoneConn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return &txDoneTx{conn: c, committed: false}, nil
}

type txDoneTx struct {
	conn      *txDoneConn
	committed bool
}

func (t *txDoneTx) Commit() error {
	t.committed = true
	return nil
}
func (t *txDoneTx) Rollback() error {
	if t.conn != nil && t.conn.connector != nil {
		t.conn.connector.rollbackCalls.Add(1)
	}
	if t.committed {
		return sql.ErrTxDone
	}
	return nil
}

func TestQuerierAdapter_RollbackGuardedByErrTxDone(t *testing.T) {
	t.Parallel()

	connector := &txDoneConnector{}
	db := sql.OpenDB(connector)
	defer db.Close()

	adapter := &Adapter{
		db:      db,
		sqlDB:   db,
		queries: orchestrator_db.New(db),
	}

	err := adapter.withTransaction(t.Context(), func(_ context.Context, _ orchestrator_dal.OrchestratorDAL) error {
		return nil
	})

	require.NoError(t, err, "withTransaction must not surface sql.ErrTxDone after a successful commit")
}

func TestQuerierAdapter_RollbackPropagatesNonErrTxDone(t *testing.T) {
	t.Parallel()

	connector := &txDoneConnector{}
	db := sql.OpenDB(connector)
	defer db.Close()

	adapter := &Adapter{
		db:      db,
		sqlDB:   db,
		queries: orchestrator_db.New(db),
	}

	sentinel := errors.New("user error")
	err := adapter.withTransaction(t.Context(), func(_ context.Context, _ orchestrator_dal.OrchestratorDAL) error {
		return sentinel
	})

	require.ErrorIs(t, err, sentinel, "user errors must surface")
	require.GreaterOrEqual(t, connector.rollbackCalls.Load(), int32(1),
		"deferred rollback must call the driver when commit was not reached")
}

var _ orchestrator_domain.TaskStore = (*Adapter)(nil)
