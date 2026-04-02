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
	"fmt"
	"slices"
	"strconv"
	"strings"

	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// describeFormatHelp is the format documentation shown in describe command help.
	describeFormatHelp = `text, json`

	// describeDefaultFormat is the default output format for describe commands.
	describeDefaultFormat = "text"

	// describeTruncateLen is the default truncation length for IDs.
	describeTruncateLen = 12
)

var (
	// describeFormats lists the output formats supported by the describe command.
	describeFormats = []string{describeDefaultFormat, "json"}

	// describeResources maps resource names to their describe handlers.
	describeResources = map[string]func(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error{
		"health":      describeHealth,
		"trace":       describeTrace,
		"task":        describeTask,
		"workflow":    describeWorkflow,
		"artefact":    describeArtefact,
		"dlq":         describeDLQ,
		"resources":   describeOpenResources,
		"ratelimiter": describeRateLimiter,
		"provider":    describeProvider,
		"providers":   describeProviders,
	}

	// describeResourceList is the sorted, comma-separated list of available
	// describe resources, derived from the describeResources dispatch map.
	describeResourceList = buildResourceList(describeResources)
)

// runDescribe dispatches to the appropriate describe subcommand.
//
// Takes cc (*CommandContext) which provides the command execution context.
// Takes arguments ([]string) which contains the resource type and any subcommand
// arguments.
//
// Returns error when no resource type is provided or the resource is unknown.
func runDescribe(ctx context.Context, cc *CommandContext, arguments []string) error {
	if len(arguments) == 0 {
		return fmt.Errorf("missing resource type\n\nAvailable resources: %s", describeResourceList)
	}

	resource := arguments[0]
	handler, ok := describeResources[resource]
	if !ok {
		return fmt.Errorf("unknown resource: %s\n\nAvailable resources: %s", resource, describeResourceList)
	}

	format := cc.Opts.Output
	if format == "table" {
		format = "text"
	}

	if err := validateOutputFormat(format, "describe", describeFormats); err != nil {
		return err
	}

	p := NewPrinter(cc.Stdout, format, cc.Opts.NoColour, cc.Opts.NoHeaders)
	p.SetLimit(cc.Opts.Limit)
	return handler(ctx, cc.Conn, p, arguments[1:])
}

// describeHealth displays the full hierarchical health tree using PrintDetail.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC client.
// Takes p (*Printer) which controls output format and styling.
// Takes arguments ([]string) which contains command-line arguments to parse.
//
// Returns error when fetching health fails or no probe matches the filter.
func describeHealth(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko describe health", "piko describe health [name] [flags]", "Show detailed health probe information.", describeFormatHelp, describeDefaultFormat, p.w)
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

	sections := buildHealthDetailSections(p, response, filter)
	if len(sections) == 0 && filter != "" {
		return fmt.Errorf("no health probe matching %q", filter)
	}
	p.PrintDetail(sections)
	return nil
}

// buildHealthDetailSections creates detail sections for health probes.
//
// Takes p (*Printer) which provides status colouring.
// Takes response (*pb.GetHealthResponse) which contains the health data.
// Takes filter (string) which optionally restricts to a named probe.
//
// Returns []DetailSection which contains the filtered sections.
func buildHealthDetailSections(p *Printer, response *pb.GetHealthResponse, filter string) []DetailSection {
	probes := []struct {
		status *pb.HealthStatus
		name   string
	}{
		{name: "Liveness", status: response.GetLiveness()},
		{name: "Readiness", status: response.GetReadiness()},
	}

	sections := make([]DetailSection, 0, len(probes))
	for _, probe := range probes {
		if !matchesFilter(probe.name, filter) {
			continue
		}
		if probe.status == nil {
			continue
		}
		sections = append(sections, healthStatusSection(p, probe.name, probe.status))
	}
	return sections
}

// healthStatusSection builds a DetailSection for a health probe.
//
// Takes p (*Printer) which provides colourised output formatting.
// Takes name (string) which specifies the section title.
// Takes status (*pb.HealthStatus) which contains the health probe data.
//
// Returns DetailSection which contains the formatted health status fields
// and any dependency subsections.
func healthStatusSection(p *Printer, name string, status *pb.HealthStatus) DetailSection {
	section := DetailSection{
		Title: name,
		Fields: []DetailField{
			{Key: "State", Value: p.ColourisedStatus(status.GetState()), IsStatus: true},
			{Key: "Message", Value: status.GetMessage()},
			{Key: "Duration", Value: status.GetDuration()},
			{Key: "Timestamp", Value: formatTimestamp(status.GetTimestampMs())},
			{Key: "Dependencies", Value: formatReady(status)},
		},
	}

	for _, dependency := range status.GetDependencies() {
		section.SubSections = append(section.SubSections, healthDependencySection(p, dependency))
	}
	return section
}

// healthDependencySection builds a DetailSection for a health dependency.
//
// Takes p (*Printer) which provides colourisation for status values.
// Takes dependency (*pb.HealthStatus) which contains the dependency health data.
//
// Returns DetailSection which represents the formatted health dependency.
func healthDependencySection(p *Printer, dependency *pb.HealthStatus) DetailSection {
	section := DetailSection{
		Title: dependency.GetName(),
		Fields: []DetailField{
			{Key: "State", Value: p.ColourisedStatus(dependency.GetState()), IsStatus: true},
			{Key: "Duration", Value: dependency.GetDuration()},
		},
	}
	if dependency.GetMessage() != "" {
		section.Fields = append(section.Fields, DetailField{Key: "Message", Value: dependency.GetMessage()})
	}
	return section
}

// describeTrace displays all spans belonging to a specific trace.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection
// for fetching trace data.
// Takes p (*Printer) which handles output formatting and table rendering.
// Takes arguments ([]string) which contains the command-line arguments including
// the trace ID.
//
// Returns error when the trace ID is missing, the gRPC request fails, or no
// spans are found for the given trace ID.
func describeTrace(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko describe trace", "piko describe trace <trace-id> [flags]", "Show all spans for a trace.", describeFormatHelp, describeDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	if len(positional) == 0 {
		return errors.New("missing trace ID\n\nUsage: piko describe trace <trace-id>")
	}

	traceID := positional[0]

	response, err := conn.MetricsClient().GetTraces(ctx, &pb.GetTracesRequest{})
	if err != nil {
		return grpcError("fetching traces", err)
	}

	traceSpans := filterSpansByTraceID(response.GetSpans(), traceID)
	if len(traceSpans) == 0 {
		return fmt.Errorf("no spans found for trace ID: %s", traceID)
	}

	if p.IsJSON() {
		return p.PrintJSON(traceSpans)
	}

	printTraceTable(p, traceID, traceSpans)
	printSpanDetails(p, traceSpans)
	return nil
}

// filterSpansByTraceID returns only the spans belonging to the given trace.
//
// Takes spans ([]*pb.Span) which is the full list of spans to filter.
// Takes traceID (string) which identifies the trace to match.
//
// Returns []*pb.Span which contains only spans with the matching trace ID.
func filterSpansByTraceID(spans []*pb.Span, traceID string) []*pb.Span {
	var result []*pb.Span
	for _, s := range spans {
		if s.GetTraceId() == traceID {
			result = append(result, s)
		}
	}
	return result
}

// printTraceTable renders the span summary table for a trace.
//
// Takes p (*Printer) which handles output formatting and colouring.
// Takes traceID (string) which identifies the trace being displayed.
// Takes spans ([]*pb.Span) which contains the spans to render.
func printTraceTable(p *Printer, traceID string, spans []*pb.Span) {
	_, _ = fmt.Fprintf(p.w, "=== Trace %s ===\n", traceID)
	_, _ = fmt.Fprintf(p.w, "Spans: %d\n\n", len(spans))

	headers := []string{"SPAN ID", "PARENT", "NAME", "SERVICE", "STATUS", "DURATION", "START"}
	rows := make([][]string, 0, len(spans))
	for _, s := range spans {
		parentID := s.GetParentSpanId()
		if parentID == "" {
			parentID = "(root)"
		} else {
			parentID = truncate(parentID, describeTruncateLen)
		}
		rows = append(rows, []string{
			truncate(s.GetSpanId(), describeTruncateLen),
			parentID,
			s.GetName(),
			s.GetServiceName(),
			p.ColourisedStatus(s.GetStatus()),
			formatNanos(s.GetDurationNs()),
			formatTimestamp(s.GetStartTimeMs()),
		})
	}
	p.PrintTable(headers, rows)
}

// printSpanDetails renders the detail sections for each span.
//
// Takes p (*Printer) which handles output formatting.
// Takes spans ([]*pb.Span) which contains the spans to render details for.
func printSpanDetails(p *Printer, spans []*pb.Span) {
	for _, s := range spans {
		sections := buildSpanDetailSections(p, s)
		if len(sections) > 0 {
			_, _ = fmt.Fprintln(p.w)
			p.PrintDetail(sections)
		}
	}
}

// buildSpanDetailSections creates detail sections for a single span's
// attributes and events.
//
// Takes s (*pb.Span) which is the span to extract details from.
//
// Returns []DetailSection which contains the formatted attribute and event
// sections for the span.
func buildSpanDetailSections(_ *Printer, s *pb.Span) []DetailSection {
	var sections []DetailSection

	attrs := s.GetAttributes()
	if len(attrs) > 0 {
		attributeKeys := make([]string, 0, len(attrs))
		for k := range attrs {
			attributeKeys = append(attributeKeys, k)
		}
		slices.Sort(attributeKeys)
		fields := make([]DetailField, 0, len(attrs))
		for _, k := range attributeKeys {
			fields = append(fields, DetailField{Key: k, Value: attrs[k]})
		}
		sections = append(sections, DetailSection{
			Title:  fmt.Sprintf("Span %s (%s)", truncate(s.GetSpanId(), describeTruncateLen), s.GetName()),
			Fields: fields,
		})
	}

	for _, e := range s.GetEvents() {
		eventFields := []DetailField{
			{Key: "Timestamp", Value: formatTimestamp(e.GetTimestampMs())},
		}
		eventAttrs := e.GetAttributes()
		eventAttrKeys := make([]string, 0, len(eventAttrs))
		for k := range eventAttrs {
			eventAttrKeys = append(eventAttrKeys, k)
		}
		slices.Sort(eventAttrKeys)
		for _, k := range eventAttrKeys {
			eventFields = append(eventFields, DetailField{Key: k, Value: eventAttrs[k]})
		}
		sections = append(sections, DetailSection{
			Title:  fmt.Sprintf("Event: %s", e.GetName()),
			Fields: eventFields,
		})
	}

	return sections
}

// resourceDescriptor bundles the three resource-specific callbacks used by
// describeResourceItems, reducing the parameter count to within lint limits.
type resourceDescriptor struct {
	// fetch retrieves items from the gRPC endpoint.
	fetch func(ctx context.Context, conn monitoringConnection, filter string) (any, error)

	// filterForJSON narrows the item collection to those matching a filter string,
	// for use in JSON output mode.
	filterForJSON func(items any, filter string) any

	// buildSections constructs DetailSection values from the items and filter.
	buildSections func(items any, filter string) []DetailSection
}

// describeResourceItems implements the shared describe pattern: parse flags,
// fetch items via gRPC, then render as JSON or detail sections.
//
// Takes resourceName (string) which names the resource for the flag set and
// error messages.
// Takes descriptor (resourceDescriptor) which bundles the fetch, filter, and
// section-building callbacks for the specific resource type.
//
// Returns error when flag parsing, the gRPC call, or output rendering fails,
// or when no sections match a non-empty filter.
func describeResourceItems(
	ctx context.Context,
	conn monitoringConnection,
	p *Printer,
	arguments []string,
	resourceName string,
	descriptor resourceDescriptor,
) error {
	fs := newResourceFlagSet(
		"piko describe "+resourceName,
		"piko describe "+resourceName+" [id] [flags]",
		"Show detailed "+resourceName+" information.",
		describeFormatHelp,
		describeDefaultFormat,
		p.w,
	)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	items, err := descriptor.fetch(ctx, conn, filter)
	if err != nil {
		return err
	}

	if p.IsJSON() {
		if filter != "" {
			return p.PrintJSON(descriptor.filterForJSON(items, filter))
		}
		return p.PrintJSON(items)
	}

	sections := descriptor.buildSections(items, filter)
	if len(sections) == 0 && filter != "" {
		return fmt.Errorf("no %s matching %q", resourceName, filter)
	}
	p.PrintDetail(sections)
	return nil
}

// describeTask displays detailed information for a specific task.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which formats the output.
// Takes arguments ([]string) which contains command-line arguments.
//
// Returns error when fetching tasks fails or no task matches the filter.
//
//nolint:dupl // type-specific describeResourceItems wrapper
func describeTask(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	return describeResourceItems(
		ctx, conn, p, arguments,
		"task",
		resourceDescriptor{
			fetch: func(ctx context.Context, conn monitoringConnection, _ string) (any, error) {
				response, err := conn.OrchestratorClient().ListRecentTasks(ctx, &pb.ListRecentTasksRequest{Limit: safeconv.IntToInt32(p.GetLimit(50))})
				if err != nil {
					return nil, grpcError("fetching tasks", err)
				}
				return response.GetTasks(), nil
			},
			filterForJSON: func(items any, filter string) any {
				return filterTasks(items.([]*pb.TaskListItem), filter)
			},
			buildSections: func(items any, filter string) []DetailSection {
				return buildTaskDetailSections(p, items.([]*pb.TaskListItem), filter)
			},
		},
	)
}

// filterTasks returns tasks matching the filter by ID.
//
// Takes tasks ([]*pb.TaskListItem) which is the list of tasks to filter.
// Takes filter (string) which is the ID pattern to match against.
//
// Returns []*pb.TaskListItem which contains only the tasks with matching IDs.
func filterTasks(tasks []*pb.TaskListItem, filter string) []*pb.TaskListItem {
	result := make([]*pb.TaskListItem, 0)
	for _, t := range tasks {
		if matchesFilter(t.GetId(), filter) {
			result = append(result, t)
		}
	}
	return result
}

// buildTaskDetailSections creates detail sections for tasks.
//
// Takes p (*Printer) which formats status values with colour.
// Takes tasks ([]*pb.TaskListItem) which contains the tasks to process.
// Takes filter (string) which limits results to matching task IDs.
//
// Returns []DetailSection which contains the formatted task details.
func buildTaskDetailSections(p *Printer, tasks []*pb.TaskListItem, filter string) []DetailSection {
	sections := make([]DetailSection, 0)
	for _, t := range tasks {
		if !matchesFilter(t.GetId(), filter) {
			continue
		}
		fields := []DetailField{
			{Key: "ID", Value: t.GetId()},
			{Key: "Workflow", Value: t.GetWorkflowId()},
			{Key: "Executor", Value: t.GetExecutor()},
			{Key: "Status", Value: p.ColourisedStatus(t.GetStatus()), IsStatus: true},
			{Key: "Priority", Value: strconv.Itoa(int(t.GetPriority()))},
			{Key: "Attempt", Value: strconv.Itoa(int(t.GetAttempt()))},
		}
		if t.GetLastError() != "" {
			fields = append(fields, DetailField{Key: "Last Error", Value: t.GetLastError()})
		}
		fields = append(fields,
			DetailField{Key: "Created", Value: formatTimestamp(t.GetCreatedAt())},
			DetailField{Key: "Updated", Value: formatTimestamp(t.GetUpdatedAt())},
		)
		sections = append(sections, DetailSection{
			Title:  fmt.Sprintf("Task %s", t.GetId()),
			Fields: fields,
		})
	}
	return sections
}

// describeWorkflow displays detailed information for a specific workflow.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles formatting of the output.
// Takes arguments ([]string) which contains command-line arguments and flags.
//
// Returns error when fetching workflows fails or no workflow matches the filter.
func describeWorkflow(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko describe workflow", "piko describe workflow [id] [flags]", "Show detailed workflow information.", describeFormatHelp, describeDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.OrchestratorClient().ListWorkflowSummary(ctx, &pb.ListWorkflowSummaryRequest{Limit: safeconv.IntToInt32(p.GetLimit(50))})
	if err != nil {
		return grpcError("fetching workflows", err)
	}

	if p.IsJSON() {
		if filter != "" {
			filtered := filterWorkflows(response.GetSummaries(), filter)
			return p.PrintJSON(filtered)
		}
		return p.PrintJSON(response.GetSummaries())
	}

	sections := buildWorkflowDetailSections(response.GetSummaries(), filter)
	if len(sections) == 0 && filter != "" {
		return fmt.Errorf("no workflow matching %q", filter)
	}
	p.PrintDetail(sections)
	return nil
}

// filterWorkflows returns workflows matching the filter by ID.
//
// Takes workflows ([]*pb.WorkflowSummary) which is the list to filter.
// Takes filter (string) which specifies the ID pattern to match against.
//
// Returns []*pb.WorkflowSummary which contains only workflows with matching IDs.
func filterWorkflows(workflows []*pb.WorkflowSummary, filter string) []*pb.WorkflowSummary {
	result := make([]*pb.WorkflowSummary, 0)
	for _, wf := range workflows {
		if matchesFilter(wf.GetWorkflowId(), filter) {
			result = append(result, wf)
		}
	}
	return result
}

// buildWorkflowDetailSections creates detail sections for workflows.
//
// Takes workflows ([]*pb.WorkflowSummary) which provides the workflow data.
// Takes filter (string) which limits results to matching workflow IDs.
//
// Returns []DetailSection which contains the formatted workflow details.
func buildWorkflowDetailSections(workflows []*pb.WorkflowSummary, filter string) []DetailSection {
	sections := make([]DetailSection, 0)
	for _, wf := range workflows {
		if !matchesFilter(wf.GetWorkflowId(), filter) {
			continue
		}
		sections = append(sections, DetailSection{
			Title: fmt.Sprintf("Workflow %s", wf.GetWorkflowId()),
			Fields: []DetailField{
				{Key: "Workflow ID", Value: wf.GetWorkflowId()},
				{Key: "Tasks", Value: strconv.FormatInt(wf.GetTaskCount(), 10)},
				{Key: "Complete", Value: strconv.FormatInt(wf.GetCompleteCount(), 10)},
				{Key: "Failed", Value: strconv.FormatInt(wf.GetFailedCount(), 10)},
				{Key: "Active", Value: strconv.FormatInt(wf.GetActiveCount(), 10)},
				{Key: "Created", Value: formatTimestamp(wf.GetCreatedAt())},
				{Key: "Updated", Value: formatTimestamp(wf.GetUpdatedAt())},
			},
		})
	}
	return sections
}

// describeArtefact displays detailed information for a specific artefact.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which formats the output.
// Takes arguments ([]string) which contains command-line arguments and flags.
//
// Returns error when fetching artefacts fails or no matching artefact is found.
//
//nolint:dupl // type-specific describeResourceItems wrapper
func describeArtefact(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	return describeResourceItems(
		ctx, conn, p, arguments,
		"artefact",
		resourceDescriptor{
			fetch: func(ctx context.Context, conn monitoringConnection, _ string) (any, error) {
				response, err := conn.RegistryClient().ListRecentArtefacts(ctx, &pb.ListRecentArtefactsRequest{Limit: safeconv.IntToInt32(p.GetLimit(50))})
				if err != nil {
					return nil, grpcError("fetching artefacts", err)
				}
				return response.GetArtefacts(), nil
			},
			filterForJSON: func(items any, filter string) any {
				return filterArtefacts(items.([]*pb.ArtefactListItem), filter)
			},
			buildSections: func(items any, filter string) []DetailSection {
				return buildArtefactDetailSections(p, items.([]*pb.ArtefactListItem), filter)
			},
		},
	)
}

// filterArtefacts returns artefacts matching the filter by ID or source path.
//
// Takes artefacts ([]*pb.ArtefactListItem) which is the list to filter.
// Takes filter (string) which is the substring to match against.
//
// Returns []*pb.ArtefactListItem which contains only matching artefacts.
func filterArtefacts(artefacts []*pb.ArtefactListItem, filter string) []*pb.ArtefactListItem {
	result := make([]*pb.ArtefactListItem, 0)
	for _, a := range artefacts {
		if matchesFilter(a.GetId(), filter) || matchesFilter(a.GetSourcePath(), filter) {
			result = append(result, a)
		}
	}
	return result
}

// buildArtefactDetailSections creates detail sections for artefacts.
//
// Takes p (*Printer) which provides colourised status formatting.
// Takes artefacts ([]*pb.ArtefactListItem) which contains the artefacts to
// display.
// Takes filter (string) which limits results to matching IDs or source paths.
//
// Returns []DetailSection which contains the formatted detail sections for
// artefacts that match the filter.
func buildArtefactDetailSections(p *Printer, artefacts []*pb.ArtefactListItem, filter string) []DetailSection {
	sections := make([]DetailSection, 0)
	for _, a := range artefacts {
		if !matchesFilter(a.GetId(), filter) && !matchesFilter(a.GetSourcePath(), filter) {
			continue
		}
		sections = append(sections, DetailSection{
			Title: fmt.Sprintf("Artefact %s", a.GetId()),
			Fields: []DetailField{
				{Key: "ID", Value: a.GetId()},
				{Key: "Source Path", Value: a.GetSourcePath()},
				{Key: "Status", Value: p.ColourisedStatus(a.GetStatus()), IsStatus: true},
				{Key: "Variants", Value: strconv.FormatInt(a.GetVariantCount(), 10)},
				{Key: "Size", Value: formatBytes(safeconv.Int64ToUint64(a.GetTotalSize()))},
				{Key: "Created", Value: formatTimestamp(a.GetCreatedAt())},
				{Key: "Updated", Value: formatTimestamp(a.GetUpdatedAt())},
			},
		})
	}
	return sections
}

// describeDLQ displays detailed dispatcher and DLQ information.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which handles output formatting.
// Takes arguments ([]string) which contains command-line arguments.
//
// Returns error when fetching the dispatcher summary fails or no dispatcher
// matches the filter.
func describeDLQ(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko describe dlq", "piko describe dlq [type] [flags]", "Show detailed dispatcher and DLQ information.", describeFormatHelp, describeDefaultFormat, p.w)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	filter := extractFilter(positional)

	response, err := conn.DispatcherClient().GetDispatcherSummary(ctx, &pb.GetDispatcherSummaryRequest{})
	if err != nil {
		return grpcError("fetching dispatcher summary", err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response.GetSummaries())
	}

	sections := buildDLQDetailSections(response.GetSummaries(), filter)
	if len(sections) == 0 && filter != "" {
		return fmt.Errorf("no dispatcher matching %q", filter)
	}
	p.PrintDetail(sections)
	return nil
}

// buildDLQDetailSections creates detail sections for dispatchers.
//
// Takes summaries ([]*pb.DispatcherSummary) which contains the dispatcher data.
// Takes filter (string) which limits results to matching dispatcher types.
//
// Returns []DetailSection which contains the formatted dispatcher details.
func buildDLQDetailSections(summaries []*pb.DispatcherSummary, filter string) []DetailSection {
	sections := make([]DetailSection, 0)
	for _, s := range summaries {
		if !matchesFilter(s.GetType(), filter) {
			continue
		}
		sections = append(sections, DetailSection{
			Title: fmt.Sprintf("Dispatcher %s", s.GetType()),
			Fields: []DetailField{
				{Key: "Type", Value: s.GetType()},
				{Key: "Queued", Value: strconv.Itoa(int(s.GetQueuedItems()))},
				{Key: "Retry Queue", Value: strconv.Itoa(int(s.GetRetryQueueSize()))},
				{Key: "Dead Letter", Value: strconv.Itoa(int(s.GetDeadLetterCount()))},
				{Key: "Total Processed", Value: strconv.FormatInt(s.GetTotalProcessed(), 10)},
				{Key: "Total Successful", Value: strconv.FormatInt(s.GetTotalSuccessful(), 10)},
				{Key: "Total Failed", Value: strconv.FormatInt(s.GetTotalFailed(), 10)},
				{Key: "Total Retries", Value: strconv.FormatInt(s.GetTotalRetries(), 10)},
				{Key: "Uptime", Value: formatDuration(s.GetUptimeMs())},
			},
		})
	}
	return sections
}

// describeOpenResources displays detailed open resource information.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection
// for fetching metrics.
// Takes p (*Printer) which controls output formatting.
// Takes arguments ([]string) which contains command-line arguments and flags.
//
// Returns error when flag parsing fails, the gRPC request fails, or no
// category matches the filter.
func describeOpenResources(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko describe resources", "piko describe resources [category] [flags]", "Show detailed open resource information.", describeFormatHelp, describeDefaultFormat, p.w)
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

	sections := buildResourceDetailSections(response, filter)
	if len(sections) == 0 && filter != "" {
		return fmt.Errorf("no resource category matching %q", filter)
	}
	p.PrintDetail(sections)
	return nil
}

// buildResourceDetailSections creates detail sections for resource categories.
//
// Takes response (*pb.GetFileDescriptorsResponse) which contains the resource
// data to format.
// Takes filter (string) which limits results to matching categories.
//
// Returns []DetailSection which contains the formatted sections for display.
func buildResourceDetailSections(response *pb.GetFileDescriptorsResponse, filter string) []DetailSection {
	sections := make([]DetailSection, 0)

	if filter == "" {
		sections = append(sections, DetailSection{
			Title: "Summary",
			Fields: []DetailField{
				{Key: "Total", Value: strconv.Itoa(int(response.GetTotal()))},
				{Key: "Categories", Value: strconv.Itoa(len(response.GetCategories()))},
				{Key: "Timestamp", Value: formatTimestamp(response.GetTimestampMs())},
			},
		})
	}

	for _, cat := range response.GetCategories() {
		if !matchesFilter(cat.GetCategory(), filter) {
			continue
		}

		catSection := DetailSection{
			Title: cat.GetCategory(),
			Fields: []DetailField{
				{Key: "Count", Value: strconv.Itoa(int(cat.GetCount()))},
			},
		}

		for _, fd := range cat.GetFds() {
			catSection.SubSections = append(catSection.SubSections, DetailSection{
				Title: fmt.Sprintf("fd %d", fd.GetFd()),
				Fields: []DetailField{
					{Key: "Target", Value: fd.GetTarget()},
					{Key: "Age", Value: formatDuration(fd.GetAgeMs())},
				},
			})
		}

		sections = append(sections, catSection)
	}
	return sections
}

// describeRateLimiter displays detailed rate limiter information.
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which formats the output.
// Takes arguments ([]string) which contains command-line arguments.
//
// Returns error when flag parsing fails or the status cannot be fetched.
func describeRateLimiter(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet("piko describe ratelimiter", "piko describe ratelimiter [flags]", "Show detailed rate limiter information.", describeFormatHelp, describeDefaultFormat, p.w)
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

	sections := buildRateLimiterDetailSections(response)
	p.PrintDetail(sections)
	return nil
}

// buildRateLimiterDetailSections creates detail sections for the rate limiter.
//
// Takes response (*pb.GetRateLimiterStatusResponse) which provides
// the rate limiter
// status data to display.
//
// Returns []DetailSection which contains the formatted sections for display.
func buildRateLimiterDetailSections(response *pb.GetRateLimiterStatusResponse) []DetailSection {
	allowedDeniedRatio := "-"
	total := response.GetTotalAllowed() + response.GetTotalDenied()
	if total > 0 {
		pct := float64(response.GetTotalAllowed()) / float64(total) * reportPercentageFactor
		allowedDeniedRatio = fmt.Sprintf("%.1f%% allowed", pct)
	}

	return []DetailSection{
		{
			Title: "Rate Limiter",
			Fields: []DetailField{
				{Key: "Token Bucket Store", Value: response.GetTokenBucketStore()},
				{Key: "Counter Store", Value: response.GetCounterStore()},
				{Key: "Fail Policy", Value: response.GetFailPolicy()},
				{Key: "Key Prefix", Value: response.GetKeyPrefix()},
			},
		},
		{
			Title: "Counters",
			Fields: []DetailField{
				{Key: "Total Checks", Value: strconv.FormatInt(response.GetTotalChecks(), 10)},
				{Key: "Total Allowed", Value: strconv.FormatInt(response.GetTotalAllowed(), 10)},
				{Key: "Total Denied", Value: strconv.FormatInt(response.GetTotalDenied(), 10)},
				{Key: "Total Errors", Value: strconv.FormatInt(response.GetTotalErrors(), 10)},
				{Key: "Allow Rate", Value: allowedDeniedRatio},
			},
		},
	}
}

// describeProvider displays detailed information about a single provider.
//
// Usage: piko describe provider <type> <name>
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which controls output formatting.
// Takes arguments ([]string) which contains the resource type, provider name, and
// any flags.
//
// Returns error when the required arguments are missing or the gRPC request
// fails.
func describeProvider(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet(
		"piko describe provider",
		"piko describe provider <type> <name> [flags]",
		"Show detailed provider information.",
		describeFormatHelp, describeDefaultFormat, p.w,
	)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	const minPositionalArgs = 2
	if len(positional) < minPositionalArgs {
		return errors.New("missing resource type and provider name\n\nUsage: piko describe provider <type> <name>")
	}

	resourceType := positional[0]
	providerName := positional[1]

	response, err := conn.ProviderInfoClient().DescribeProvider(ctx, &pb.DescribeProviderRequest{
		ResourceType: resourceType,
		Name:         providerName,
	})
	if err != nil {
		return grpcError(fmt.Sprintf("describing %s provider %q", resourceType, providerName), err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	sections := buildProviderDetailSections(response)
	sections = appendSubResourceSections(ctx, conn, sections, resourceType, providerName)
	p.PrintDetail(sections)
	return nil
}

// appendSubResourceSections fetches sub-resources for a provider and appends
// them as a detail section if any are found.
//
// Takes ctx (context.Context) which controls the request lifecycle.
// Takes conn (monitoringConnection) which provides the gRPC client.
// Takes sections ([]DetailSection) which holds the existing sections to extend.
// Takes resourceType (string) which identifies the provider's resource type.
// Takes providerName (string) which identifies the provider by name.
//
// Returns []DetailSection which is the original sections with any sub-resource
// section appended.
func appendSubResourceSections(ctx context.Context, conn monitoringConnection, sections []DetailSection, resourceType, providerName string) []DetailSection {
	subResp, err := conn.ProviderInfoClient().ListSubResources(ctx, &pb.ListSubResourcesRequest{
		ResourceType: resourceType,
		ProviderName: providerName,
	})
	if err != nil || len(subResp.GetRows()) == 0 {
		return sections
	}

	subFields := buildSubResourceFields(subResp)
	subName := subResp.GetSubResourceName()
	if len(subName) > 0 {
		subName = strings.ToUpper(subName[:1]) + subName[1:]
	}

	return append(sections, DetailSection{
		Title:  fmt.Sprintf("%s (%d)", subName, len(subResp.GetRows())),
		Fields: subFields,
	})
}

// buildSubResourceFields creates detail fields from sub-resource rows,
// combining non-identity column values into a summary string per row.
//
// Takes response (*pb.ListSubResourcesResponse) which contains the sub-resource
// data to format.
//
// Returns []DetailField which contains one field per sub-resource row.
func buildSubResourceFields(response *pb.ListSubResourcesResponse) []DetailField {
	fields := make([]DetailField, 0, len(response.GetRows()))
	for _, row := range response.GetRows() {
		var valueParts []string
		for _, col := range response.GetColumns() {
			if col.GetKey() == "name" || col.GetKey() == "namespace" {
				continue
			}
			if v := row.GetValues()[col.GetKey()]; v != "" {
				valueParts = append(valueParts, fmt.Sprintf("%s %s", v, strings.ToLower(col.GetHeader())))
			}
		}
		value := strings.Join(valueParts, ", ")
		if value == "" {
			value = "-"
		}
		fields = append(fields, DetailField{
			Key:   row.GetName(),
			Value: value,
		})
	}
	return fields
}

// describeProviders displays a service-level overview for a resource type, or
// delegates to describeProvider when a provider name is also given.
//
// Usage:
// piko describe providers <type>          - service-level overview
// piko describe providers <type> <name>   - single provider detail
//
// Takes conn (*provider_grpc.Connection) which provides the gRPC connection.
// Takes p (*Printer) which controls output formatting.
// Takes arguments ([]string) which contains the resource type and
// optional provider
// name.
//
// Returns error when the resource type is missing or the gRPC request fails.
func describeProviders(ctx context.Context, conn monitoringConnection, p *Printer, arguments []string) error {
	fs := newResourceFlagSet(
		"piko describe providers",
		"piko describe providers <type> [name] [flags]",
		"Show service-level provider overview, or detail for a named provider.",
		describeFormatHelp, describeDefaultFormat, p.w,
	)
	positional, err := parseInterspersed(fs, arguments)
	if err != nil {
		return helpOrError(err)
	}

	if len(positional) == 0 {
		return errors.New("missing resource type\n\nUsage: piko describe providers <type> [name]")
	}

	const minProviderDetailArgs = 2
	if len(positional) >= minProviderDetailArgs {
		return describeProvider(ctx, conn, p, positional)
	}

	resourceType := positional[0]

	response, err := conn.ProviderInfoClient().DescribeResourceType(ctx, &pb.DescribeResourceTypeRequest{
		ResourceType: resourceType,
	})
	if err != nil {
		return grpcError(fmt.Sprintf("describing %s resource type", resourceType), err)
	}

	if p.IsJSON() {
		return p.PrintJSON(response)
	}

	sections := buildProviderDetailSections(response)
	p.PrintDetail(sections)
	return nil
}

// buildProviderDetailSections creates detail sections from a provider describe
// response.
//
// Takes response (*pb.DescribeProviderResponse) which contains the provider detail
// data.
//
// Returns []DetailSection which contains the formatted sections for display.
func buildProviderDetailSections(response *pb.DescribeProviderResponse) []DetailSection {
	pbSections := response.GetSections()
	sections := make([]DetailSection, 0, len(pbSections))

	for _, s := range pbSections {
		fields := make([]DetailField, 0, len(s.GetEntries()))
		for _, e := range s.GetEntries() {
			fields = append(fields, DetailField{
				Key:   e.GetKey(),
				Value: e.GetValue(),
			})
		}
		sections = append(sections, DetailSection{
			Title:  s.GetTitle(),
			Fields: fields,
		})
	}

	return sections
}
