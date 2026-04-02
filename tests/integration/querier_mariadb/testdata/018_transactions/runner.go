package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")

	conn, err := sql.Open("mysql", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `INSERT INTO accounts (id, name, balance) VALUES
		(1, 'Alice', 100),
		(2, 'Bob', 200)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	err = queries.RunInTx(ctx, conn, func(txQueries *db.Queries) error {
		err := txQueries.UpdateBalance(ctx, db.UpdateBalanceParams{
			P1: int32(50),
			P2: int32(1),
		})
		if err != nil {
			return err
		}
		return txQueries.UpdateBalance(ctx, db.UpdateBalanceParams{
			P1: int32(250),
			P2: int32(2),
		})
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "RunInTx commit:", err)
		os.Exit(1)
	}

	afterCommit, err := queries.ListAccounts(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListAccounts after commit:", err)
		os.Exit(1)
	}
	err = queries.RunInTx(ctx, conn, func(txQueries *db.Queries) error {
		err := txQueries.UpdateBalance(ctx, db.UpdateBalanceParams{
			P1: int32(0),
			P2: int32(1),
		})
		if err != nil {
			return err
		}
		return errors.New("simulated failure")
	})
	if err == nil {
		fmt.Fprintln(os.Stderr, "expected error from failed transaction")
		os.Exit(1)
	}

	afterRollback, err := queries.ListAccounts(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListAccounts after rollback:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"after_commit":   afterCommit,
		"after_rollback": afterRollback,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
