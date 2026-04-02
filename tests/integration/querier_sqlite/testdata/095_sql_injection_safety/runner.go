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

	_, err = conn.ExecContext(ctx, `CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		role TEXT NOT NULL,
		secret TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `INSERT INTO users (id, name, role, secret) VALUES
		(1, 'Alice', 'admin', 'hunter2'),
		(2, 'Bob', 'user', 'password123'),
		(3, 'Charlie', 'user', 'qwerty')`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)
	result := map[string]any{}

	normalQuery, err := queries.SearchUsers(ctx).Where("role", "=", "admin").All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "normal query:", err)
		os.Exit(1)
	}
	result["normal_query"] = normalQuery

	tautologyResult, err := queries.SearchUsers(ctx).Where("name", "=", "' OR '1'='1").All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tautology:", err)
		os.Exit(1)
	}
	result["value_tautology"] = tautologyResult

	unionResult, err := queries.SearchUsers(ctx).Where("name", "=", "' UNION SELECT id, secret, role FROM users --").All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "union:", err)
		os.Exit(1)
	}
	result["value_union_injection"] = unionResult

	dropResult, err := queries.SearchUsers(ctx).Where("name", "=", "'; DROP TABLE users; --").All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "drop:", err)
		os.Exit(1)
	}
	result["value_drop_injection"] = dropResult

	afterDropCheck, err := queries.SearchUsers(ctx).Where("role", "=", "admin").All(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "post-drop check:", err)
		os.Exit(1)
	}
	result["table_still_exists"] = afterDropCheck

	result["where_column_injection"] = expectPanic(func() {
		queries.SearchUsers(ctx).Where("id; DROP TABLE users; --", "=", 1).All(ctx)
	})

	result["where_column_subquery"] = expectPanic(func() {
		queries.SearchUsers(ctx).Where("(SELECT secret FROM users LIMIT 1)", "=", 1).All(ctx)
	})

	result["where_operator_injection"] = expectPanic(func() {
		queries.SearchUsers(ctx).Where("id", "= 1 OR 1=1; --", 1).All(ctx)
	})

	result["orderby_column_injection"] = expectPanic(func() {
		queries.SearchUsers(ctx).OrderBy("id; DROP TABLE users; --", "ASC").All(ctx)
	})

	result["orderby_column_subquery"] = expectPanic(func() {
		queries.SearchUsers(ctx).OrderBy("(CASE WHEN (SELECT 1) THEN id ELSE name END)", "ASC").All(ctx)
	})

	result["orderby_direction_injection"] = expectPanic(func() {
		queries.SearchUsers(ctx).OrderBy("id", "ASC; DROP TABLE users; --").All(ctx)
	})

	result["orderby_direction_subquery"] = expectPanic(func() {
		queries.SearchUsers(ctx).OrderBy("id", "ASC, (SELECT secret FROM users LIMIT 1)").All(ctx)
	})

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func expectPanic(function func()) map[string]any {
	recovered := false
	message := ""

	func() {
		defer func() {
			if recovery := recover(); recovery != nil {
				recovered = true
				message = fmt.Sprintf("%v", recovery)
			}
		}()
		function()
	}()

	return map[string]any{
		"panicked": recovered,
		"message":  message,
	}
}
