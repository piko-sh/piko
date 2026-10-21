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
	"errors"
	"time"
)

var (
	// ErrLeaseLost reports a node lease that no longer exists or has expired. The node's
	// claims may already have been recovered by another node, so it must stop executing and
	// register again.
	ErrLeaseLost = errors.New("orchestrator: node lease lost")

	// ErrStoreUnsupported is returned by a store that does not implement an operation.
	ErrStoreUnsupported = errors.New("orchestrator: operation not supported by this task store")

	// ErrStoreDegraded reports a store whose state differs from its durable log and refuses
	// further writes until restart.
	ErrStoreDegraded = errors.New("orchestrator: task store degraded; restart to reload it from its log")
)

// ClaimRequest describes a claim on a delivered task attempt.
type ClaimRequest struct {
	// Now is the claim time, stored as the task's update time.
	Now time.Time

	// Into optionally receives the claimed task or premature-delivery state. Stores may
	// return a separately allocated task instead.
	Into *Task

	// ID identifies the task.
	ID string

	// Owner identifies the node instance that will execute the task.
	Owner string

	// Attempt is the attempt the delivery carries. The claim fails unless it is still the
	// stored attempt.
	Attempt int

	// DueTolerance is how long before its due time a retrying or scheduled task may still be
	// claimed, absorbing clock skew between nodes and second-precision stored times.
	DueTolerance time.Duration
}

// Settlement is the state a claimed task settles into.
type Settlement struct {
	// Task holds the COMPLETE, FAILED, or RETRYING state for the claimed attempt.
	Task *Task

	// Successor is a recurring task's next run, created with the settle, or nil. A successor
	// whose deduplication key is already held is skipped; it never fails the settle.
	Successor *Task
}

// SettleOutcome reports what a settle created besides the settled state.
type SettleOutcome struct {
	// Successor is the successor that was created, or nil when there was none or its key was
	// already held.
	Successor *Task

	// Rerun is the pending task created from a rerun requested while the settled task was
	// active, or nil. It holds the settled task's deduplication key and must be published.
	Rerun *Task
}

// TransitionOutcome reports what a transition created besides the moved task.
type TransitionOutcome struct {
	// Rerun is the pending task created from a rerun requested while the moved task was
	// active, when the transition took it to a terminal status; nil otherwise. It holds the
	// moved task's deduplication key and must be published.
	Rerun *Task
}

// Transition changes a task's status outside a claim, fenced by its stored attempt and
// status.
type Transition struct {
	// Task holds the new state. Its Attempt must equal the stored attempt.
	Task *Task

	// From lists the statuses the stored task may be in.
	From []TaskStatus
}

// RerunRequest asks for an active task's work to run again once it settles, because its
// inputs changed while it ran. Only the latest request per active task is kept.
type RerunRequest struct {
	// Payload is the rerun's payload.
	Payload map[string]any

	// ID identifies the task the rerun becomes.
	ID string

	// Config holds the rerun's execution settings.
	Config TaskConfig
}

// DispatchQuery selects a page of due tasks to publish.
type DispatchQuery struct {
	// Now is the time tasks must be due by.
	Now time.Time

	// AfterTime and AfterID continue after the previous page's last task; a zero AfterTime
	// starts at the beginning.
	AfterTime time.Time

	// PublishedBefore limits tasks by their last publication time; zero includes every due
	// task.
	PublishedBefore time.Time

	// AfterID identifies the previous page's last task.
	AfterID string

	// Limit caps the page size.
	Limit int
}

// OrphanQuery selects processing tasks whose executor is gone.
type OrphanQuery struct {
	// Now is the current time, against which node leases are checked.
	Now time.Time

	// StaleBefore restricts the query to tasks last written before this time.
	StaleBefore time.Time

	// Limit caps the number of tasks returned.
	Limit int
}

// NodeLease records that a node instance is alive and executing tasks.
type NodeLease struct {
	// StartedAt is when the instance registered.
	StartedAt time.Time

	// RenewedAt is when the lease was last renewed.
	RenewedAt time.Time

	// ExpiresAt is when the lease lapses unless renewed.
	ExpiresAt time.Time

	// Instance uniquely identifies the running instance.
	Instance string

	// NodeID identifies the node the instance runs on.
	NodeID string
}

// TaskStoreV2 is the compare-and-set task store API. Each operation is a single fenced
// write, so the store alone decides which node executes a task, whatever the transport
// delivers.
type TaskStoreV2 interface {
	// ClaimTask claims a delivered attempt by incrementing its attempt and assigning this
	// node as owner.
	//
	// The stored task must match the delivered attempt and be pending, or scheduled or
	// retrying and due.
	//
	// Takes request (ClaimRequest) which describes the claim.
	//
	// Returns *Task which is a copy of the claimed task.
	// Returns error which is ErrObsoleteDelivery when the attempt is no longer claimable, or
	// ErrPrematureDelivery, with a copy of the stored task, when it is not yet due.
	ClaimTask(ctx context.Context, request ClaimRequest) (*Task, error)

	// SettleTask writes a claimed task's final or retry state, creates its successor and any
	// requested rerun, and frees its owner, all at once.
	//
	// Takes settlement (Settlement) which holds the new state.
	//
	// Returns SettleOutcome which reports the tasks created.
	// Returns error which is ErrObsoleteDelivery when the stored task is no longer
	// processing the claimed attempt.
	SettleTask(ctx context.Context, settlement Settlement) (SettleOutcome, error)

	// TransitionTask moves a task to a new state if it is still at the expected attempt and
	// in one of the expected statuses. A task moved to a terminal status with a rerun
	// pending gets its rerun created as a pending task in the same write.
	//
	// Takes transition (Transition) which describes the move.
	//
	// Returns TransitionOutcome which reports the rerun created, if any.
	// Returns error which is ErrObsoleteDelivery when the stored task has moved on, or
	// ErrTaskNotFound when it does not exist.
	TransitionTask(ctx context.Context, transition Transition) (TransitionOutcome, error)

	// RequestRerun records that the active task holding a deduplication key must run again
	// once it settles.
	//
	// Takes key (string) which is the deduplication key.
	// Takes rerun (RerunRequest) which describes the rerun.
	//
	// Returns bool which is true when the request was recorded, or false when no active task
	// holds the key and the caller should dispatch the work itself.
	// Returns error when the store cannot be written.
	RequestRerun(ctx context.Context, key string, rerun RerunRequest) (bool, error)

	// CountUnsettled returns how many foreground tasks tagged with buildTag have not
	// settled.
	//
	// Takes buildTag (string) which identifies the build.
	//
	// Returns int64 which counts unsettled foreground tasks for the build.
	// Returns error when the count cannot be read.
	CountUnsettled(ctx context.Context, buildTag string) (int64, error)

	// ListFailedForBuild returns the failed tasks tagged with buildTag.
	//
	// Takes buildTag (string) which identifies the build.
	//
	// Returns []*Task which contains the build's failed tasks.
	// Returns error when the tasks cannot be read.
	ListFailedForBuild(ctx context.Context, buildTag string) ([]*Task, error)

	// DispatchablePage reads a page of due pending and retrying tasks, ordered by execution
	// time and ID.
	//
	// Takes query (DispatchQuery) which selects due tasks and the page boundary.
	//
	// Returns []*Task which contains due tasks ordered by execution time and ID.
	// Returns error when the page cannot be read.
	DispatchablePage(ctx context.Context, query DispatchQuery) ([]*Task, error)

	// MarkPublished records when tasks were last published.
	//
	// Takes ids ([]string) which identifies the tasks.
	// Takes at (time.Time) which is the timestamp to record.
	//
	// Returns error when publication times cannot be recorded.
	MarkPublished(ctx context.Context, ids []string, at time.Time) error

	// ListOrphanedTasks returns processing tasks whose owner holds no live lease and that
	// were last written before the query's cutoff.
	//
	// Takes query (OrphanQuery) which selects orphaned tasks by age and result limit.
	//
	// Returns []*Task which contains processing tasks without a live owner lease.
	// Returns error when the tasks cannot be read.
	ListOrphanedTasks(ctx context.Context, query OrphanQuery) ([]*Task, error)

	// RegisterNode records a node instance's lease, replacing any earlier one.
	//
	// Takes lease (NodeLease) which describes the node instance and its expiry.
	//
	// Returns error when the lease cannot be recorded.
	RegisterNode(ctx context.Context, lease NodeLease) error

	// RenewNode extends a node instance's lease.
	//
	// Takes lease (NodeLease) which describes the node instance and its expiry.
	//
	// Returns error which is ErrLeaseLost when the lease no longer exists.
	RenewNode(ctx context.Context, lease NodeLease) error

	// DeregisterNode removes a node instance's lease.
	//
	// Takes instance (string) which identifies the node instance.
	//
	// Returns error when the lease cannot be removed.
	DeregisterNode(ctx context.Context, instance string) error

	// TouchOwned stamps every task an instance is processing as written now, so they are not
	// taken for orphans while the instance is alive.
	//
	// Takes instance (string) which identifies the node instance.
	// Takes at (time.Time) which is the timestamp to record.
	//
	// Returns int64 which is the number of tasks updated.
	// Returns error when the update fails.
	TouchOwned(ctx context.Context, instance string, at time.Time) (int64, error)

	// ListExpiredNodes returns instances whose leases lapsed before now.
	//
	// Takes now (time.Time) which is the time used to check lease expiry.
	// Takes limit (int) which caps the number of results or deletions.
	//
	// Returns []string which identifies the expired node instances.
	// Returns error when the leases cannot be read.
	ListExpiredNodes(ctx context.Context, now time.Time, limit int) ([]string, error)

	// PurgeTerminal deletes completed and failed tasks last written before a cutoff.
	//
	// Takes before (time.Time) which is the update-time cutoff for deletion.
	// Takes limit (int) which caps the number of results or deletions.
	//
	// Returns int which is the number of tasks deleted.
	// Returns error when deletion fails.
	PurgeTerminal(ctx context.Context, before time.Time, limit int) (int, error)
}

// UnimplementedTaskStoreV2 implements TaskStoreV2 with ErrStoreUnsupported, for test
// doubles that only exercise the original TaskStore methods.
type UnimplementedTaskStoreV2 struct{}

var (
	_ TaskStoreV2 = UnimplementedTaskStoreV2{}
)

// ClaimTask returns ErrStoreUnsupported.
//
// Takes arg2 (ClaimRequest) which is ignored.
//
// Returns *Task which is always nil.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) ClaimTask(context.Context, ClaimRequest) (*Task, error) {
	return nil, ErrStoreUnsupported
}

// SettleTask returns ErrStoreUnsupported.
//
// Takes arg2 (Settlement) which is ignored.
//
// Returns SettleOutcome which is always zero.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) SettleTask(context.Context, Settlement) (SettleOutcome, error) {
	return SettleOutcome{}, ErrStoreUnsupported
}

// TransitionTask returns ErrStoreUnsupported.
//
// Takes arg2 (Transition) which is ignored.
//
// Returns TransitionOutcome which is always zero.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) TransitionTask(context.Context, Transition) (TransitionOutcome, error) {
	return TransitionOutcome{}, ErrStoreUnsupported
}

// RequestRerun returns ErrStoreUnsupported.
//
// Takes arg2 (string) which is ignored.
// Takes arg3 (RerunRequest) which is ignored.
//
// Returns bool which is always false.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) RequestRerun(context.Context, string, RerunRequest) (bool, error) {
	return false, ErrStoreUnsupported
}

// CountUnsettled returns ErrStoreUnsupported.
//
// Takes arg2 (string) which is ignored.
//
// Returns int64 which is always zero.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) CountUnsettled(context.Context, string) (int64, error) {
	return 0, ErrStoreUnsupported
}

// ListFailedForBuild returns ErrStoreUnsupported.
//
// Takes arg2 (string) which is ignored.
//
// Returns []*Task which is always nil.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) ListFailedForBuild(context.Context, string) ([]*Task, error) {
	return nil, ErrStoreUnsupported
}

// DispatchablePage returns ErrStoreUnsupported.
//
// Takes arg2 (DispatchQuery) which is ignored.
//
// Returns []*Task which is always nil.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) DispatchablePage(context.Context, DispatchQuery) ([]*Task, error) {
	return nil, ErrStoreUnsupported
}

// MarkPublished returns ErrStoreUnsupported.
//
// Takes arg2 ([]string) which is ignored.
// Takes arg3 (time.Time) which is ignored.
//
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) MarkPublished(context.Context, []string, time.Time) error {
	return ErrStoreUnsupported
}

// ListOrphanedTasks returns ErrStoreUnsupported.
//
// Takes arg2 (OrphanQuery) which is ignored.
//
// Returns []*Task which is always nil.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) ListOrphanedTasks(context.Context, OrphanQuery) ([]*Task, error) {
	return nil, ErrStoreUnsupported
}

// RegisterNode returns ErrStoreUnsupported.
//
// Takes arg2 (NodeLease) which is ignored.
//
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) RegisterNode(context.Context, NodeLease) error {
	return ErrStoreUnsupported
}

// RenewNode returns ErrStoreUnsupported.
//
// Takes arg2 (NodeLease) which is ignored.
//
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) RenewNode(context.Context, NodeLease) error {
	return ErrStoreUnsupported
}

// DeregisterNode returns ErrStoreUnsupported.
//
// Takes arg2 (string) which is ignored.
//
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) DeregisterNode(context.Context, string) error {
	return ErrStoreUnsupported
}

// TouchOwned returns ErrStoreUnsupported.
//
// Takes arg2 (string) which is ignored.
// Takes arg3 (time.Time) which is ignored.
//
// Returns int64 which is always zero.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) TouchOwned(context.Context, string, time.Time) (int64, error) {
	return 0, ErrStoreUnsupported
}

// ListExpiredNodes returns ErrStoreUnsupported.
//
// Takes arg2 (time.Time) which is ignored.
// Takes arg3 (int) which is ignored.
//
// Returns []string which is always nil.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) ListExpiredNodes(context.Context, time.Time, int) ([]string, error) {
	return nil, ErrStoreUnsupported
}

// PurgeTerminal returns ErrStoreUnsupported.
//
// Takes arg2 (time.Time) which is ignored.
// Takes arg3 (int) which is ignored.
//
// Returns int which is always zero.
// Returns error which is always ErrStoreUnsupported.
func (UnimplementedTaskStoreV2) PurgeTerminal(context.Context, time.Time, int) (int, error) {
	return 0, ErrStoreUnsupported
}
