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
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDSN(t *testing.T) {
	testCases := []struct {
		name          string
		dsn           string
		expectedError string
		expected      Config
	}{
		{
			name:     "valid",
			dsn:      "myaccount/mydb?token=secret123",
			expected: Config{APIToken: "secret123", AccountID: "myaccount", DatabaseID: "mydb"},
		},
		{name: "missing token", dsn: "myaccount/mydb", expectedError: "token"},
		{name: "missing slash", dsn: "noslashhere?token=secret", expectedError: "expected format"},
		{name: "empty account", dsn: "/mydb?token=secret", expectedError: "accountID is empty"},
		{name: "empty database", dsn: "myaccount/?token=secret", expectedError: "databaseID is empty"},
		{name: "malformed query", dsn: "myaccount/mydb?token=%zz", expectedError: "invalid DSN query"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config, err := parseDSN(testCase.dsn)
			if testCase.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), testCase.expectedError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, testCase.expected, config)
		})
	}
}

func TestStringifyNamedParamsNil(t *testing.T) {
	args := []driver.NamedValue{
		{Ordinal: 1, Value: nil},
	}
	result, err := stringifyNamedParams(args)
	require.ErrorIs(t, err, ErrNullParamUnsupported)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parameter 1")
}

func TestStringifyNamedParamsNilNotFirst(t *testing.T) {
	args := []driver.NamedValue{
		{Ordinal: 1, Value: "ok"},
		{Ordinal: 2, Value: nil},
	}
	result, err := stringifyNamedParams(args)
	require.ErrorIs(t, err, ErrNullParamUnsupported)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parameter 2")
}

func TestStringifyNamedParamsString(t *testing.T) {
	args := []driver.NamedValue{
		{Ordinal: 1, Value: "hello"},
	}
	result, err := stringifyNamedParams(args)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "hello", result[0])
}

func TestStringifyNamedParamsInt64(t *testing.T) {
	args := []driver.NamedValue{
		{Ordinal: 1, Value: int64(42)},
	}
	result, err := stringifyNamedParams(args)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "42", result[0])
}

func TestStringifyNamedParamsFloat64(t *testing.T) {
	args := []driver.NamedValue{
		{Ordinal: 1, Value: float64(3.14)},
	}
	result, err := stringifyNamedParams(args)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, strconv.FormatFloat(3.14, 'g', -1, 64), result[0])
}

func TestStringifyNamedParamsBool(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		value    bool
	}{
		{name: "true", value: true, expected: "1"},
		{name: "false", value: false, expected: "0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []driver.NamedValue{
				{Ordinal: 1, Value: test.value},
			}
			result, err := stringifyNamedParams(args)
			require.NoError(t, err)
			require.Len(t, result, 1)
			assert.Equal(t, test.expected, result[0])
		})
	}
}

func TestStringifyNamedParamsBytes(t *testing.T) {
	data := []byte("binary data")
	args := []driver.NamedValue{
		{Ordinal: 1, Value: data},
	}
	result, err := stringifyNamedParams(args)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, base64.StdEncoding.EncodeToString(data), result[0])
}

func TestStringifyNamedParamsTime(t *testing.T) {
	timestamp := time.Date(2026, 3, 27, 12, 0, 0, 123456789, time.FixedZone("CEST", 2*60*60))
	args := []driver.NamedValue{
		{Ordinal: 1, Value: timestamp},
	}
	result, err := stringifyNamedParams(args)
	require.NoError(t, err)
	require.Len(t, result, 1)

	assert.Equal(t, timestamp.UTC().Format(time.RFC3339Nano), result[0])
}

func TestDriverName(t *testing.T) {
	assert.Equal(t, "d1", DriverName())
}

func TestBuildBatch(t *testing.T) {
	testCases := []struct {
		name           string
		expectedSQL    string
		statements     []batchStatement
		expectedParams []string
	}{
		{
			name: "parameters follow statement order",
			statements: []batchStatement{
				{query: "INSERT INTO a (x) VALUES (?)", params: []string{"a1"}},
				{query: "INSERT INTO b (x, y) VALUES (?, ?)", params: []string{"b1", "b2"}},
				{query: "INSERT INTO c (x) VALUES (?)", params: []string{"c1"}},
			},
			expectedSQL: "BEGIN;\n" +
				"INSERT INTO a (x) VALUES (?)\n;\n" +
				"INSERT INTO b (x, y) VALUES (?, ?)\n;\n" +
				"INSERT INTO c (x) VALUES (?)\n;\n" +
				"COMMIT;",
			expectedParams: []string{"a1", "b1", "b2", "c1"},
		},
		{
			name: "statements without parameters",
			statements: []batchStatement{
				{query: "DELETE FROM a", params: nil},
				{query: "DELETE FROM b", params: nil},
			},
			expectedSQL:    "BEGIN;\nDELETE FROM a\n;\nDELETE FROM b\n;\nCOMMIT;",
			expectedParams: nil,
		},
		{
			name: "trailing line comment cannot swallow the separator",
			statements: []batchStatement{
				{query: "DELETE FROM a -- clear a", params: nil},
				{query: "DELETE FROM b;  \n", params: nil},
			},
			expectedSQL:    "BEGIN;\nDELETE FROM a -- clear a\n;\nDELETE FROM b\n;\nCOMMIT;",
			expectedParams: nil,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			sql, params := buildBatch(testCase.statements)

			assert.Equal(t, testCase.expectedSQL, sql)
			assert.Equal(t, testCase.expectedParams, params)
		})
	}
}

func TestValidateConfig(t *testing.T) {
	testCases := []struct {
		name          string
		expectedError string
		config        Config
	}{
		{name: "complete", config: Config{APIToken: "t", AccountID: "a", DatabaseID: "d"}},
		{name: "missing token", config: Config{AccountID: "a", DatabaseID: "d"}, expectedError: "APIToken"},
		{name: "missing account", config: Config{APIToken: "t", DatabaseID: "d"}, expectedError: "AccountID"},
		{name: "missing database", config: Config{APIToken: "t", AccountID: "a"}, expectedError: "DatabaseID"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateConfig(testCase.config)
			if testCase.expectedError == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedError)

			database, openErr := Open(testCase.config)
			require.Error(t, openErr)
			assert.Nil(t, database)
		})
	}
}

func TestOpenSharesOneClientAcrossConnections(t *testing.T) {
	recorder := newRecordingServer(t, http.StatusOK, unorderedDuplicateColumnsResult)
	database, err := Open(Config{APIToken: "t", AccountID: "a", DatabaseID: "d"},
		withBaseURL(recorder.server.URL), WithRequestsPerSecond(testRequestsPerSecond))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, database.Close()) })

	ctx := context.Background()
	first, err := database.Conn(ctx)
	require.NoError(t, err)
	second, err := database.Conn(ctx)
	require.NoError(t, err)

	var clients []*d1Client
	for _, connection := range []*sql.Conn{first, second} {
		require.NoError(t, connection.Raw(func(driverConn any) error {
			d1Connection, ok := driverConn.(*d1Conn)
			require.True(t, ok)
			clients = append(clients, d1Connection.client)
			return nil
		}))
		require.NoError(t, connection.Close())
	}
	require.Len(t, clients, 2)
	assert.Same(t, clients[0], clients[1])

	var zeta, alpha, firstName, secondName string
	require.NoError(t, database.QueryRowContext(ctx, "SELECT zeta, alpha, a.name, b.name FROM a JOIN b").
		Scan(&zeta, &alpha, &firstName, &secondName))
	assert.Equal(t, []string{"z1", "a1", "first", "second"}, []string{zeta, alpha, firstName, secondName})
}

func TestDriverOpenAndOpenConnector(t *testing.T) {
	d1 := &d1Driver{}

	connection, err := d1.Open("account/database?token=secret")
	require.NoError(t, err)
	d1Connection, ok := connection.(*d1Conn)
	require.True(t, ok)
	assert.Same(t, d1Connection.client, d1Connection.ownedClient)
	assert.NoError(t, connection.Close())

	connector, err := d1.OpenConnector("account/database?token=secret")
	require.NoError(t, err)
	assert.Same(t, d1, connector.Driver())

	shared, err := connector.Connect(context.Background())
	require.NoError(t, err)
	sharedConnection, ok := shared.(*d1Conn)
	require.True(t, ok)
	assert.Nil(t, sharedConnection.ownedClient)
	assert.NoError(t, shared.Close())

	cancelled, cancel := context.WithCancelCause(context.Background())
	cancel(errors.New("caller gave up"))
	_, err = connector.Connect(cancelled)
	require.ErrorIs(t, err, context.Canceled)

	closer, ok := connector.(io.Closer)
	require.True(t, ok)
	assert.NoError(t, closer.Close())
}

func TestDriverRejectsMalformedDSN(t *testing.T) {
	d1 := &d1Driver{}

	connection, err := d1.Open("not-a-dsn")
	require.Error(t, err)
	assert.Nil(t, connection)

	connector, err := d1.OpenConnector("account/database")
	require.Error(t, err)
	assert.Nil(t, connector)
}
