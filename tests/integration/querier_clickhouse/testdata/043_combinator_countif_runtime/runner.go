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

	rows := []db.InsertEventParams{
		{ID: 1, Category: "sale", Amount: 100, UserID: 10},
		{ID: 2, Category: "sale", Amount: 250, UserID: 10},
		{ID: 3, Category: "sale", Amount: 75, UserID: 11},
		{ID: 4, Category: "refund", Amount: 30, UserID: 12},
		{ID: 5, Category: "view", Amount: 0, UserID: 13},
	}
	for _, row := range rows {
		if err := queries.InsertEvent(ctx, row); err != nil {
			return fmt.Errorf("insert event %d: %w", row.ID, err)
		}
	}

	row, err := queries.FilteredCounts(ctx)
	if err != nil {
		return fmt.Errorf("filtered counts: %w", err)
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"sale_count": row.SaleCount,
		"sale_total": row.SaleTotal,
		"sale_users": row.SaleUsers,
	})
}
