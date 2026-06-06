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
	"strings"

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
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		separator := "?"
		if strings.Contains(dsn, "?") {
			separator = "&"
		}
		dsn = dsn + separator + "mutations_sync=2"
	}
	conn, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	ctx := context.Background()
	queries := db.New(conn)

	if err := queries.InsertEntity(ctx, db.InsertEntityParams{ID: 1, Label: "original"}); err != nil {
		return fmt.Errorf("insert 1: %w", err)
	}
	if err := queries.InsertEntity(ctx, db.InsertEntityParams{ID: 2, Label: "untouched"}); err != nil {
		return fmt.Errorf("insert 2: %w", err)
	}
	if err := queries.UpdateLabel(ctx, db.UpdateLabelParams{NewLabel: "renamed", ID: 1}); err != nil {
		return fmt.Errorf("alter update: %w", err)
	}

	rows, err := queries.All(ctx)
	if err != nil {
		return fmt.Errorf("select: %w", err)
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"rows": rows,
	})
}
