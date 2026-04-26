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

package provider_grpc

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sync"
	"time"

	"piko.sh/piko/cmd/piko/internal/inspector"
	"piko.sh/piko/cmd/piko/internal/tui/tui_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/logger"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
	"piko.sh/piko/wdk/safeconv"
)

var _ tui_domain.ResourceProvider = (*ResourceProvider)(nil)

const (
	// defaultRecentTasksLimit is the default limit for recent tasks queries.
	defaultRecentTasksLimit = 100

	// defaultWorkflowSummaryLimit is the number of workflows to fetch by default.
	defaultWorkflowSummaryLimit = 50

	// defaultRecentArtefactsLimit is the default limit for recent artefacts queries.
	defaultRecentArtefactsLimit = 100

	// percentageMultiplier is the multiplier used for percentage calculations.
	percentageMultiplier = 100

	// kindOrchestratorTask is the resource kind for orchestrator tasks.
	kindOrchestratorTask = "task"

	// kindOrchestratorWorkflow is the resource kind for orchestrator workflows.
	kindOrchestratorWorkflow = "workflow"

	// kindRegistryArtefact is the resource kind for registry artefacts.
	kindRegistryArtefact = "artefact"

	// kindRegistryVariant is the resource kind identifier for registry variants.
	kindRegistryVariant = "variant"

	// metadataKeyWorkflowID is the metadata key for storing the workflow identifier.
	metadataKeyWorkflowID = "workflow_id"

	// metadataKeyExecutor is the metadata key for the task executor name.
	metadataKeyExecutor = "executor"

	// metadataKeyPriority is the metadata key for task priority level.
	metadataKeyPriority = "priority"

	// metadataKeyAttempt is the metadata key for the task attempt number.
	metadataKeyAttempt = "attempt"

	// metadataKeyLastError is the metadata key for the last error message.
	metadataKeyLastError = "last_error"

	// metadataKeyTaskCount is the metadata key for the total task count.
	metadataKeyTaskCount = "task_count"

	// metadataKeyCompleteCount is the metadata key for the count of completed tasks.
	metadataKeyCompleteCount = "complete_count"

	// metadataKeyFailedCount is the metadata key for the number of failed tasks.
	metadataKeyFailedCount = "failed_count"

	// metadataKeyActiveCount is the metadata key for the count of active items.
	metadataKeyActiveCount = "active_count"

	// metadataKeyProgress is the metadata key for workflow completion percentage.
	metadataKeyProgress = "progress"

	// metadataKeyVariantCount is the metadata key for storing the variant count.
	metadataKeyVariantCount = "variant_count"

	// metadataKeyTotalSize is the metadata key for the total size of an artefact.
	metadataKeyTotalSize = "total_size"

	// metadataKeySourcePath is the metadata key for the artefact source path.
	metadataKeySourcePath = "source_path"

	// idDisplayLength is the number of characters to show when shortening IDs.
	idDisplayLength = 8

	// intFormat is the format string for formatting integers as decimal strings.
	intFormat = "%d"
)

// ResourceProvider provides access to remote resources using gRPC.
// It implements tui_domain.ResourceProvider and io.Closer.
type ResourceProvider struct {
	// lastError stores the most recent error from a refresh operation.
	// Placed first for optimal alignment (16 bytes for error interface).
	lastError error

	// conn holds the gRPC connection and service clients.
	conn *Connection

	// summary holds resource counts grouped by kind and status.
	summary map[string]map[tui_domain.ResourceStatus]int

	// tasks holds the cached orchestrator tasks for listing.
	tasks []tui_domain.Resource

	// workflows holds the cached workflow resources from the orchestrator.
	workflows []tui_domain.Resource

	// artefacts holds the cached registry artefact resources.
	artefacts []tui_domain.Resource

	// interval specifies the refresh interval for the resource provider.
	interval time.Duration

	// mu guards concurrent access to tasks, workflows, and summary fields.
	mu sync.RWMutex
}

// NewResourceProvider creates a new ResourceProvider with the given connection
// and refresh interval.
//
// Takes conn (*Connection) which is the shared gRPC connection.
// Takes interval (time.Duration) which sets how often data is refreshed.
//
// Returns *ResourceProvider which is ready to use.
func NewResourceProvider(conn *Connection, interval time.Duration) *ResourceProvider {
	return &ResourceProvider{
		conn:      conn,
		lastError: nil,
		tasks:     make([]tui_domain.Resource, 0),
		workflows: make([]tui_domain.Resource, 0),
		artefacts: make([]tui_domain.Resource, 0),
		summary:   make(map[string]map[tui_domain.ResourceStatus]int),
		mu:        sync.RWMutex{},
		interval:  interval,
	}
}

// Name returns the provider name.
//
// Returns string which is the identifier for this resource provider.
func (*ResourceProvider) Name() string {
	return "grpc-resources"
}

// Health checks if the gRPC connection is healthy.
//
// Returns error when the health check fails.
func (p *ResourceProvider) Health(ctx context.Context) error {
	_, err := p.conn.healthClient.GetHealth(ctx, &pb.GetHealthRequest{})
	if err != nil {
		return fmt.Errorf("checking resource provider health via gRPC: %w", err)
	}
	return nil
}

// Close releases resources.
//
// Returns error when the resources cannot be released.
func (*ResourceProvider) Close() error {
	return nil
}

// RefreshInterval returns the refresh interval.
//
// Returns time.Duration which is the interval between resource refreshes.
func (p *ResourceProvider) RefreshInterval() time.Duration {
	return p.interval
}

// Refresh fetches the latest data via gRPC.
//
// Returns error when the data cannot be fetched, though partial failures are
// tolerated and logged.
//
// Safe for concurrent use. Updates internal state under a mutex lock.
func (p *ResourceProvider) Refresh(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	startTime := time.Now()
	defer func() {
		duration := float64(time.Since(startTime).Milliseconds())
		GRPCCallDuration.Record(ctx, duration)
	}()

	GRPCCallCount.Add(ctx, 1)

	taskSummary, tasks, workflows, orchestratorErr := p.fetchOrchestratorData(ctx)
	if orchestratorErr != nil {
		GRPCCallErrorCount.Add(ctx, 1)
		l.Warn("Failed to fetch orchestrator data", logger.Error(orchestratorErr))
	}

	artefactSummary, artefacts, registryErr := p.fetchRegistryData(ctx)
	if registryErr != nil {
		GRPCCallErrorCount.Add(ctx, 1)
		l.Warn("Failed to fetch registry data", logger.Error(registryErr))
	}

	summary := make(map[string]map[tui_domain.ResourceStatus]int)
	maps.Copy(summary, taskSummary)
	maps.Copy(summary, artefactSummary)

	combined := errors.Join(orchestratorErr, registryErr)

	p.mu.Lock()
	p.tasks = tasks
	p.workflows = workflows
	p.artefacts = artefacts
	p.summary = summary
	p.lastError = combined
	p.mu.Unlock()

	return combined
}

// Kinds returns the resource kinds this provider supports.
//
// Returns []string which contains the kind identifiers for orchestrator tasks,
// workflows, and registry artefacts.
func (*ResourceProvider) Kinds() []string {
	return []string{kindOrchestratorTask, kindOrchestratorWorkflow, kindRegistryArtefact}
}

// List returns resources of the specified kind.
//
// Takes kind (string) which specifies the resource type to retrieve.
//
// Returns []tui_domain.Resource which contains copies of the matching resources.
// Returns error when the kind is not recognised.
//
// Safe for concurrent use; protects internal state with a read lock.
func (p *ResourceProvider) List(_ context.Context, kind string) ([]tui_domain.Resource, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	switch kind {
	case kindOrchestratorTask:
		result := make([]tui_domain.Resource, len(p.tasks))
		copy(result, p.tasks)
		return result, nil
	case kindOrchestratorWorkflow:
		result := make([]tui_domain.Resource, len(p.workflows))
		copy(result, p.workflows)
		return result, nil
	case kindRegistryArtefact:
		result := make([]tui_domain.Resource, len(p.artefacts))
		copy(result, p.artefacts)
		return result, nil
	default:
		return nil, tui_domain.ErrInvalidResourceKind
	}
}

// Summary returns aggregate counts by status.
//
// Returns map[string]map[tui_domain.ResourceStatus]int which contains counts
// grouped by resource kind and status.
// Returns error when retrieval fails.
//
// Safe for concurrent use; acquires a read lock on the provider.
func (p *ResourceProvider) Summary(_ context.Context) (map[string]map[tui_domain.ResourceStatus]int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]map[tui_domain.ResourceStatus]int)
	for kind, statusCounts := range p.summary {
		result[kind] = make(map[tui_domain.ResourceStatus]int)
		maps.Copy(result[kind], statusCounts)
	}

	return result, nil
}

// fetchOrchestratorData retrieves orchestrator data via gRPC.
//
// Returns map[string]map[tui_domain.ResourceStatus]int which contains
// task counts grouped by kind and status.
// Returns []tui_domain.Resource which contains the recent tasks.
// Returns []tui_domain.Resource which contains the workflow summaries.
// Returns error when fetching task summary, recent tasks, or workflow
// data fails.
func (p *ResourceProvider) fetchOrchestratorData(ctx context.Context) (
	summary map[string]map[tui_domain.ResourceStatus]int,
	tasks []tui_domain.Resource,
	workflows []tui_domain.Resource,
	err error,
) {
	summary = make(map[string]map[tui_domain.ResourceStatus]int)

	taskSummaryResp, err := p.conn.orchestratorClient.GetTaskSummary(ctx, &pb.GetTaskSummaryRequest{})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fetching task summary: %w", err)
	}

	summary[kindOrchestratorTask] = make(map[tui_domain.ResourceStatus]int)
	for _, item := range taskSummaryResp.GetSummaries() {
		status := mapTaskStatus(item.GetStatus())
		summary[kindOrchestratorTask][status] = safeconv.Int64ToInt(item.GetCount())
	}

	tasksResp, err := p.conn.orchestratorClient.ListRecentTasks(ctx, &pb.ListRecentTasksRequest{
		Limit: defaultRecentTasksLimit,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fetching recent tasks: %w", err)
	}

	for _, task := range tasksResp.GetTasks() {
		tasks = append(tasks, convertTask(task))
	}

	workflowsResp, err := p.conn.orchestratorClient.ListWorkflowSummary(ctx, &pb.ListWorkflowSummaryRequest{
		Limit: defaultWorkflowSummaryLimit,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fetching workflow summary: %w", err)
	}

	for _, workflow := range workflowsResp.GetSummaries() {
		workflows = append(workflows, convertWorkflow(workflow))
	}

	return summary, tasks, workflows, nil
}

// fetchRegistryData retrieves registry data via gRPC.
//
// Returns map[string]map[tui_domain.ResourceStatus]int which contains resource
// counts grouped by kind and status.
// Returns []tui_domain.Resource which contains the most recent artefacts.
// Returns error when fetching artefact summary, variant summary, or recent
// artefacts fails.
func (p *ResourceProvider) fetchRegistryData(ctx context.Context) (
	map[string]map[tui_domain.ResourceStatus]int,
	[]tui_domain.Resource,
	error,
) {
	summary := make(map[string]map[tui_domain.ResourceStatus]int)

	artefactSummaryResp, err := p.conn.registryClient.GetArtefactSummary(ctx, &pb.GetArtefactSummaryRequest{})
	if err != nil {
		return nil, nil, fmt.Errorf("fetching artefact summary: %w", err)
	}

	summary[kindRegistryArtefact] = make(map[tui_domain.ResourceStatus]int)
	for _, item := range artefactSummaryResp.GetSummaries() {
		status := mapArtefactStatus(item.GetStatus())
		summary[kindRegistryArtefact][status] = safeconv.Int64ToInt(item.GetCount())
	}

	variantSummaryResp, err := p.conn.registryClient.GetVariantSummary(ctx, &pb.GetVariantSummaryRequest{})
	if err != nil {
		return nil, nil, fmt.Errorf("fetching variant summary: %w", err)
	}

	summary[kindRegistryVariant] = make(map[tui_domain.ResourceStatus]int)
	for _, item := range variantSummaryResp.GetSummaries() {
		status := mapArtefactStatus(item.GetStatus())
		summary[kindRegistryVariant][status] = safeconv.Int64ToInt(item.GetCount())
	}

	artefactsResp, err := p.conn.registryClient.ListRecentArtefacts(ctx, &pb.ListRecentArtefactsRequest{
		Limit: defaultRecentArtefactsLimit,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("fetching recent artefacts: %w", err)
	}

	pbArtefacts := artefactsResp.GetArtefacts()
	artefacts := make([]tui_domain.Resource, 0, len(pbArtefacts))
	for _, artefact := range pbArtefacts {
		artefacts = append(artefacts, convertArtefact(artefact))
	}

	return summary, artefacts, nil
}

// convertTask converts a protobuf task to a TUI resource.
//
// Takes task (*pb.TaskListItem) which is the protobuf task to convert.
//
// Returns tui_domain.Resource which is the converted TUI resource with
// metadata including workflow ID, executor, priority, and attempt count.
func convertTask(task *pb.TaskListItem) tui_domain.Resource {
	id := task.GetId()
	displayID := id
	if len(id) > idDisplayLength {
		displayID = id[:idDisplayLength]
	}

	return tui_domain.Resource{
		Kind:       kindOrchestratorTask,
		ID:         id,
		Name:       fmt.Sprintf("%s (%s)", task.GetExecutor(), displayID),
		Status:     mapTaskStatus(task.GetStatus()),
		StatusText: task.GetStatus(),
		Metadata: map[string]string{
			metadataKeyWorkflowID: task.GetWorkflowId(),
			metadataKeyExecutor:   task.GetExecutor(),
			metadataKeyPriority:   mapPriority(task.GetPriority()),
			metadataKeyAttempt:    fmt.Sprintf(intFormat, task.GetAttempt()),
			metadataKeyLastError:  task.GetLastError(),
		},
		Children:  nil,
		CreatedAt: time.Unix(task.GetCreatedAt(), 0),
		UpdatedAt: time.Unix(task.GetUpdatedAt(), 0),
	}
}

// convertWorkflow converts a protobuf workflow summary to a TUI resource.
//
// Takes workflow (*pb.WorkflowSummary) which provides the workflow data to
// convert.
//
// Returns tui_domain.Resource which contains the formatted workflow data for
// display.
func convertWorkflow(workflow *pb.WorkflowSummary) tui_domain.Resource {
	id := workflow.GetWorkflowId()
	displayID := id
	if len(id) > idDisplayLength {
		displayID = id[:idDisplayLength] + "..."
	}

	taskCount := workflow.GetTaskCount()
	completeCount := workflow.GetCompleteCount()
	failedCount := workflow.GetFailedCount()
	activeCount := workflow.GetActiveCount()

	var status tui_domain.ResourceStatus
	var statusText string
	switch {
	case failedCount > 0:
		status = tui_domain.ResourceStatusUnhealthy
		statusText = "FAILED"
	case activeCount > 0:
		status = tui_domain.ResourceStatusPending
		statusText = "ACTIVE"
	case completeCount == taskCount:
		status = tui_domain.ResourceStatusHealthy
		statusText = "COMPLETE"
	default:
		status = tui_domain.ResourceStatusUnknown
		statusText = "UNKNOWN"
	}

	progress := int64(0)
	if taskCount > 0 {
		progress = (completeCount * percentageMultiplier) / taskCount
	}

	return tui_domain.Resource{
		Kind:       kindOrchestratorWorkflow,
		ID:         id,
		Name:       displayID,
		Status:     status,
		StatusText: statusText,
		Metadata: map[string]string{
			metadataKeyTaskCount:     fmt.Sprintf(intFormat, taskCount),
			metadataKeyCompleteCount: fmt.Sprintf(intFormat, completeCount),
			metadataKeyFailedCount:   fmt.Sprintf(intFormat, failedCount),
			metadataKeyActiveCount:   fmt.Sprintf(intFormat, activeCount),
			metadataKeyProgress:      fmt.Sprintf(intFormat+"%%", progress),
		},
		Children:  nil,
		CreatedAt: time.Unix(workflow.GetCreatedAt(), 0),
		UpdatedAt: time.Unix(workflow.GetUpdatedAt(), 0),
	}
}

// convertArtefact converts a protobuf artefact to a TUI resource.
//
// Takes artefact (*pb.ArtefactListItem) which is the protobuf artefact to
// convert.
//
// Returns tui_domain.Resource which is the converted TUI resource.
func convertArtefact(artefact *pb.ArtefactListItem) tui_domain.Resource {
	id := artefact.GetId()
	name := id
	if sourcePath := artefact.GetSourcePath(); sourcePath != "" {
		name = sourcePath
	}

	return tui_domain.Resource{
		Kind:       kindRegistryArtefact,
		ID:         id,
		Name:       name,
		Status:     mapArtefactStatus(artefact.GetStatus()),
		StatusText: artefact.GetStatus(),
		Metadata: map[string]string{
			metadataKeyVariantCount: fmt.Sprintf(intFormat, artefact.GetVariantCount()),
			metadataKeyTotalSize:    inspector.FormatBytes(safeconv.Int64ToUint64(artefact.GetTotalSize())),
			metadataKeySourcePath:   artefact.GetSourcePath(),
		},
		Children:  nil,
		CreatedAt: time.Unix(artefact.GetCreatedAt(), 0),
		UpdatedAt: time.Unix(artefact.GetUpdatedAt(), 0),
	}
}

// mapTaskStatus converts a status string to ResourceStatus.
//
// Takes status (string) which is the task status to convert.
//
// Returns tui_domain.ResourceStatus which represents the mapped status value.
func mapTaskStatus(status string) tui_domain.ResourceStatus {
	switch status {
	case "COMPLETE":
		return tui_domain.ResourceStatusHealthy
	case "FAILED":
		return tui_domain.ResourceStatusUnhealthy
	case "PROCESSING":
		return tui_domain.ResourceStatusDegraded
	case "PENDING", "SCHEDULED", "RETRYING":
		return tui_domain.ResourceStatusPending
	default:
		return tui_domain.ResourceStatusUnknown
	}
}

// mapArtefactStatus converts a status string to ResourceStatus.
//
// Takes status (string) which is the artefact status to convert.
//
// Returns tui_domain.ResourceStatus which is the mapped resource status.
func mapArtefactStatus(status string) tui_domain.ResourceStatus {
	switch status {
	case "READY":
		return tui_domain.ResourceStatusHealthy
	case "STALE":
		return tui_domain.ResourceStatusDegraded
	case "PENDING":
		return tui_domain.ResourceStatusPending
	default:
		return tui_domain.ResourceStatusUnknown
	}
}

// mapPriority converts a priority integer to a human-readable string.
//
// Takes priority (int32) which is the numeric priority level to convert.
//
// Returns string which is the human-readable priority label.
func mapPriority(priority int32) string {
	switch priority {
	case 0:
		return "Low"
	case 1:
		return "Normal"
	case 2:
		return "High"
	default:
		return fmt.Sprintf("P%d", priority)
	}
}
