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
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"querier_test_runner/db"
)

func main() {
	connectionString := os.Getenv("DATABASE_URL")
	schemaName := os.Getenv("DATABASE_SCHEMA")

	conn, err := sql.Open("pgx", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	baseline, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")
	samples := []struct {
		offsetMinutes int
		source        int64
		value         float64
	}{
		{0, 1, 10.0},
		{30, 1, 12.0},
		{75, 1, 14.0},
		{15, 2, 5.0},
		{90, 2, 8.0},
	}
	for _, sample := range samples {
		value := sample.value
		err = queries.InsertPulse(ctx, db.InsertPulseParams{
			Ts:     baseline.Add(time.Duration(sample.offsetMinutes) * time.Minute),
			Source: sample.source,
			Value:  &value,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	rowCount, err := queries.CountPulses(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if _, err := conn.ExecContext(ctx,
		fmt.Sprintf("CALL refresh_continuous_aggregate('%s.hourly_pulses', NULL, NULL)", schemaName),
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	rows, err := conn.QueryContext(ctx, `
		SELECT bucket, source, total_value, sample_count
		FROM hourly_pulses
		ORDER BY bucket, source
	`)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer rows.Close()

	type bucketEntry struct {
		BucketISO   string  `json:"bucket"`
		Source      int64   `json:"source"`
		TotalValue  float64 `json:"total_value"`
		SampleCount int64   `json:"sample_count"`
	}
	entries := []bucketEntry{}
	for rows.Next() {
		var bucket time.Time
		var source, sampleCount int64
		var total sql.NullFloat64
		if scanErr := rows.Scan(&bucket, &source, &total, &sampleCount); scanErr != nil {
			fmt.Fprintln(os.Stderr, scanErr)
			os.Exit(1)
		}
		entries = append(entries, bucketEntry{
			BucketISO:   bucket.UTC().Format(time.RFC3339),
			Source:      source,
			TotalValue:  total.Float64,
			SampleCount: sampleCount,
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"raw_row_count": rowCount,
		"hourly_cagg":   entries,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
