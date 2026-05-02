package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"querier_test_runner/db"
)

func mustParseTime(layout string, value string) time.Time {
	parsed, err := time.Parse(layout, value)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return parsed
}


func main() {
	connectionString := os.Getenv("DATABASE_URL")

	conn, err := sql.Open("mysql", connectionString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx := context.Background()

	queries := db.New(conn)

	err = queries.InsertEvent(ctx, db.InsertEventParams{
		Title:     "Conference",
		EventDate: mustParseTime("2006-01-02", "2025-06-15"),
		StartTime: "09:00:00",
		CreatedAt: mustParseTime("2006-01-02 15:04:05", "2025-01-10 08:00:00"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertEvent 1:", err)
		os.Exit(1)
	}

	err = queries.InsertEvent(ctx, db.InsertEventParams{
		Title:     "Workshop",
		EventDate: mustParseTime("2006-01-02", "2025-06-20"),
		StartTime: "14:30:00",
		CreatedAt: mustParseTime("2006-01-02 15:04:05", "2025-01-10 12:00:00"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertEvent 2:", err)
		os.Exit(1)
	}

	err = queries.InsertEvent(ctx, db.InsertEventParams{
		Title:     "Meetup",
		EventDate: mustParseTime("2006-01-02", "2025-03-01"),
		StartTime: "18:00:00",
		CreatedAt: mustParseTime("2006-01-02 15:04:05", "2025-01-05 10:00:00"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertEvent 3:", err)
		os.Exit(1)
	}

	event, err := queries.GetEvent(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetEvent:", err)
		os.Exit(1)
	}

	formatted, err := queries.GetFormattedDate(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetFormattedDate:", err)
		os.Exit(1)
	}

	daysBetween, err := queries.GetDaysBetween(ctx, db.GetDaysBetweenParams{
		ID:  int32(1),
		ID2: int32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetDaysBetween:", err)
		os.Exit(1)
	}

	hoursBetween, err := queries.GetHoursBetween(ctx, db.GetHoursBetweenParams{
		ID:  int32(1),
		ID2: int32(2),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetHoursBetween:", err)
		os.Exit(1)
	}

	byDate, err := queries.ListByDate(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByDate:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"event":         event,
		"formatted":     formatted,
		"days_between":  daysBetween,
		"hours_between": hoursBetween,
		"by_date":       byDate,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
