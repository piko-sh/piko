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

package annotator_domain

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
)

// recordingFSReader wraps an FSReaderPort and records the path of every CSS file it
// reads.
//
// The partial expander reads external stylesheets only when resolving CSS @import
// statements, so the recorded set is the external stylesheet dependencies of a
// component's <style> blocks. The build watches those files and folds their contents into
// the input hash, so editing an imported stylesheet invalidates the cache and rebuilds in
// dev-i mode. Safe for concurrent use; sibling components may be processed in parallel.
type recordingFSReader struct {
	// inner is the underlying reader that performs the actual I/O.
	inner FSReaderPort

	// paths is the set of recorded .css file paths.
	paths map[string]struct{}

	// mu guards paths.
	mu sync.Mutex
}

// newRecordingFSReader wraps inner so that every .css file it reads is recorded.
//
// Takes inner (FSReaderPort) which performs the underlying file reads.
//
// Returns *recordingFSReader which is ready to use as an FSReaderPort.
func newRecordingFSReader(inner FSReaderPort) *recordingFSReader {
	return &recordingFSReader{
		inner: inner,
		paths: make(map[string]struct{}),
	}
}

// ReadFile records the path when it is a .css file, then delegates to the inner reader.
//
// Takes ctx (context.Context) which controls cancellation.
// Takes filePath (string) which is the path to read.
//
// Returns the file contents and any read error from the inner reader.
//
// Concurrency: safe for concurrent use; guarded by mu.
func (r *recordingFSReader) ReadFile(ctx context.Context, filePath string) ([]byte, error) {
	if strings.HasSuffix(strings.ToLower(filePath), ".css") {
		r.mu.Lock()
		r.paths[filePath] = struct{}{}
		r.mu.Unlock()
	}
	return r.inner.ReadFile(ctx, filePath)
}

// recordedPaths returns the sorted, de-duplicated set of recorded .css file paths.
//
// Returns []string which contains the recorded stylesheet paths (nil if none were read).
//
// Concurrency: safe for concurrent use; guarded by mu.
func (r *recordingFSReader) recordedPaths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.paths) == 0 {
		return nil
	}
	return slices.Sorted(maps.Keys(r.paths))
}
