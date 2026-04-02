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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func TestParseIgnoreFile(t *testing.T) {
	t.Parallel()

	t.Run("parses valid .pikoignore file with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		ignoreContent := "node_modules\n# comment line\n*.log\n\ndist\n"
		require.NoError(t, sandbox.WriteFile(".pikoignore", []byte(ignoreContent), 0600))

		patterns, err := parseIgnoreFile("/project/.pikoignore", sandbox, nil)

		require.NoError(t, err)
		assert.Equal(t, []string{"node_modules", "*.log", "dist"}, patterns)
	})

	t.Run("returns nil when file does not exist", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		patterns, err := parseIgnoreFile("/project/.pikoignore", sandbox, nil)

		require.NoError(t, err)
		assert.Nil(t, patterns)
	})

	t.Run("returns error when Open fails with non-NotExist error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile(".pikoignore", []byte("test"), 0600))
		sandbox.OpenErr = errors.New("permission denied")

		patterns, err := parseIgnoreFile("/project/.pikoignore", sandbox, nil)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
		assert.Nil(t, patterns)
	})

	t.Run("skips empty lines and comments", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		ignoreContent := "\n\n# This is a comment\n  # indented comment\npattern1\n   \npattern2\n"
		require.NoError(t, sandbox.WriteFile(".pikoignore", []byte(ignoreContent), 0600))

		patterns, err := parseIgnoreFile("/project/.pikoignore", sandbox, nil)

		require.NoError(t, err)
		assert.Equal(t, []string{"pattern1", "pattern2"}, patterns)
	})

	t.Run("handles empty file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile(".pikoignore", []byte(""), 0600))

		patterns, err := parseIgnoreFile("/project/.pikoignore", sandbox, nil)

		require.NoError(t, err)
		assert.Nil(t, patterns)
	})
}

func TestNewIgnoreMatcher(t *testing.T) {
	t.Parallel()

	t.Run("combines default patterns with .pikoignore and config patterns", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile(".pikoignore", []byte("custom_ignore\n"), 0600))

		matcher := newIgnoreMatcher("/project", []string{"config_pattern"}, withIgnoreMatcherSandbox(sandbox))

		assert.True(t, matcher.Matches(".git"))
		assert.True(t, matcher.Matches("node_modules"))

		assert.True(t, matcher.Matches("custom_ignore"))

		assert.True(t, matcher.Matches("config_pattern"))

		assert.False(t, matcher.Matches("random_file"))
	})

	t.Run("works without .pikoignore file", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		matcher := newIgnoreMatcher("/project", []string{"my_pattern"}, withIgnoreMatcherSandbox(sandbox))

		assert.True(t, matcher.Matches(".git"))

		assert.True(t, matcher.Matches("my_pattern"))
	})

	t.Run("handles sandbox errors gracefully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.OpenErr = errors.New("disk error")

		matcher := newIgnoreMatcher("/project", []string{"fallback"}, withIgnoreMatcherSandbox(sandbox))

		assert.True(t, matcher.Matches(".git"))
		assert.True(t, matcher.Matches("fallback"))
	})
}

func TestIgnoreMatcher_Matches(t *testing.T) {
	t.Parallel()

	t.Run("matches glob patterns", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile(".pikoignore", []byte("*.tmp\ntest_*\n"), 0600))

		matcher := newIgnoreMatcher("/project", nil, withIgnoreMatcherSandbox(sandbox))

		assert.True(t, matcher.Matches("file.tmp"))
		assert.True(t, matcher.Matches("test_file.go"))
		assert.False(t, matcher.Matches("file.txt"))
	})
}

func TestParseIgnoreFile_WithNilSandbox(t *testing.T) {
	t.Parallel()

	t.Run("should create its own sandbox when no injected sandbox is provided", func(t *testing.T) {
		t.Parallel()

		patterns, err := parseIgnoreFile("/tmp/nonexistent_dir_test/.pikoignore", nil, nil)

		assert.NoError(t, err)
		assert.Nil(t, patterns)
	})
}

func TestParseIgnoreFile_PatternsWithWhitespace(t *testing.T) {
	t.Parallel()

	t.Run("should trim leading and trailing whitespace from patterns", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		content := "  node_modules  \n  dist  \n# comment\n  *.log  \n"
		require.NoError(t, sandbox.WriteFile(".pikoignore", []byte(content), 0600))

		patterns, err := parseIgnoreFile("/project/.pikoignore", sandbox, nil)

		require.NoError(t, err)
		assert.Equal(t, []string{"node_modules", "dist", "*.log"}, patterns)
	})
}

func TestParseIgnoreFile_OnlyComments(t *testing.T) {
	t.Parallel()

	t.Run("should return nil for a file containing only comments", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		content := "# comment 1\n# comment 2\n# comment 3\n"
		require.NoError(t, sandbox.WriteFile(".pikoignore", []byte(content), 0600))

		patterns, err := parseIgnoreFile("/project/.pikoignore", sandbox, nil)

		require.NoError(t, err)
		assert.Nil(t, patterns)
	})
}

func TestIgnoreMatcher_DefaultPatterns(t *testing.T) {
	t.Parallel()

	t.Run("should match all default ignore patterns", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		matcher := newIgnoreMatcher("/project", nil, withIgnoreMatcherSandbox(sandbox))

		defaultPatternExamples := []string{
			".git", ".svn", "node_modules", "vendor",
			"__pycache__", ".venv", "dist", "build",
			".idea", ".vscode", ".DS_Store",
		}

		for _, pattern := range defaultPatternExamples {
			assert.True(t, matcher.Matches(pattern), "should match default pattern: %s", pattern)
		}
	})
}
