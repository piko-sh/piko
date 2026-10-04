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

package db_engine_duckdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/db"
)

var (
	errHealthQueryRefused = errors.New("query refused")
)

func TestCheckHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		answers map[string]string
		name    string
		want    []db.DatabaseHealthDiagnostic
	}{
		{
			name: "every probe answers",
			answers: map[string]string{
				"SELECT database_size FROM duckdb_databases() WHERE database_name = current_database()": "12 MiB",
				"SELECT current_setting('memory_limit')":                                                "8 GiB",
				"SELECT current_setting('threads')":                                                     "4",
			},
			want: []db.DatabaseHealthDiagnostic{
				{Name: "database_size", Value: "12 MiB", State: "", Message: ""},
				{Name: "memory_limit", Value: "8 GiB", State: "", Message: ""},
				{Name: "threads", Value: "4", State: "", Message: ""},
			},
		},
		{
			name:    "every probe fails",
			answers: map[string]string{},
			want: []db.DatabaseHealthDiagnostic{
				{Name: "database_size", Value: "", State: "UNHEALTHY", Message: "query failed: query refused"},
				{Name: "memory_limit", Value: "", State: "UNHEALTHY", Message: "query failed: query refused"},
				{Name: "threads", Value: "", State: "UNHEALTHY", Message: "query failed: query refused"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			database := sql.OpenDB(&fakeHealthConnector{answers: testCase.answers})
			t.Cleanup(func() { assert.NoError(t, database.Close()) })

			diagnostics := NewDuckDBEngine().CheckHealth(context.Background(), database)

			require.Equal(t, testCase.want, diagnostics)
		})
	}
}

type fakeHealthConnector struct {
	answers map[string]string
}

func (c *fakeHealthConnector) Connect(context.Context) (driver.Conn, error) {
	return &fakeHealthConn{answers: c.answers}, nil
}

func (c *fakeHealthConnector) Driver() driver.Driver {
	return fakeHealthDriver{}
}

type fakeHealthDriver struct{}

func (fakeHealthDriver) Open(string) (driver.Conn, error) {
	return nil, errHealthQueryRefused
}

type fakeHealthConn struct {
	answers map[string]string
}

func (c *fakeHealthConn) Prepare(query string) (driver.Stmt, error) {
	return &fakeHealthStatement{answers: c.answers, query: query}, nil
}

func (*fakeHealthConn) Close() error {
	return nil
}

func (*fakeHealthConn) Begin() (driver.Tx, error) {
	return nil, errHealthQueryRefused
}

type fakeHealthStatement struct {
	answers map[string]string
	query   string
}

func (*fakeHealthStatement) Close() error {
	return nil
}

func (*fakeHealthStatement) NumInput() int {
	return -1
}

func (*fakeHealthStatement) Exec([]driver.Value) (driver.Result, error) {
	return nil, errHealthQueryRefused
}

func (s *fakeHealthStatement) Query([]driver.Value) (driver.Rows, error) {
	answer, known := s.answers[s.query]
	if !known {
		return nil, errHealthQueryRefused
	}
	return &fakeHealthRows{value: answer, served: false}, nil
}

type fakeHealthRows struct {
	value  string
	served bool
}

func (*fakeHealthRows) Columns() []string {
	return []string{"value"}
}

func (*fakeHealthRows) Close() error {
	return nil
}

func (r *fakeHealthRows) Next(destination []driver.Value) error {
	if r.served {
		return io.EOF
	}
	r.served = true
	destination[0] = r.value
	return nil
}
