package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"

	"querier_test_runner/db"
)

func main() {
	conn, err := sql.Open("duckdb", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, `CREATE TABLE events (id INTEGER PRIMARY KEY, name VARCHAR NOT NULL, data JSON NOT NULL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO events (id, name, data) VALUES
		(1, 'signup', '{"name": "signup", "category": "user", "user": {"email": "alice@example.com"}}'),
		(2, 'purchase', '{"name": "purchase", "category": "billing", "user": {"email": "bob@example.com"}}'),
		(3, 'login', '{"name": "login", "category": "user", "user": {"email": "charlie@example.com"}}')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	eventName, err := queries.GetEventName(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetEventName:", err)
		os.Exit(1)
	}

	nestedValue, err := queries.GetNestedValue(ctx, int32(2))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetNestedValue:", err)
		os.Exit(1)
	}

	byCategory, err := queries.ListByCategory(ctx, "user")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByCategory:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"get_event_name":   eventName,
		"get_nested_value": nestedValue,
		"list_by_category": byCategory,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
