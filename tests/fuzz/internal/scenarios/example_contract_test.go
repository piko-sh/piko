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

package scenarios

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
)

func TestPrivateExampleListenerPreservesAuthoredOptions(t *testing.T) {
	for _, name := range publiccontracts.LocalExamples {
		t.Run(name, func(t *testing.T) {
			from := filepath.Join("../../../..", "examples", "scenarios", name, "src", "cmd", "main", "main.go")
			original, err := os.ReadFile(from)
			require.NoError(t, err)
			directory := t.TempDir()
			mainFile := filepath.Join(directory, "cmd", "main", "main.go")
			require.NoError(t, os.MkdirAll(filepath.Dir(mainFile), 0o700))
			require.NoError(t, os.WriteFile(mainFile, original, 0o600))
			require.NoError(t, configureExampleListener(directory))
			local, err := os.ReadFile(mainFile)
			require.NoError(t, err)
			assert.Equal(t, string(original), strings.Replace(string(local), "newLocalExample(", "piko.New(", 1), "changed authored setup")
			helper, err := os.ReadFile(filepath.Join(filepath.Dir(mainFile), "piko_fuzz_listener.go"))
			require.NoError(t, err)
			for _, want := range []string{"piko.WithBindAddress(\"127.0.0.1\")", "piko.WithHealthEnabled(false)", "piko.WithAutoNextPort(false)", "piko.WithPort(port)"} {
				assert.Contains(t, string(helper), want)
			}
		})
	}
}

func TestConfigureExampleListenerRequiresOneConstructor(t *testing.T) {
	directory := t.TempDir()
	mainFile := filepath.Join(directory, "cmd", "main", "main.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(mainFile), 0o700))
	require.NoError(t, os.WriteFile(mainFile, []byte("package main\nfunc main(){}\n"), 0o600))
	require.ErrorIs(t, configureExampleListener(directory), errExampleEntryPoint)
}

func TestExampleWorkspaceRebasesLocalReplacements(t *testing.T) {
	repository := t.TempDir()
	source := filepath.Join(repository, "examples", "demo", "src")
	app := t.TempDir()
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(app, "go.mod"), []byte("module example.org/demo\n\ngo 1.27\n\nreplace piko.sh/piko => ../../..\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(repository, "go.work"), []byte("go 1.27\n\nuse ./sub\n"), 0o600))
	t.Setenv("GOWORK", "")
	workspace, err := exampleWorkspace(repository, source, app, work)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(work, "go.work"), workspace)
	rewritten, err := os.ReadFile(filepath.Join(app, "go.mod"))
	require.NoError(t, err)
	assert.Contains(t, string(rewritten), "=> "+repository)
	written, err := os.ReadFile(workspace)
	require.NoError(t, err)
	assert.Contains(t, string(written), app)
	assert.Contains(t, string(written), filepath.Join(repository, "sub"))
	assert.NoFileExists(t, workspace+".sum")
}

func TestExampleReceiptKeepsSuiteFailureWithUndecodableCounts(t *testing.T) {
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, "example-counts.json"), []byte("{"), 0o600))
	suiteFailure := errors.New("browser suite failed")

	_, err := exampleReceipt(work, "counter", publiccontracts.Receipt{}, suiteFailure)

	require.ErrorIs(t, err, suiteFailure)
	assert.ErrorContains(t, err, "decode assertion counts")
}

func TestExampleReceiptMergesCounts(t *testing.T) {
	work := t.TempDir()
	counts := `{"Steps":2,"Assertions":3,"HTTPStatusVerified":true,"BrowserVisited":true}`
	require.NoError(t, os.WriteFile(filepath.Join(work, "example-counts.json"), []byte(counts), 0o600))

	receipt, err := exampleReceipt(work, "counter", publiccontracts.Receipt{}, nil)

	require.NoError(t, err)
	assert.Equal(t, 3, receipt.Assertions)
	assert.True(t, receipt.BrowserVisited)
}

func TestExampleReceiptRequiresPositiveAssertions(t *testing.T) {
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, "example-counts.json"), []byte(`{}`), 0o600))

	_, err := exampleReceipt(work, "counter", publiccontracts.Receipt{}, nil)

	require.ErrorIs(t, err, errExampleUnverified)
}

func TestExampleReceiptReportsMissingCounts(t *testing.T) {
	_, err := exampleReceipt(t.TempDir(), "counter", publiccontracts.Receipt{}, nil)

	require.ErrorContains(t, err, "read assertion counts")
}

func TestStageExampleCopiesAuthoredSourcesAndSpecification(t *testing.T) {
	example := t.TempDir()
	source := filepath.Join(example, "src")
	writeExampleFile(t, filepath.Join(example, exampleSpecFile), `{"steps":[]}`)
	writeExampleFile(t, filepath.Join(source, "go.mod"), "module example.org/demo\n")
	writeExampleFile(t, filepath.Join(source, "cmd", "main", "main.go"), "package main\n")
	writeExampleFile(t, filepath.Join(source, "go.work"), "go 1.27\n")
	writeExampleFile(t, filepath.Join(source, "go.work.sum"), "")
	for _, generated := range []string{"dist", ".piko", "bin", "node_modules", ".git"} {
		writeExampleFile(t, filepath.Join(source, generated, "ignored.txt"), generated)
	}
	stage := func() (string, string, string) {
		t.Helper()
		work := t.TempDir()
		app := filepath.Join(work, "src")
		digest, err := stageExample(source, app, work, "demo")
		require.NoError(t, err)
		return digest, app, work
	}

	digest, app, work := stage()

	assert.Len(t, digest, 64)
	assert.FileExists(t, filepath.Join(app, "go.mod"))
	assert.FileExists(t, filepath.Join(app, "cmd", "main", "main.go"))
	for _, skipped := range []string{"go.work", "go.work.sum", "dist", ".piko", "bin", "node_modules", ".git"} {
		assert.NoFileExists(t, filepath.Join(app, skipped))
		assert.NoDirExists(t, filepath.Join(app, skipped))
	}
	spec, err := os.ReadFile(filepath.Join(work, exampleSpecFile))
	require.NoError(t, err)
	assert.JSONEq(t, `{"steps":[]}`, string(spec))
	again, _, _ := stage()
	assert.Equal(t, digest, again, "staging the same sources changed the digest")
	writeExampleFile(t, filepath.Join(source, "cmd", "main", "main.go"), "package main\n\nfunc main() {}\n")
	changed, _, _ := stage()
	assert.NotEqual(t, digest, changed, "an authored edit kept the digest")
	writeExampleFile(t, filepath.Join(example, exampleSpecFile), `{"steps":[1]}`)
	respecified, _, _ := stage()
	assert.NotEqual(t, changed, respecified, "a specification edit kept the digest")
}

func TestStageExampleRejectsAuthoredSymlink(t *testing.T) {
	example := t.TempDir()
	source := filepath.Join(example, "src")
	writeExampleFile(t, filepath.Join(example, exampleSpecFile), "{}")
	writeExampleFile(t, filepath.Join(source, "target.txt"), "target")
	require.NoError(t, os.Symlink("target.txt", filepath.Join(source, "link.txt")))
	work := t.TempDir()

	_, err := stageExample(source, filepath.Join(work, "src"), work, "demo")

	require.ErrorIs(t, err, errExampleSymlink)
	assert.ErrorContains(t, err, "example/demo: stage sources")
}

func TestStageExampleRejectsTooManyFiles(t *testing.T) {
	example := t.TempDir()
	source := filepath.Join(example, "src")
	writeExampleFile(t, filepath.Join(example, exampleSpecFile), "{}")
	for index := range maxExampleFiles + 1 {
		writeExampleFile(t, filepath.Join(source, "file-"+strconv.Itoa(index)+".txt"), "x")
	}
	work := t.TempDir()

	_, err := stageExample(source, filepath.Join(work, "src"), work, "demo")

	require.ErrorIs(t, err, errExampleTooLarge)
}

func TestStageExampleRequiresTestSpecification(t *testing.T) {
	source := filepath.Join(t.TempDir(), "src")
	writeExampleFile(t, filepath.Join(source, "go.mod"), "module example.org/demo\n")
	work := t.TempDir()

	_, err := stageExample(source, filepath.Join(work, "src"), work, "demo")

	require.ErrorContains(t, err, "example/demo: read test specification")
}

func TestStageExampleReportsMissingSources(t *testing.T) {
	work := t.TempDir()

	_, err := stageExample(filepath.Join(t.TempDir(), "missing"), filepath.Join(work, "src"), work, "demo")

	require.ErrorContains(t, err, "example/demo: stage sources")
}

func TestRunPublicExampleKeepsTheSourceDigestWhenTheBuildFails(t *testing.T) {
	repository := t.TempDir()
	name := publiccontracts.LocalExamples[1]
	example := filepath.Join(repository, "examples", "scenarios", name)
	writeExampleFile(t, filepath.Join(example, exampleSpecFile), "{}")
	writeExampleFile(t, filepath.Join(example, "src", "cmd", "main", "main.go"), "package main\n")

	receipt, err := RunPublic(t.Context(), repository, t.TempDir(), &casefile.Case{Target: "example-contracts", Index: 1})

	require.ErrorContains(t, err, "example/"+name+": prepare workspace")
	assert.Equal(t, "example/"+name, receipt.Contract)
	assert.Len(t, receipt.SourceSHA256, 64)
	assert.Zero(t, receipt.Assertions)
}

func TestRunPublicExampleReportsAStagingFailure(t *testing.T) {
	name := publiccontracts.LocalExamples[0]

	receipt, err := RunPublic(t.Context(), t.TempDir(), t.TempDir(), &casefile.Case{Target: "example-contracts", Index: 0})

	require.ErrorContains(t, err, "example/"+name+": stage sources")
	assert.Equal(t, "example/"+name, receipt.Contract)
	assert.Empty(t, receipt.SourceSHA256)
}

func TestBuildExampleReportsGenerationAndEntryPointFailures(t *testing.T) {
	t.Setenv("GOWORK", "")
	t.Setenv("GOFLAGS", "")
	testCases := []struct {
		name      string
		generator bool
		message   string
	}{
		{name: "missing generator", generator: false, message: "example/demo/run"},
		{name: "unsupported entry point", generator: true, message: errExampleEntryPoint.Error()},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repository := t.TempDir()
			writeExampleFile(t, filepath.Join(repository, "go.work"), "go 1.27\n")
			source := filepath.Join(repository, "examples", "demo", "src")
			app, work := t.TempDir(), t.TempDir()
			writeExampleFile(t, filepath.Join(app, "go.mod"), "module example.org/demo\n\ngo 1.27\n")
			writeExampleFile(t, filepath.Join(app, "cmd", "main", "main.go"), "package main\n\nfunc main() {}\n")
			if testCase.generator {
				writeExampleFile(t, filepath.Join(app, "cmd", "generator", "main.go"), "package main\n\nfunc main() {}\n")
			}

			err := buildExample(t.Context(), repository, source, app, work, "demo")

			require.ErrorContains(t, err, testCase.message)
		})
	}
}

func TestExampleWorkspaceCopiesWorkspaceSums(t *testing.T) {
	repository := t.TempDir()
	app, work := t.TempDir(), t.TempDir()
	writeExampleFile(t, filepath.Join(app, "go.mod"), "module example.org/demo\n\ngo 1.27\n")
	writeExampleFile(t, filepath.Join(repository, "go.work"), "go 1.27\n\nuse ./sub\n\nreplace example.org/local => ./local\n")
	writeExampleFile(t, filepath.Join(repository, "go.work.sum"), "example.org/x v1.0.0 h1:abc=\n")
	t.Setenv("GOWORK", "")

	workspace, err := exampleWorkspace(repository, filepath.Join(repository, "src"), app, work)

	require.NoError(t, err)
	sums, err := os.ReadFile(workspace + ".sum")
	require.NoError(t, err)
	assert.Equal(t, "example.org/x v1.0.0 h1:abc=\n", string(sums))
	written, err := os.ReadFile(workspace)
	require.NoError(t, err)
	assert.Contains(t, string(written), filepath.Join(repository, "local"))
}

func TestExampleWorkspaceRejectsInvalidModuleFiles(t *testing.T) {
	testCases := []struct {
		name      string
		module    string
		workspace string
		message   string
	}{
		{name: "missing go.mod", module: "", workspace: "go 1.27\n", message: "read go.mod"},
		{name: "invalid go.mod", module: "module\n", workspace: "go 1.27\n", message: "parse go.mod"},
		{name: "missing workspace", module: "module example.org/demo\n", workspace: "", message: "read workspace"},
		{name: "invalid workspace", module: "module example.org/demo\n", workspace: "use (\n", message: "parse workspace"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repository := t.TempDir()
			app := t.TempDir()
			if testCase.module != "" {
				writeExampleFile(t, filepath.Join(app, "go.mod"), testCase.module)
			}
			if testCase.workspace != "" {
				writeExampleFile(t, filepath.Join(repository, "go.work"), testCase.workspace)
			}
			t.Setenv("GOWORK", "")

			_, err := exampleWorkspace(repository, filepath.Join(repository, "src"), app, t.TempDir())

			require.ErrorContains(t, err, testCase.message)
		})
	}
}

func TestResolvePathKeepsAbsolutePaths(t *testing.T) {
	base := t.TempDir()
	absolute := filepath.Join(t.TempDir(), "module")

	assert.Equal(t, absolute, resolvePath(base, absolute))
	assert.Equal(t, filepath.Join(base, "module"), resolvePath(base, "module"))
}

func writeExampleFile(t *testing.T, name, contents string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o700))
	require.NoError(t, os.WriteFile(name, []byte(contents), 0o600))
}
