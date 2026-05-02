package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
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
		groupID       int64
		sample        float64
	}{
		{0, 1, 10.0},
		{10, 1, 12.0},
		{20, 1, 14.0},
		{30, 1, 16.0},
		{40, 1, 18.0},
		{0, 2, 5.0},
		{15, 2, 5.0},
		{30, 2, 5.0},
		{45, 2, 5.0},
	}
	for _, sample := range samples {
		err = queries.InsertMeasurement(ctx, db.InsertMeasurementParams{
			Ts:      baseline.Add(time.Duration(sample.offsetMinutes) * time.Minute),
			GroupID: sample.groupID,
			Sample:  sample.sample,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	rows, err := queries.StatsByGroup(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	type groupStats struct {
		GroupID     int64   `json:"group_id"`
		SampleCount int64   `json:"sample_count"`
		Mean        float64 `json:"mean_value"`
		Stddev      float64 `json:"stddev_value"`
	}
	out := []groupStats{}
	for _, row := range rows {
		var (
			count  int64
			mean   float64
			stddev float64
		)
		if row.SampleCount != nil {
			count = *row.SampleCount
		}
		if row.MeanValue != nil {
			mean = *row.MeanValue
		}
		if row.StddevValue != nil {
			stddev = *row.StddevValue
		}
		if math.IsNaN(stddev) {
			stddev = 0
		}
		out = append(out, groupStats{
			GroupID:     row.GroupID,
			SampleCount: count,
			Mean:        math.Round(mean*1000) / 1000,
			Stddev:      math.Round(stddev*1000) / 1000,
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"stats_by_group": out,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
