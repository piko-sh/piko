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
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// getFormatHelp is the format documentation shown in get command help.
	getFormatHelp = `table, wide, json`

	// getDefaultFormat is the default output format for get commands.
	getDefaultFormat = "table"

	// getDefaultLimit is the default item limit for get commands.
	getDefaultLimit = 20

	// getTruncateLen is the default truncation length used in table rendering.
	getTruncateLen = 3
)

var (
	// getFormats lists the output formats supported by the get command.
	getFormats = []string{"table", "wide", "json"}

	// getResourceList is the sorted, comma-separated list of available get
	// resources, derived from the getResources dispatch map.
	getResourceList = buildResourceList(getResources)

	// getResources maps resource names to their handler functions.
	getResources = map[string]func(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error{
		"health":      getHealth,
		"tasks":       getTasks,
		"workflows":   getWorkflows,
		"artefacts":   getArtefacts,
		"variants":    getVariants,
		"metrics":     getMetrics,
		"traces":      getTraces,
		"resources":   getOpenResources,
		"dlq":         getDLQ,
		"ratelimiter": getRateLimiter,
		"providers":   getProviders,
	}

	// healthColumns defines the column layout for each resource type.
	healthColumns = []Column{
		{Header: "PROBE"},
		{Header: "STATE"},
		{Header: "READY"},
		{Header: "MESSAGE"},
		{Header: "DURATION", WideOnly: true},
		{Header: "TIMESTAMP", WideOnly: true},
	}

	taskColumns = []Column{
		{Header: "ID"},
		{Header: "WORKFLOW"},
		{Header: "EXECUTOR"},
		{Header: "STATUS"},
		{Header: "ATTEMPT"},
		{Header: "UPDATED"},
		{Header: "PRIORITY", WideOnly: true},
		{Header: "LAST ERROR", WideOnly: true},
		{Header: "CREATED", WideOnly: true},
	}

	workflowColumns = []Column{
		{Header: "WORKFLOW ID"},
		{Header: "TASKS"},
		{Header: "COMPLETE"},
		{Header: "FAILED"},
		{Header: "ACTIVE"},
		{Header: "UPDATED"},
		{Header: "CREATED", WideOnly: true},
	}

	artefactColumns = []Column{
		{Header: "ID"},
		{Header: "SOURCE PATH"},
		{Header: "STATUS"},
		{Header: "VARIANTS"},
		{Header: "SIZE"},
		{Header: "UPDATED"},
		{Header: "CREATED", WideOnly: true},
	}

	variantColumns = []Column{
		{Header: "STATUS"},
		{Header: "COUNT"},
	}

	metricColumns = []Column{
		{Header: "NAME"},
		{Header: "TYPE"},
		{Header: "DATA POINTS"},
		{Header: "UNIT", WideOnly: true},
		{Header: "DESCRIPTION", WideOnly: true},
	}

	traceColumns = []Column{
		{Header: "TRACE ID"},
		{Header: "SPAN ID"},
		{Header: "NAME"},
		{Header: "SERVICE"},
		{Header: "STATUS"},
		{Header: "DURATION"},
		{Header: "KIND", WideOnly: true},
		{Header: "START", WideOnly: true},
		{Header: "STATUS MESSAGE", WideOnly: true},
	}

	resourceColumns = []Column{
		{Header: "CATEGORY"},
		{Header: "COUNT"},
	}

	dlqSummaryColumns = []Column{
		{Header: "TYPE"},
		{Header: "QUEUED"},
		{Header: "DLQ"},
		{Header: "PROCESSED"},
		{Header: "FAILED"},
		{Header: "RETRY", WideOnly: true},
		{Header: "SUCCESSFUL", WideOnly: true},
		{Header: "RETRIES", WideOnly: true},
		{Header: "UPTIME", WideOnly: true},
	}

	dlqEntryColumns = []Column{
		{Header: "ID"},
		{Header: "TYPE"},
		{Header: "ERROR"},
		{Header: "ATTEMPTS"},
		{Header: "ADDED"},
		{Header: "FIRST ATTEMPT", WideOnly: true},
		{Header: "LAST ATTEMPT", WideOnly: true},
	}

	rateLimiterColumns = []Column{
		{Header: "TOKEN BUCKET"},
		{Header: "COUNTER"},
		{Header: "FAIL POLICY"},
		{Header: "CHECKS"},
		{Header: "ALLOWED"},
		{Header: "DENIED"},
		{Header: "KEY PREFIX", WideOnly: true},
		{Header: "ERRORS", WideOnly: true},
	}
)

// runGet dispatches to the appropriate get subcommand.
//
// Takes cc (*CommandContext) which provides the command execution context.
// Takes arguments ([]string) which contains the resource type and
// any subarguments.
//
// Returns error when the resource type is missing or unknown.
func runGet(ctx context.Context, cc *CommandContext, arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("missing resource type\n\nAvailable resources: %s", getResourceList)
	}

	resource := arguments[0]
	handler, ok := getResources[resource]
	if !ok {
		return fmt.Errorf("unknown resource: %s\n\nAvailable resources: %s", resource, getResourceList)
	}

	if err := validateOutputFormat(cc.Opts.Output, "get", getFormats); err != nil {
		return err
	}

	p := NewPrinter(cc.Stdout, cc.Opts.Output, cc.Opts.NoColour, cc.Opts.NoHeaders)
	p.SetLimit(cc.Opts.Limit)
	return handler(ctx, cc.Conn, p, arguments[1:])
}

// getHealth displays the liveness and readiness summary.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains command-line arguments.
//
// Returns error when fetching health data fails.
func getHealth(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get health", "piko get health [name] [flags]", "Display health probe status.", getFormatHelp, getDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.HealthClient().GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return grpcError("fetching health", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	rows := buildHealthRows(p, response, filter)
	p.PrintResource(healthColumns, rows)
	return nil
}

// buildHealthRows creates table rows for health probes, applying an optional
// name filter.
//
// Takes p (*Printer) which handles output formatting.
// Takes response (*pb.GetHealthResponse) which contains the health data.
// Takes filter (string) which optionally restricts results to a named probe.
//
// Returns [][]string which contains the filtered rows.
func buildHealthRows(p *Printer, response *pb.GetHealthResponse, filter string) [][]string {
	probes := []struct {
		status *pb.HealthStatus
		name   string
	}{
		{name: "Liveness", status: response.GetLiveness()},
		{name: "Readiness", status: response.GetReadiness()},
	}

	rows := make([][]string, 0, len(probes))
	for _, probe := range probes {
		if !matchesFilter(probe.name, filter) {
			continue
		}
		rows = append(rows, healthRow(p, probe.name, probe.status))
	}
	return rows
}

// healthRow builds a table row for a health probe.
//
// Takes p (*Printer) which formats and colourises the output.
// Takes probe (string) which is the name of the health probe.
// Takes status (*pb.HealthStatus) which contains the probe's health state.
//
// Returns []string which is the formatted row for table display.
func healthRow(p *Printer, probe string, status *pb.HealthStatus) []string {
	if status == nil {
		return []string{probe, p.ColourisedStatus("unknown"), outputDash, outputDash, outputDash, outputDash}
	}
	return []string{
		probe,
		p.ColourisedStatus(status.GetState()),
		formatReady(status),
		status.GetMessage(),
		status.GetDuration(),
		formatTimestamp(status.GetTimestampMs()),
	}
}

// formatReady counts healthy dependencies and formats as "x/y".
//
// Takes status (*pb.HealthStatus) which contains the dependency list.
//
// Returns string which is the ready count in "x/y" format.
func formatReady(status *pb.HealthStatus) string {
	deps := status.GetDependencies()
	if len(deps) == 0 {
		return "-"
	}
	healthy := 0
	for _, d := range deps {
		if strings.EqualFold(d.GetState(), "healthy") {
			healthy++
		}
	}
	return fmt.Sprintf("%d/%d", healthy, len(deps))
}

// getTasks displays recent tasks.
//
// Takes conn (*provider_grpc.Connection) which provides the orchestrator
// client for fetching tasks.
// Takes p (*Printer) which handles output formatting as JSON or table.
// Takes arguments ([]string) which may contain a name filter and limit flag.
//
// Returns error when fetching tasks from the orchestrator fails.
func getTasks(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get tasks", "piko get tasks [name] [flags]", "Display recent tasks.", getFormatHelp, getDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.OrchestratorClient().ListRecentTasks(ctx, &pb.ListRecentTasksRequest{Limit: safeconv.IntToInt32(p.GetLimit(getDefaultLimit))})
	if err != nil {
		return grpcError("fetching tasks", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response.GetTasks())
	}

	rows := make([][]string, 0, len(response.GetTasks()))
	for _, t := range response.GetTasks() {
		if !matchesFilter(t.GetId(), filter) {
			continue
		}
		rows = append(rows, []string{
			t.GetId(),
			t.GetWorkflowId(),
			t.GetExecutor(),
			p.ColourisedStatus(t.GetStatus()),
			strconv.Itoa(int(t.GetAttempt())),
			formatTimestamp(t.GetUpdatedAt()),
			strconv.Itoa(int(t.GetPriority())),
			t.GetLastError(),
			formatTimestamp(t.GetCreatedAt()),
		})
	}
	p.PrintResource(taskColumns, rows)
	return nil
}

// getWorkflows displays workflow summaries.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which may contain a name filter and limit flag.
//
// Returns error when the workflow list cannot be fetched.
func getWorkflows(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get workflows", "piko get workflows [name] [flags]", "Display workflow summaries.", getFormatHelp, getDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.OrchestratorClient().ListWorkflowSummary(ctx, &pb.ListWorkflowSummaryRequest{Limit: safeconv.IntToInt32(p.GetLimit(getDefaultLimit))})
	if err != nil {
		return grpcError("fetching workflows", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response.GetSummaries())
	}

	rows := make([][]string, 0, len(response.GetSummaries()))
	for _, w := range response.GetSummaries() {
		if !matchesFilter(w.GetWorkflowId(), filter) {
			continue
		}
		rows = append(rows, []string{
			w.GetWorkflowId(),
			strconv.FormatInt(w.GetTaskCount(), 10),
			strconv.FormatInt(w.GetCompleteCount(), 10),
			strconv.FormatInt(w.GetFailedCount(), 10),
			strconv.FormatInt(w.GetActiveCount(), 10),
			formatTimestamp(w.GetUpdatedAt()),
			formatTimestamp(w.GetCreatedAt()),
		})
	}
	p.PrintResource(workflowColumns, rows)
	return nil
}

// getArtefacts displays recently updated artefacts.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which may contain a name filter and limit flag.
//
// Returns error when fetching artefacts from the registry fails.
func getArtefacts(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get artefacts", "piko get artefacts [name] [flags]", "Display recently updated artefacts.", getFormatHelp, getDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.RegistryClient().ListRecentArtefacts(ctx, &pb.ListRecentArtefactsRequest{Limit: safeconv.IntToInt32(p.GetLimit(getDefaultLimit))})
	if err != nil {
		return grpcError("fetching artefacts", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response.GetArtefacts())
	}

	rows := make([][]string, 0, len(response.GetArtefacts()))
	for _, a := range response.GetArtefacts() {
		if !matchesFilter(a.GetId(), filter) && !matchesFilter(a.GetSourcePath(), filter) {
			continue
		}
		rows = append(rows, []string{
			a.GetId(),
			a.GetSourcePath(),
			p.ColourisedStatus(a.GetStatus()),
			strconv.FormatInt(a.GetVariantCount(), 10),
			formatBytes(safeconv.Int64ToUint64(a.GetTotalSize())),
			formatTimestamp(a.GetUpdatedAt()),
			formatTimestamp(a.GetCreatedAt()),
		})
	}
	p.PrintResource(artefactColumns, rows)
	return nil
}

// getVariants displays variant summary by status.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection
// to the registry service.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains command-line arguments to parse.
//
// Returns error when argument parsing fails or the variant summary cannot be
// fetched.
func getVariants(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get variants", "piko get variants [flags]", "Display variant summary by status.", getFormatHelp, getDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.RegistryClient().GetVariantSummary(ctx, &pb.GetVariantSummaryRequest{})
	if err != nil {
		return grpcError("fetching variant summary", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response.GetSummaries())
	}

	rows := make([][]string, 0, len(response.GetSummaries()))
	for _, s := range response.GetSummaries() {
		if !matchesFilter(s.GetStatus(), filter) {
			continue
		}
		rows = append(rows, []string{
			p.ColourisedStatus(s.GetStatus()),
			strconv.FormatInt(s.GetCount(), 10),
		})
	}
	p.PrintResource(variantColumns, rows)
	return nil
}

// getMetrics displays all OTEL metrics.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains command-line arguments to parse.
//
// Returns error when argument parsing fails or metrics cannot be fetched.
func getMetrics(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get metrics", "piko get metrics [name] [flags]", "Display OTEL metrics.", getFormatHelp, getDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.MetricsClient().GetMetrics(ctx, &pb.GetMetricsRequest{})
	if err != nil {
		return grpcError("fetching metrics", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response.GetMetrics())
	}

	rows := make([][]string, 0, len(response.GetMetrics()))
	for _, m := range response.GetMetrics() {
		if !matchesFilter(m.GetName(), filter) {
			continue
		}
		rows = append(rows, []string{
			m.GetName(),
			m.GetType(),
			strconv.Itoa(len(m.GetDataPoints())),
			m.GetUnit(),
			truncate(m.GetDescription(), 60),
		})
	}
	p.PrintResource(metricColumns, rows)
	return nil
}

// getTraces displays recent trace spans.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains command-line flags for filtering.
//
// Returns error when fetching traces from the server fails.
func getTraces(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get traces", "piko get traces [name] [flags]", "Display recent trace spans.", getFormatHelp, getDefaultFormat, p.w)
	errorsOnly := fs.Bool("errors", false, "Show only error spans")
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.MetricsClient().GetTraces(ctx, &pb.GetTracesRequest{})
	if err != nil {
		return grpcError("fetching traces", err)
	}

	spans := response.GetSpans()
	if *errorsOnly {
		spans = filterErrorSpans(spans)
	}
	if effectiveLimit := p.GetLimit(getDefaultLimit); len(spans) > effectiveLimit {
		spans = spans[:effectiveLimit]
	}

	if p.IsJSON() {
		return p.PrintJSON(spans)
	}

	rows := make([][]string, 0, len(spans))
	for _, s := range spans {
		if !matchesFilter(s.GetTraceId(), filter) && !matchesFilter(s.GetName(), filter) {
			continue
		}
		rows = append(rows, []string{
			truncate(s.GetTraceId(), 16),
			truncate(s.GetSpanId(), 12),
			s.GetName(),
			s.GetServiceName(),
			p.ColourisedStatus(s.GetStatus()),
			formatNanos(s.GetDurationNs()),
			s.GetKind(),
			formatTimestamp(s.GetStartTimeMs()),
			s.GetStatusMessage(),
		})
	}
	p.PrintResource(traceColumns, rows)
	return nil
}

// getOpenResources displays open resources grouped by category.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection
// for fetching metrics.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains command-line arguments and flags.
//
// Returns error when argument parsing fails or the gRPC request fails.
func getOpenResources(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get resources", "piko get resources [name] [flags]", "Display open resources grouped by category.", getFormatHelp, getDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.MetricsClient().GetFileDescriptors(ctx, &pb.GetFileDescriptorsRequest{})
	if err != nil {
		return grpcError("fetching resources", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	rows := make([][]string, 0, len(response.GetCategories()))
	for _, c := range response.GetCategories() {
		if !matchesFilter(c.GetCategory(), filter) {
			continue
		}
		rows = append(rows, []string{
			c.GetCategory(),
			strconv.Itoa(int(c.GetCount())),
		})
	}
	if filter == "" {
		rows = append(rows, []string{"TOTAL", strconv.Itoa(int(response.GetTotal()))})
	}
	p.PrintResource(resourceColumns, rows)
	return nil
}

// getRateLimiter displays the current rate limiter status and counters.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains command-line arguments.
//
// Returns error when fetching rate limiter status fails.
func getRateLimiter(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get ratelimiter", "piko get ratelimiter [flags]", "Display rate limiter status and counters.", getFormatHelp, getDefaultFormat, p.w)
	if _, err := parseInterspersed(fs, arguments); err != nil {
		return helpOrError(err)
	}

	response, err := conn.RateLimiterClient().GetRateLimiterStatus(ctx, &pb.GetRateLimiterStatusRequest{})
	if err != nil {
		return grpcError("fetching rate limiter status", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	p.PrintResource(rateLimiterColumns, [][]string{{
		response.GetTokenBucketStore(),
		response.GetCounterStore(),
		response.GetFailPolicy(),
		strconv.FormatInt(response.GetTotalChecks(), 10),
		strconv.FormatInt(response.GetTotalAllowed(), 10),
		strconv.FormatInt(response.GetTotalDenied(), 10),
		response.GetKeyPrefix(),
		strconv.FormatInt(response.GetTotalErrors(), 10),
	}})
	return nil
}

// getDLQ displays dispatcher summaries or DLQ entries for a specific type.
//
// When no arguments are provided, displays a summary of all dispatchers.
// When a dispatcher type is specified, displays the DLQ entries for that type.
//
// Usage: piko get dlq [type] [--limit N]
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains the optional
// dispatcher type and flags.
//
// Returns error when the gRPC request fails or the dispatcher type is invalid.
func getDLQ(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	filter := extractFilter(arguments)
	remaining := argsAfterFilter(arguments, filter)

	if filter != "" {
		return getDLQEntries(ctx, conn, p, filter, remaining)
	}
	return getDLQSummary(ctx, conn, p, remaining)
}

// getDLQSummary displays a summary of all dispatchers.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles the output formatting.
// Takes arguments ([]string) which may contain flags.
//
// Returns error when fetching the dispatcher summary fails.
func getDLQSummary(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko get dlq", "piko get dlq [type] [flags]", "Display dispatcher summaries or DLQ entries.", getFormatHelp, getDefaultFormat, p.w)
	if _, err := parseInterspersed(fs, arguments); err != nil {
		return helpOrError(err)
	}

	response, err := conn.DispatcherClient().GetDispatcherSummary(ctx, &pb.GetDispatcherSummaryRequest{})
	if err != nil {
		return grpcError("fetching dispatcher summary", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response.GetSummaries())
	}

	rows := make([][]string, 0, len(response.GetSummaries()))
	for _, s := range response.GetSummaries() {
		rows = append(rows, []string{
			s.GetType(),
			strconv.Itoa(int(s.GetQueuedItems())),
			strconv.Itoa(int(s.GetDeadLetterCount())),
			strconv.FormatInt(s.GetTotalProcessed(), 10),
			strconv.FormatInt(s.GetTotalFailed(), 10),
			strconv.Itoa(int(s.GetRetryQueueSize())),
			strconv.FormatInt(s.GetTotalSuccessful(), 10),
			strconv.FormatInt(s.GetTotalRetries(), 10),
			formatDuration(s.GetUptimeMs()),
		})
	}
	p.PrintResource(dlqSummaryColumns, rows)
	return nil
}

// getDLQEntries displays DLQ entries for a specific dispatcher type.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes p (*Printer) which handles output formatting.
// Takes dispatcherType (string) which identifies the dispatcher to query.
// Takes arguments ([]string) which may contain flags like --limit.
//
// Returns error when fetching the DLQ entries fails.
func getDLQEntries(ctx context.Context, conn monitoringConnection, p *Printer, dispatcherType string, arguments []string) error {
	fs := newResourceFlagSet("piko get dlq", "piko get dlq <type> [flags]", "Display DLQ entries for a dispatcher.", getFormatHelp, getDefaultFormat, p.w)
	if _, err := parseInterspersed(fs, arguments); err != nil {
		return helpOrError(err)
	}

	response, err := conn.DispatcherClient().ListDLQEntries(ctx, &pb.ListDLQEntriesRequest{
		DispatcherType: dispatcherType,
		Limit:          safeconv.IntToInt32(p.GetLimit(getDefaultLimit)),
	})
	if err != nil {
		return grpcError(fmt.Sprintf("fetching DLQ entries for %s", dispatcherType), err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response.GetEntries())
	}

	rows := make([][]string, 0, len(response.GetEntries()))
	for _, e := range response.GetEntries() {
		rows = append(rows, []string{
			truncate(e.GetId(), 16),
			e.GetType(),
			truncate(e.GetOriginalError(), 40),
			strconv.Itoa(int(e.GetTotalAttempts())),
			formatTimestamp(e.GetAddedAtMs()),
			formatTimestamp(e.GetFirstAttemptMs()),
			formatTimestamp(e.GetLastAttemptMs()),
		})
	}
	p.PrintResource(dlqEntryColumns, rows)
	return nil
}

// getProviders displays registered providers for a resource type, or lists
// available resource types if none is specified.
//
// Usage:
// piko get providers           - lists available resource types
// piko get providers email     - lists email providers
// piko get providers storage   - lists storage providers
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which optionally contains the resource type.
//
// Returns error when the gRPC request fails.
func getProviders(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	if len(arguments) == 0 {
		return getProviderResourceTypes(ctx, conn, p)
	}
	return getProviderList(ctx, conn, p, arguments)
}

// getProviderResourceTypes lists all available resource types.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles output formatting.
//
// Returns error when the gRPC request fails.
func getProviderResourceTypes(ctx context.Context, conn monitoringConnection, p *Printer) error {
	response, err := conn.ProviderInfoClient().ListResourceTypes(ctx, &pb.ListResourceTypesRequest{})
	if err != nil {
		return grpcError("fetching resource types", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	types := response.GetResourceTypes()
	if len(types) == 0 {
		_, _ = fmt.Fprintln(p.w, "No resource types registered.")
		return nil
	}

	headers := []string{"RESOURCE TYPE"}
	rows := make([][]string, len(types))
	for i, t := range types {
		rows[i] = []string{t}
	}
	p.PrintTable(headers, rows)

	_, _ = fmt.Fprint(p.w, "\nUsage: piko get providers <type>\n")
	return nil
}

// getProviderList lists providers for a specific resource type, or
// sub-resources when a provider name is given and the service supports it.
//
// When a provider name is given (e.g. `piko get providers cache otter`), the
// command first tries ListSubResources. If the service supports sub-resources,
// a table of sub-resources is shown (e.g. namespaces). If the service does
// not support sub-resources, it falls back to the filtered provider list.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains the resource type and optional flags.
//
// Returns error when the gRPC request fails.
func getProviderList(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	resourceType := arguments[0]
	remaining := arguments[1:]

	fs := newResourceFlagSet("piko get providers", "piko get providers <type> [name] [flags]", "Display registered providers for a resource type.", getFormatHelp, getDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, remaining)
	if err != nil {
		return helpOrError(err)
	}

	providerName := extractFilter(positional)

	if providerName != "" {
		subResp, subErr := conn.ProviderInfoClient().ListSubResources(ctx, &pb.ListSubResourcesRequest{
			ResourceType: resourceType,
			ProviderName: providerName,
		})
		if subErr == nil {
			if p.IsJSON() {
				return p.PrintJSON(subResp)
			}
			columns, rows := buildSubResourceRows(subResp)
			p.PrintResource(columns, rows)
			return nil
		}
	}

	response, err := conn.ProviderInfoClient().ListProviders(ctx, &pb.ListProvidersRequest{
		ResourceType: resourceType,
	})
	if err != nil {
		return grpcError(fmt.Sprintf("fetching %s providers", resourceType), err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	columns, rows := buildProviderRows(p, response, providerName)
	p.PrintResource(columns, rows)
	return nil
}

// buildProviderRows creates columns and rows from the gRPC provider list
// response. A DEFAULT indicator column is prepended with a green dot for the
// default provider.
//
// Takes p (*Printer) which provides colouring support.
// Takes response (*pb.ListProvidersResponse) which contains column definitions
// and provider rows.
// Takes filter (string) which optionally restricts results to a named provider.
//
// Returns []Column which contains the column definitions for the table.
// Returns [][]string which contains the formatted rows.
func buildProviderRows(p *Printer, response *pb.ListProvidersResponse, filter string) ([]Column, [][]string) {
	greenDot := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("●")
	if p.noColour {
		greenDot = "*"
	}

	pbColumns := response.GetColumns()
	columns := make([]Column, 0, len(pbColumns)+1)
	columns = append(columns, Column{Header: "DEFAULT"})
	for _, col := range pbColumns {
		columns = append(columns, Column{
			Header:   col.GetHeader(),
			WideOnly: col.GetWideOnly(),
		})
	}

	pbRows := response.GetRows()
	rows := make([][]string, 0, len(pbRows))
	for _, row := range pbRows {
		if !matchesFilter(row.GetName(), filter) {
			continue
		}

		indicator := " "
		if row.GetIsDefault() {
			indicator = greenDot
		}

		cells := make([]string, 0, len(pbColumns)+1)
		cells = append(cells, indicator)
		for _, col := range pbColumns {
			cells = append(cells, row.GetValues()[col.GetKey()])
		}
		rows = append(rows, cells)
	}

	return columns, rows
}

// buildSubResourceRows creates columns and rows from a ListSubResources
// response. Unlike buildProviderRows, no DEFAULT indicator column is prepended.
//
// Takes response (*pb.ListSubResourcesResponse) which contains the sub-resource
// column definitions and rows.
//
// Returns []Column which contains the column definitions for the table.
// Returns [][]string which contains the formatted rows.
func buildSubResourceRows(response *pb.ListSubResourcesResponse) ([]Column, [][]string) {
	pbColumns := response.GetColumns()
	columns := make([]Column, len(pbColumns))
	for i, col := range pbColumns {
		columns[i] = Column{
			Header:   col.GetHeader(),
			WideOnly: col.GetWideOnly(),
		}
	}

	pbRows := response.GetRows()
	rows := make([][]string, 0, len(pbRows))
	for _, row := range pbRows {
		cells := make([]string, len(pbColumns))
		for i, col := range pbColumns {
			cells[i] = row.GetValues()[col.GetKey()]
		}
		rows = append(rows, cells)
	}

	return columns, rows
}

// newResourceFlagSet creates a FlagSet for a resource command with formatted
// help output.
//
// Takes name (string) which identifies the flag set.
// Takes usage (string) which is the usage line shown in help.
// Takes description (string) which explains what the command does.
// Takes formatHelp (string) which lists supported output formats for this
// command type (e.g. "table, wide, json" for get, "text, json" for describe).
// Takes defaultFormat (string) which is the default output format.
// Takes w (io.Writer) which receives help output.
//
// Returns *flag.FlagSet which is configured for resource flag parsing.
func newResourceFlagSet(name, usage, description, formatHelp, defaultFormat string, w io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(w)
	fs.Usage = func() {
		_, _ = fmt.Fprintf(w, "Usage: %s\n\n%s\n", usage, description)

		hasFlags := false
		fs.VisitAll(func(_ *flag.Flag) { hasFlags = true })
		if hasFlags {
			_, _ = fmt.Fprint(w, "\nFlags:\n")
			fs.PrintDefaults()
		}

		_, _ = fmt.Fprintf(w, `
Global Flags:
  -o, --output string      Output format: %s (default %q)
  -n, --limit int          Maximum number of items to return
  -e, --endpoint string    gRPC monitoring server address (default "127.0.0.1:9091")
  -t, --timeout duration   Connection and request timeout (default 5s)
      --no-colour          Disable coloured output
      --raw                Disable coloured output (alias for --no-colour)
      --no-headers         Omit table headers from output
`, formatHelp, defaultFormat)
	}
	return fs
}

// helpOrError returns nil for help requests and the original error otherwise.
//
// Takes err (error) which is the flag parsing error.
//
// Returns error which is nil for ErrHelp or the original error.
func helpOrError(err error) error {
	if errors.Is(err, flag.ErrHelp) {
		return nil
	}
	return err
}

// filterErrorSpans returns only spans with error status.
//
// Takes spans ([]*pb.Span) which is the collection of spans to filter.
//
// Returns []*pb.Span which contains only spans with error status.
func filterErrorSpans(spans []*pb.Span) []*pb.Span {
	result := make([]*pb.Span, 0)
	for _, s := range spans {
		if strings.EqualFold(s.GetStatus(), "error") {
			result = append(result, s)
		}
	}
	return result
}

// truncate shortens a string to the given length, appending "..." if truncated.
//
// Takes s (string) which is the string to shorten.
// Takes maxLen (int) which is the maximum length of the result.
//
// Returns string which is the original string or a truncated version with
// "..." appended.
func truncate(s string, maxLen int) string {
	if maxLen <= getTruncateLen || len(s) <= maxLen {
		return s
	}
	return s[:maxLen-getTruncateLen] + "..."
}
