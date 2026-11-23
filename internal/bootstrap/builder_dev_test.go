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

package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

type schedulingOrchestratorService struct {
	orchestrator_domain.OrchestratorService
}

func (*schedulingOrchestratorService) GetTaskDispatcher() orchestrator_domain.TaskDispatcher {
	return nil
}

func (*schedulingOrchestratorService) Schedule(context.Context, *orchestrator_domain.Task, time.Time) (*orchestrator_domain.WorkflowReceipt, error) {
	return nil, nil
}

func TestBuildDevDaemon_AssemblesAWatchingDaemon(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	buildTestProject(t, baseDir)
	c := newResolvedContainer(t, baseDir, WithDevWidget(), WithDevHotreload())

	daemon, err := Daemon(context.Background(), runModeDev, c, &Dependencies{AppRouter: chi.NewRouter()})

	require.NoError(t, err)
	assert.NotNil(t, daemon)
	waitForInitialTasks(t, c)
	require.NotNil(t, c.seoProductionMode)
	assert.False(t, *c.seoProductionMode)
}

func TestBuildDevDaemon_FailsWithoutAGeneratedManifest(t *testing.T) {
	baseDir := newTestProject(t, map[string]string{"pages/index.pk": testProjectIndexPage})
	c := newResolvedContainer(t, baseDir, WithOrchestratorService(&schedulingOrchestratorService{}))

	daemon, err := Daemon(context.Background(), runModeDev, c, &Dependencies{AppRouter: chi.NewRouter()})

	require.Error(t, err)
	assert.Nil(t, daemon)
	assert.Contains(t, err.Error(), "piko generate all")
}
