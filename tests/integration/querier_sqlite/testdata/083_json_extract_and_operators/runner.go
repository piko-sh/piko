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

	_, err = conn.ExecContext(ctx, `CREATE TABLE events (id INTEGER PRIMARY KEY, name TEXT NOT NULL, data TEXT NOT NULL)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO events (id, name, data) VALUES
		(1, 'signup', '{"name":"Alice","category":"auth","user":{"email":"alice@test.com"}}'),
		(2, 'purchase', '{"name":"Bob","category":"sales","amount":99.99,"user":{"email":"bob@test.com"}}'),
		(3, 'login', '{"name":"Charlie","category":"auth","user":{"email":"charlie@test.com"}}')`)
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

	byCategory, err := queries.ListByCategory(ctx, "auth")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByCategory:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"event_name":   eventName,
		"nested_value": nestedValue,
		"by_category":  byCategory,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
