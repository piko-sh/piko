package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"querier_test_runner/db"
)

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

	err = queries.InsertTicket(ctx, db.InsertTicketParams{
		P1: "Fix login bug",
		P2: "high",
		P3: "open",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertTicket 1:", err)
		os.Exit(1)
	}

	err = queries.InsertTicket(ctx, db.InsertTicketParams{
		P1: "Update documentation",
		P2: "low",
		P3: "in_progress",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertTicket 2:", err)
		os.Exit(1)
	}

	err = queries.InsertTicket(ctx, db.InsertTicketParams{
		P1: "Database migration",
		P2: "high",
		P3: "open",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertTicket 3:", err)
		os.Exit(1)
	}

	err = queries.InsertTicket(ctx, db.InsertTicketParams{
		P1: "Add unit tests",
		P2: "medium",
		P3: "closed",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "InsertTicket 4:", err)
		os.Exit(1)
	}

	ticket, err := queries.GetTicket(ctx, int32(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "GetTicket:", err)
		os.Exit(1)
	}

	highPriority, err := queries.ListByPriority(ctx, "high")
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListByPriority:", err)
		os.Exit(1)
	}

	allTickets, err := queries.ListAll(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ListAll:", err)
		os.Exit(1)
	}

	result := map[string]any{
		"ticket":        ticket,
		"high_priority": highPriority,
		"all_tickets":   allTickets,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
