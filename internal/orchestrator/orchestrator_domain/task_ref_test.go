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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/testutil/allocbudget"
)

func TestTaskRef_RoundTrips(t *testing.T) {
	t.Parallel()

	for _, ref := range []TaskRef{
		{ID: NewTaskID(), Attempt: 0, Priority: PriorityNormal},
		{ID: NewTaskID(), Attempt: 127, Priority: PriorityHigh, Flags: TaskRefBackground},
		{ID: "a-much-longer-task-identifier-than-a-uuid-would-ever-be-in-practice", Attempt: 100_000, Priority: PriorityLow},
	} {
		encoded := AppendTaskRef(nil, ref)
		decoded, err := DecodeTaskRef(encoded)
		require.NoError(t, err)
		assert.Equal(t, ref, decoded)
	}
}

func TestTaskRef_FitsItsDocumentedSize(t *testing.T) {
	t.Parallel()

	encoded := AppendTaskRef(nil, TaskRef{ID: NewTaskID(), Attempt: 127, Priority: PriorityHigh, Flags: TaskRefBackground})
	assert.LessOrEqual(t, len(encoded), MaxTaskRefSize)
}

func TestTaskRef_RejectsMalformedBytes(t *testing.T) {
	t.Parallel()

	valid := AppendTaskRef(nil, TaskRef{ID: "task", Attempt: 3})
	for name, data := range map[string][]byte{
		"empty":           nil,
		"json":            []byte(`{"id":"task"}`),
		"truncated":       valid[:len(valid)-1],
		"trailing bytes":  append(append([]byte(nil), valid...), 'x'),
		"empty id":        AppendTaskRef(nil, TaskRef{}),
		"unknown version": append([]byte{9}, valid[1:]...),
	} {
		_, err := DecodeTaskRef(data)
		assert.Error(t, err, name)
	}
}

func TestTaskRef_RefOfCarriesTheDelivery(t *testing.T) {
	t.Parallel()

	task := NewTask("executor", nil)
	task.Attempt = 2
	task.Config.Priority = PriorityLow
	task.Background = true
	ref := RefOf(task)
	assert.Equal(t, TaskRef{ID: task.ID, Attempt: 2, Priority: PriorityLow, Flags: TaskRefBackground}, ref)
}

func TestAllocBudget_TaskRef(t *testing.T) {
	ref := TaskRef{ID: NewTaskID(), Attempt: 1, Priority: PriorityNormal}
	buffer := make([]byte, 0, MaxTaskRefSize)
	encoded := AppendTaskRef(nil, ref)

	t.Run("op=encode", func(t *testing.T) {
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 0}, func() {
			buffer = AppendTaskRef(buffer[:0], ref)
		})
	})
	t.Run("op=decode", func(t *testing.T) {
		allocbudget.Assert(t, allocbudget.Budget{Allocs: 1}, func() {
			if _, err := DecodeTaskRef(encoded); err != nil {
				t.Fatal(err)
			}
		})
	})
}

func TestAllocBudget_LocalHandoff(t *testing.T) {
	pool := NewExecutionPool(ExecutionPoolConfig{LaneCapacity: 1024},
		func(context.Context, TaskRef, *ExecutionSlot) {},
		func(context.Context, *Task, *ExecutionSlot) {},
		nil)
	ref := TaskRef{ID: NewTaskID(), Attempt: 1, Priority: PriorityHigh}

	allocbudget.Assert(t, allocbudget.Budget{Allocs: 0, Runs: 500}, func() {
		if !pool.Publish(ref) {
			t.Fatal("the ring refused a reference")
		}
		if queued := <-pool.lanes[PriorityHigh].refs; queued.ID != ref.ID {
			t.Fatal("the reference was not queued")
		}
	})
}
