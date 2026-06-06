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
		dsn = dsn + separator + "allow_experimental_lightweight_delete=1&mutations_sync=2"
	}
	conn, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer conn.Close()

	ctx := context.Background()
	queries := db.New(conn)

	const bigID = uint64(18_000_000_000_000_000_000)
	metrics := []db.InsertMetricParams{
		{ID: 1, Sequence: 5_000_000_000, Value: 7},
		{ID: 2, Sequence: -5_000_000_000, Value: 8},
		{ID: bigID, Sequence: 9_000_000_000, Value: 9},
	}
	for index, metric := range metrics {
		if err := queries.InsertMetric(ctx, metric); err != nil {
			return fmt.Errorf("insert %d: %w", index, err)
		}
	}

	big, err := queries.GetMetric(ctx, bigID)
	if err != nil {
		return fmt.Errorf("get big metric: %w", err)
	}

	before, err := queries.CountMetrics(ctx)
	if err != nil {
		return fmt.Errorf("count before: %w", err)
	}

	if err := queries.PruneMetricsBelow(ctx, uint64(2)); err != nil {
		return fmt.Errorf("prune: %w", err)
	}

	after, err := queries.CountMetrics(ctx)
	if err != nil {
		return fmt.Errorf("count after: %w", err)
	}

	var _ int64 = before.Total

	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"big_id":       big.ID,
		"big_sequence": big.Sequence,
		"big_value":    big.Value,
		"total_before": before.Total,
		"total_after":  after.Total,
	})
}
