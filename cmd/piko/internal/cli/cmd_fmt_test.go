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

package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/wdk/safedisk"
)

func newMockSandboxOpener(builder *safedisk.MockSandbox) sandboxOpener {
	return func(_ string, _ safedisk.Mode) (safedisk.Sandbox, error) {
		return builder, nil
	}
}

func newFailingSandboxOpener(err error) sandboxOpener {
	return func(_ string, _ safedisk.Mode) (safedisk.Sandbox, error) {
		return nil, err
	}
}

func TestReadFileContent(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadOnly)
		mock.AddFile("hello.pk", []byte("hello world"))

		opener := newMockSandboxOpener(mock)
		content, err := readFileContent("/tmp/test/hello.pk", opener)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Equal(t, "hello world", string(content), "got %q, want %q", string(content), "hello world")
	})

	t.Run("file not found", func(t *testing.T) {
		t.Parallel()
		mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadOnly)

		opener := newMockSandboxOpener(mock)
		_, err := readFileContent("/tmp/test/missing.pk", opener)
		require.Error(t, err, "expected error for missing file, got nil")
		assert.Contains(t, err.Error(), "file not found", "expected 'file not found' error, got: %v", err)
	})

	t.Run("sandbox creation failure", func(t *testing.T) {
		t.Parallel()
		opener := newFailingSandboxOpener(errors.New("sandbox failed"))
		_, err := readFileContent("/tmp/test/hello.pk", opener)
		require.Error(t, err, "expected error, got nil")
		assert.Contains(t, err.Error(), "sandbox", "expected sandbox error, got: %v", err)
	})
}

func TestOutputCheckMode(t *testing.T) {
	t.Parallel()

	t.Run("changed prints needs formatting", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		err := outputCheckMode(&buffer, "test.pk", true)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Contains(t, buffer.String(), "needs formatting", "expected 'needs formatting' message, got: %q", buffer.String())
		assert.Contains(t, buffer.String(), "test.pk", "expected path in output, got: %q", buffer.String())
	})

	t.Run("unchanged prints nothing", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		err := outputCheckMode(&buffer, "test.pk", false)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Equal(t, 0, buffer.Len(), "expected no output for unchanged file, got: %q", buffer.String())
	})
}

func TestOutputDryRunMode(t *testing.T) {
	t.Parallel()

	t.Run("changed prints would format", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		err := outputDryRunMode(&buffer, "test.pk", true)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Contains(t, buffer.String(), "Would format:", "expected 'Would format:' message, got: %q", buffer.String())
		assert.Contains(t, buffer.String(), "test.pk", "expected path in output, got: %q", buffer.String())
	})

	t.Run("unchanged prints nothing", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		err := outputDryRunMode(&buffer, "test.pk", false)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Equal(t, 0, buffer.Len(), "expected no output for unchanged file, got: %q", buffer.String())
	})
}

func TestOutputListMode(t *testing.T) {
	t.Parallel()

	t.Run("changed prints path", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		err := outputListMode(&buffer, "test.pk", true)
		require.NoError(t, err, "unexpected error: %v", err)
		output := strings.TrimSpace(buffer.String())
		assert.Equal(t, "test.pk", output, "expected %q, got %q", "test.pk", output)
	})

	t.Run("unchanged prints nothing", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		err := outputListMode(&buffer, "test.pk", false)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Equal(t, 0, buffer.Len(), "expected no output for unchanged file, got: %q", buffer.String())
	})
}

func TestOutputWriteMode(t *testing.T) {
	t.Parallel()

	t.Run("unchanged skips write", func(t *testing.T) {
		t.Parallel()
		var stdout bytes.Buffer
		stats := &Statistics{
			stdout: &stdout,
		}
		err := outputWriteMode(stats, "/tmp/test/hello.pk", []byte("content"), false)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Equal(t, 0, stats.formatted, "expected formatted=0, got %d", stats.formatted)
		assert.Equal(t, 0, stdout.Len(), "expected no output for unchanged file, got: %q", stdout.String())
	})

	t.Run("changed writes file", func(t *testing.T) {
		t.Parallel()
		mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadWrite)
		var stdout bytes.Buffer
		stats := &Statistics{
			newSandbox: newMockSandboxOpener(mock),
			stdout:     &stdout,
		}
		formatted := []byte("formatted content")
		err := outputWriteMode(stats, "/tmp/test/hello.pk", formatted, true)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Equal(t, 1, stats.formatted, "expected formatted=1, got %d", stats.formatted)
		assert.Contains(t, stdout.String(), "Formatted:", "expected 'Formatted:' message, got: %q", stdout.String())
	})

	t.Run("sandbox creation failure", func(t *testing.T) {
		t.Parallel()
		var stdout bytes.Buffer
		stats := &Statistics{
			newSandbox: newFailingSandboxOpener(errors.New("write sandbox failed")),
			stdout:     &stdout,
		}
		err := outputWriteMode(stats, "/tmp/test/hello.pk", []byte("content"), true)
		require.Error(t, err, "expected error, got nil")
		assert.Contains(t, err.Error(), "write sandbox", "expected write sandbox error, got: %v", err)
	})
}

func TestProcessFile(t *testing.T) {
	t.Parallel()

	t.Run("unchanged file", func(t *testing.T) {
		t.Parallel()
		mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadOnly)
		mock.AddFile("hello.pk", []byte("hello"))

		var stdout, stderr bytes.Buffer
		stats := &Statistics{
			formatter: &mockFileFormatter{
				FormatFunc: func(_ context.Context, source []byte) ([]byte, error) {
					return source, nil
				},
			},
			newSandbox: newMockSandboxOpener(mock),
			stdout:     &stdout,
			stderr:     &stderr,
		}
		flags := &formatFlags{write: true}
		err := processFile(context.Background(), stats, "/tmp/test/hello.pk", flags)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Equal(t, 1, stats.total, "expected total=1, got %d", stats.total)
		assert.Equal(t, 0, stats.needsFormatting, "expected needsFormatting=0, got %d", stats.needsFormatting)
	})

	t.Run("changed file in check mode", func(t *testing.T) {
		t.Parallel()
		mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadOnly)
		mock.AddFile("hello.pk", []byte("hello"))

		var stdout, stderr bytes.Buffer
		stats := &Statistics{
			formatter: &mockFileFormatter{
				FormatFunc: func(_ context.Context, _ []byte) ([]byte, error) {
					return []byte("hello formatted"), nil
				},
			},
			newSandbox: newMockSandboxOpener(mock),
			stdout:     &stdout,
			stderr:     &stderr,
		}
		flags := &formatFlags{check: true}
		err := processFile(context.Background(), stats, "/tmp/test/hello.pk", flags)
		require.NoError(t, err, "unexpected error: %v", err)
		assert.Equal(t, 1, stats.needsFormatting, "expected needsFormatting=1, got %d", stats.needsFormatting)
		assert.Contains(t, stdout.String(), "needs formatting", "expected 'needs formatting' message, got: %q", stdout.String())
	})

	t.Run("format error", func(t *testing.T) {
		t.Parallel()
		mock := safedisk.NewMockSandbox("/tmp/test", safedisk.ModeReadOnly)
		mock.AddFile("hello.pk", []byte("hello"))

		var stdout, stderr bytes.Buffer
		stats := &Statistics{
			formatter: &mockFileFormatter{
				FormatFunc: func(_ context.Context, _ []byte) ([]byte, error) {
					return nil, errors.New("parse error")
				},
			},
			newSandbox: newMockSandboxOpener(mock),
			stdout:     &stdout,
			stderr:     &stderr,
		}
		flags := &formatFlags{write: true}
		err := processFile(context.Background(), stats, "/tmp/test/hello.pk", flags)
		require.Error(t, err, "expected error, got nil")
		assert.Contains(t, err.Error(), "formatting", "expected formatting error, got: %v", err)
	})
}

func TestPrintSummaryAndGetExitCode(t *testing.T) {
	t.Parallel()

	t.Run("no files returns current exit code", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, total: 0}
		flags := &formatFlags{}
		got := printSummaryAndGetExitCode(stats, flags, 0)
		assert.Equal(t, 0, got, "expected exit code 0, got %d", got)
	})

	t.Run("check mode with needs formatting returns 1", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, total: 3, needsFormatting: 2}
		flags := &formatFlags{check: true}
		got := printSummaryAndGetExitCode(stats, flags, 0)
		assert.Equal(t, 1, got, "expected exit code 1, got %d", got)
	})

	t.Run("check mode all formatted returns current", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, total: 3, needsFormatting: 0}
		flags := &formatFlags{check: true}
		got := printSummaryAndGetExitCode(stats, flags, 0)
		assert.Equal(t, 0, got, "expected exit code 0, got %d", got)
	})

	t.Run("dry run prints summary", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, total: 5, needsFormatting: 3}
		flags := &formatFlags{dryRun: true}
		got := printSummaryAndGetExitCode(stats, flags, 0)
		assert.Equal(t, 0, got, "expected exit code 0, got %d", got)
		assert.Contains(t, stdout.String(), "Dry run complete", "expected dry run summary, got: %q", stdout.String())
	})

	t.Run("list mode with needs formatting returns 1", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, total: 5, needsFormatting: 2}
		flags := &formatFlags{list: true}
		got := printSummaryAndGetExitCode(stats, flags, 0)
		assert.Equal(t, 1, got, "expected exit code 1, got %d", got)
	})

	t.Run("list mode with no changes returns current", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, total: 5, needsFormatting: 0}
		flags := &formatFlags{list: true}
		got := printSummaryAndGetExitCode(stats, flags, 0)
		assert.Equal(t, 0, got, "expected exit code 0, got %d", got)
	})

	t.Run("write mode with errors returns 1", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, total: 5, formatted: 3, errors: 2}
		flags := &formatFlags{write: true}
		got := printSummaryAndGetExitCode(stats, flags, 0)
		assert.Equal(t, 1, got, "expected exit code 1, got %d", got)
	})

	t.Run("write mode no errors returns current", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, total: 5, formatted: 5, errors: 0}
		flags := &formatFlags{write: true}
		got := printSummaryAndGetExitCode(stats, flags, 0)
		assert.Equal(t, 0, got, "expected exit code 0, got %d", got)
	})
}

func TestPrintCheckSummary(t *testing.T) {
	t.Parallel()

	t.Run("needs formatting returns 1", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, needsFormatting: 3}
		got := printCheckSummary(stats, 0)
		assert.Equal(t, 1, got, "expected exit code 1, got %d", got)
		assert.Contains(t, stderr.String(), "Check failed", "expected 'Check failed' in stderr, got: %q", stderr.String())
	})

	t.Run("all formatted returns current", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, needsFormatting: 0}
		got := printCheckSummary(stats, 0)
		assert.Equal(t, 0, got, "expected exit code 0, got %d", got)
		assert.Contains(t, stdout.String(), "properly formatted", "expected 'properly formatted' message, got: %q", stdout.String())
	})
}

func TestPrintWriteSummary(t *testing.T) {
	t.Parallel()

	t.Run("with errors returns 1", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, formatted: 3, errors: 2}
		got := printWriteSummary(stats, 0)
		assert.Equal(t, 1, got, "expected exit code 1, got %d", got)
		assert.Contains(t, stderr.String(), "Errors:", "expected 'Errors:' in stderr, got: %q", stderr.String())
	})

	t.Run("no errors returns current", func(t *testing.T) {
		t.Parallel()
		var stdout, stderr bytes.Buffer
		stats := &Statistics{stdout: &stdout, stderr: &stderr, formatted: 5, errors: 0}
		got := printWriteSummary(stats, 0)
		assert.Equal(t, 0, got, "expected exit code 0, got %d", got)
		assert.Contains(t, stdout.String(), "Formatted", "expected 'Formatted' message, got: %q", stdout.String())
	})
}

func TestFmtUsage(t *testing.T) {
	t.Parallel()

	var buffer bytes.Buffer
	fmtUsage(&buffer)
	output := buffer.String()

	expectedStrings := []string{
		"piko fmt",
		"Usage:",
		"-r",
		"-n",
		"-check",
		"-w",
		"-l",
		"Examples:",
	}

	for _, want := range expectedStrings {
		assert.Contains(t, output, want, "usage output missing %q\nfull output:\n%s", want, output)
	}
}
