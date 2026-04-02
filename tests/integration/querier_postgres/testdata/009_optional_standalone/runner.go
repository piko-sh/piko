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

func ptrString(s string) *string { return &s }

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

	_, err = conn.ExecContext(ctx, `INSERT INTO users (name, email, role) VALUES
		('Alice', 'alice@test.com', 'admin'),
		('Bob', 'bob@test.com', 'user'),
		('Alice', 'alice2@test.com', 'user'),
		('Charlie', 'charlie@test.com', 'admin')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	noFilter, err := queries.SearchUsers(ctx, db.SearchUsersParams{})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchUsers (no filter):", err)
		os.Exit(1)
	}
	byName, err := queries.SearchUsers(ctx, db.SearchUsersParams{
		Name: ptrString("Alice"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchUsers (by name):", err)
		os.Exit(1)
	}
	byRole, err := queries.SearchUsers(ctx, db.SearchUsersParams{
		Role: ptrString("admin"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchUsers (by role):", err)
		os.Exit(1)
	}
	byNameAndRole, err := queries.SearchUsers(ctx, db.SearchUsersParams{
		Name: ptrString("Alice"),
		Role: ptrString("user"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "SearchUsers (by name and role):", err)
		os.Exit(1)
	}

	result := map[string]any{
		"no_filter":        noFilter,
		"by_name":          byName,
		"by_role":          byRole,
		"by_name_and_role": byNameAndRole,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
