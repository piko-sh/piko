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

	_, err = conn.ExecContext(ctx, `CREATE TABLE sales (
		id INTEGER PRIMARY KEY,
		employee TEXT NOT NULL,
		amount INTEGER NOT NULL,
		sale_date TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO sales (id, employee, amount, sale_date) VALUES
		(1, 'Alice', 100, '2025-01'),
		(2, 'Bob', 200, '2025-02'),
		(3, 'Alice', 150, '2025-03'),
		(4, 'Bob', 50, '2025-04'),
		(5, 'Alice', 300, '2025-05'),
		(6, 'Bob', 175, '2025-06')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	runningTotal, err := queries.ListWithRunningTotal(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListWithRunningTotal:", err)
		os.Exit(1)
	}

	lagLead, err := queries.ListWithLagLead(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListWithLagLead:", err)
		os.Exit(1)
	}

	rankByEmployee, err := queries.ListWithRankByEmployee(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListWithRankByEmployee:", err)
		os.Exit(1)
	}

	ntile, err := queries.ListWithNtile(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListWithNtile:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"running_total":    runningTotal,
		"lag_lead":         lagLead,
		"rank_by_employee": rankByEmployee,
		"ntile":            ntile,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
