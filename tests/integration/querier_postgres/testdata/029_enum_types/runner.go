package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")
	schemaName := os.Getenv("DATABASE_SCHEMA")

	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	acc1, err := queries.InsertAccount(ctx, db.InsertAccountParams{
		P1: "alice",
		P2: "active",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	acc2, err := queries.InsertAccount(ctx, db.InsertAccountParams{
		P1: "bob",
		P2: "inactive",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	acc3, err := queries.InsertAccount(ctx, db.InsertAccountParams{
		P1: "charlie",
		P2: "active",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	acc4, err := queries.InsertAccount(ctx, db.InsertAccountParams{
		P1: "diana",
		P2: "suspended",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	activeAccounts, err := queries.ListByStatus(ctx, "active")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	allAccounts, err := queries.ListAllAccounts(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	result := map[string]any{
		"acc_1":           acc1,
		"acc_2":           acc2,
		"acc_3":           acc3,
		"acc_4":           acc4,
		"active_accounts": activeAccounts,
		"all_accounts":    allAccounts,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
