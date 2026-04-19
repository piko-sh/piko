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
	"io"
	"os"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsurePikoProject(t *testing.T) {
	t.Chdir(t.TempDir())

	var stderr bytes.Buffer
	require.NotZero(t, ensurePikoProject(&stderr), "expected an empty dir to be rejected as a non-Piko project")

	makePikoProject(t)
	stderr.Reset()
	require.Zero(t, ensurePikoProject(&stderr), stderr.String())
}

func TestRunGenerate_NotAProject(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	require.Equal(t, 1, RunGenerateWithIO(nil, &stdout, &stderr), "expected exit 1 outside a Piko project")
}

func TestRunGenerate_RunsGeneratorWithMode(t *testing.T) {
	cases := []struct {
		name      string
		arguments []string
		wantMode  string
	}{
		{name: "default mode", arguments: nil, wantMode: "all"},
		{name: "explicit mode", arguments: []string{"manifest"}, wantMode: "manifest"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			makePikoProject(t)
			var calls [][]string
			stubGoRunner(t, &calls)

			var stdout, stderr bytes.Buffer
			require.Equal(t, 0, RunGenerateWithIO(testCase.arguments, &stdout, &stderr), stderr.String())
			require.Len(t, calls, 1)
			assert.Equal(t, []string{"run", "./cmd/generator", testCase.wantMode}, calls[0])
		})
	}
}

func TestRunGenerate_RejectsBadModeAndExtraArgs(t *testing.T) {
	cases := []struct {
		name      string
		arguments []string
	}{
		{name: "unknown mode", arguments: []string{"bogus"}},
		{name: "extra argument", arguments: []string{"all", "extra"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			makePikoProject(t)
			var calls [][]string
			stubGoRunner(t, &calls)

			var stdout, stderr bytes.Buffer
			assert.Equal(t, 1, RunGenerateWithIO(testCase.arguments, &stdout, &stderr))
			assert.NotEmpty(t, stderr.String(), "expected a message explaining the rejection")
			assert.Empty(t, calls, "the generator must not run when arguments are rejected")
		})
	}
}

func TestRunBuild_UnknownArg(t *testing.T) {
	t.Chdir(t.TempDir())
	makePikoProject(t)

	var stdout, stderr bytes.Buffer
	require.Equal(t, 1, RunBuildWithIO([]string{"--bogus"}, &stdout, &stderr), "expected exit 1 for an unknown argument")
}

func TestRunBuild_FlagParsing(t *testing.T) {
	t.Chdir(t.TempDir())
	makePikoProject(t)

	cases := []struct {
		name      string
		arguments []string
		wantCode  int
	}{
		{name: "equals form output rejected mode", arguments: []string{"--output=bin/x", "--mode=bogus"}, wantCode: 1},
		{name: "unknown mode", arguments: []string{"--mode", "bogus"}, wantCode: 1},
		{name: "dangling output value", arguments: []string{"--output"}, wantCode: 1},
		{name: "unknown flag", arguments: []string{"--bogus"}, wantCode: 1},
		{name: "unknown positional", arguments: []string{"positional"}, wantCode: 1},
		{name: "help exits zero", arguments: []string{"--help"}, wantCode: 0},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			var calls [][]string
			stubGoRunner(t, &calls)

			var stdout, stderr bytes.Buffer
			code := RunBuildWithIO(testCase.arguments, &stdout, &stderr)
			assert.Equal(t, testCase.wantCode, code, "stderr: %s", stderr.String())
			assert.Empty(t, calls, "no case here should reach the go toolchain")
		})
	}
}

func TestRunBuild_HelpGoesToStdout(t *testing.T) {
	t.Chdir(t.TempDir())
	var calls [][]string
	stubGoRunner(t, &calls)

	var stdout, stderr bytes.Buffer
	code := RunBuildWithIO([]string{"--help"}, &stdout, &stderr)
	assert.Equal(t, 0, code, "help must exit zero even outside a Piko project")
	assert.Contains(t, stdout.String(), "Usage: piko build", "help must be written to stdout")
	assert.Empty(t, stderr.String(), "help must not be written to stderr")
	assert.Empty(t, calls, "help must not run the go toolchain")
}

func TestRunBuild_GeneratesThenBuilds(t *testing.T) {
	cases := []struct {
		name      string
		arguments []string
		wantBuild []string
	}{
		{name: "embed by default", arguments: nil, wantBuild: []string{"build", "-tags", "piko_embed", "-o", "bin/app", "./cmd/main"}},
		{name: "no embed omits tag", arguments: []string{"--no-embed"}, wantBuild: []string{"build", "-o", "bin/app", "./cmd/main"}},
		{name: "output shorthand", arguments: []string{"-o", "custom/app"}, wantBuild: []string{"build", "-tags", "piko_embed", "-o", "custom/app", "./cmd/main"}},
		{name: "output long equals", arguments: []string{"--output=custom/app"}, wantBuild: []string{"build", "-tags", "piko_embed", "-o", "custom/app", "./cmd/main"}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			makePikoProject(t)
			var calls [][]string
			stubGoRunner(t, &calls)

			var stdout, stderr bytes.Buffer
			code := RunBuildWithIO(testCase.arguments, &stdout, &stderr)
			require.Equal(t, 0, code, stderr.String())
			require.Len(t, calls, 2, "expected the generator then the build, in that order")
			assert.Equal(t, []string{"run", "./cmd/generator", "all"}, calls[0], "the generator must run first")
			assert.Equal(t, testCase.wantBuild, calls[1], "the build must follow the generator")
		})
	}
}

func TestRunBuild_GeneratorFailureAbortsBuild(t *testing.T) {
	t.Chdir(t.TempDir())
	makePikoProject(t)

	var calls [][]string
	previous := runGoCommand
	runGoCommand = func(_ context.Context, _, _ io.Writer, args ...string) int {
		calls = append(calls, slices.Clone(args))
		return 2
	}
	t.Cleanup(func() { runGoCommand = previous })

	var stdout, stderr bytes.Buffer
	code := RunBuildWithIO(nil, &stdout, &stderr)
	assert.Equal(t, 2, code, "the generator exit code must propagate")
	require.Len(t, calls, 1, "the build must not run after a generator failure")
	assert.Equal(t, []string{"run", "./cmd/generator", "all"}, calls[0])
}

func TestRunDev_RejectsUnknownArguments(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	code := RunDevWithIO([]string{"--anything"}, &stdout, &stderr)
	assert.Equal(t, 1, code, "expected exit 1 for an unknown dev argument")
	assert.NotEmpty(t, stderr.String(), "expected an error message naming the unknown argument")
}

func TestRunDevInterpreted_RejectsUnknownArguments(t *testing.T) {
	t.Chdir(t.TempDir())

	var stdout, stderr bytes.Buffer
	code := RunDevInterpretedWithIO([]string{"--anything"}, &stdout, &stderr)
	assert.Equal(t, 1, code, "expected exit 1 for an unknown dev-i argument")
	assert.Contains(t, stderr.String(), "piko dev-i", "expected the error message to name the dev-i command")
}

func TestRunDev_GeneratesThenRuns(t *testing.T) {
	cases := []struct {
		name     string
		run      func([]string, io.Writer, io.Writer) int
		wantMode string
	}{
		{name: "dev", run: RunDevWithIO, wantMode: "dev"},
		{name: "dev-i", run: RunDevInterpretedWithIO, wantMode: "dev-i"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			makePikoProject(t)
			var calls [][]string
			stubGoRunner(t, &calls)

			var stdout, stderr bytes.Buffer
			code := testCase.run(nil, &stdout, &stderr)
			require.Equal(t, 0, code, stderr.String())
			require.Len(t, calls, 2, "expected the generator then the dev server")
			assert.Equal(t, []string{"run", "./cmd/generator", "all"}, calls[0])
			assert.Equal(t, []string{"run", "./cmd/main", testCase.wantMode}, calls[1])
		})
	}
}

func makePikoProject(t *testing.T) {
	t.Helper()
	for _, directory := range requiredProjectDirs {
		require.NoError(t, os.MkdirAll(directory, 0o755))
	}
}

func stubGoRunner(t *testing.T, calls *[][]string) {
	t.Helper()
	previous := runGoCommand
	runGoCommand = func(_ context.Context, _, _ io.Writer, args ...string) int {
		*calls = append(*calls, slices.Clone(args))
		return 0
	}
	t.Cleanup(func() { runGoCommand = previous })
}
