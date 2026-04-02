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

package querier_duckdb_test

import (
	"database/sql"
	"testing"

	_ "github.com/duckdb/duckdb-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openDuckDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("duckdb", "")
	require.NoError(t, err)
	t.Cleanup(func() { database.Close() })
	return database
}

func TestBasicCRUD(t *testing.T) {
	database := openDuckDB(t)

	_, err := database.Exec(`CREATE TABLE items (
		id INTEGER PRIMARY KEY,
		name VARCHAR NOT NULL,
		price DOUBLE NOT NULL
	)`)
	require.NoError(t, err)

	result, err := database.Exec(`INSERT INTO items (id, name, price) VALUES ($1, $2, $3)`, 1, "Widget", 9.99)
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	var id int
	var name string
	var price float64
	err = database.QueryRow(`SELECT id, name, price FROM items WHERE id = $1`, 1).Scan(&id, &name, &price)
	require.NoError(t, err)
	assert.Equal(t, 1, id)
	assert.Equal(t, "Widget", name)
	assert.InDelta(t, 9.99, price, 0.001)

	_, err = database.Exec(`UPDATE items SET price = $1 WHERE id = $2`, 12.50, 1)
	require.NoError(t, err)

	err = database.QueryRow(`SELECT price FROM items WHERE id = $1`, 1).Scan(&price)
	require.NoError(t, err)
	assert.InDelta(t, 12.50, price, 0.001)

	_, err = database.Exec(`DELETE FROM items WHERE id = $1`, 1)
	require.NoError(t, err)

	err = database.QueryRow(`SELECT id FROM items WHERE id = $1`, 1).Scan(&id)
	assert.Equal(t, sql.ErrNoRows, err)
}

func TestInsertReturning(t *testing.T) {
	database := openDuckDB(t)

	_, err := database.Exec(`CREATE TABLE products (
		id INTEGER PRIMARY KEY,
		name VARCHAR NOT NULL
	)`)
	require.NoError(t, err)

	var returnedID int
	var returnedName string
	err = database.QueryRow(
		`INSERT INTO products (id, name) VALUES ($1, $2) RETURNING id, name`,
		1, "Gadget",
	).Scan(&returnedID, &returnedName)
	require.NoError(t, err)
	assert.Equal(t, 1, returnedID)
	assert.Equal(t, "Gadget", returnedName)
}

func TestDuckDBTypes(t *testing.T) {
	database := openDuckDB(t)

	_, err := database.Exec(`CREATE TABLE type_test (
		id INTEGER PRIMARY KEY,
		big_val BIGINT,
		huge_val HUGEINT,
		bool_val BOOLEAN,
		float_val DOUBLE,
		varchar_val VARCHAR,
		blob_val BLOB,
		ts_val TIMESTAMP,
		date_val DATE,
		json_val JSON
	)`)
	require.NoError(t, err)

	_, err = database.Exec(
		`INSERT INTO type_test (id, big_val, huge_val, bool_val, float_val, varchar_val, ts_val, date_val, json_val)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		1, int64(9223372036854775807), "170141183460469231731687303715884105727",
		true, 3.14159, "hello", "2026-01-15 10:30:00", "2026-01-15",
		`{"key": "value"}`,
	)
	require.NoError(t, err)

	var bigVal int64
	var boolVal bool
	var floatVal float64
	var varcharVal string
	err = database.QueryRow(
		`SELECT big_val, bool_val, float_val, varchar_val FROM type_test WHERE id = $1`, 1,
	).Scan(&bigVal, &boolVal, &floatVal, &varcharVal)
	require.NoError(t, err)
	assert.Equal(t, int64(9223372036854775807), bigVal)
	assert.True(t, boolVal)
	assert.InDelta(t, 3.14159, floatVal, 0.0001)
	assert.Equal(t, "hello", varcharVal)
}

func TestJSONOperators(t *testing.T) {
	database := openDuckDB(t)

	_, err := database.Exec(`CREATE TABLE events (
		id INTEGER PRIMARY KEY,
		payload JSON NOT NULL
	)`)
	require.NoError(t, err)

	_, err = database.Exec(
		`INSERT INTO events (id, payload) VALUES ($1, $2)`,
		1, `{"name": "click", "count": 42}`,
	)
	require.NoError(t, err)

	var name string
	err = database.QueryRow(`SELECT payload->>'name' FROM events WHERE id = $1`, 1).Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "click", name)
}

func TestAggregateQueries(t *testing.T) {
	database := openDuckDB(t)

	_, err := database.Exec(`CREATE TABLE sales (
		id INTEGER PRIMARY KEY,
		amount DOUBLE NOT NULL
	)`)
	require.NoError(t, err)

	_, err = database.Exec(`INSERT INTO sales VALUES (1, 10.00), (2, 20.00), (3, 30.00)`)
	require.NoError(t, err)

	var count int
	err = database.QueryRow(`SELECT count(*) FROM sales`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	var total float64
	err = database.QueryRow(`SELECT sum(amount) FROM sales`).Scan(&total)
	require.NoError(t, err)
	assert.InDelta(t, 60.00, total, 0.01)
}

func TestTransactionSupport(t *testing.T) {
	database := openDuckDB(t)

	_, err := database.Exec(`CREATE TABLE accounts (
		id INTEGER PRIMARY KEY,
		balance DOUBLE NOT NULL
	)`)
	require.NoError(t, err)

	_, err = database.Exec(`INSERT INTO accounts VALUES (1, 100.00)`)
	require.NoError(t, err)

	transaction, err := database.Begin()
	require.NoError(t, err)

	_, err = transaction.Exec(`UPDATE accounts SET balance = balance - 50 WHERE id = 1`)
	require.NoError(t, err)

	err = transaction.Rollback()
	require.NoError(t, err)

	var balance float64
	err = database.QueryRow(`SELECT balance FROM accounts WHERE id = 1`).Scan(&balance)
	require.NoError(t, err)
	assert.InDelta(t, 100.00, balance, 0.01)

	transaction, err = database.Begin()
	require.NoError(t, err)

	_, err = transaction.Exec(`UPDATE accounts SET balance = balance - 25 WHERE id = 1`)
	require.NoError(t, err)

	err = transaction.Commit()
	require.NoError(t, err)

	err = database.QueryRow(`SELECT balance FROM accounts WHERE id = 1`).Scan(&balance)
	require.NoError(t, err)
	assert.InDelta(t, 75.00, balance, 0.01)
}
