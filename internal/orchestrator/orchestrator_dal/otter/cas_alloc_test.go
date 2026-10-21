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
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/orchestrator/orchestrator_domain"
	"piko.sh/piko/internal/testutil/allocbudget"
)

const (
	casAllocTasks = 600
)

func newCASAllocStore(t *testing.T) *DAL {
	t.Helper()
	opened, err := NewOtterDAL(Config{})
	require.NoError(t, err)
	t.Cleanup(func() { _ = opened.Close() })
	dal, ok := opened.(*DAL)
	require.True(t, ok)
	now := time.Now()
	for i := range casAllocTasks {
		require.NoError(t, dal.CreateTask(context.Background(), &orchestrator_domain.Task{
			ID: "cas-" + strconv.Itoa(i), WorkflowID: "cas-" + strconv.Itoa(i), Executor: "bench",
			Status: orchestrator_domain.StatusPending, Payload: map[string]any{"artefactID": "a"},
			ExecuteAt: now, CreatedAt: now, UpdatedAt: now, DeduplicationKey: "cas-" + strconv.Itoa(i),
		}))
	}
	return dal
}

func TestAllocBudget_OtterClaim(t *testing.T) {
	dal := newCASAllocStore(t)
	ctx := context.Background()
	ids := make([]string, casAllocTasks)
	for i := range ids {
		ids[i] = "cas-" + strconv.Itoa(i)
	}
	next := 0
	now := time.Now()
	allocbudget.Assert(t, allocbudget.Budget{Allocs: 6, Bytes: 1536, Runs: 500}, func() {
		if _, err := dal.ClaimTask(ctx, orchestrator_domain.ClaimRequest{Now: now, ID: ids[next], Owner: "node"}); err != nil {
			t.Fatal(err)
		}
		next++
	})
}

func TestAllocBudget_OtterSettle(t *testing.T) {
	dal := newCASAllocStore(t)
	ctx := context.Background()
	now := time.Now()
	claimed := make([]*orchestrator_domain.Task, casAllocTasks)
	for i := range claimed {
		task, err := dal.ClaimTask(ctx, orchestrator_domain.ClaimRequest{Now: now, ID: "cas-" + strconv.Itoa(i), Owner: "node"})
		require.NoError(t, err)
		task.Status = orchestrator_domain.StatusComplete
		claimed[i] = task
	}
	next := 0
	allocbudget.Assert(t, allocbudget.Budget{Allocs: 3, Bytes: 768, Runs: 500}, func() {
		if _, err := dal.SettleTask(ctx, orchestrator_domain.Settlement{Task: claimed[next]}); err != nil {
			t.Fatal(err)
		}
		next++
	})
}
