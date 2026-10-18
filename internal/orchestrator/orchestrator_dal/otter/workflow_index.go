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

// workflowIndex maps workflows to task IDs, storing single-task workflows inline.
//
// The DAL lock protects it; writes require the write lock and reads accept either lock.
type workflowIndex struct {
	// byWorkflow holds each workflow's tasks.
	byWorkflow map[string]workflowEntry

	// byTask maps each indexed task to its workflow.
	byTask map[string]string
}

// workflowEntry is one workflow's tasks.
type workflowEntry struct {
	// rest holds the tasks after the first, or is nil.
	rest map[string]struct{}

	// first is one of the workflow's tasks.
	first string
}

// newWorkflowIndex returns an empty index.
//
// Returns *workflowIndex which is ready for use.
func newWorkflowIndex() *workflowIndex {
	return &workflowIndex{
		byWorkflow: make(map[string]workflowEntry),
		byTask:     make(map[string]string),
	}
}

// AddSingle indexes a task under a workflow, moving it from any other workflow.
//
// Takes workflowID (string) which identifies the workflow.
// Takes taskID (string) which identifies the task.
func (x *workflowIndex) AddSingle(workflowID, taskID string) {
	if current, ok := x.byTask[taskID]; ok {
		if current == workflowID {
			return
		}
		x.RemoveSingle(current, taskID)
	}
	x.byTask[taskID] = workflowID
	entry, ok := x.byWorkflow[workflowID]
	if !ok {
		x.byWorkflow[workflowID] = workflowEntry{rest: nil, first: taskID}
		return
	}
	if entry.rest == nil {
		entry.rest = make(map[string]struct{}, 1)
	}
	entry.rest[taskID] = struct{}{}
	x.byWorkflow[workflowID] = entry
}

// RemoveSingle removes a task from a workflow.
//
// Takes workflowID (string) which identifies the workflow.
// Takes taskID (string) which identifies the task.
func (x *workflowIndex) RemoveSingle(workflowID, taskID string) {
	entry, ok := x.byWorkflow[workflowID]
	if !ok {
		return
	}
	if current, indexed := x.byTask[taskID]; indexed && current == workflowID {
		delete(x.byTask, taskID)
	}
	switch {
	case entry.first == taskID && len(entry.rest) == 0:
		delete(x.byWorkflow, workflowID)
		return
	case entry.first == taskID:
		for next := range entry.rest {
			entry.first = next
			delete(entry.rest, next)
			break
		}
	default:
		delete(entry.rest, taskID)
	}
	if len(entry.rest) == 0 {
		entry.rest = nil
	}
	x.byWorkflow[workflowID] = entry
}

// RemoveKey removes a task from whichever workflow holds it.
//
// Takes taskID (string) which identifies the task.
func (x *workflowIndex) RemoveKey(taskID string) {
	if workflowID, ok := x.byTask[taskID]; ok {
		x.RemoveSingle(workflowID, taskID)
	}
}

// Get returns a workflow's tasks.
//
// Takes workflowID (string) which identifies the workflow.
//
// Returns map[string]struct{} which is a set of the task IDs the caller owns, or nil when
// the workflow has no tasks.
func (x *workflowIndex) Get(workflowID string) map[string]struct{} {
	entry, ok := x.byWorkflow[workflowID]
	if !ok {
		return nil
	}
	tasks := make(map[string]struct{}, 1+len(entry.rest))
	tasks[entry.first] = struct{}{}
	for taskID := range entry.rest {
		tasks[taskID] = struct{}{}
	}
	return tasks
}
