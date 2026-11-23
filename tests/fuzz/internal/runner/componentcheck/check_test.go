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

package componentcheck

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/compiler/compiler_dto"
)

func TestDiagnosticErrorsJoinsEveryErrorDiagnostic(t *testing.T) {
	diagnostics := []compiler_dto.CompilationDiagnostic{
		{Severity: compiler_dto.DiagnosticSeverityError, Message: "first failure"},
		{Severity: compiler_dto.DiagnosticSeverityWarning, Message: "ignored warning"},
		{Severity: compiler_dto.DiagnosticSeverityError, Message: "second failure"},
	}

	err := diagnosticErrors(diagnostics)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "first failure")
	assert.Contains(t, err.Error(), "second failure")
	assert.NotContains(t, err.Error(), "ignored warning")
	joined, ok := errors.AsType[interface {
		error
		Unwrap() []error
	}](err)
	require.True(t, ok, "error does not expose its joined diagnostics")
	assert.Len(t, joined.Unwrap(), 2)
}

func TestDiagnosticErrorsAcceptsWarningsOnly(t *testing.T) {
	diagnostics := []compiler_dto.CompilationDiagnostic{
		{Severity: compiler_dto.DiagnosticSeverityWarning, Message: "warning"},
		{Severity: "info", Message: "information"},
	}

	assert.NoError(t, diagnosticErrors(diagnostics))
	assert.NoError(t, diagnosticErrors(nil))
}

func TestVerifyRejectsInputsBeforeCompiling(t *testing.T) {
	t.Chdir(t.TempDir())

	require.NoError(t, Verify(t.Context(), ""), "an empty tag selects no component")
	assert.ErrorContains(t, Verify(t.Context(), "../escape"), "invalid component name")
	assert.ErrorIs(t, Verify(t.Context(), "missing-card"), fs.ErrNotExist)
}

func TestVerifyRejectsOversizedSource(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, os.Mkdir("components", 0700))
	oversized := strings.Repeat("x", sourceMax+1)
	require.NoError(t, os.WriteFile(filepath.Join("components", "huge-card.pkc"), []byte(oversized), 0600))

	assert.ErrorIs(t, Verify(t.Context(), "huge-card"), errSourceOversize)
}

func writeComponent(t *testing.T, tag, source string) {
	t.Helper()
	require.NoError(t, os.MkdirAll("components", 0700))
	require.NoError(t, os.WriteFile(filepath.Join("components", tag+".pkc"), []byte(source), 0600))
}

func TestVerifyAcceptsCompilingComponent(t *testing.T) {
	t.Chdir(t.TempDir())
	writeComponent(t, "greeting-card", `<template name="greeting-card"><p>{{ state.message }}</p></template>
<script lang="ts">
    const state = {
        message: "Hello" as string
    };
</script>`)

	assert.NoError(t, Verify(t.Context(), "greeting-card"))
}

func TestVerifyRejectsComponentWithDifferentTag(t *testing.T) {
	t.Chdir(t.TempDir())
	writeComponent(t, "named-card", `<template name="other-card"><p>text</p></template>
<script lang="ts">
    const state = {
        message: "Hello" as string
    };
</script>`)

	assert.ErrorContains(t, Verify(t.Context(), "named-card"), "compiler did not produce <named-card>")
}

func TestVerifyReportsCompilationFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	writeComponent(t, "broken-card", `<template name="broken-card"><p>{{ state.message( }}</p></template>
<script lang="ts">
    const state = {
        message: "Hello" as string
    };
    function broken( {
</script>`)

	assert.Error(t, Verify(t.Context(), "broken-card"))
}
