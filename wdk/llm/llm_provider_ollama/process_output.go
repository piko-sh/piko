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

package llm_provider_ollama

import (
	"bytes"
	"context"
	"sync"
	"unicode/utf8"

	"piko.sh/piko/wdk/logger"
)

const (
	// maxOutputLineBytes caps one relayed line of managed Ollama output; longer lines are
	// cut at this length and the remainder is counted and discarded.
	maxOutputLineBytes = 64 << 10
)

// outputRelay is an io.Writer that relays the managed Ollama server's output to the
// logger one line at a time.
//
// It never blocks or fails a write, so the subprocess can never stall on a full pipe, and
// it bounds the memory held for a partial line.
type outputRelay struct {
	// l receives one trace entry per relayed line.
	l logger.Logger

	// pending holds the current partial line.
	pending []byte

	// droppedBytes counts the bytes of the current line cut off by maxOutputLineBytes.
	droppedBytes int

	// mu guards pending and droppedBytes.
	mu sync.Mutex
}

// Write relays every complete line in data and buffers any trailing partial line.
//
// Takes data ([]byte) which is the next chunk of subprocess output.
//
// Returns int which is always len(data).
// Returns error which is always nil, so output is never left unread.
//
// Safe for concurrent use.
func (r *outputRelay) Write(data []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	written := len(data)
	for len(data) > 0 {
		line, rest, found := bytes.Cut(data, []byte{'\n'})
		r.appendPending(line)
		if !found {
			break
		}
		r.emit()
		data = rest
	}
	return written, nil
}

// Flush relays any buffered partial line.
//
// Safe for concurrent use.
func (r *outputRelay) Flush() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.pending) > 0 || r.droppedBytes > 0 {
		r.emit()
	}
}

// appendPending adds chunk to the partial line, cutting it at a rune boundary once the
// line reaches maxOutputLineBytes.
//
// Takes chunk ([]byte) which is the next part of the current line.
func (r *outputRelay) appendPending(chunk []byte) {
	room := maxOutputLineBytes - len(r.pending)
	if len(chunk) <= room {
		r.pending = append(r.pending, chunk...)
		return
	}

	cut := max(room, 0)
	for cut > 0 && !utf8.RuneStart(chunk[cut]) {
		cut--
	}
	r.pending = append(r.pending, chunk[:cut]...)
	r.droppedBytes += len(chunk) - cut
}

// emit logs the buffered line and resets the buffer.
func (r *outputRelay) emit() {
	line := string(bytes.TrimSuffix(r.pending, []byte{'\r'}))
	if r.droppedBytes > 0 {
		r.l.Trace("Ollama output",
			logger.String("output", line),
			logger.Int("truncated_bytes", r.droppedBytes),
		)
	} else {
		r.l.Trace("Ollama output", logger.String("output", line))
	}
	r.pending = r.pending[:0]
	r.droppedBytes = 0
}

// newOutputRelay creates an output relay that logs through the context logger.
//
// Returns *outputRelay which is ready to be used as a subprocess output writer.
func newOutputRelay(ctx context.Context) *outputRelay {
	_, l := logger.From(ctx, log)
	return &outputRelay{
		l:            l,
		pending:      nil,
		droppedBytes: 0,
		mu:           sync.Mutex{},
	}
}
