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
		sensorID      int64
		value         float64
	}{
		{0, 1, 10.0},
		{30, 1, 15.0},
		{60, 1, 12.0},
		{0, 2, 100.0},
		{45, 2, 110.0},
		{90, 2, 105.0},
	}
	for _, sample := range samples {
		err = queries.InsertSensorReading(ctx, db.InsertSensorReadingParams{
			Ts:       baseline.Add(time.Duration(sample.offsetMinutes) * time.Minute),
			SensorID: sample.sensorID,
			Value:    sample.value,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	rows, err := queries.FirstLastBySensor(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	type firstLast struct {
		SensorID   int64   `json:"sensor_id"`
		FirstValue float64 `json:"first_value"`
		LastValue  float64 `json:"last_value"`
	}
	out := []firstLast{}
	for _, row := range rows {
		out = append(out, firstLast{
			SensorID:   row.SensorID,
			FirstValue: asFloat64(row.FirstValue),
			LastValue:  asFloat64(row.LastValue),
		})
	}

	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{
		"first_last_by_sensor": out,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func asFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case *float64:
		if typed == nil {
			return 0
		}
		return *typed
	case nil:
		return 0
	}
	return 0
}
