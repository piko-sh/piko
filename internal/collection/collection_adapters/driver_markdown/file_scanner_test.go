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

package driver_markdown

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewFileScanner(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadOnly)
	defer func() { _ = sandbox.Close() }()

	scanner := newFileScanner(sandbox)

	require.NotNil(t, scanner)
	assert.Equal(t, sandbox, scanner.sandbox)
}

func TestFileScanner_ScanDirectory(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for empty directory", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.MkdirAll(".", 0755))

		scanner := newFileScanner(sandbox)
		files, err := scanner.scanDirectory(context.Background(), ".")

		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("discovers markdown files", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.MkdirAll(".", 0755))

		require.NoError(t, sandbox.MkdirAll("blog", 0755))
		require.NoError(t, sandbox.WriteFile("README.md", []byte("# Readme"), 0644))
		require.NoError(t, sandbox.WriteFile("blog/post1.md", []byte("# Post 1"), 0644))
		require.NoError(t, sandbox.WriteFile("blog/post2.MD", []byte("# Post 2"), 0644))

		scanner := newFileScanner(sandbox)
		files, err := scanner.scanDirectory(context.Background(), ".")

		require.NoError(t, err)
		assert.Len(t, files, 3)

		paths := make(map[string]bool)
		for _, f := range files {
			paths[f.relativePath] = true
			assert.NotEmpty(t, f.absolutePath)
			assert.Greater(t, f.size, int64(0))
			assert.NotZero(t, f.modTime)
		}
		assert.True(t, paths["README.md"])
		assert.True(t, paths["blog/post1.md"])
		assert.True(t, paths["blog/post2.MD"])
	})

	t.Run("ignores non-markdown files", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.MkdirAll(".", 0755))
		require.NoError(t, sandbox.WriteFile("readme.md", []byte("# Readme"), 0644))
		require.NoError(t, sandbox.WriteFile("config.json", []byte("{}"), 0644))
		require.NoError(t, sandbox.WriteFile("script.js", []byte(""), 0644))
		require.NoError(t, sandbox.WriteFile("styles.css", []byte(""), 0644))

		scanner := newFileScanner(sandbox)
		files, err := scanner.scanDirectory(context.Background(), ".")

		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "readme.md", files[0].relativePath)
	})

	t.Run("scans subdirectory", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.MkdirAll(".", 0755))
		require.NoError(t, sandbox.MkdirAll("docs/api", 0755))
		require.NoError(t, sandbox.WriteFile("docs/api/reference.md", []byte("# API"), 0644))

		scanner := newFileScanner(sandbox)
		files, err := scanner.scanDirectory(context.Background(), "docs")

		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Equal(t, "api/reference.md", files[0].relativePath)
	})
}

func TestFileScanner_ScanDirectory_SkipDirectories(t *testing.T) {
	t.Parallel()

	skipDirs := []string{
		"node_modules",
		"vendor",
		"dist",
		"build",
		"out",
		".next",
		".cache",
		".git",
		".hidden",
		"coverage",
		"__pycache__",
	}

	for _, directory := range skipDirs {
		t.Run("skips "+directory, func(t *testing.T) {
			t.Parallel()

			sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadWrite)
			defer func() { _ = sandbox.Close() }()

			require.NoError(t, sandbox.MkdirAll(".", 0755))
			require.NoError(t, sandbox.MkdirAll(directory, 0755))
			require.NoError(t, sandbox.WriteFile(directory+"/should-ignore.md", []byte("# Ignore"), 0644))
			require.NoError(t, sandbox.WriteFile("keep.md", []byte("# Keep"), 0644))

			scanner := newFileScanner(sandbox)
			files, err := scanner.scanDirectory(context.Background(), ".")

			require.NoError(t, err)
			assert.Len(t, files, 1)
			assert.Equal(t, "keep.md", files[0].relativePath)
		})
	}
}

func TestFileScanner_ScanDirectory_Errors(t *testing.T) {
	t.Parallel()

	t.Run("returns error when Stat fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.StatErr = errors.New("permission denied")

		scanner := newFileScanner(sandbox)
		files, err := scanner.scanDirectory(context.Background(), ".")

		require.Error(t, err)
		assert.Nil(t, files)
		assert.Contains(t, err.Error(), "cannot access directory")
	})

	t.Run("returns error when path is not a directory", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.WriteFile("file.txt", []byte("content"), 0644))

		scanner := newFileScanner(sandbox)
		files, err := scanner.scanDirectory(context.Background(), "file.txt")

		require.Error(t, err)
		assert.Nil(t, files)
		assert.Contains(t, err.Error(), "is not a directory")
	})

	t.Run("returns error when WalkDir fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		require.NoError(t, sandbox.MkdirAll(".", 0755))
		sandbox.WalkDirErr = errors.New("walk error")

		scanner := newFileScanner(sandbox)
		files, err := scanner.scanDirectory(context.Background(), ".")

		require.Error(t, err)
		assert.Nil(t, files)
		assert.Contains(t, err.Error(), "error scanning directory")
	})
}

func TestIsMarkdownFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "lowercase .md", filename: "readme.md", want: true},
		{name: "uppercase .MD", filename: "README.MD", want: true},
		{name: "mixed case .Md", filename: "Document.Md", want: true},
		{name: "json file", filename: "config.json", want: false},
		{name: "txt file", filename: "notes.txt", want: false},
		{name: "no extension", filename: "README", want: false},
		{name: "markdown in middle", filename: "file.md.bak", want: false},
		{name: "empty string", filename: "", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isMarkdownFile(tc.filename))
		})
	}
}

func TestFileScanner_ShouldSkipDirectory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		directory string
		want      bool
	}{
		{name: "root dot", directory: ".", want: false},
		{name: "normal directory", directory: "src", want: false},
		{name: "docs directory", directory: "docs", want: false},
		{name: "hidden directory", directory: ".hidden", want: true},
		{name: "git directory", directory: ".git", want: true},
		{name: "node_modules", directory: "node_modules", want: true},
		{name: "vendor", directory: "vendor", want: true},
		{name: "dist", directory: "dist", want: true},
		{name: "build", directory: "build", want: true},
		{name: "out", directory: "out", want: true},
		{name: ".next", directory: ".next", want: true},
		{name: ".cache", directory: ".cache", want: true},
		{name: "coverage", directory: "coverage", want: true},
		{name: "__pycache__", directory: "__pycache__", want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, shouldSkipDirectory(tc.directory))
		})
	}
}

func TestProcessWalkEntry_ContextCancelled(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadOnly)
	defer func() { _ = sandbox.Close() }()
	scanner := newFileScanner(sandbox)

	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(fmt.Errorf("test: simulating cancelled context"))

	var files []*discoveredFile
	entry := &mockDirEntry{name: "post.md", isDir: false}
	err := scanner.processWalkEntry(ctx, ".", "post.md", entry, nil, &files)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestProcessWalkEntry_WalkError(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadOnly)
	defer func() { _ = sandbox.Close() }()
	scanner := newFileScanner(sandbox)

	var files []*discoveredFile
	entry := &mockDirEntry{name: "post.md", isDir: false}
	err := scanner.processWalkEntry(context.Background(), ".", "post.md", entry, errors.New("walk error"), &files)

	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestBuildDiscoveredFile_InfoError(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/content", safedisk.ModeReadOnly)
	defer func() { _ = sandbox.Close() }()
	scanner := newFileScanner(sandbox)

	entry := &mockDirEntry{
		name:  "post.md",
		isDir: false,
		infoFunc: func() (fs.FileInfo, error) {
			return nil, errors.New("info error")
		},
	}

	file, err := scanner.buildDiscoveredFile(context.Background(), ".", "post.md", entry)

	require.Error(t, err)
	assert.Nil(t, file)
	assert.Contains(t, err.Error(), "getting file info")
}
