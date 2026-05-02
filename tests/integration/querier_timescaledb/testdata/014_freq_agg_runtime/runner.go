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

	_, err = conn.ExecContext(ctx, fmt.Sprintf("SET search_path TO %s, toolkit_experimental, public", schemaName))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	queries := db.New(conn)

	baseline, _ := time.Parse(time.RFC3339, "2026-01-01T00:00:00Z")

	deviceSamples := []struct {
		device int64
		count  int
	}{
		{1, 100},
		{2, 50},
		{3, 20},
		{4, 10},
		{5, 5},
		{6, 2},
		{7, 1},
	}
	rowOffset := 0
	for _, sample := range deviceSamples {
		for index := 0; index < sample.count; index++ {
			err = queries.InsertDeviceEvent(ctx, db.InsertDeviceEventParams{
				Ts:       baseline.Add(time.Duration(rowOffset) * time.Second),
				DeviceID: sample.device,
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			rowOffset++
		}
	}

	rows, err := queries.TopDevices(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	type topEntry struct {
		DeviceID int64   `json:"device_id"`
		MaxFreq  float64 `json:"max_freq"`
		MinFreq  float64 `json:"min_freq"`
	}
	derefFloat := func(value *float64) float64 {
		if value == nil {
			return 0
		}
		return *value
	}
	derefInt := func(value *int64) int64 {
		if value == nil {
			return 0
		}
		return *value
	}
	out := []topEntry{}
	for _, row := range rows {
		out = append(out, topEntry{
			DeviceID: derefInt(row.DeviceID),
			MaxFreq:  derefFloat(row.MaxFreq),
			MinFreq:  derefFloat(row.MinFreq),
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"top_devices": out,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
