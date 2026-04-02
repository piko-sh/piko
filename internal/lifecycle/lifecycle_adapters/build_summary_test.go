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
	"testing"
	"time"

	"piko.sh/piko/internal/colour"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
)

func TestFormatBuildSummary(t *testing.T) {

	disableColour := func(t *testing.T) {
		t.Helper()
		previous := colour.Enabled()
		colour.SetEnabled(false)
		t.Cleanup(func() { colour.SetEnabled(previous) })
	}

	t.Run("AllSuccess", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched: 5,
			TotalCompleted:  5,
			TotalFailed:     0,
			Duration:        1200 * time.Millisecond,
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "5 completed")
		assert.Contains(t, output, "5 tasks dispatched")

		assert.NotContains(t, output, "✗")
	})

	t.Run("WithFailures", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched: 10,
			TotalCompleted:  7,
			TotalFailed:     3,
			Duration:        1200 * time.Millisecond,
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "✗")
		assert.Contains(t, output, "3 failed")
	})

	t.Run("WithFatalFailures", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched:  10,
			TotalCompleted:   6,
			TotalFailed:      4,
			TotalFatalFailed: 2,
			Duration:         1200 * time.Millisecond,
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "4 failed")
		assert.Contains(t, output, "2 fatal")
	})

	t.Run("WithRetries", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched: 8,
			TotalCompleted:  8,
			TotalRetried:    3,
			Duration:        1200 * time.Millisecond,
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "↻")
		assert.Contains(t, output, "3 retried")
	})

	t.Run("TimedOut", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched: 10,
			TotalCompleted:  4,
			TotalFailed:     6,
			TimedOut:        true,
			Duration:        1200 * time.Millisecond,
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "Build timed out")
	})

	t.Run("FailureDetails", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched: 3,
			TotalCompleted:  1,
			TotalFailed:     2,
			Duration:        1200 * time.Millisecond,
			Failures: []lifecycle_domain.BuildFailure{
				{
					ArtefactID: "components/button.pk",
					Executor:   "go-codegen",
					Profile:    "default",
					Error:      "template rendering failed",
					Attempt:    2,
					IsFatal:    false,
				},
			},
		}

		output := FormatBuildSummary(result)
		require.Contains(t, output, "go-codegen")

		assert.Contains(t, output, "components/button.pk")
		assert.Contains(t, output, "attempt 2")
		assert.Contains(t, output, "template rendering failed")
	})

	t.Run("FatalFailureDetails", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched:  2,
			TotalCompleted:   1,
			TotalFailed:      1,
			TotalFatalFailed: 1,
			Duration:         1200 * time.Millisecond,
			Failures: []lifecycle_domain.BuildFailure{
				{
					ArtefactID: "components/header.pk",
					Executor:   "css-minifier",
					Profile:    "production",
					Error:      "unsupported syntax",
					Attempt:    1,
					IsFatal:    true,
				},
			},
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "fatal")
		assert.Contains(t, output, "not retried")
		assert.Contains(t, output, "css-minifier")
	})

	t.Run("StripsOrchestratorSentinel", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched: 1,
			TotalFailed:     1,
			Duration:        100 * time.Millisecond,
			Failures: []lifecycle_domain.BuildFailure{
				{
					Executor: "compile-component",
					Error:    "some error: capability: fatal error: orchestrator: fatal error",
					Attempt:  1,
					IsFatal:  true,
				},
			},
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "some error")
		assert.NotContains(t, output, ": orchestrator: fatal error")
		assert.NotContains(t, output, ": capability: fatal error")
	})

	t.Run("DiagnosticSnippetFormat", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched: 1,
			TotalFailed:     1,
			Duration:        100 * time.Millisecond,
			Failures: []lifecycle_domain.BuildFailure{
				{
					ArtefactID: "components/my-widget.pkc",
					Executor:   "compile-component",
					Error:      "capability 'compile-component' failed: typescript parsing errors in my-widget.ts: The symbol \"v\" has already been declared (line 25, col 10)\n    25 |     const v = 2;\n       |           ^",
					Attempt:    1,
					IsFatal:    true,
				},
			},
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "fatal")
		assert.Contains(t, output, `The symbol "v" has already been declared`)
		assert.Contains(t, output, "components/my-widget.pkc:25:10")
		assert.Contains(t, output, "compile-component, attempt 1")
		assert.Contains(t, output, "25 |")
		assert.Contains(t, output, "const v = 2;")
		assert.Contains(t, output, "^")
		assert.NotContains(t, output, "capability 'compile-component' failed")
	})

	t.Run("PlainErrorFallback", func(t *testing.T) {

		disableColour(t)

		result := &lifecycle_domain.BuildResult{
			TotalDispatched: 1,
			TotalFailed:     1,
			Duration:        100 * time.Millisecond,
			Failures: []lifecycle_domain.BuildFailure{
				{
					ArtefactID: "components/button.pk",
					Executor:   "go-codegen",
					Error:      "template rendering failed: missing field",
					Attempt:    2,
				},
			},
		}

		output := FormatBuildSummary(result)

		assert.Contains(t, output, "go-codegen failed")
		assert.Contains(t, output, "template rendering failed: missing field")
		assert.Contains(t, output, "attempt 2")
	})
}

func TestCleanBuildError(t *testing.T) {
	t.Parallel()

	t.Run("strips orchestrator sentinel", func(t *testing.T) {
		t.Parallel()

		got := cleanBuildError("parse failed: capability: fatal error: orchestrator: fatal error")

		assert.Equal(t, "parse failed", got)
	})

	t.Run("strips capability sentinel only", func(t *testing.T) {
		t.Parallel()

		got := cleanBuildError("parse failed: capability: fatal error")

		assert.Equal(t, "parse failed", got)
	})

	t.Run("returns unchanged when no sentinel", func(t *testing.T) {
		t.Parallel()

		got := cleanBuildError("normal error message")

		assert.Equal(t, "normal error message", got)
	})

	t.Run("handles empty string", func(t *testing.T) {
		t.Parallel()

		got := cleanBuildError("")

		assert.Equal(t, "", got)
	})
}

func TestExtractBuildErrorSnippet(t *testing.T) {
	t.Parallel()

	t.Run("extracts full diagnostic from wrapped error", func(t *testing.T) {
		t.Parallel()

		input := `capability 'compile-component' failed: typescript parsing errors in test.ts: The symbol "v" has already been declared (line 25, col 10)
    25 |     const v = 2;
       |           ^`

		snippet := extractBuildErrorSnippet(input)

		require.NotNil(t, snippet)
		assert.Equal(t, `The symbol "v" has already been declared`, snippet.coreMessage)
		assert.Equal(t, 25, snippet.line)
		assert.Equal(t, 10, snippet.column)
		assert.Equal(t, "    const v = 2;", snippet.sourceText)
		assert.Equal(t, "          ^", snippet.caretLine)
	})

	t.Run("returns nil for plain errors", func(t *testing.T) {
		t.Parallel()

		snippet := extractBuildErrorSnippet("template rendering failed")

		assert.Nil(t, snippet)
	})

	t.Run("returns nil for empty string", func(t *testing.T) {
		t.Parallel()

		snippet := extractBuildErrorSnippet("")

		assert.Nil(t, snippet)
	})

	t.Run("extracts without source snippet", func(t *testing.T) {
		t.Parallel()

		input := `The symbol "x" has already been declared (line 1, col 5)`

		snippet := extractBuildErrorSnippet(input)

		require.NotNil(t, snippet)
		assert.Equal(t, `The symbol "x" has already been declared`, snippet.coreMessage)
		assert.Equal(t, 1, snippet.line)
		assert.Equal(t, 5, snippet.column)
		assert.Empty(t, snippet.sourceText)
	})
}

func TestFormatGeneratorSummary(t *testing.T) {

	disableColour := func(t *testing.T) {
		t.Helper()
		previous := colour.Enabled()
		colour.SetEnabled(false)
		t.Cleanup(func() { colour.SetEnabled(previous) })
	}

	t.Run("AllCategories", func(t *testing.T) {
		disableColour(t)

		result := &GeneratorResult{
			Pages:     3,
			Partials:  5,
			Emails:    2,
			Artefacts: 10,
			Duration:  1300 * time.Millisecond,
		}

		output := FormatGeneratorSummary(result)

		assert.Contains(t, output, "Generator Summary")
		assert.Contains(t, output, "10 artefacts generated")
		assert.Contains(t, output, "3 pages")
		assert.Contains(t, output, "5 partials")
		assert.Contains(t, output, "2 emails")
	})

	t.Run("ZeroCounts", func(t *testing.T) {
		disableColour(t)

		result := &GeneratorResult{
			Duration: 100 * time.Millisecond,
		}

		output := FormatGeneratorSummary(result)

		assert.Contains(t, output, "0 pages")
		assert.Contains(t, output, "0 partials")
		assert.Contains(t, output, "0 emails")
		assert.Contains(t, output, "0 artefacts")
	})

	t.Run("SingleItems", func(t *testing.T) {
		disableColour(t)

		result := &GeneratorResult{
			Pages:     1,
			Partials:  1,
			Emails:    1,
			Artefacts: 3,
			Duration:  500 * time.Millisecond,
		}

		output := FormatGeneratorSummary(result)

		assert.Contains(t, output, "1 page\n")
		assert.Contains(t, output, "1 partial\n")
		assert.Contains(t, output, "1 email\n")
		assert.NotContains(t, output, "1 pages")
		assert.NotContains(t, output, "1 partials")
		assert.NotContains(t, output, "1 emails")
	})

	t.Run("DurationFormatted", func(t *testing.T) {
		disableColour(t)

		result := &GeneratorResult{
			Artefacts: 5,
			Duration:  2456 * time.Millisecond,
		}

		output := FormatGeneratorSummary(result)

		assert.Contains(t, output, "2.46s")
	})
}

func TestPluralise(t *testing.T) {
	t.Parallel()

	t.Run("zero uses plural", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "pages", pluralise("page", 0))
	})

	t.Run("one uses singular", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "page", pluralise("page", 1))
	})

	t.Run("many uses plural", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "pages", pluralise("page", 5))
	})
}

func TestExtractCoreMessage(t *testing.T) {
	t.Parallel()

	t.Run("strips wrapping chain", func(t *testing.T) {
		t.Parallel()

		got := extractCoreMessage(`capability 'x' failed: compile error: The symbol "v" has already been declared `)

		assert.Equal(t, `The symbol "v" has already been declared`, got)
	})

	t.Run("returns as-is when no chain", func(t *testing.T) {
		t.Parallel()

		got := extractCoreMessage("Something went wrong ")

		assert.Equal(t, "Something went wrong", got)
	})

	t.Run("handles quoted colon in message", func(t *testing.T) {
		t.Parallel()

		got := extractCoreMessage(`errors in test.ts: Cannot find name "foo" `)

		assert.Equal(t, `Cannot find name "foo"`, got)
	})
}
