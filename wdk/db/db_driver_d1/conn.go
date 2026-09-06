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
	"fmt"
)

// Compile-time interface checks.
var (
	_ driver.Conn = (*d1Conn)(nil)

	_ driver.ConnBeginTx = (*d1Conn)(nil)
)

var (
	// ErrUnsupportedTransactionOptions reports that BeginTx was asked for an isolation level
	// or access mode D1's batched transactions cannot provide. Callers can match it with
	// errors.Is.
	ErrUnsupportedTransactionOptions = errors.New("db_driver_d1: unsupported transaction options")
)

// d1Conn implements driver.Conn and driver.ConnBeginTx for Cloudflare D1. Since D1 is
// accessed over HTTP, there is no persistent connection to manage; the HTTP connection
// pool and rate limiter belong to the shared client.
type d1Conn struct {
	// client performs the D1 API calls; it is shared by every connection of the database
	// handle.
	client *d1Client

	// ownedClient is the client to release on Close when this connection was opened directly
	// through driver.Driver.Open rather than through a shared connector; nil otherwise.
	ownedClient *d1Client

	// activeTx holds the currently active transaction, or nil when no transaction is in
	// progress. When set, ExecContext on statements routes through the transaction's batch
	// instead of executing immediately.
	activeTx *d1Tx
}

// Prepare returns a prepared statement bound to this connection.
//
// Takes query (string) which is the SQL statement to prepare.
//
// Returns driver.Stmt which is the prepared statement.
// Returns error which is always nil for D1 (preparation is deferred).
func (c *d1Conn) Prepare(query string) (driver.Stmt, error) {
	return &d1Stmt{
		conn:  c,
		query: query,
	}, nil
}

// Close releases the connection's own client when it has one. Connections created through
// a shared connector hold no resources of their own.
//
// Returns error which is always nil.
func (c *d1Conn) Close() error {
	if c.ownedClient != nil {
		c.ownedClient.close()
	}
	return nil
}

// Begin starts a new transaction. D1 does not support interactive transactions, so
// statements are collected and executed as a single batch on Commit, which is bounded by
// the configured request timeout.
//
// Returns driver.Tx which collects statements for batch execution.
// Returns error when a transaction is already active.
func (c *d1Conn) Begin() (driver.Tx, error) {
	return c.begin(context.Background())
}

// BeginTx starts a new transaction with the given context and options.
//
// The context is captured so the deferred batch Commit can honour cancellation and
// deadlines. D1 executes the batch atomically, which is equivalent to serialisable
// isolation, so the default and serialisable levels are accepted; any other level and
// read-only transactions are rejected.
//
// Takes options (driver.TxOptions) which carries the requested isolation level and access
// mode.
//
// Returns driver.Tx which collects statements for batch execution.
// Returns error which wraps ErrUnsupportedTransactionOptions when the options cannot be
// honoured, or an error when a transaction is already active.
func (c *d1Conn) BeginTx(ctx context.Context, options driver.TxOptions) (driver.Tx, error) {
	if err := validateTxOptions(options); err != nil {
		return nil, err
	}
	return c.begin(ctx)
}

// begin creates a deferred D1 transaction, capturing ctx so the eventual batched Commit
// can honour cancellation and deadlines.
//
// Returns driver.Tx which collects statements for batch execution.
// Returns error when a transaction is already active.
func (c *d1Conn) begin(ctx context.Context) (driver.Tx, error) {
	if c.activeTx != nil {
		return nil, errors.New("db_driver_d1: a transaction is already active on this connection")
	}

	tx := &d1Tx{
		conn:       c,
		ctx:        ctx,
		statements: make([]batchStatement, 0),
		committed:  false,
	}
	c.activeTx = tx
	return tx, nil
}

// newConn returns a connection that issues its calls through client.
//
// Takes client (*d1Client) which performs the D1 API calls.
// Takes ownedClient (*d1Client) which is released on Close, or nil when the client is
// shared and owned by a connector.
//
// Returns *d1Conn which is the new connection.
func newConn(client *d1Client, ownedClient *d1Client) *d1Conn {
	return &d1Conn{
		client:      client,
		ownedClient: ownedClient,
		activeTx:    nil,
	}
}

// validateTxOptions rejects transaction options D1's batched transactions cannot honour.
//
// Takes options (driver.TxOptions) which carries the requested isolation level and access
// mode.
//
// Returns error wrapping ErrUnsupportedTransactionOptions for a read-only transaction or
// an isolation level other than the default or serialisable, nil otherwise.
func validateTxOptions(options driver.TxOptions) error {
	if options.ReadOnly {
		return fmt.Errorf("%w: read-only transactions are not supported", ErrUnsupportedTransactionOptions)
	}
	switch sql.IsolationLevel(options.Isolation) {
	case sql.LevelDefault, sql.LevelSerializable:
		return nil
	default:
		return fmt.Errorf("%w: isolation level %s is not supported",
			ErrUnsupportedTransactionOptions, sql.IsolationLevel(options.Isolation))
	}
}
