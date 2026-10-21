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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pipit.sh/pipit"
)

func TestCompileAndExecuteListsEveryProblemWithAHint(t *testing.T) {
	t.Parallel()

	err := newTestInterpreter(t).CompileAndExecute(context.Background(), "example.com/site", map[string]map[string]string{
		"typed":    {"typed.go": "package typed\n\nvar Value int = \"not a number\"\n"},
		"embedded": {"embedded.go": "package embedded\n\nimport _ \"embed\"\n\n//go:embed notes.txt\nvar Notes string\n"},
		"guarded": {
			"guarded.go":      "package guarded\n\nfunc Value() string { return notes }\n",
			"notes_native.go": "//go:build !interp\n\npackage guarded\n\nimport _ \"embed\"\n\n//go:embed notes.txt\nvar notes string\n",
			"notes_interp.go": "//go:build interp\n\npackage guarded\n\nvar notes = \"read at run time\"\n",
		},
	})

	require.Error(t, err)
	require.ErrorIs(t, err, pipit.ErrTypeCheck, "the type error is still matched through the aggregate")
	require.ErrorIs(t, err, pipit.ErrEmbedUnsupported, "the embed problem is still matched through the aggregate")

	message := err.Error()
	assert.Contains(t, message, `dev-i could not compile module "example.com/site" (2 problems):`)
	assert.Contains(t, message, "1. //go:embed in a package dev-i compiles from source")
	assert.Contains(t, message, "embedded.go")
	assert.Contains(t, message, "2. type error")
	assert.Contains(t, message, "example.com/site/typed")
	assert.Contains(t, message, "hint: dev-i type-checks the files the interp build tag selects",
		"piko advises on the type error, for which pipit gives no hint")
	assert.Equal(t, 1, strings.Count(message, "//go:build !interp, and declare"),
		"the embed problem keeps pipit's hint and does not get a second one")
	assert.Equal(t, 1, strings.Count(message, "note: dev-i runs from the project root"),
		"piko adds where dev-i reads relative paths from, once")
	assert.NotContains(t, message, "notes_native.go", "a //go:embed guarded by //go:build !interp is not a problem")
	assert.NotContains(t, message, "example.com/site/guarded")
}

func TestCompileProblemsErrorFormatsEachProblem(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		cause        error
		name         string
		wantContains []string
		wantMissing  []string
	}{
		{
			name:  "a single problem is counted in the singular and keeps its whole message",
			cause: fmt.Errorf("compiling program: %w: package example.com/site: oops", pipit.ErrTypeCheck),
			wantContains: []string{
				`dev-i could not compile module "example.com/site" (1 problem):`,
				"  1. type error\n     compiling program: type check failed: package example.com/site: oops",
				"     hint: dev-i type-checks the files the interp build tag selects",
			},
		},
		{
			name: "a hint pipit already gave is not repeated",
			cause: &pipit.ProblemList{Problems: []error{
				fmt.Errorf("%w at page.go:3:1\n\nhint: pipit's own advice", pipit.ErrEmbedUnsupported),
				fmt.Errorf("compiling BuildAST: %w in function \"BuildAST\"", pipit.ErrRegisterOverflow),
			}},
			wantContains: []string{
				"(2 problems):",
				"  1. //go:embed in a package dev-i compiles from source\n",
				"\n\n     hint: pipit's own advice\n\n     note: dev-i runs from the project root",
				"  2. function too large for the interpreter\n",
				"     hint: if the function is a page's or partial's generated BuildAST, move part of its template into a partial",
			},
		},
		{
			name: "a package missing from the symbol registry is told to extract it",
			cause: fmt.Errorf("%w: example.com/site: %w", pipit.ErrTypeCheck,
				errors.Join(pipit.ErrPackageNotInRegistry, errors.New("could not import gopkg.in/yaml.v3"))),
			wantContains: []string{
				"  1. package without a symbol table\n",
				"note: in a piko project the manifest is piko-symbols.yaml",
			},
			wantMissing: []string{"hint:"},
		},
		{
			name:         "a problem no sentinel identifies is shown without a hint",
			cause:        errors.New("something else went wrong"),
			wantContains: []string{"  1. compile error\n     something else went wrong"},
			wantMissing:  []string{"hint:"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := newCompileProblemsError("example.com/site", testCase.cause)

			require.ErrorIs(t, err, testCase.cause, "the aggregate unwraps to pipit's error")
			message := err.Error()
			for _, fragment := range testCase.wantContains {
				assert.Contains(t, message, fragment)
			}
			for _, fragment := range testCase.wantMissing {
				assert.NotContains(t, message, fragment)
			}
			assert.NotContains(t, message, " \n", "indentation leaves no trailing spaces on blank lines")
		})
	}
}

func TestCompileAndExecuteNotesWhereAMissingPackageIsRegistered(t *testing.T) {
	t.Parallel()

	err := newTestInterpreter(t).CompileAndExecute(context.Background(), "example.com/site", map[string]map[string]string{
		"lister": {"lister.go": "package lister\n\nimport \"example.com/unregistered/widgets\"\n\nvar Count = widgets.Count\n"},
	})
	require.ErrorIs(t, err, pipit.ErrPackageNotInRegistry)
	message := err.Error()
	assert.Contains(t, message, "1. package without a symbol table")
	assert.Equal(t, 1, strings.Count(message, "hint: "), "pipit's hint is shown once and piko adds none: %s", message)
	assert.Equal(t, 1, strings.Count(message, "note: in a piko project the manifest is piko-symbols.yaml"))
}
