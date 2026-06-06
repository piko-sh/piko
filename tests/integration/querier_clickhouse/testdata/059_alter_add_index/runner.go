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

	seeds := []db.InsertMetricParams{
		{ID: 1, Value: 10},
		{ID: 2, Value: 20},
		{ID: 3, Value: 30},
		{ID: 4, Value: 40},
		{ID: 5, Value: 50},
		{ID: 6, Value: 60},
		{ID: 7, Value: 70},
		{ID: 8, Value: 80},
		{ID: 9, Value: 90},
		{ID: 10, Value: 100},
	}
	for _, seed := range seeds {
		if err := queries.InsertMetric(ctx, seed); err != nil {
			return fmt.Errorf("insert metric %d: %w", seed.ID, err)
		}
	}

	rows, err := queries.FilterByValue(ctx, db.FilterByValueParams{
		Lower: 30,
		Upper: 70,
	})
	if err != nil {
		return fmt.Errorf("filter by value: %w", err)
	}

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"rows": rows,
	})
}
