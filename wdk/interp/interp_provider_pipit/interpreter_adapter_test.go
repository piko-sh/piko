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

package interp_provider_pipit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validProgram   = "package main\n\nvar Value = 1\n"
	invalidProgram = "package main\n\nvar Value int = \"not a number\"\n"
)

func newTestInterpreter(t *testing.T, options ...ProviderOption) *interpreterAdapter {
	t.Helper()
	interpreter, err := NewProvider(options...).NewInterpreterPool().Get()
	require.NoError(t, err)
	adapter, ok := interpreter.(*interpreterAdapter)
	require.True(t, ok, "pool returned %T", interpreter)
	return adapter
}

func TestCompileAndExecuteErrorsNameEachStageOnce(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		source     string
		wantPhrase string
	}{
		{name: "compile failure", source: invalidProgram, wantPhrase: "compiling program"},
		{name: "init failure", source: "package main\n\nfunc init() {\n\tpanic(\"init failed\")\n}\n", wantPhrase: "executing init functions"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			err := newTestInterpreter(t).CompileAndExecute(context.Background(), "main", map[string]map[string]string{
				"main": {"main.go": testCase.source},
			})
			require.Error(t, err)
			assert.Equal(t, 1, strings.Count(err.Error(), testCase.wantPhrase), err.Error())
		})
	}
}

func TestBytecodeEmission(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name         string
		source       string
		wantFiles    []string
		missingFiles []string
		wantErr      bool
	}{
		{
			name:         "successful compilation emits bytecode and disassembly",
			source:       validProgram,
			wantFiles:    []string{"compiled/bytecode-main.txt", "compiled/bytecode-main.bin", "compiled/bytecode-main.pkasm", "source/emit_module/main/main.go"},
			missingFiles: []string{"compiled/bytecode-main.error.txt"},
		},
		{
			name:         "failed compilation emits the error instead of bytecode",
			source:       invalidProgram,
			wantErr:      true,
			wantFiles:    []string{"compiled/bytecode-main.txt", "compiled/bytecode-main.error.txt", "source/emit_module/main/main.go"},
			missingFiles: []string{"compiled/bytecode-main.bin", "compiled/bytecode-main.pkasm"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			directory := t.TempDir()
			err := newTestInterpreter(t, WithBytecodeEmission(directory)).CompileAndExecute(context.Background(), "emit/module", map[string]map[string]string{
				"main": {"main.go": testCase.source},
			})
			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			for _, file := range testCase.wantFiles {
				assert.FileExists(t, filepath.Join(directory, file))
			}
			for _, file := range testCase.missingFiles {
				assert.NoFileExists(t, filepath.Join(directory, file))
			}
		})
	}
}

func TestBytecodeEmissionErrorRecordsCompileFailure(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	err := newTestInterpreter(t, WithBytecodeEmission(directory)).CompileAndExecute(context.Background(), "main", map[string]map[string]string{
		"main": {"main.go": invalidProgram},
	})
	require.Error(t, err)
	content, readErr := os.ReadFile(filepath.Join(directory, "compiled", "bytecode-main.error.txt"))
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "compiling program")
}

func TestBytecodeEmissionToUnusableDirectoryStillCompiles(t *testing.T) {
	t.Parallel()
	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, []byte("not a directory"), 0o600))
	err := newTestInterpreter(t, WithBytecodeEmission(blocker)).CompileAndExecute(context.Background(), "main", map[string]map[string]string{
		"main": {"main.go": validProgram},
	})
	require.NoError(t, err)
}

func TestBytecodeFileSuffix(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		packages   map[string]map[string]string
		name       string
		wantSuffix string
		wantPaths  []string
	}{
		{name: "single package", packages: map[string]map[string]string{"pages/home": nil}, wantSuffix: "pages_home", wantPaths: []string{"pages_home"}},
		{name: "root package", packages: map[string]map[string]string{"": nil}, wantSuffix: "_root", wantPaths: []string{"_root"}},
		{name: "several packages", packages: map[string]map[string]string{"b": nil, "a": nil}, wantPaths: []string{"a", "b"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			suffix, paths := bytecodeFileSuffix(testCase.packages)
			assert.Equal(t, testCase.wantPaths, paths)
			if testCase.wantSuffix != "" {
				assert.Equal(t, testCase.wantSuffix, suffix)
				return
			}
			assert.True(t, strings.HasPrefix(suffix, "batch-") && strings.HasSuffix(suffix, "-2pkgs"), suffix)
		})
	}
}

func TestPoolClonesShareTheSymbolRegistry(t *testing.T) {
	t.Parallel()
	pool := NewProvider().NewInterpreterPool()
	first, err := pool.Get()
	require.NoError(t, err)
	require.NoError(t, first.CompileAndExecute(context.Background(), "mod", map[string]map[string]string{
		"util": {"util.go": "package util\n\nfunc Answer() int { return 1 }\n"},
	}))
	require.True(t, first.HasRegisteredPackage("mod/util"))

	second, err := pool.Get()
	require.NoError(t, err)
	assert.True(t, second.HasRegisteredPackage("mod/util"), "a clone from the same pool sees packages compiled by another clone")

	fresh, err := NewProvider().NewInterpreterPool().Get()
	require.NoError(t, err)
	assert.False(t, fresh.HasRegisteredPackage("mod/util"), "a new pool starts with a fresh registry")
}
