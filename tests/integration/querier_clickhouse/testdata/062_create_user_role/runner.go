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

	if err := queries.InsertNoop(ctx, 1); err != nil {
		return fmt.Errorf("insert noop: %w", err)
	}

	userCount, err := queries.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	roleCount, err := queries.CountRoles(ctx)
	if err != nil {
		return fmt.Errorf("count roles: %w", err)
	}

	if _, err := conn.ExecContext(ctx, "DROP USER IF EXISTS test_user"); err != nil {
		return fmt.Errorf("drop user: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "DROP ROLE IF EXISTS test_reader"); err != nil {
		return fmt.Errorf("drop role: %w", err)
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"user_count": userCount.UserCount,
		"role_count": roleCount.RoleCount,
	})
}
