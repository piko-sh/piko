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

func ptrString(s string) *string { return &s }
func ptrInt32(v int32) *int32    { return &v }

func main() {
	conn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TABLE employees (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		department TEXT NOT NULL,
		level INTEGER NOT NULL,
		active INTEGER NOT NULL DEFAULT 1
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO employees (id, name, department, level, active) VALUES
		(1, 'Alice', 'eng', 3, 1),
		(2, 'Bob', 'eng', 1, 1),
		(3, 'Charlie', 'sales', 2, 1),
		(4, 'Dave', 'eng', 2, 0),
		(5, 'Eve', 'sales', 3, 1)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	noFilter, err := queries.SearchEmployees(ctx, db.SearchEmployeesParams{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchEmployees (no filter):", err)
		os.Exit(1)
	}
	byDepartment, err := queries.SearchEmployees(ctx, db.SearchEmployeesParams{
		Department: ptrString("eng"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchEmployees (by department):", err)
		os.Exit(1)
	}
	byMinLevel, err := queries.SearchEmployees(ctx, db.SearchEmployeesParams{
		MinLevel: ptrInt32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchEmployees (by min level):", err)
		os.Exit(1)
	}
	byActive, err := queries.SearchEmployees(ctx, db.SearchEmployeesParams{
		Active: ptrInt32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchEmployees (by active):", err)
		os.Exit(1)
	}
	byDeptAndLevel, err := queries.SearchEmployees(ctx, db.SearchEmployeesParams{
		Department: ptrString("eng"),
		MinLevel:   ptrInt32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchEmployees (by dept and level):", err)
		os.Exit(1)
	}
	allFilters, err := queries.SearchEmployees(ctx, db.SearchEmployeesParams{
		Department: ptrString("eng"),
		MinLevel:   ptrInt32(2),
		Active:     ptrInt32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchEmployees (all filters):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"no_filter":         noFilter,
		"by_department":     byDepartment,
		"by_min_level":      byMinLevel,
		"by_active":         byActive,
		"by_dept_and_level": byDeptAndLevel,
		"all_filters":       allFilters,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
