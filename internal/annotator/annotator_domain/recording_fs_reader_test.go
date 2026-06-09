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
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFSReader struct {
	mu        sync.Mutex
	content   []byte
	err       error
	readPaths []string
}

func (s *stubFSReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	s.mu.Lock()
	s.readPaths = append(s.readPaths, filePath)
	s.mu.Unlock()
	return s.content, s.err
}

func TestRecordingFSReaderRecordsCSSImports(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		reads    []string
		expected []string
	}{
		{name: "records a css read", reads: []string{"/a/style.css"}, expected: []string{"/a/style.css"}},
		{name: "ignores non-css reads", reads: []string{"/a/main.pk", "/a/app.js"}, expected: nil},
		{name: "matches uppercase extension", reads: []string{"/a/Theme.CSS"}, expected: []string{"/a/Theme.CSS"}},
		{name: "de-duplicates repeated paths", reads: []string{"/a/x.css", "/a/x.css"}, expected: []string{"/a/x.css"}},
		{name: "returns paths sorted", reads: []string{"/a/z.css", "/a/a.css"}, expected: []string{"/a/a.css", "/a/z.css"}},
		{name: "returns nil when nothing read", reads: nil, expected: nil},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recorder := newRecordingFSReader(&stubFSReader{content: []byte("body")})
			for _, path := range testCase.reads {
				_, err := recorder.ReadFile(t.Context(), path)
				require.NoError(t, err)
			}
			assert.Equal(t, testCase.expected, recorder.recordedPaths())
		})
	}
}

func TestRecordingFSReaderDelegatesToInner(t *testing.T) {
	t.Parallel()

	t.Run("propagates content and records the css path", func(t *testing.T) {
		t.Parallel()

		inner := &stubFSReader{content: []byte("body content")}
		recorder := newRecordingFSReader(inner)

		content, err := recorder.ReadFile(t.Context(), "/a/style.css")

		require.NoError(t, err)
		assert.Equal(t, []byte("body content"), content)
		assert.Equal(t, []string{"/a/style.css"}, inner.readPaths)
		assert.Equal(t, []string{"/a/style.css"}, recorder.recordedPaths())
	})

	t.Run("propagates the inner error while still recording", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("read failed")
		recorder := newRecordingFSReader(&stubFSReader{err: sentinel})

		_, err := recorder.ReadFile(t.Context(), "/a/style.css")

		require.ErrorIs(t, err, sentinel)
		assert.Equal(t, []string{"/a/style.css"}, recorder.recordedPaths())
	})
}

func TestRecordingFSReaderConcurrentReadsAreRaceFree(t *testing.T) {
	t.Parallel()

	recorder := newRecordingFSReader(&stubFSReader{content: []byte("body")})
	const readerCount = 32

	var wg sync.WaitGroup
	for i := range readerCount {
		wg.Go(func() {
			_, _ = recorder.ReadFile(t.Context(), fmt.Sprintf("/a/file%02d.css", i))
		})
	}
	wg.Wait()

	assert.Len(t, recorder.recordedPaths(), readerCount)
}
