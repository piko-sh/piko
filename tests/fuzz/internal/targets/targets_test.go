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

package targets

import (
	"context"
	"errors"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestWorkerResultClassification(t *testing.T) {
	for _, testCase := range []struct {
		output string
		want   string
	}{
		{output: "warnings\nPIKO_TARGET_RESULT {\"Property\":\"form-value\",\"Detail\":\"wrong count 7\"}\nexit status 1", want: "binder/form-value:"},
		{output: "# package build failed", want: "infrastructure:"},
		{output: "PIKO_TARGET_RESULT malformed", want: "infrastructure:"},
		{output: "PIKO_TARGET_RESULT {\"Detail\":\"creating cache directory: no space\",\"Infrastructure\":true}", want: "infrastructure: binder worker: creating cache directory: no space"},
		{output: "PIKO_TARGET_RESULT {\"Property\":\"unclassified\",\"Detail\":\"parse failed\"}", want: "binder/unclassified: parse failed"},
	} {
		err := workerResult(testCase.output, errors.New("exit status 1"), "binder")
		require.Error(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), testCase.want), "got %v, want prefix %q", err, testCase.want)
	}
	assert.NoError(t, workerResult("anything", nil, "binder"))
}

func TestPropertyFailureSubject(t *testing.T) {
	failure := &propertyFailure{Target: "binder", Property: "form-value", Source: "binder/form", Detail: "wrong"}
	assert.Equal(t, "binder/form-value: wrong", failure.Error())
	assert.Equal(t, "binder/form", failure.FailureSubject())
}

func TestTargetCatalogues(t *testing.T) {
	list := List()
	assert.True(t, slices.IsSorted(list))
	assert.Contains(t, list, "binder-safe")
	assert.True(t, IsApplication("pages"))
	assert.False(t, IsApplication("binder"))
	assert.True(t, UsesPackageWorker("binder-safe"))
	assert.True(t, IsPublicContract("example-contracts"))
	for _, name := range BrowserNames() {
		assert.True(t, IsApplication(name), name)
		assert.True(t, IsBrowser(name), name)
	}
	assert.False(t, IsBrowser("pages"))
	for name, fuzzer := range Fuzzers() {
		assert.NotEmpty(t, fuzzer.Package, name)
		assert.True(t, strings.HasPrefix(fuzzer.Function, "Fuzz"), name)
	}
}

func TestRunRejectsUnknownTarget(t *testing.T) {
	err := Run(t.Context(), t.TempDir(), t.TempDir(), &casefile.Case{Target: "no-such-target"})
	require.ErrorIs(t, err, ErrUnknownTarget)
}

func TestPrivateDirectoryIsAbsoluteAndRemovable(t *testing.T) {
	directory, err := privateDirectory(t.TempDir(), "target-")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(directory))
	assert.DirExists(t, directory)

	require.NoError(t, removeDirectory(directory))
	assert.NoDirExists(t, directory)
}

func TestPrivateDirectoryReportsInfrastructureFailure(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	_, err := privateDirectory(missing, "target-")

	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "infrastructure: create target- directory:"), err.Error())
}

func TestRunWithWorkersReportsInfrastructureForMissingWorkDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	err := RunWithWorkers(t.Context(), t.TempDir(), missing, "", &casefile.Case{Target: "binder"})

	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "infrastructure:"), err.Error())
}

func TestRunPublicContractReportsInfrastructureForMissingWorkDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	_, err := RunPublicContract(t.Context(), t.TempDir(), missing, &casefile.Case{Target: "example-contracts"})

	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "infrastructure:"), err.Error())
}

func TestRunWorkerClassifiesCachedWorkerResult(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		script string
		tag    string
		want   string
	}{
		{name: "pass", script: "exit 0", want: ""},
		{name: "property", script: `echo 'PIKO_TARGET_RESULT {"Property":"form-value","Detail":"wrong"}' >&2; exit 1`, want: "binder/form-value: wrong"},
		{name: "infrastructure", script: `echo 'PIKO_TARGET_RESULT {"Detail":"disk full","Infrastructure":true}' >&2; exit 1`, want: "infrastructure: binder worker: disk full"},
		{name: "safe build", script: `echo 'PIKO_TARGET_RESULT {"Property":"form-value","Detail":"wrong"}' >&2; exit 1`, tag: "safe", want: "binder-safe/form-value: wrong"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			workers := t.TempDir()
			binary := filepath.Join(workers, "target-worker")
			target := "binder"
			if testCase.tag != "" {
				binary += "-" + testCase.tag
				target = binderSafeTarget
			}
			require.NoError(t, os.WriteFile(binary, []byte("#!/bin/sh\n"+testCase.script+"\n"), 0o700))

			err := runWorker(t.Context(), t.TempDir(), t.TempDir(), workers, "case", testCase.tag, target)

			if testCase.want == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, testCase.want)
		})
	}
}

func TestRunWorkerLabelsDefaultBinderSafeBuild(t *testing.T) {
	workers := t.TempDir()
	script := "#!/bin/sh\necho 'PIKO_TARGET_RESULT {\"Property\":\"p\",\"Detail\":\"d\"}' >&2\nexit 1\n"
	require.NoError(t, os.WriteFile(filepath.Join(workers, "target-worker"), []byte(script), 0o700))

	err := runWorker(t.Context(), t.TempDir(), t.TempDir(), workers, "case", "", binderSafeTarget)

	require.EqualError(t, err, "binder-safe/default/p: d")
}

func TestRunWorkerReturnsCancellationCause(t *testing.T) {
	workers := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workers, "target-worker"), []byte("#!/bin/sh\nexit 0\n"), 0o700))
	cause := errors.New("campaign stopped")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)

	err := runWorker(ctx, t.TempDir(), t.TempDir(), workers, "case", "", "binder")

	require.ErrorIs(t, err, cause)
}

func TestFuzzersResolveToFuzzFunctions(t *testing.T) {
	repository := repositoryRoot(t)
	buildContext := build.Default
	buildContext.BuildTags = []string{"fuzz"}
	for name, fuzzer := range Fuzzers() {
		t.Run(name, func(t *testing.T) {
			relative, found := strings.CutPrefix(fuzzer.Package, "piko.sh/piko/")
			require.True(t, found, "fuzzer package %s is outside the Piko module", fuzzer.Package)
			directory := filepath.Join(repository, filepath.FromSlash(relative))
			assert.True(t, declaresFuzzFunction(t, &buildContext, directory, fuzzer.Function), "%s has no func %s(*testing.F)", fuzzer.Package, fuzzer.Function)
		})
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(root, "go.work"))
	return root
}

func declaresFuzzFunction(t *testing.T, buildContext *build.Context, directory, function string) bool {
	t.Helper()
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	files := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		matched, err := buildContext.MatchFile(directory, entry.Name())
		require.NoError(t, err)
		if !matched {
			continue
		}
		file, err := parser.ParseFile(files, filepath.Join(directory, entry.Name()), nil, parser.SkipObjectResolution)
		require.NoError(t, err)
		for _, declaration := range file.Decls {
			if isFuzzFunction(declaration, function) {
				return true
			}
		}
	}
	return false
}

func isFuzzFunction(declaration ast.Decl, function string) bool {
	decl, ok := declaration.(*ast.FuncDecl)
	if !ok || decl.Recv != nil || decl.Name.Name != function || len(decl.Type.Params.List) != 1 {
		return false
	}
	parameter := decl.Type.Params.List[0]
	if len(parameter.Names) > 1 {
		return false
	}
	pointer, ok := parameter.Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := pointer.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "testing" && selector.Sel.Name == "F"
}
