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

package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"piko.sh/piko/internal/orchestrator/orchestrator_adapters"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

// Service is the main orchestrator service interface.
type Service = orchestrator_domain.OrchestratorService

// Task represents a unit of work to be run by the orchestrator.
type Task = orchestrator_domain.Task

// TaskConfig holds configuration for a task.
type TaskConfig = orchestrator_domain.TaskConfig

// WorkflowReceipt represents the result of scheduling a workflow.
type WorkflowReceipt = orchestrator_domain.WorkflowReceipt

// TaskExecutor is the interface for executing tasks.
type TaskExecutor = orchestrator_domain.TaskExecutor

const (
	// defaultWorkerCount is the default number of worker goroutines.
	defaultWorkerCount = 4

	// defaultSchedulerIntervalSeconds is the default interval for the scheduler.
	defaultSchedulerIntervalSeconds = 10

	// PriorityLow is the lowest task priority level.
	PriorityLow = orchestrator_domain.PriorityLow

	// PriorityNormal is the default priority level for standard operations.
	PriorityNormal = orchestrator_domain.PriorityNormal

	// PriorityHigh is the highest priority level for task scheduling.
	PriorityHigh = orchestrator_domain.PriorityHigh
)

var (
	// errTaskStoreNil is returned when the orchestrator configuration has a
	// nil TaskStore.
	errTaskStoreNil = errors.New("config: TaskStore cannot be nil")

	// errEventBusNil is returned when the orchestrator configuration has a
	// nil EventBus.
	errEventBusNil = errors.New("config: EventBus cannot be nil (required for event-driven orchestration)")
)

// Config holds the settings for the orchestrator service.
type Config struct {
	// TaskStore provides persistence for task state and retrieval.
	TaskStore orchestrator_domain.TaskStore

	// EventBus handles event-driven communication between components.
	EventBus orchestrator_domain.EventBus

	// WorkerCount specifies the number of workers; 0 or negative uses the default.
	WorkerCount int

	// SchedulerInterval is time between scheduler runs; 0 uses the default.
	SchedulerInterval time.Duration

	// DispatcherInterval is the time between dispatch cycles; defaults to 1 second
	// if not set or non-positive.
	DispatcherInterval time.Duration
}

// Validate reports whether the configuration is consistent.
//
// Returns error when TaskStore or EventBus is nil.
func (c *Config) Validate() error {
	if c.TaskStore == nil {
		return errTaskStoreNil
	}
	if c.EventBus == nil {
		return errEventBusNil
	}
	return nil
}

// NewTask creates a new task with the given executor name and payload.
//
// Takes executor (string) which specifies the name of the task executor.
// Takes payload (map[string]any) which contains the data for the task.
//
// Returns *Task which is the newly created task.
func NewTask(executor string, payload map[string]any) *Task {
	return orchestrator_domain.NewTask(executor, payload)
}

// NewService creates a new orchestrator service with the given configuration.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation through background goroutines.
// Takes config (Config) which specifies the orchestrator settings including worker
// count, scheduler interval, and dispatcher interval. Default values are used
// for any unset or invalid fields.
//
// Returns Service which is the configured orchestrator ready for use.
// Returns error when validation fails or the task dispatcher cannot be created.
func NewService(ctx context.Context, config Config) (Service, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validating orchestrator config: %w", err)
	}

	if config.WorkerCount <= 0 {
		config.WorkerCount = defaultWorkerCount
	}
	if config.SchedulerInterval <= 0 {
		config.SchedulerInterval = defaultSchedulerIntervalSeconds * time.Second
	}
	if config.DispatcherInterval <= 0 {
		config.DispatcherInterval = 1 * time.Second
	}

	dispatcherConfig := orchestrator_domain.DefaultDispatcherConfig()
	taskDispatcher := orchestrator_adapters.CreateTaskDispatcher(ctx, dispatcherConfig, config.EventBus, config.TaskStore)

	service := orchestrator_domain.NewService(
		ctx,
		config.TaskStore,
		config.EventBus,
		orchestrator_domain.WithTaskDispatcher(taskDispatcher),
	)

	return service, nil
}
