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
package db_catalogue_postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	errFakeUnsupported = errors.New("fake database: operation not supported")
)

type fakeQueryResult struct {
	err      error
	nextErr  error
	closeErr error
	columns  []string
	rows     [][]driver.Value
}

type fakeQueryHandler struct {
	respond func(arguments []driver.NamedValue) fakeQueryResult
	marker  string
}

type fakeConnector struct {
	handlers []fakeQueryHandler
}

func (connector *fakeConnector) Connect(context.Context) (driver.Conn, error) {
	return &fakeConnection{handlers: connector.handlers}, nil
}

func (*fakeConnector) Driver() driver.Driver {
	return fakeDriver{}
}

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) {
	return nil, errFakeUnsupported
}

type fakeConnection struct {
	handlers []fakeQueryHandler
}

func (*fakeConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errFakeUnsupported
}

func (*fakeConnection) Close() error {
	return nil
}

func (*fakeConnection) Begin() (driver.Tx, error) {
	return nil, errFakeUnsupported
}

func (connection *fakeConnection) QueryContext(
	_ context.Context,
	query string,
	arguments []driver.NamedValue,
) (driver.Rows, error) {
	for _, handler := range connection.handlers {
		if !strings.Contains(query, handler.marker) {
			continue
		}
		result := handler.respond(arguments)
		if result.err != nil {
			return nil, result.err
		}
		return &fakeRows{result: result, position: 0}, nil
	}
	return nil, fmt.Errorf("fake database: unexpected query %q", query)
}

type fakeRows struct {
	result   fakeQueryResult
	position int
}

func (rows *fakeRows) Columns() []string {
	return rows.result.columns
}

func (rows *fakeRows) Close() error {
	return rows.result.closeErr
}

func (rows *fakeRows) Next(destination []driver.Value) error {
	if rows.position >= len(rows.result.rows) {
		if rows.result.nextErr != nil {
			return rows.result.nextErr
		}
		return io.EOF
	}
	copy(destination, rows.result.rows[rows.position])
	rows.position++
	return nil
}

func openFakeDatabase(t *testing.T, handlers ...fakeQueryHandler) *sql.DB {
	t.Helper()

	database := sql.OpenDB(&fakeConnector{handlers: handlers})
	t.Cleanup(func() { assert.NoError(t, database.Close()) })
	return database
}

func fixedResult(columns []string, rows ...[]driver.Value) func([]driver.NamedValue) fakeQueryResult {
	return func([]driver.NamedValue) fakeQueryResult {
		return fakeQueryResult{columns: columns, rows: rows, err: nil, nextErr: nil, closeErr: nil}
	}
}

func argumentString(arguments []driver.NamedValue, index int) string {
	if index >= len(arguments) {
		return ""
	}
	value, ok := arguments[index].Value.(string)
	if !ok {
		return ""
	}
	return value
}
