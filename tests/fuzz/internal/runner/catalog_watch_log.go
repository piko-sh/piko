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

package runner

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	// catalogueLineLimit bounds one retained source-watch log line in bytes.
	catalogueLineLimit = 2048

	// catalogueEventLimit bounds the number of retained source-watch events.
	catalogueEventLimit = 32

	// truncationMarker marks a retained line that was cut at its byte limit.
	truncationMarker = " [truncated]"
)

// catalogueWatchLog retains source-watch events separately from general output.
type catalogueWatchLog struct {
	// line holds a partial source-watch log line.
	line []byte

	// events holds completed source-watch events.
	events []string

	// mu protects the mutable state.
	mu sync.Mutex

	// truncated records that the partial line exceeded its byte limit.
	truncated bool
}

// Write accepts worker output and records recognised source-watch events.
//
// Takes data ([]byte) which holds the process-output bytes.
//
// Returns int which is the number of accepted bytes.
// Returns error which is always nil.
//
// Concurrency: Safe for concurrent use. The log mutex serialises writes and snapshots.
func (l *catalogueWatchLog) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	total := len(data)
	for len(data) > 0 {
		fragment, rest, complete := bytes.Cut(data, []byte{'\n'})
		l.appendFragment(fragment)
		if !complete {
			break
		}
		line := string(l.line)
		if l.truncated {
			line += truncationMarker
		}
		l.line, l.truncated = l.line[:0], false
		data = rest
		if catalogueWatchEvent(line) {
			l.record(line)
		}
	}
	return total, nil
}

// edit records the source edit in the watch history.
//
// Takes step (string) which identifies the current history step.
// Takes source (string) which holds the saved source text.
//
// Concurrency: Safe for concurrent use. The log mutex serialises edit markers and worker
// events.
func (l *catalogueWatchLog) edit(step, source string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.record(fmt.Sprintf("%s wrote %s %s sha256=%x", time.Now().UTC().Format(time.RFC3339Nano), step, catalogueLeaf, sha256.Sum256([]byte(source))))
}

// record appends a bounded source-watch event.
//
// Takes event (string) which is the browser or source-watch event.
func (l *catalogueWatchLog) record(event string) {
	if len(l.events) < catalogueEventLimit {
		l.events = append(l.events, event)
	}
}

// appendFragment appends output to the partial line, cutting on a rune boundary at the
// byte limit. The caller must hold the log mutex.
//
// Takes fragment ([]byte) which holds the next output bytes for the line.
func (l *catalogueWatchLog) appendFragment(fragment []byte) {
	if l.truncated {
		return
	}
	room := catalogueLineLimit - len(l.line)
	if len(fragment) > room {
		l.truncated = true
		cut := room
		for cut > 0 && !utf8.RuneStart(fragment[cut]) {
			cut--
		}
		fragment = fragment[:cut]
	}
	l.line = append(l.line, fragment...)
}

// snapshot copies the completed watch history.
//
// Returns []string which is a copy of the recorded watcher events.
//
// Concurrency: Safe for concurrent use. The log mutex protects the copied history.
func (l *catalogueWatchLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.events)
}

// catalogueWatchEvent reports whether a worker log line is a recognised source-watch
// event.
//
// Takes line (string) which is one complete worker log line.
//
// Returns bool which reports a watcher start, dirty marking, or a rebuild event for the
// catalogue leaf.
func catalogueWatchEvent(line string) bool {
	if strings.Contains(line, `msg="File watcher started"`) || strings.Contains(line, "Targeted Dirty Marking Complete") {
		return true
	}
	if !strings.Contains(line, catalogueLeaf) {
		return false
	}
	for _, message := range []string{
		`msg="Received file event"`,
		`msg="Starting targeted rebuild"`,
		`msg="Targeted rebuild complete"`,
		`msg="Targeted rebuild failed,`,
		`msg="Targeted rebuild abandoned;`,
	} {
		if strings.Contains(line, message) {
			return true
		}
	}
	return false
}
