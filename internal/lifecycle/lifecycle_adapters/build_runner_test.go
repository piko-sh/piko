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

package lifecycle_adapters

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func TestNewBuildService(t *testing.T) {
	t.Parallel()

	t.Run("creates build service with all fields initialised", func(t *testing.T) {
		t.Parallel()

		websiteConfig := &config.WebsiteConfig{}
		pathsConfig := lifecycle_domain.LifecyclePathsConfig{
			BaseDir: "/project",
		}

		service := NewBuildService(
			websiteConfig,
			pathsConfig,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		)

		require.NotNil(t, service)

		bs, ok := service.(*buildService)
		require.True(t, ok, "Expected *buildService type")
		assert.Equal(t, websiteConfig, bs.websiteConfig)
		assert.Equal(t, "/project", bs.pathsConfig.BaseDir)
	})

	t.Run("returns BuilderAdapter interface", func(t *testing.T) {
		t.Parallel()

		websiteConfig := &config.WebsiteConfig{}

		service := NewBuildService(
			websiteConfig,
			lifecycle_domain.LifecyclePathsConfig{},
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		)

		require.NotNil(t, service)
		_, ok := service.(*buildService)
		require.True(t, ok, "Expected *buildService type")
	})
}

func TestBuildService_getDirectories(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no directories configured", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:             "/project",
				AssetsSourceDir:     "",
				ComponentsSourceDir: "",
			},
		}

		dirs := bs.getDirectories()

		assert.Empty(t, dirs)
	})

	t.Run("returns assets directory when configured", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:             "/project",
				AssetsSourceDir:     "lib",
				ComponentsSourceDir: "",
			},
		}

		dirs := bs.getDirectories()

		require.Len(t, dirs, 1)
		assert.Equal(t, "/project/lib", dirs[0])
	})

	t.Run("returns components directory when configured", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:             "/project",
				AssetsSourceDir:     "",
				ComponentsSourceDir: "components",
			},
		}

		dirs := bs.getDirectories()

		require.Len(t, dirs, 1)
		assert.Equal(t, "/project/components", dirs[0])
	})

	t.Run("returns both directories when both configured", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:             "/project",
				AssetsSourceDir:     "lib",
				ComponentsSourceDir: "components",
			},
		}

		dirs := bs.getDirectories()

		require.Len(t, dirs, 2)
		assert.Contains(t, dirs, "/project/lib")
		assert.Contains(t, dirs, "/project/components")
	})

	t.Run("preserves order - assets before components", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:             "/project",
				AssetsSourceDir:     "lib",
				ComponentsSourceDir: "components",
			},
		}

		dirs := bs.getDirectories()

		require.Len(t, dirs, 2)
		assert.Equal(t, "/project/lib", dirs[0])
		assert.Equal(t, "/project/components", dirs[1])
	})

	t.Run("handles nested directory paths", func(t *testing.T) {
		t.Parallel()

		bs := &buildService{
			pathsConfig: lifecycle_domain.LifecyclePathsConfig{
				BaseDir:             "/home/user/project",
				AssetsSourceDir:     "src/assets/images",
				ComponentsSourceDir: "src/components/ui",
			},
		}

		dirs := bs.getDirectories()

		require.Len(t, dirs, 2)
		assert.Equal(t, "/home/user/project/src/assets/images", dirs[0])
		assert.Equal(t, "/home/user/project/src/components/ui", dirs[1])
	})
}

func TestPollingConstants(t *testing.T) {
	t.Parallel()

	t.Run("pollInterval is reasonable", func(t *testing.T) {
		t.Parallel()

		assert.GreaterOrEqual(t, pollInterval, 10*time.Millisecond)
		assert.LessOrEqual(t, pollInterval, time.Second)
	})

	t.Run("defaultTimeout is reasonable", func(t *testing.T) {
		t.Parallel()

		assert.GreaterOrEqual(t, defaultTimeout, time.Minute)
		assert.LessOrEqual(t, defaultTimeout, 30*time.Minute)
	})

	t.Run("pollInterval is less than defaultTimeout", func(t *testing.T) {
		t.Parallel()

		assert.Less(t, pollInterval, defaultTimeout/100)
	})
}

func TestFieldConstants(t *testing.T) {
	t.Parallel()

	t.Run("fieldError constant is correct", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "error", fieldError)
	})
}

type stubTaskDispatcher struct {
	orchestrator_domain.TaskDispatcher
	failedTasksErr error
	failedTasks    []orchestrator_domain.FailedTaskSummary
	stats          orchestrator_domain.DispatcherStats
}

func (s *stubTaskDispatcher) Stats() orchestrator_domain.DispatcherStats { return s.stats }
func (s *stubTaskDispatcher) FailedTasks(_ context.Context) ([]orchestrator_domain.FailedTaskSummary, error) {
	return s.failedTasks, s.failedTasksErr
}
func (s *stubTaskDispatcher) IsIdle() bool { return true }
func (s *stubTaskDispatcher) Dispatch(_ context.Context, _ *orchestrator_domain.Task) error {
	return nil
}
func (s *stubTaskDispatcher) DispatchDelayed(_ context.Context, _ *orchestrator_domain.Task, _ time.Time) error {
	return nil
}
func (s *stubTaskDispatcher) RegisterExecutor(_ context.Context, _ string, _ orchestrator_domain.TaskExecutor) {
}
func (s *stubTaskDispatcher) Start(_ context.Context) error { return nil }
func (s *stubTaskDispatcher) SetBuildTag(_ string)          {}
func (s *stubTaskDispatcher) BuildTag() string              { return "" }

type stubOrchestratorService struct {
	dispatcher *stubTaskDispatcher
}

func (s *stubOrchestratorService) RegisterExecutor(_ context.Context, _ string, _ orchestrator_domain.TaskExecutor) error {
	return nil
}
func (s *stubOrchestratorService) Dispatch(_ context.Context, _ *orchestrator_domain.Task) (*orchestrator_domain.WorkflowReceipt, error) {
	return nil, nil
}
func (s *stubOrchestratorService) Schedule(_ context.Context, _ *orchestrator_domain.Task, _ time.Time) (*orchestrator_domain.WorkflowReceipt, error) {
	return nil, nil
}
func (s *stubOrchestratorService) Run(_ context.Context)                {}
func (s *stubOrchestratorService) Stop()                                {}
func (s *stubOrchestratorService) ActiveTasks(_ context.Context) int64  { return 0 }
func (s *stubOrchestratorService) PendingTasks(_ context.Context) int64 { return 0 }
func (s *stubOrchestratorService) GetTaskDispatcher() orchestrator_domain.TaskDispatcher {
	return s.dispatcher
}
func (s *stubOrchestratorService) DispatchDirect(_ context.Context, _ *orchestrator_domain.Task) (*orchestrator_domain.WorkflowReceipt, error) {
	return nil, nil
}

var _ orchestrator_domain.TaskDispatcher = (*stubTaskDispatcher)(nil)
var _ orchestrator_domain.OrchestratorService = (*stubOrchestratorService)(nil)

func TestBuildService_buildResult(t *testing.T) {
	t.Parallel()

	t.Run("returns zero counts when no tasks were dispatched", func(t *testing.T) {
		t.Parallel()

		dispatcher := &stubTaskDispatcher{
			stats: orchestrator_domain.DispatcherStats{},
		}
		bs := &buildService{
			orchestratorService: &stubOrchestratorService{dispatcher: dispatcher},
		}

		zeroStats := orchestrator_domain.DispatcherStats{}
		result := bs.buildResult(context.Background(), time.Now(), zeroStats)

		require.NotNil(t, result)
		assert.Equal(t, int64(0), result.TotalDispatched)
		assert.Equal(t, int64(0), result.TotalCompleted)
		assert.Equal(t, int64(0), result.TotalFailed)
		assert.Equal(t, int64(0), result.TotalFatalFailed)
		assert.Equal(t, int64(0), result.TotalRetried)
		assert.Empty(t, result.Failures)
		assert.False(t, result.HasFailures())
	})

	t.Run("maps dispatcher stats to result fields", func(t *testing.T) {
		t.Parallel()

		dispatcher := &stubTaskDispatcher{
			stats: orchestrator_domain.DispatcherStats{
				TasksDispatched:  20,
				TasksCompleted:   18,
				TasksFailed:      0,
				TasksFatalFailed: 0,
				TasksRetried:     1,
			},
		}
		bs := &buildService{
			orchestratorService: &stubOrchestratorService{dispatcher: dispatcher},
		}

		zeroStats := orchestrator_domain.DispatcherStats{}
		result := bs.buildResult(context.Background(), time.Now(), zeroStats)

		require.NotNil(t, result)
		assert.Equal(t, int64(20), result.TotalDispatched)
		assert.Equal(t, int64(18), result.TotalCompleted)
		assert.Equal(t, int64(0), result.TotalFailed)
		assert.Equal(t, int64(0), result.TotalFatalFailed)
		assert.Equal(t, int64(1), result.TotalRetried)
		assert.Empty(t, result.Failures)
	})

	t.Run("populates failures when tasks have failed", func(t *testing.T) {
		t.Parallel()

		dispatcher := &stubTaskDispatcher{
			stats: orchestrator_domain.DispatcherStats{
				TasksDispatched:  5,
				TasksCompleted:   3,
				TasksFailed:      2,
				TasksFatalFailed: 1,
				TasksRetried:     1,
			},
			failedTasks: []orchestrator_domain.FailedTaskSummary{
				{
					TaskID:     "task-1",
					WorkflowID: "artefact-a",
					Executor:   "minify_css",
					LastError:  "syntax error",
					Attempt:    3,
					IsFatal:    false,
				},
				{
					TaskID:     "task-2",
					WorkflowID: "artefact-b",
					Executor:   "compile_component",
					LastError:  "fatal: missing template",
					Attempt:    1,
					IsFatal:    true,
				},
			},
		}
		bs := &buildService{
			orchestratorService: &stubOrchestratorService{dispatcher: dispatcher},
		}

		zeroStats := orchestrator_domain.DispatcherStats{}
		result := bs.buildResult(context.Background(), time.Now(), zeroStats)

		require.NotNil(t, result)
		assert.Equal(t, int64(2), result.TotalFailed)
		assert.Equal(t, int64(1), result.TotalFatalFailed)
		assert.True(t, result.HasFailures())

		require.Len(t, result.Failures, 2)

		assert.Equal(t, "artefact-a", result.Failures[0].ArtefactID)
		assert.Equal(t, "minify_css", result.Failures[0].Executor)
		assert.Equal(t, "syntax error", result.Failures[0].Error)
		assert.Equal(t, 3, result.Failures[0].Attempt)
		assert.False(t, result.Failures[0].IsFatal)

		assert.Equal(t, "artefact-b", result.Failures[1].ArtefactID)
		assert.Equal(t, "compile_component", result.Failures[1].Executor)
		assert.Equal(t, "fatal: missing template", result.Failures[1].Error)
		assert.Equal(t, 1, result.Failures[1].Attempt)
		assert.True(t, result.Failures[1].IsFatal)
	})

	t.Run("handles FailedTasks retrieval error gracefully", func(t *testing.T) {
		t.Parallel()

		dispatcher := &stubTaskDispatcher{
			stats: orchestrator_domain.DispatcherStats{
				TasksDispatched: 3,
				TasksCompleted:  2,
				TasksFailed:     1,
			},
			failedTasksErr: errors.New("store unavailable"),
		}
		bs := &buildService{
			orchestratorService: &stubOrchestratorService{dispatcher: dispatcher},
		}

		zeroStats := orchestrator_domain.DispatcherStats{}
		result := bs.buildResult(context.Background(), time.Now(), zeroStats)

		require.NotNil(t, result)
		assert.Equal(t, int64(1), result.TotalFailed)
		assert.True(t, result.HasFailures())
		assert.Empty(t, result.Failures, "Failures slice should be empty when retrieval fails")
	})

	t.Run("computes delta stats from start snapshot", func(t *testing.T) {
		t.Parallel()

		dispatcher := &stubTaskDispatcher{
			stats: orchestrator_domain.DispatcherStats{
				TasksDispatched:  30,
				TasksCompleted:   28,
				TasksFailed:      1,
				TasksFatalFailed: 1,
				TasksRetried:     2,
			},
			failedTasks: []orchestrator_domain.FailedTaskSummary{
				{
					TaskID:     "task-new",
					WorkflowID: "artefact-x",
					Executor:   "minify_js",
					LastError:  "new error",
					Attempt:    1,
					IsFatal:    true,
				},
			},
		}
		bs := &buildService{
			orchestratorService: &stubOrchestratorService{dispatcher: dispatcher},
		}

		startStats := orchestrator_domain.DispatcherStats{
			TasksDispatched:  20,
			TasksCompleted:   18,
			TasksFailed:      0,
			TasksFatalFailed: 0,
			TasksRetried:     1,
		}
		result := bs.buildResult(context.Background(), time.Now(), startStats)

		require.NotNil(t, result)
		assert.Equal(t, int64(10), result.TotalDispatched, "should be delta: 30-20")
		assert.Equal(t, int64(10), result.TotalCompleted, "should be delta: 28-18")
		assert.Equal(t, int64(1), result.TotalFailed, "should be delta: 1-0")
		assert.Equal(t, int64(1), result.TotalFatalFailed, "should be delta: 1-0")
		assert.Equal(t, int64(1), result.TotalRetried, "should be delta: 2-1")
		assert.True(t, result.HasFailures())
		require.Len(t, result.Failures, 1)
		assert.Equal(t, "artefact-x", result.Failures[0].ArtefactID)
	})

	t.Run("sets positive duration from build start time", func(t *testing.T) {
		t.Parallel()

		dispatcher := &stubTaskDispatcher{
			stats: orchestrator_domain.DispatcherStats{
				TasksDispatched: 1,
				TasksCompleted:  1,
			},
		}
		bs := &buildService{
			orchestratorService: &stubOrchestratorService{dispatcher: dispatcher},
		}

		startTime := time.Now().Add(-500 * time.Millisecond)
		zeroStats := orchestrator_domain.DispatcherStats{}
		result := bs.buildResult(context.Background(), startTime, zeroStats)

		require.NotNil(t, result)
		assert.Greater(t, result.Duration, time.Duration(0), "Duration should be positive")
		assert.GreaterOrEqual(t, result.Duration, 500*time.Millisecond,
			"Duration should be at least as long as the elapsed time")
	})
}
