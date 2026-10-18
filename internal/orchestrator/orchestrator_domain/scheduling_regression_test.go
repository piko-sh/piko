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

package orchestrator_domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/clock"
)

func TestSchedulePersistsDueTimeBeforeDispatch(t *testing.T) {
	store := NewFakeTaskStore()
	service := &orchestratorService{taskStore: store, clock: clock.RealClock(), taskInsertChan: make(chan *Task, 1), receipts: make(map[string][]*WorkflowReceipt)}
	task := NewTask("test", nil)
	due := time.Now().Add(time.Hour)
	_, err := service.Schedule(t.Context(), task, due)
	require.NoError(t, err)
	queued := <-service.taskInsertChan
	require.Equal(t, due.UTC(), queued.ScheduledExecuteAt)
	require.NoError(t, store.CreateTask(t.Context(), queued))
	stored, err := store.GetTasksByID(t.Context(), []string{task.ID})
	require.NoError(t, err)
	require.True(t, stored[0].ScheduledExecuteAt.Equal(due))
}
