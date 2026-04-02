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

//go:build bench

package inspector_test_bench_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func getSourceContentsForBenchmark(b *testing.B, root string) map[string][]byte {
	b.Helper()
	sourceContents := make(map[string][]byte)
	absRoot, err := filepath.Abs(root)
	require.NoError(b, err)

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		require.NoError(b, err)
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			content, readErr := os.ReadFile(path)
			require.NoError(b, readErr)
			sourceContents[path] = content
		}
		return nil
	})
	require.NoError(b, err)
	return sourceContents
}

func cloneSourceContents(sources map[string][]byte) map[string][]byte {
	clone := make(map[string][]byte, len(sources))
	for path, content := range sources {

		contentCopy := make([]byte, len(content))
		copy(contentCopy, content)
		clone[path] = contentCopy
	}
	return clone
}

func modifyFileContent(sources map[string][]byte, relPath string, baseDir string) {
	fullPath := filepath.Join(baseDir, relPath)
	if content, exists := sources[fullPath]; exists {

		sources[fullPath] = append(content, []byte("\n// Modified for benchmark\n")...)
	}
}

func modifyMultipleFiles(sources map[string][]byte, relPaths []string, baseDir string) {
	for _, relPath := range relPaths {
		modifyFileContent(sources, relPath, baseDir)
	}
}
