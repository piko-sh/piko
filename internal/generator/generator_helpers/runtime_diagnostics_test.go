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

package generator_helpers

import (
	"testing"

	"piko.sh/piko/internal/colour"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_dto"
)

func init() {

	colour.SetEnabled(false)
}

func TestAppendDiagnostic(t *testing.T) {
	t.Parallel()

	t.Run("append to empty slice", func(t *testing.T) {
		t.Parallel()

		var diagnostics []*generator_dto.RuntimeDiagnostic
		diagnostics = AppendDiagnostic(diagnostics, &generator_dto.RuntimeDiagnostic{
			Severity:   generator_dto.Error,
			Message:    "test message",
			SourcePath: "file.pk",
			Expression: "expr",
			Line:       10,
			Column:     5,
		})
		require.Len(t, diagnostics, 1)
		d := diagnostics[0]
		assert.Equal(t, generator_dto.Error, d.Severity)
		assert.Equal(t, "test message", d.Message)
		assert.Equal(t, "file.pk", d.SourcePath)
		assert.Equal(t, "expr", d.Expression)
		assert.Equal(t, 10, d.Line)
		assert.Equal(t, 5, d.Column)
	})

	t.Run("append multiple", func(t *testing.T) {
		t.Parallel()

		var diagnostics []*generator_dto.RuntimeDiagnostic
		diagnostics = AppendDiagnostic(diagnostics, &generator_dto.RuntimeDiagnostic{Severity: generator_dto.Error, Message: "first", SourcePath: "a.pk", Expression: "x", Line: 1, Column: 1})
		diagnostics = AppendDiagnostic(diagnostics, &generator_dto.RuntimeDiagnostic{Severity: generator_dto.Warning, Message: "second", SourcePath: "b.pk", Expression: "y", Line: 2, Column: 3})
		require.Len(t, diagnostics, 2)
		assert.Equal(t, "first", diagnostics[0].Message)
		assert.Equal(t, "second", diagnostics[1].Message)
	})
}

func TestFormatRuntimeDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("empty diagnostics returns empty string", func(t *testing.T) {
		t.Parallel()

		result := FormatRuntimeDiagnostics(nil, "")
		assert.Equal(t, "", result)
	})

	t.Run("empty slice returns empty string", func(t *testing.T) {
		t.Parallel()

		result := FormatRuntimeDiagnostics([]*generator_dto.RuntimeDiagnostic{}, "")
		assert.Equal(t, "", result)
	})

	t.Run("single error diagnostic", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Error,
				Message:    "undefined variable",
				SourcePath: "page.pk",
				Expression: "foo.bar",
				Line:       5,
				Column:     10,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "")
		assert.Contains(t, result, "1 runtime issue")
		assert.Contains(t, result, "error")
		assert.Contains(t, result, "undefined variable")
		assert.Contains(t, result, "page.pk")
		assert.Contains(t, result, "5:10")
		assert.Contains(t, result, "foo.bar")
		assert.Contains(t, result, "^^^^^^^")
	})

	t.Run("warning severity", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Warning,
				Message:    "deprecated usage",
				SourcePath: "page.pk",
				Line:       1,
				Column:     1,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "")
		assert.Contains(t, result, "warning")
		assert.Contains(t, result, "deprecated usage")
	})

	t.Run("info severity", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Info,
				Message:    "informational",
				SourcePath: "page.pk",
				Line:       1,
				Column:     1,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "")
		assert.Contains(t, result, "info")
	})

	t.Run("default severity", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Debug,
				Message:    "debug info",
				SourcePath: "page.pk",
				Line:       1,
				Column:     1,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "")
		assert.Contains(t, result, "debug info")
	})

	t.Run("baseDir joins relative path", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Error,
				Message:    "err",
				SourcePath: "templates/page.pk",
				Line:       1,
				Column:     1,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "/project")
		assert.Contains(t, result, "/project/templates/page.pk")
	})

	t.Run("absolute path not joined with baseDir", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Error,
				Message:    "err",
				SourcePath: "/abs/path/page.pk",
				Line:       1,
				Column:     1,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "/project")
		assert.Contains(t, result, "/abs/path/page.pk")
		assert.NotContains(t, result, "/project/abs")
	})

	t.Run("empty expression omits expression line", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Error,
				Message:    "err",
				SourcePath: "page.pk",
				Expression: "",
				Line:       1,
				Column:     1,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "")
		assert.NotContains(t, result, "^^^")
	})

	t.Run("column 0 uses column 1 for caret", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Error,
				Message:    "err",
				SourcePath: "page.pk",
				Expression: "x",
				Line:       1,
				Column:     0,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "")
		assert.Contains(t, result, "^")
	})

	t.Run("multiple diagnostics", func(t *testing.T) {
		t.Parallel()

		diagnostics := []*generator_dto.RuntimeDiagnostic{
			{
				Severity:   generator_dto.Error,
				Message:    "first error",
				SourcePath: "a.pk",
				Line:       1,
				Column:     1,
			},
			{
				Severity:   generator_dto.Warning,
				Message:    "second warning",
				SourcePath: "b.pk",
				Line:       2,
				Column:     5,
			},
		}
		result := FormatRuntimeDiagnostics(diagnostics, "")
		assert.Contains(t, result, "2 runtime issue")
		assert.Contains(t, result, "first error")
		assert.Contains(t, result, "second warning")
	})
}

func TestValidatePikoElementTagName(t *testing.T) {
	t.Parallel()

	t.Run("valid tag passes through", func(t *testing.T) {
		t.Parallel()

		var diagnostics []*generator_dto.RuntimeDiagnostic
		result, diagnostics := ValidatePikoElementTagName("h2", diagnostics, "test.pk", "state.Tag", 1, 1)
		assert.Equal(t, "h2", result)
		assert.Empty(t, diagnostics)
	})

	t.Run("piko:a passes through", func(t *testing.T) {
		t.Parallel()

		var diagnostics []*generator_dto.RuntimeDiagnostic
		result, diagnostics := ValidatePikoElementTagName("piko:a", diagnostics, "test.pk", "state.Tag", 1, 1)
		assert.Equal(t, "piko:a", result)
		assert.Empty(t, diagnostics)
	})

	t.Run("empty tag falls back to div", func(t *testing.T) {
		t.Parallel()

		var diagnostics []*generator_dto.RuntimeDiagnostic
		result, diagnostics := ValidatePikoElementTagName("", diagnostics, "test.pk", "state.Tag", 1, 1)
		assert.Equal(t, "div", result)
		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "empty tag name")
	})

	t.Run("piko:partial rejected", func(t *testing.T) {
		t.Parallel()

		var diagnostics []*generator_dto.RuntimeDiagnostic
		result, diagnostics := ValidatePikoElementTagName("piko:partial", diagnostics, "test.pk", "state.Tag", 5, 10)
		assert.Equal(t, "div", result)
		require.Len(t, diagnostics, 1)
		assert.Contains(t, diagnostics[0].Message, "cannot target 'piko:partial'")
		assert.Equal(t, 5, diagnostics[0].Line)
		assert.Equal(t, 10, diagnostics[0].Column)
	})

	t.Run("piko:slot rejected", func(t *testing.T) {
		t.Parallel()

		var diagnostics []*generator_dto.RuntimeDiagnostic
		result, diagnostics := ValidatePikoElementTagName("piko:slot", diagnostics, "test.pk", "state.Tag", 1, 1)
		assert.Equal(t, "div", result)
		require.Len(t, diagnostics, 1)
	})

	t.Run("piko:element rejected", func(t *testing.T) {
		t.Parallel()

		var diagnostics []*generator_dto.RuntimeDiagnostic
		result, diagnostics := ValidatePikoElementTagName("piko:element", diagnostics, "test.pk", "state.Tag", 1, 1)
		assert.Equal(t, "div", result)
		require.Len(t, diagnostics, 1)
	})
}
