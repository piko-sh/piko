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

	"github.com/cloudflare/cloudflare-go"
)

// Compile-time interface checks.
var (
	_ driver.Conn = (*d1Conn)(nil)

	_ driver.ConnBeginTx = (*d1Conn)(nil)
)

// d1Conn implements driver.Conn and driver.ConnBeginTx for Cloudflare D1.
// Since D1 is accessed over HTTP, there is no persistent connection to manage;
// Close is a no-op.
type d1Conn struct {
	// api is the Cloudflare API client used to execute D1 queries.
	api *cloudflare.API

	// rc is the Cloudflare resource container (account identifier).
	rc *cloudflare.ResourceContainer

	// activeTx holds the currently active transaction, or nil when no
	// transaction is in progress. When set, ExecContext on statements routes
	// through the transaction's batch instead of executing immediately.
	activeTx *d1Tx

	// databaseID is the UUID of the D1 database.
	databaseID string
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

// Close is a no-op since D1 connections are stateless HTTP calls.
//
// Returns error which is always nil.
func (*d1Conn) Close() error {
	return nil
}

// Begin starts a new transaction. D1 does not support interactive transactions,
// so statements are collected and executed as a single batch on Commit.
//
// Returns driver.Tx which collects statements for batch execution.
// Returns error when a transaction is already active.
func (c *d1Conn) Begin() (driver.Tx, error) {
	if c.activeTx != nil {
		return nil, errors.New("db_driver_d1: a transaction is already active on this connection")
	}

	tx := &d1Tx{
		conn:       c,
		statements: make([]batchStatement, 0),
	}
	c.activeTx = tx
	return tx, nil
}

// BeginTx starts a new transaction with the given context and options. D1 does
// not support isolation levels or read-only transactions, so opts is ignored.
//
// Takes ctx (context.Context) which is unused since D1 transactions are
// deferred to Commit time.
// Takes opts (driver.TxOptions) which is ignored.
//
// Returns driver.Tx which collects statements for batch execution.
// Returns error when a transaction is already active.
func (c *d1Conn) BeginTx(_ context.Context, _ driver.TxOptions) (driver.Tx, error) {
	return c.Begin()
}
