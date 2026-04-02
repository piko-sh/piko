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
	_, err = conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "PRAGMA foreign_keys:", err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE TABLE departments (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE TABLE employees (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		dept_id INTEGER NOT NULL REFERENCES departments(id) ON DELETE CASCADE ON UPDATE CASCADE
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	err = queries.InsertDepartment(ctx, db.InsertDepartmentParams{
		P1: int32(1),
		P2: "Engineering",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert dept 1:", err)
		os.Exit(1)
	}

	err = queries.InsertDepartment(ctx, db.InsertDepartmentParams{
		P1: int32(2),
		P2: "Sales",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert dept 2:", err)
		os.Exit(1)
	}
	err = queries.InsertEmployee(ctx, db.InsertEmployeeParams{
		P1: int32(1),
		P2: "Alice",
		P3: int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert emp 1:", err)
		os.Exit(1)
	}

	err = queries.InsertEmployee(ctx, db.InsertEmployeeParams{
		P1: int32(2),
		P2: "Bob",
		P3: int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert emp 2:", err)
		os.Exit(1)
	}

	err = queries.InsertEmployee(ctx, db.InsertEmployeeParams{
		P1: int32(3),
		P2: "Charlie",
		P3: int32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert emp 3:", err)
		os.Exit(1)
	}
	beforeDelete, err := queries.ListEmployees(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListEmployees before:", err)
		os.Exit(1)
	}
	err = queries.DeleteDepartment(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "DeleteDepartment:", err)
		os.Exit(1)
	}
	afterDelete, err := queries.ListEmployees(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListEmployees after:", err)
		os.Exit(1)
	}
	countAfter, err := queries.CountEmployees(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "CountEmployees:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"before_delete": beforeDelete,
		"after_delete":  afterDelete,
		"count_after":   countAfter,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
