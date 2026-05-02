package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "modernc.org/sqlite"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TABLE accounts (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		balance INTEGER NOT NULL CHECK (balance >= 0),
		status TEXT NOT NULL CHECK (status IN ('active', 'inactive', 'suspended'))
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	err = queries.InsertAccount(ctx, db.InsertAccountParams{
		ID:      int64(1),
		Name:    "Alice",
		Balance: int32(100),
		Status:  "active",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert Alice:", err)
		os.Exit(1)
	}

	err = queries.InsertAccount(ctx, db.InsertAccountParams{
		ID:      int64(2),
		Name:    "Bob",
		Balance: int32(0),
		Status:  "inactive",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert Bob:", err)
		os.Exit(1)
	}
	negativeBalanceError := false
	err = queries.InsertAccount(ctx, db.InsertAccountParams{
		ID:      int64(3),
		Name:    "Charlie",
		Balance: int32(-50),
		Status:  "active",
	})
	if err != nil {
		negativeBalanceError = true
	}
	invalidStatusError := false
	err = queries.InsertAccount(ctx, db.InsertAccountParams{
		ID:      int64(4),
		Name:    "Diana",
		Balance: int32(200),
		Status:  "banned",
	})
	if err != nil {
		invalidStatusError = true
	}
	validAccount, err := queries.GetAccount(ctx, int64(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetAccount:", err)
		os.Exit(1)
	}
	activeAccounts, err := queries.ListActive(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListActive:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"valid_account":          validAccount,
		"insert_count":           2,
		"negative_balance_error": negativeBalanceError,
		"invalid_status_error":   invalidStatusError,
		"active_accounts":        activeAccounts,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
