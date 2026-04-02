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

package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// defaultWatchInterval is the default interval between watch updates.
	defaultWatchInterval = 2 * time.Second
)

var (
	// watchResourceList is the sorted, comma-separated list of available watch
	// resources, derived from the watchResources dispatch map.
	watchResourceList = buildResourceList(watchResources)

	// watchResources maps resource names to their watch handler functions.
	watchResources = map[string]func(ctx context.Context, conn monitoringConnection, w io.Writer, p *Printer, interval time.Duration) error{
		"health":    watchHealth,
		"tasks":     watchTasks,
		"artefacts": watchArtefacts,
		"metrics":   watchMetrics,
	}
)

// watchUpdate is the interface shared by all streaming watch
// update messages.
type watchUpdate interface {
	// GetTimestampMs returns the update timestamp in
	// milliseconds since the Unix epoch.
	GetTimestampMs() int64
}

// watchStream abstracts a gRPC server streaming receiver so
// that the shared streaming loop can work with different
// update types.
type watchStream[T watchUpdate] interface {
	// Recv receives the next update from the stream.
	Recv() (T, error)
}

// statusSummary provides a common interface for task and
// artefact summary types that share Status and Count fields.
type statusSummary interface {
	// GetStatus returns the status label for this summary
	// entry.
	GetStatus() string

	// GetCount returns the number of items with this status.
	GetCount() int64
}

// runWatch dispatches to the appropriate watch subcommand.
//
// Takes cc (*CommandContext) which provides the command execution context.
// Takes arguments ([]string) which specifies the resource type and optional flags.
//
// Returns error when the resource type is missing or unknown.
func runWatch(ctx context.Context, cc *CommandContext, arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("missing resource type\n\nAvailable resources: %s", watchResourceList)
	}

	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(cc.Stderr)
	interval := fs.Duration("interval", defaultWatchInterval, "Update interval")
	if err := fs.Parse(arguments[1:]); err != nil {
		return helpOrError(err)
	}

	resource := arguments[0]
	handler, ok := watchResources[resource]
	if !ok {
		return fmt.Errorf("unknown resource: %s\n\nAvailable resources: %s", resource, watchResourceList)
	}

	watchFormats := []string{"table", "json"}
	if err := validateOutputFormat(cc.Opts.Output, "watch", watchFormats); err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, false)
	return handler(ctx, cc.Conn, cc.Stdout, p, *interval)
}

// watchHealth streams health updates using the gRPC server streaming API.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which controls the output format.
// Takes interval (time.Duration) which sets the update frequency.
//
// Returns error when the stream cannot be started or an error occurs
// while receiving updates.
func watchHealth(ctx context.Context, conn monitoringConnection, w io.Writer, p *Printer, interval time.Duration) error {
	stream, err := conn.HealthClient().WatchHealth(ctx, &pb.WatchHealthRequest{
		IntervalMs: interval.Milliseconds(),
	})
	if err != nil {
		return grpcError("starting health watch", err)
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			if isStreamDone(err) {
				return nil
			}
			return fmt.Errorf("receiving health update: %w", err)
		}

		if p.IsJSON() {
			_ = p.PrintJSON(update)
			continue
		}

		clearScreen(w)
		_, _ = fmt.Fprintf(w, "=== Health (updated %s) ===\n\n", formatTimestamp(update.GetTimestampMs()))
		if update.GetLiveness() != nil {
			_, _ = fmt.Fprintln(w, "--- Liveness ---")
			printHealthTree(w, p, update.GetLiveness(), 0)
			_, _ = fmt.Fprintln(w)
		}
		if update.GetReadiness() != nil {
			_, _ = fmt.Fprintln(w, "--- Readiness ---")
			printHealthTree(w, p, update.GetReadiness(), 0)
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintln(w, "Press Ctrl+C to stop watching.")
	}
}

// streamWatch runs the shared receive-render loop for all streaming watch
// handlers.
//
// Takes stream (watchStream[T]) which provides the gRPC update stream.
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which controls output formatting.
// Takes resourceName (string) which is the label shown in the header.
// Takes render (func(io.Writer, *Printer, T)) which renders resource-specific
// content for each update.
//
// Returns error when the stream fails with a non-terminal error.
func streamWatch[T watchUpdate](
	stream watchStream[T],
	w io.Writer,
	p *Printer,
	resourceName string,
	render func(io.Writer, *Printer, T),
) error {
	for {
		update, err := stream.Recv()
		if err != nil {
			if isStreamDone(err) {
				return nil
			}
			return fmt.Errorf("receiving %s update: %w", resourceName, err)
		}

		if p.IsJSON() {
			_ = p.PrintJSON(update)
			continue
		}

		clearScreen(w)
		_, _ = fmt.Fprintf(w, "=== %s (updated %s) ===\n\n",
			resourceName, formatTimestamp(update.GetTimestampMs()))
		render(w, p, update)
		_, _ = fmt.Fprintln(w, "\nPress Ctrl+C to stop watching.")
	}
}

// watchTasks streams task updates using the gRPC server streaming API.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which handles output formatting and colouring.
// Takes interval (time.Duration) which sets the update frequency.
//
// Returns error when the stream cannot be started or fails during receive.
func watchTasks(ctx context.Context, conn monitoringConnection, w io.Writer, p *Printer, interval time.Duration) error {
	stream, err := conn.OrchestratorClient().WatchTasks(ctx, &pb.WatchTasksRequest{
		IntervalMs: interval.Milliseconds(),
	})
	if err != nil {
		return grpcError("starting task watch", err)
	}

	return streamWatch(stream, w, p, "Tasks", renderTaskUpdate)
}

// renderTaskUpdate renders the text output for a single task watch update.
//
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which provides colourised status formatting.
// Takes update (*pb.TasksUpdate) which contains the task data to render.
func renderTaskUpdate(w io.Writer, p *Printer, update *pb.TasksUpdate) {
	renderStatusSummaries(w, p, update.GetSummaries())
	renderRecentTasks(w, p, update.GetRecentTasks())
}

// renderStatusSummaries writes a status summary section for any slice of
// status/count pairs.
//
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which provides colourised status formatting.
// Takes summaries ([]T) which contains the status counts.
func renderStatusSummaries[T statusSummary](w io.Writer, p *Printer, summaries []T) {
	if len(summaries) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "--- Summary ---")
	for _, s := range summaries {
		_, _ = fmt.Fprintf(w, "  %s: %d\n", p.ColourisedStatus(s.GetStatus()), s.GetCount())
	}
	_, _ = fmt.Fprintln(w)
}

// renderRecentTasks writes the recent tasks table.
//
// Takes w (io.Writer) which receives the table header label.
// Takes p (*Printer) which renders the table.
// Takes tasks ([]*pb.TaskListItem) which contains the tasks to display.
func renderRecentTasks(w io.Writer, p *Printer, tasks []*pb.TaskListItem) {
	if len(tasks) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "--- Recent Tasks ---")
	headers := []string{"ID", "WORKFLOW", "EXECUTOR", "STATUS", "ATTEMPT", "UPDATED"}
	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		rows = append(rows, []string{
			t.GetId(),
			t.GetWorkflowId(),
			t.GetExecutor(),
			p.ColourisedStatus(t.GetStatus()),
			strconv.Itoa(int(t.GetAttempt())),
			formatTimestamp(t.GetUpdatedAt()),
		})
	}
	p.PrintTable(headers, rows)
}

// watchArtefacts streams artefact updates using the gRPC server streaming API.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which handles output formatting and colouring.
// Takes interval (time.Duration) which sets the update frequency.
//
// Returns error when the stream cannot be started or an unexpected error
// occurs during receive.
func watchArtefacts(ctx context.Context, conn monitoringConnection, w io.Writer, p *Printer, interval time.Duration) error {
	stream, err := conn.RegistryClient().WatchArtefacts(ctx, &pb.WatchArtefactsRequest{
		IntervalMs: interval.Milliseconds(),
	})
	if err != nil {
		return grpcError("starting artefact watch", err)
	}

	return streamWatch(stream, w, p, "Artefacts", renderArtefactUpdate)
}

// renderArtefactUpdate renders the text output for a single artefact watch
// update.
//
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which provides colourised status formatting.
// Takes update (*pb.ArtefactsUpdate) which contains the artefact data to render.
func renderArtefactUpdate(w io.Writer, p *Printer, update *pb.ArtefactsUpdate) {
	renderStatusSummaries(w, p, update.GetSummaries())
	renderRecentArtefacts(w, p, update.GetRecentArtefacts())
}

// renderRecentArtefacts writes the recent artefacts table.
//
// Takes w (io.Writer) which receives the table header label.
// Takes p (*Printer) which renders the table.
// Takes artefacts ([]*pb.ArtefactListItem) which contains the artefacts to
// display.
func renderRecentArtefacts(w io.Writer, p *Printer, artefacts []*pb.ArtefactListItem) {
	if len(artefacts) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "--- Recent Artefacts ---")
	headers := []string{"ID", "SOURCE PATH", "STATUS", "VARIANTS", "SIZE", "UPDATED"}
	rows := make([][]string, 0, len(artefacts))
	for _, a := range artefacts {
		rows = append(rows, []string{
			a.GetId(),
			a.GetSourcePath(),
			p.ColourisedStatus(a.GetStatus()),
			strconv.FormatInt(a.GetVariantCount(), 10),
			formatBytes(safeconv.Int64ToUint64(a.GetTotalSize())),
			formatTimestamp(a.GetUpdatedAt()),
		})
	}
	p.PrintTable(headers, rows)
}

// watchMetrics streams metric updates using the gRPC server streaming API.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes w (io.Writer) which receives the formatted output.
// Takes p (*Printer) which handles output formatting.
// Takes interval (time.Duration) which sets the update frequency.
//
// Returns error when the stream cannot be started or an error occurs
// while receiving updates.
func watchMetrics(ctx context.Context, conn monitoringConnection, w io.Writer, p *Printer, interval time.Duration) error {
	stream, err := conn.MetricsClient().WatchMetrics(ctx, &pb.WatchMetricsRequest{
		IntervalMs: interval.Milliseconds(),
	})
	if err != nil {
		return grpcError("starting metrics watch", err)
	}

	for {
		update, err := stream.Recv()
		if err != nil {
			if isStreamDone(err) {
				return nil
			}
			return fmt.Errorf("receiving metrics update: %w", err)
		}

		if p.IsJSON() {
			_ = p.PrintJSON(update)
			continue
		}

		clearScreen(w)
		_, _ = fmt.Fprintf(w, "=== Metrics (updated %s) ===\n\n", formatTimestamp(update.GetTimestampMs()))

		headers := []string{"NAME", "TYPE", "UNIT", "DATA POINTS"}
		rows := make([][]string, 0, len(update.GetMetrics()))
		for _, m := range update.GetMetrics() {
			rows = append(rows, []string{
				m.GetName(),
				m.GetType(),
				m.GetUnit(),
				strconv.Itoa(len(m.GetDataPoints())),
			})
		}
		p.PrintTable(headers, rows)
		_, _ = fmt.Fprintln(w, "\nPress Ctrl+C to stop watching.")
	}
}

// clearScreen writes ANSI escape sequences to clear the terminal.
//
// Takes w (io.Writer) which receives the escape sequences.
func clearScreen(w io.Writer) {
	_, _ = fmt.Fprint(w, "\033[H\033[2J")
}

// isStreamDone reports whether the error indicates the stream has ended.
//
// Takes err (error) which is the error to check.
//
// Returns bool which is true when the error is io.EOF or context.Canceled.
func isStreamDone(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, context.Canceled)
}
