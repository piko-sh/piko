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

package main

import (
	"io/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko"
)

func TestParseArgumentsSelectsWork(t *testing.T) {
	for _, testCase := range []struct {
		name      string
		arguments []string
		want      invocation
	}{
		{name: "no arguments", arguments: nil, want: invocation{mode: piko.GenerateModeAll, componentsOnly: false}},
		{name: "manifest", arguments: []string{flagManifest}, want: invocation{mode: piko.GenerateModeManifest, componentsOnly: false}},
		{name: "components only", arguments: []string{flagCompileComponents}, want: invocation{mode: piko.GenerateModeAll, componentsOnly: true}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := parseArguments(testCase.arguments)
			require.NoError(t, err)
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestParseArgumentsRejectsUnknownArguments(t *testing.T) {
	for _, arguments := range [][]string{
		{"--manfest"},
		{"manifest"},
		{flagManifest, flagCompileComponents},
	} {
		_, err := parseArguments(arguments)
		require.ErrorIs(t, err, errUsage, "%q", arguments)
	}
}

func TestRunRejectsUnknownArgumentBeforeWork(t *testing.T) {
	t.Setenv("PIKO_FUZZ_TAG", "no-such-component")

	err := run(t.Context(), []string{"--unknown"})

	require.ErrorIs(t, err, errUsage)
}

func TestHostOptionsFollowEnvironment(t *testing.T) {
	t.Setenv("PIKO_FUZZ_SHIPPED", "")
	t.Setenv("PIKO_FUZZ_PARTIAL_COMPONENTS", "")
	t.Setenv("PIKO_FUZZ_COLLECTIONS", "")
	base, err := hostOptions()
	require.NoError(t, err)

	t.Setenv("PIKO_FUZZ_PARTIAL_COMPONENTS", "true")
	t.Setenv("PIKO_FUZZ_COLLECTIONS", "true")
	extended, err := hostOptions()
	require.NoError(t, err)
	assert.Len(t, extended, len(base)+2)

	t.Setenv("PIKO_FUZZ_SHIPPED", "true")
	shipped, err := hostOptions()
	require.NoError(t, err)
	assert.Greater(t, len(shipped), len(extended))
}

func TestRunStopsAfterComponentCheckWhenNoTagSelected(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PIKO_FUZZ_TAG", "")

	require.NoError(t, run(t.Context(), []string{flagCompileComponents}))

	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	assert.Empty(t, entries, "component check generated output")
}

func TestRunReportsComponentCompileFailure(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PIKO_FUZZ_TAG", "missing-card")

	err := run(t.Context(), []string{flagCompileComponents})

	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.ErrorContains(t, err, "pkc-compile")
}

func TestRunGeneratesManifestForEmptyProject(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PIKO_FUZZ_TAG", "")
	t.Setenv("PIKO_FUZZ_SHIPPED", "")
	t.Setenv("PIKO_FUZZ_PARTIAL_COMPONENTS", "")
	t.Setenv("PIKO_FUZZ_COLLECTIONS", "")
	require.NoError(t, os.Mkdir("pages", 0o700))
	require.NoError(t, os.WriteFile("go.mod", []byte("module example.test/site\n\ngo 1.25\n"), 0o600))

	assert.NoError(t, run(t.Context(), []string{flagManifest}))
}

func TestRunReportsGenerationFailureWithoutPagesDirectory(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("PIKO_FUZZ_TAG", "")
	t.Setenv("PIKO_FUZZ_SHIPPED", "")
	t.Setenv("PIKO_FUZZ_PARTIAL_COMPONENTS", "")
	t.Setenv("PIKO_FUZZ_COLLECTIONS", "")

	err := run(t.Context(), nil)

	assert.ErrorContains(t, err, "pages source directory")
}
