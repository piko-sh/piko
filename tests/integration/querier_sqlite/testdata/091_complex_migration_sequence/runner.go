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
		email TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration 001:", err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'user'`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration 002a:", err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `ALTER TABLE users ADD COLUMN created_at TEXT NOT NULL DEFAULT '2025-01-01'`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration 002b:", err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE INDEX idx_users_email ON users (email)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration 003a:", err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE INDEX idx_users_role ON users (role)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration 003b:", err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE TABLE posts (
		id INTEGER PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id),
		title TEXT NOT NULL,
		body TEXT NOT NULL,
		created_at TEXT NOT NULL DEFAULT '2025-01-01'
	)`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration 004:", err)
		os.Exit(1)
	}

	_, err = conn.ExecContext(ctx, `CREATE VIEW user_post_counts AS
		SELECT u.id, u.name, u.role, COUNT(p.id) AS post_count
		FROM users u
		LEFT JOIN posts p ON p.user_id = u.id
		GROUP BY u.id`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration 005:", err)
		os.Exit(1)
	}

	queries := db.New(conn)
	err = queries.InsertUser(ctx, db.InsertUserParams{
		P1: int32(1),
		P2: "Alice",
		P3: "alice@example.com",
		P4: "admin",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert user 1:", err)
		os.Exit(1)
	}

	err = queries.InsertUser(ctx, db.InsertUserParams{
		P1: int32(2),
		P2: "Bob",
		P3: "bob@example.com",
		P4: "user",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert user 2:", err)
		os.Exit(1)
	}

	err = queries.InsertUser(ctx, db.InsertUserParams{
		P1: int32(3),
		P2: "Charlie",
		P3: "charlie@example.com",
		P4: "user",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert user 3:", err)
		os.Exit(1)
	}
	err = queries.InsertPost(ctx, db.InsertPostParams{
		P1: int32(1),
		P2: int32(1),
		P3: "First Post",
		P4: "Hello world",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert post 1:", err)
		os.Exit(1)
	}

	err = queries.InsertPost(ctx, db.InsertPostParams{
		P1: int32(2),
		P2: int32(1),
		P3: "Second Post",
		P4: "More content",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert post 2:", err)
		os.Exit(1)
	}

	err = queries.InsertPost(ctx, db.InsertPostParams{
		P1: int32(3),
		P2: int32(2),
		P3: "Bob's Post",
		P4: "Bob writes here",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert post 3:", err)
		os.Exit(1)
	}

	err = queries.InsertPost(ctx, db.InsertPostParams{
		P1: int32(4),
		P2: int32(2),
		P3: "Another from Bob",
		P4: "More from Bob",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert post 4:", err)
		os.Exit(1)
	}

	err = queries.InsertPost(ctx, db.InsertPostParams{
		P1: int32(5),
		P2: int32(3),
		P3: "Charlie's Post",
		P4: "Charlie joins in",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "insert post 5:", err)
		os.Exit(1)
	}
	postCounts, err := queries.GetUserPostCounts(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetUserPostCounts:", err)
		os.Exit(1)
	}
	regularUsers, err := queries.ListUsersByRole(ctx, "user")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListUsersByRole:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"post_counts":   postCounts,
		"regular_users": regularUsers,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
