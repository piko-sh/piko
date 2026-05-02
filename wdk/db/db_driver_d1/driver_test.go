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
	"database/sql/driver"
	"encoding/base64"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDSNValid(t *testing.T) {
	config, err := parseDSN("myaccount/mydb?token=secret123")
	require.NoError(t, err)

	assert.Equal(t, "myaccount", config.AccountID)
	assert.Equal(t, "mydb", config.DatabaseID)
	assert.Equal(t, "secret123", config.APIToken)
}

func TestParseDSNMissingToken(t *testing.T) {
	_, err := parseDSN("myaccount/mydb")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token")
}

func TestParseDSNMissingSlash(t *testing.T) {
	_, err := parseDSN("noslashhere?token=secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected format")
}

func TestParseDSNEmptyAccountID(t *testing.T) {
	_, err := parseDSN("/mydb?token=secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "accountID is empty")
}

func TestParseDSNEmptyDatabaseID(t *testing.T) {
	_, err := parseDSN("myaccount/?token=secret")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "databaseID is empty")
}

func TestStringifyNamedParamsNil(t *testing.T) {
	args := []driver.NamedValue{
		{Ordinal: 1, Value: nil},
	}
	result, err := stringifyNamedParams(args)
	require.ErrorIs(t, err, errNullParamUnsupported)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "parameter 1")
}

func TestStringifyNamedParamsNilNotFirst(t *testing.T) {
	args := []driver.NamedValue{
		{Ordinal: 1, Value: "ok"},
		{Ordinal: 2, Value: nil},
	}
	result, err := stringifyNamedParams(args)
	require.ErrorIs(t, err, errNullParamUnsupported)
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
		value    bool
		expected string
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

func TestBuildBatchParameterOrdering(t *testing.T) {
	statements := []batchStatement{
		{query: "INSERT INTO a (x) VALUES (?)", params: []string{"a1"}},
		{query: "INSERT INTO b (x, y) VALUES (?, ?)", params: []string{"b1", "b2"}},
		{query: "INSERT INTO c (x) VALUES (?)", params: []string{"c1"}},
	}

	sql, params := buildBatch(statements)

	expectedSQL := "BEGIN;\n" +
		"INSERT INTO a (x) VALUES (?);\n" +
		"INSERT INTO b (x, y) VALUES (?, ?);\n" +
		"INSERT INTO c (x) VALUES (?);\n" +
		"COMMIT;"
	assert.Equal(t, expectedSQL, sql)

	assert.Equal(t, []string{"a1", "b1", "b2", "c1"}, params)
}

func TestBuildBatchEmptyParameters(t *testing.T) {
	statements := []batchStatement{
		{query: "DELETE FROM a", params: nil},
		{query: "DELETE FROM b", params: nil},
	}

	sql, params := buildBatch(statements)

	assert.Equal(t, "BEGIN;\nDELETE FROM a;\nDELETE FROM b;\nCOMMIT;", sql)
	assert.Empty(t, params)
}
