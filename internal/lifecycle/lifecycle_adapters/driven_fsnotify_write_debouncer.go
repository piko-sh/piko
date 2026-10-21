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
	"cmp"
	"slices"
	"time"

	"github.com/fsnotify/fsnotify"
)

// pendingWrite is a write or chmod event held back until its path goes quiet.
type pendingWrite struct {
	// firstSeen is when the first held-back event for the path arrived; the wait is capped
	// at maxWait from it.
	firstSeen time.Time

	// lastSeen is when the latest event for the path arrived; the quiet period runs from it.
	lastSeen time.Time

	// op accumulates the operations of every event folded in, so a write followed by a chmod
	// is still reported as a write.
	op fsnotify.Op
}

// writeDebouncer coalesces write and chmod events until a path is quiet or the maximum
// wait expires.
//
// The watcher goroutine owns it. It is not safe for concurrent use.
type writeDebouncer struct {
	// pending maps a path to its held-back event.
	pending map[string]*pendingWrite

	// quietPeriod is how long a path must go without events before its event is released.
	quietPeriod time.Duration

	// maxWait caps how long a continuous burst can hold back its event.
	maxWait time.Duration
}

// newWriteDebouncer creates an empty writeDebouncer.
//
// Takes quietPeriod (time.Duration) which is how long a path must be quiet before its
// event is released.
// Takes maxWait (time.Duration) which caps how long a burst can hold back its event.
//
// Returns *writeDebouncer which holds no events.
func newWriteDebouncer(quietPeriod, maxWait time.Duration) *writeDebouncer {
	return &writeDebouncer{
		pending:     make(map[string]*pendingWrite),
		quietPeriod: quietPeriod,
		maxWait:     maxWait,
	}
}

// isCoalescedEvent reports whether an event only changes a file's content or mode, and so
// is held back by the writeDebouncer. Create, remove, and rename events change which
// files exist and are forwarded straight away.
//
// Takes ev (fsnotify.Event) which is the raw event to classify.
//
// Returns bool which is true for write and chmod events.
func isCoalescedEvent(ev fsnotify.Event) bool {
	return !ev.Has(fsnotify.Create) && !ev.Has(fsnotify.Remove) && !ev.Has(fsnotify.Rename)
}

// record holds back a write or chmod event, folding it into any event already pending for
// the same path and restarting that path's quiet period.
//
// Takes ev (fsnotify.Event) which is the event to hold back.
// Takes now (time.Time) which is when the event arrived.
func (d *writeDebouncer) record(ev fsnotify.Event, now time.Time) {
	if existing, ok := d.pending[ev.Name]; ok {
		existing.lastSeen = now
		existing.op |= ev.Op
		return
	}
	d.pending[ev.Name] = &pendingWrite{firstSeen: now, lastSeen: now, op: ev.Op}
}

// discard drops any event held back for a path. It is called when a create, remove, or
// rename for the path is forwarded, which supersedes the held-back write.
//
// Takes path (string) which is the path whose pending event is dropped.
func (d *writeDebouncer) discard(path string) {
	delete(d.pending, path)
}

// takeDue removes and returns the held-back events that are due by now, sorted by path so
// the order is deterministic.
//
// Takes now (time.Time) which is the current time.
//
// Returns []fsnotify.Event which holds one event per due path.
func (d *writeDebouncer) takeDue(now time.Time) []fsnotify.Event {
	var due []fsnotify.Event
	for path, write := range d.pending {
		if now.Before(d.dueAt(write)) {
			continue
		}
		due = append(due, fsnotify.Event{Name: path, Op: write.op})
		delete(d.pending, path)
	}
	slices.SortFunc(due, func(a, b fsnotify.Event) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return due
}

// nextDeadline reports when the earliest held-back event becomes due.
//
// Returns time.Time which is the earliest due time.
// Returns bool which is false when no event is held back.
func (d *writeDebouncer) nextDeadline() (time.Time, bool) {
	var earliest time.Time
	found := false
	for _, write := range d.pending {
		due := d.dueAt(write)
		if !found || due.Before(earliest) {
			earliest = due
			found = true
		}
	}
	return earliest, found
}

// dueAt returns the earlier of the quiet-period deadline and the maximum-wait deadline.
//
// Takes write (*pendingWrite) which is the held-back event.
//
// Returns time.Time which is the release time.
func (d *writeDebouncer) dueAt(write *pendingWrite) time.Time {
	quietAt := write.lastSeen.Add(d.quietPeriod)
	capAt := write.firstSeen.Add(d.maxWait)
	if capAt.Before(quietAt) {
		return capAt
	}
	return quietAt
}
