// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"querier_clickhouse_runner/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	conn, err := sql.Open("clickhouse", os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	ctx := context.Background()
	queries := db.New(conn)

	users := []db.InsertUserParams{
		{ID: 1, Name: "alice"},
		{ID: 2, Name: "bob"},
	}
	for _, user := range users {
		if err := queries.InsertUser(ctx, user); err != nil {
			return fmt.Errorf("insert user %d: %w", user.ID, err)
		}
	}

	sessions := []db.InsertSessionParams{
		{ID: 10, UserID: 1, Duration: 30},
		{ID: 11, UserID: 1, Duration: 60},
		{ID: 12, UserID: 2, Duration: 45},
	}
	for _, session := range sessions {
		if err := queries.InsertSession(ctx, session); err != nil {
			return fmt.Errorf("insert session %d: %w", session.ID, err)
		}
	}

	out, err := queries.SessionsByUser(ctx)
	if err != nil {
		return fmt.Errorf("sessions by user: %w", err)
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"rows": out,
	})
}
