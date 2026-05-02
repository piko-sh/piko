package main

import (
	"context"
	"database/sql"
	"encoding/json"
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

	queries := db.New(conn)
	err = queries.InsertDepartment(ctx, db.InsertDepartmentParams{
		ID:   int32(1),
		Name: "Engineering",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert dept 1:", err)
		os.Exit(1)
	}

	err = queries.InsertDepartment(ctx, db.InsertDepartmentParams{
		ID:   int32(2),
		Name: "Sales",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert dept 2:", err)
		os.Exit(1)
	}
	err = queries.InsertEmployee(ctx, db.InsertEmployeeParams{
		ID:     int32(1),
		Name:   "Alice",
		DeptID: int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert emp 1:", err)
		os.Exit(1)
	}

	err = queries.InsertEmployee(ctx, db.InsertEmployeeParams{
		ID:     int32(2),
		Name:   "Bob",
		DeptID: int32(1),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert emp 2:", err)
		os.Exit(1)
	}

	err = queries.InsertEmployee(ctx, db.InsertEmployeeParams{
		ID:     int32(3),
		Name:   "Charlie",
		DeptID: int32(2),
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
