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
)

func Test_newScheduledTaskHeap(t *testing.T) {
	t.Parallel()

	heap := newScheduledTaskHeap()

	if heap == nil {
		t.Fatal("newScheduledTaskHeap returned nil")
	}
	if heap.Len() != 0 {
		t.Errorf("new heap should be empty, got len=%d", heap.Len())
	}
}

func Test_scheduledTaskHeap_PushAndPop(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name          string
		tasks         []*Task
		expectedOrder []string
	}{
		{
			name:          "single task",
			tasks:         []*Task{{ID: "task-1", ScheduledExecuteAt: baseTime}},
			expectedOrder: []string{"task-1"},
		},
		{
			name: "tasks in order",
			tasks: []*Task{
				{ID: "task-1", ScheduledExecuteAt: baseTime},
				{ID: "task-2", ScheduledExecuteAt: baseTime.Add(1 * time.Hour)},
				{ID: "task-3", ScheduledExecuteAt: baseTime.Add(2 * time.Hour)},
			},
			expectedOrder: []string{"task-1", "task-2", "task-3"},
		},
		{
			name: "tasks in reverse order",
			tasks: []*Task{
				{ID: "task-3", ScheduledExecuteAt: baseTime.Add(2 * time.Hour)},
				{ID: "task-2", ScheduledExecuteAt: baseTime.Add(1 * time.Hour)},
				{ID: "task-1", ScheduledExecuteAt: baseTime},
			},
			expectedOrder: []string{"task-1", "task-2", "task-3"},
		},
		{
			name: "tasks in random order",
			tasks: []*Task{
				{ID: "task-2", ScheduledExecuteAt: baseTime.Add(1 * time.Hour)},
				{ID: "task-4", ScheduledExecuteAt: baseTime.Add(3 * time.Hour)},
				{ID: "task-1", ScheduledExecuteAt: baseTime},
				{ID: "task-3", ScheduledExecuteAt: baseTime.Add(2 * time.Hour)},
			},
			expectedOrder: []string{"task-1", "task-2", "task-3", "task-4"},
		},
		{
			name: "tasks with same time preserve insertion for equal times",
			tasks: []*Task{
				{ID: "task-a", ScheduledExecuteAt: baseTime},
				{ID: "task-b", ScheduledExecuteAt: baseTime},
			},
			expectedOrder: []string{"task-a", "task-b"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			heap := newScheduledTaskHeap()

			for _, task := range tc.tasks {
				heap.Push(task)
			}

			if heap.Len() != len(tc.tasks) {
				t.Errorf("expected len=%d, got len=%d", len(tc.tasks), heap.Len())
			}

			for i, expectedID := range tc.expectedOrder {
				task := heap.Pop()
				if task == nil {
					t.Fatalf("pop %d: expected task, got nil", i)
				}
				if task.ID != expectedID {
					t.Errorf("pop %d: expected ID=%s, got ID=%s", i, expectedID, task.ID)
				}
			}

			if heap.Len() != 0 {
				t.Errorf("heap should be empty after popping all, got len=%d", heap.Len())
			}
		})
	}
}

func Test_scheduledTaskHeap_Peek(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	t.Run("peek empty heap returns nil", func(t *testing.T) {
		t.Parallel()

		heap := newScheduledTaskHeap()
		task := heap.Peek()

		if task != nil {
			t.Errorf("expected nil, got task with ID=%s", task.ID)
		}
	})

	t.Run("peek returns earliest without removing", func(t *testing.T) {
		t.Parallel()

		heap := newScheduledTaskHeap()
		heap.Push(&Task{ID: "task-2", ScheduledExecuteAt: baseTime.Add(1 * time.Hour)})
		heap.Push(&Task{ID: "task-1", ScheduledExecuteAt: baseTime})

		task := heap.Peek()
		if task == nil || task.ID != "task-1" {
			t.Errorf("expected task-1, got %v", task)
		}

		task = heap.Peek()
		if task == nil || task.ID != "task-1" {
			t.Errorf("second peek: expected task-1, got %v", task)
		}

		if heap.Len() != 2 {
			t.Errorf("expected len=2, got len=%d", heap.Len())
		}
	})
}

func Test_scheduledTaskHeap_PopEmpty(t *testing.T) {
	t.Parallel()

	heap := newScheduledTaskHeap()
	task := heap.Pop()

	if task != nil {
		t.Errorf("expected nil from empty heap, got task with ID=%s", task.ID)
	}
}

func Test_scheduledTaskHeap_Len(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	heap := newScheduledTaskHeap()

	if heap.Len() != 0 {
		t.Errorf("empty heap: expected len=0, got len=%d", heap.Len())
	}

	heap.Push(&Task{ID: "task-1", ScheduledExecuteAt: baseTime})
	if heap.Len() != 1 {
		t.Errorf("after 1 push: expected len=1, got len=%d", heap.Len())
	}

	heap.Push(&Task{ID: "task-2", ScheduledExecuteAt: baseTime.Add(time.Hour)})
	heap.Push(&Task{ID: "task-3", ScheduledExecuteAt: baseTime.Add(2 * time.Hour)})
	if heap.Len() != 3 {
		t.Errorf("after 3 pushes: expected len=3, got len=%d", heap.Len())
	}

	heap.Pop()
	if heap.Len() != 2 {
		t.Errorf("after 1 pop: expected len=2, got len=%d", heap.Len())
	}
}

func Test_scheduledTaskHeap_Less(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	heap := &scheduledTaskHeap{
		{ID: "task-1", ScheduledExecuteAt: baseTime},
		{ID: "task-2", ScheduledExecuteAt: baseTime.Add(time.Hour)},
	}

	if !heap.Less(0, 1) {
		t.Error("expected task at index 0 to be less than task at index 1")
	}

	if heap.Less(1, 0) {
		t.Error("expected task at index 1 to NOT be less than task at index 0")
	}
}

func Test_scheduledTaskHeap_Swap(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	heap := &scheduledTaskHeap{
		{ID: "task-1", ScheduledExecuteAt: baseTime},
		{ID: "task-2", ScheduledExecuteAt: baseTime.Add(time.Hour)},
	}

	heap.Swap(0, 1)

	if (*heap)[0].ID != "task-2" {
		t.Errorf("after swap: expected index 0 to be task-2, got %s", (*heap)[0].ID)
	}
	if (*heap)[1].ID != "task-1" {
		t.Errorf("after swap: expected index 1 to be task-1, got %s", (*heap)[1].ID)
	}
}

func Test_scheduledTaskHeap_ManyTasks(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	heap := newScheduledTaskHeap()

	for i := 99; i >= 0; i-- {
		heap.Push(&Task{
			ID:                 "",
			ScheduledExecuteAt: baseTime.Add(time.Duration(i) * time.Minute),
		})
	}

	if heap.Len() != 100 {
		t.Fatalf("expected len=100, got len=%d", heap.Len())
	}

	var lastTime time.Time
	for i := range 100 {
		task := heap.Pop()
		if task == nil {
			t.Fatalf("pop %d: unexpected nil", i)
		}
		if !lastTime.IsZero() && task.ScheduledExecuteAt.Before(lastTime) {
			t.Errorf("pop %d: task time %v is before previous time %v",
				i, task.ScheduledExecuteAt, lastTime)
		}
		lastTime = task.ScheduledExecuteAt
	}
}

func Test_scheduledTaskHeap_InterleavedPushPop(t *testing.T) {
	t.Parallel()

	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	heap := newScheduledTaskHeap()

	heap.Push(&Task{ID: "task-3", ScheduledExecuteAt: baseTime.Add(2 * time.Hour)})
	heap.Push(&Task{ID: "task-1", ScheduledExecuteAt: baseTime})

	task := heap.Pop()
	if task == nil {
		t.Fatal("first pop: expected task-1, got nil")
	}
	if task.ID != "task-1" {
		t.Errorf("first pop: expected task-1, got %s", task.ID)
	}

	heap.Push(&Task{ID: "task-2", ScheduledExecuteAt: baseTime.Add(1 * time.Hour)})
	heap.Push(&Task{ID: "task-0", ScheduledExecuteAt: baseTime.Add(-1 * time.Hour)})

	task = heap.Pop()
	if task == nil {
		t.Fatal("second pop: expected task-0, got nil")
	}
	if task.ID != "task-0" {
		t.Errorf("second pop: expected task-0, got %s", task.ID)
	}

	task = heap.Pop()
	if task == nil {
		t.Fatal("third pop: expected task-2, got nil")
	}
	if task.ID != "task-2" {
		t.Errorf("third pop: expected task-2, got %s", task.ID)
	}

	task = heap.Pop()
	if task == nil {
		t.Fatal("fourth pop: expected task-3, got nil")
	}
	if task.ID != "task-3" {
		t.Errorf("fourth pop: expected task-3, got %s", task.ID)
	}

	if heap.Len() != 0 {
		t.Errorf("expected empty heap, got len=%d", heap.Len())
	}
}
