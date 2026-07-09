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

package log_collector_grpcfb

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

const (
	// breadcrumbRingCap is the number of recent forwarded log lines retained for the
	// breadcrumb trail attached to errors.
	breadcrumbRingCap = 256

	// maxBreadcrumbs is how many trail entries are attached to a single error.
	maxBreadcrumbs = 20

	// maxBreadcrumbMsg bounds a retained breadcrumb message (UTF-8 safe) so the ring's
	// footprint stays bounded.
	maxBreadcrumbMsg = 256
)

// breadcrumb is one retained log line in the trail leading up to an error.
type breadcrumb struct {
	// level is the record's log level (e.g. INFO), surfaced as the crumb's level.
	level string

	// logger is the originating logger/component, surfaced as the crumb's category.
	logger string

	// message is the length-bounded log message retained for the trail.
	message string

	// traceID is the trace the line belonged to, used to scope a trail to one request.
	traceID string

	// tsMs is the line's timestamp in Unix milliseconds.
	tsMs int64
}

// breadcrumbRing is a fixed-size, mutex-guarded ring of the most recent forwarded log
// lines.
type breadcrumbRing struct {
	// buffer is the fixed-capacity backing store; len(buffer) is the ring capacity.
	buffer []breadcrumb

	// mu serialises concurrent add and recent calls.
	mu sync.Mutex

	// position is the index the next add writes to (the oldest entry once the ring is full).
	position int

	// size is the number of populated entries, saturating at len(buffer).
	size int
}

// newBreadcrumbRing builds an empty ring with the given fixed capacity.
//
// Takes capacity (int) which is the number of entries the ring retains.
//
// Returns *breadcrumbRing which is the initialised, empty ring.
func newBreadcrumbRing(capacity int) *breadcrumbRing {
	return &breadcrumbRing{buffer: make([]breadcrumb, capacity)}
}

// add records a line as the newest entry, evicting the oldest when full.
//
// Takes b (breadcrumb) which is the line to retain; its message is bounded and copied.
//
// Concurrency: safe for concurrent callers; serialised by r.mu.
func (r *breadcrumbRing) add(b breadcrumb) {
	if r == nil {
		return
	}
	msg, _ := telemetry_grpcfb.TruncateUTF8(b.message, maxBreadcrumbMsg)
	b.message = strings.Clone(msg)
	r.mu.Lock()
	r.buffer[r.position] = b
	r.position = (r.position + 1) % len(r.buffer)
	if r.size < len(r.buffer) {
		r.size++
	}
	r.mu.Unlock()
}

// recent returns up to limit trail entries oldest-first.
//
// Takes limit (int) which bounds how many of the newest matching entries are returned.
// Takes traceID (string) which, when non-empty, restricts the trail to that trace.
//
// Returns []breadcrumb which is the matching entries oldest-first (nil for a nil ring).
//
// Concurrency: safe for concurrent callers; serialised by r.mu.
func (r *breadcrumbRing) recent(limit int, traceID string) []breadcrumb {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]breadcrumb, 0, r.size)
	start := (r.position - r.size + len(r.buffer)) % len(r.buffer)
	for i := 0; i < r.size; i++ {
		b := r.buffer[(start+i)%len(r.buffer)]
		if traceID != "" && b.traceID != traceID {
			continue
		}
		out = append(out, b)
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// breadcrumbsJSON serialises a trail as the JSON array the issue detail reads
// ([{ts,category,message,level}]); "" when empty.
//
// Takes crumbs ([]breadcrumb) which is the trail to serialise, oldest-first.
//
// Returns string which is the JSON array, or "" when empty or marshalling fails.
func breadcrumbsJSON(crumbs []breadcrumb) string {
	if len(crumbs) == 0 {
		return ""
	}
	type bc struct {
		TS       string `json:"ts"`
		Category string `json:"category"`
		Message  string `json:"message"`
		Level    string `json:"level"`
	}
	out := make([]bc, 0, len(crumbs))
	for _, c := range crumbs {
		out = append(out, bc{
			TS:       time.UnixMilli(c.tsMs).Format("15:04:05"),
			Category: c.logger,
			Message:  c.message,
			Level:    c.level,
		})
	}
	b, err := json.Marshal(out)
	if err != nil {
		return ""
	}
	return string(b)
}
