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
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

var (
	_ driver.Tx = (*d1Tx)(nil)
)

// d1Tx implements driver.Tx for D1 with client-side batching.
//
// D1 does not support interactive transactions. Statements are collected into a batch and
// flushed as a single request wrapped in BEGIN/COMMIT when Commit is called. Rollback
// discards the collected statements, since no server-side state exists to revert.
type d1Tx struct {
	// conn is the parent connection used to execute the final batch.
	conn *d1Conn

	// ctx is the context captured at BeginTx so the batched Commit network call can honour
	// cancellation and deadlines. The database/sql driver.Tx interface gives Commit no
	// context of its own, so the transaction must carry it - the sanctioned exception to not
	// storing a context in a struct.
	ctx context.Context

	// statements holds the SQL statements queued for batch execution.
	statements []batchStatement

	// committed tracks whether Commit has already been called.
	committed bool
}

// batchStatement holds a single SQL statement and its stringified parameters for batch
// execution within a transaction.
type batchStatement struct {
	// query is the SQL statement text.
	query string

	// params holds the stringified parameter values.
	params []string
}

// Commit flushes all collected statements as a single batch request wrapped in
// BEGIN/COMMIT. If no statements were collected, Commit is a no-op.
//
// Returns error when the batch execution fails or Commit has already been called.
func (tx *d1Tx) Commit() error {
	if tx.committed {
		return errors.New("db_driver_d1: transaction already committed")
	}
	tx.committed = true
	tx.conn.activeTx = nil

	if len(tx.statements) == 0 {
		return nil
	}

	sql, allParams := buildBatch(tx.statements)

	commitCtx := tx.ctx
	if commitCtx == nil {
		commitCtx = context.Background()
	}

	results, err := tx.conn.api.QueryD1Database(commitCtx, tx.conn.rc, cloudflare.QueryD1DatabaseParams{
		DatabaseID: tx.conn.databaseID,
		SQL:        sql,
		Parameters: allParams,
	})
	if err != nil {
		return fmt.Errorf("db_driver_d1: commit: %w", err)
	}

	for index, result := range results {
		if successErr := checkD1Success(result); successErr != nil {
			return fmt.Errorf("db_driver_d1: commit: statement %d: %w", index, successErr)
		}
	}

	return nil
}

// buildBatch concatenates the queued statements into a single BEGIN/COMMIT-wrapped SQL
// string and flattens their parameters into one slice.
//
// The D1 HTTP API binds a single '?'-sequence across the whole batch, so the parameters
// are appended in statement order, each statement's bindings following the previous
// statement's. Callers must therefore queue statements in the order their placeholders
// should consume the flattened parameter list.
//
// Takes statements ([]batchStatement) which are the queued statements in execution order.
//
// Returns string which is the BEGIN/COMMIT-wrapped batch SQL.
// Returns []string which holds every statement's parameters concatenated in order.
func buildBatch(statements []batchStatement) (string, []string) {
	var batch strings.Builder
	var allParams []string

	batch.WriteString("BEGIN;\n")
	for _, statement := range statements {
		batch.WriteString(statement.query)
		batch.WriteString(";\n")
		allParams = append(allParams, statement.params...)
	}
	batch.WriteString("COMMIT;")

	return batch.String(), allParams
}

// Rollback discards all collected statements. Since D1 transactions are client-side only
// (no server-side state exists until Commit), there is nothing to roll back on the
// server.
//
// Returns error when the transaction has already been committed.
func (tx *d1Tx) Rollback() error {
	if tx.committed {
		return errors.New("db_driver_d1: cannot rollback a committed transaction")
	}
	tx.statements = nil
	tx.conn.activeTx = nil
	return nil
}

// addStatement appends a statement to the transaction batch.
//
// Takes query (string) which is the SQL statement text.
// Takes params ([]string) which are the stringified parameter values.
func (tx *d1Tx) addStatement(query string, params []string) {
	tx.statements = append(tx.statements, batchStatement{
		query:  query,
		params: params,
	})
}
