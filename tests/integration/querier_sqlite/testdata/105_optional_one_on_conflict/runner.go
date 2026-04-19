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

	_, err = conn.ExecContext(ctx, `CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT NOT NULL UNIQUE)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	firstRow, firstOK, err := queries.InsertUserReturning(ctx, "a@b.com")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	secondRow, secondOK, secondErr := queries.InsertUserReturning(ctx, "a@b.com")
	if secondErr != nil {
		fmt.Fprintln(os.Stderr, secondErr)
		os.Exit(1)
	}

	result := map[string]any{
		"first_ok":       firstOK,
		"first_email":    firstRow.Email,
		"second_ok":      secondOK,
		"second_email":   secondRow.Email,
		"second_err_nil": secondErr == nil,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
