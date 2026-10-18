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

package otter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
)

func taggedTask(id, tag string, status orchestrator_domain.TaskStatus) *orchestrator_domain.Task {
	task := makeTask(id, id)
	task.BuildTag = tag
	task.Status = status
	return task
}

func unsettled(t *testing.T, d *DAL, tag string) int64 {
	t.Helper()
	count, err := d.CountUnsettled(context.Background(), tag)
	require.NoError(t, err)
	return count
}

func TestCountUnsettled_FollowsStatusChanges(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	require.NoError(t, d.CreateTask(ctx, taggedTask("a", "build-1", orchestrator_domain.StatusPending)))
	require.NoError(t, d.CreateTask(ctx, taggedTask("b", "build-1", orchestrator_domain.StatusScheduled)))
	require.NoError(t, d.CreateTask(ctx, taggedTask("c", "build-2", orchestrator_domain.StatusPending)))
	assert.Equal(t, int64(2), unsettled(t, d, "build-1"))
	assert.Equal(t, int64(1), unsettled(t, d, "build-2"))

	for _, status := range []orchestrator_domain.TaskStatus{
		orchestrator_domain.StatusProcessing,
		orchestrator_domain.StatusRetrying,
	} {
		require.NoError(t, d.UpdateTask(ctx, taggedTask("a", "build-1", status)))
		assert.Equal(t, int64(2), unsettled(t, d, "build-1"), "%s is still unsettled", status)
	}

	require.NoError(t, d.UpdateTask(ctx, taggedTask("a", "build-1", orchestrator_domain.StatusComplete)))
	require.NoError(t, d.UpdateTask(ctx, taggedTask("b", "build-1", orchestrator_domain.StatusFailed)))
	assert.Zero(t, unsettled(t, d, "build-1"))
	assert.Equal(t, int64(1), unsettled(t, d, "build-2"))
}

func TestCountUnsettled_IgnoresBackgroundAndUntagged(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()

	background := taggedTask("gc", "build-1", orchestrator_domain.StatusScheduled)
	background.Background = true
	require.NoError(t, d.CreateTask(ctx, background))
	require.NoError(t, d.CreateTask(ctx, taggedTask("untagged", "", orchestrator_domain.StatusPending)))

	assert.Zero(t, unsettled(t, d, "build-1"))
	assert.Zero(t, unsettled(t, d, ""))
}

func TestCountUnsettled_RollbackRestoresCount(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()
	require.NoError(t, d.CreateTask(ctx, taggedTask("kept", "build-1", orchestrator_domain.StatusPending)))

	errAbort := errors.New("abort")
	err := d.RunAtomic(ctx, func(ctx context.Context, store orchestrator_domain.TaskStore) error {
		if err := store.CreateTask(ctx, taggedTask("added", "build-1", orchestrator_domain.StatusPending)); err != nil {
			return err
		}
		if err := store.UpdateTask(ctx, taggedTask("kept", "build-1", orchestrator_domain.StatusComplete)); err != nil {
			return err
		}
		return errAbort
	})
	require.ErrorIs(t, err, errAbort)
	assert.Equal(t, int64(1), unsettled(t, d, "build-1"), "the rolled-back create and settle leave the count as it was")
}

func TestCountUnsettled_RebuiltAfterRestart(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()
	require.NoError(t, d.CreateTask(ctx, taggedTask("a", "build-1", orchestrator_domain.StatusPending)))
	require.NoError(t, d.CreateTask(ctx, taggedTask("b", "build-1", orchestrator_domain.StatusComplete)))

	d.RebuildIndexes(ctx)
	assert.Equal(t, int64(1), unsettled(t, d, "build-1"))
}

func TestListFailedForBuild(t *testing.T) {
	d := newTestDAL(t)
	ctx := context.Background()
	require.NoError(t, d.CreateTask(ctx, taggedTask("failed", "build-1", orchestrator_domain.StatusFailed)))
	require.NoError(t, d.CreateTask(ctx, taggedTask("other-build", "build-2", orchestrator_domain.StatusFailed)))
	require.NoError(t, d.CreateTask(ctx, taggedTask("done", "build-1", orchestrator_domain.StatusComplete)))

	failed, err := d.ListFailedForBuild(ctx, "build-1")
	require.NoError(t, err)
	require.Len(t, failed, 1)
	assert.Equal(t, "failed", failed[0].ID)
}
