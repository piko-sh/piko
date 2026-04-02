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

package monitoring_transport_grpc

import (
	"context"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/registry/registry_domain"
)

type mockRenderCacheStatsProvider struct {
	componentCacheSize int
	svgCacheSize       int
}

func (m *mockRenderCacheStatsProvider) GetComponentCacheSize() int {
	return m.componentCacheSize
}

func (m *mockRenderCacheStatsProvider) GetSVGCacheSize() int {
	return m.svgCacheSize
}

var _ RenderCacheStatsProvider = (*mockRenderCacheStatsProvider)(nil)

type mockOrchestratorInspector struct {
	taskSummaryError      error
	recentTasksError      error
	workflowSummaryError  error
	taskSummaryReturn     []orchestrator_domain.TaskSummary
	recentTasksReturn     []orchestrator_domain.TaskListItem
	workflowSummaryReturn []orchestrator_domain.WorkflowSummary
}

func (m *mockOrchestratorInspector) ListTaskSummary(_ context.Context) ([]orchestrator_domain.TaskSummary, error) {
	return m.taskSummaryReturn, m.taskSummaryError
}

func (m *mockOrchestratorInspector) ListRecentTasks(_ context.Context, _ int32) ([]orchestrator_domain.TaskListItem, error) {
	return m.recentTasksReturn, m.recentTasksError
}

func (m *mockOrchestratorInspector) ListWorkflowSummary(_ context.Context, _ int32) ([]orchestrator_domain.WorkflowSummary, error) {
	return m.workflowSummaryReturn, m.workflowSummaryError
}

var _ orchestrator_domain.OrchestratorInspector = (*mockOrchestratorInspector)(nil)

type mockRegistryInspector struct {
	artefactSummaryError  error
	variantSummaryError   error
	recentArtefactsError  error
	artefactSummaryReturn []registry_domain.ArtefactSummary
	variantSummaryReturn  []registry_domain.VariantSummary
	recentArtefactsReturn []registry_domain.ArtefactListItem
}

func (m *mockRegistryInspector) ListArtefactSummary(_ context.Context) ([]registry_domain.ArtefactSummary, error) {
	return m.artefactSummaryReturn, m.artefactSummaryError
}

func (m *mockRegistryInspector) ListVariantSummary(_ context.Context) ([]registry_domain.VariantSummary, error) {
	return m.variantSummaryReturn, m.variantSummaryError
}

func (m *mockRegistryInspector) ListRecentArtefacts(_ context.Context, _ int32) ([]registry_domain.ArtefactListItem, error) {
	return m.recentArtefactsReturn, m.recentArtefactsError
}

var _ registry_domain.RegistryInspector = (*mockRegistryInspector)(nil)
