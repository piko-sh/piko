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

package campaign

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvedLocalOverrideAndContentIdentity(t *testing.T) {
	ctx := t.Context()
	parent := t.TempDir()
	repository := filepath.Join(parent, "app")
	dependency := filepath.Join(parent, "dep")
	for _, directory := range []string{repository, dependency} {
		require.NoError(t, os.Mkdir(directory, 0o700))
	}
	writeFixture(t, filepath.Join(repository, "go.mod"), "module example.test/app\n\ngo 1.27.0\nrequire example.test/dep v0.0.0\n")
	writeFixture(t, filepath.Join(dependency, "go.mod"), "module example.test/dep\n\ngo 1.27.0\n")
	work := filepath.Join(parent, "go.work")
	writeFixture(t, work, "go 1.27.0\n\nuse "+repository+"\nreplace example.test/dep => ./dep\n")
	t.Setenv("GOWORK", work)
	t.Setenv("PIKO_FUZZ_STATE", filepath.Join(parent, "state"))
	git := func(arguments ...string) {
		t.Helper()
		command := exec.CommandContext(ctx, "git", arguments...)
		command.Dir = dependency
		output, err := command.CombinedOutput()
		require.NoError(t, err, "git: %s", output)
	}
	git("init", "-q")
	git("add", "go.mod")
	git("-c", "user.name=Fixture", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture")
	record := func() Provenance {
		t.Helper()
		provenance, err := RecordProvenance(ctx, repository)
		require.NoError(t, err)
		require.Empty(t, provenance.ResolutionProblem)
		require.Empty(t, provenance.Sources[dependency].Problem)
		require.NotEmpty(t, provenance.Sources[dependency].Revision)
		require.Equal(t, "example.test/dep", provenance.LocalModules[dependency])
		require.Contains(t, provenance.Resolved["example.test/dep"], dependency)
		return provenance
	}
	one := record()
	writeFixture(t, filepath.Join(dependency, "extra.go"), "package dep\n")
	two := record()
	writeFixture(t, filepath.Join(dependency, "extra.go"), "package abc\n")
	three := record()
	assert.Equal(t, one.Sources[dependency].Revision, three.Sources[dependency].Revision)
	assert.NotEqual(t, one.Sources[dependency].DirtySHA256, two.Sources[dependency].DirtySHA256)
	assert.NotEqual(t, two.Sources[dependency].DirtySHA256, three.Sources[dependency].DirtySHA256)
	_, err := os.Stat(work + ".sum")
	assert.True(t, errors.Is(err, fs.ErrNotExist), "probe modified source workspace")
}

func TestSourceRevisionReportsIncompleteHash(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "main.go"), []byte("package main"), 0o600))
	budget := int64(2)
	revision := sourceRevision(t.Context(), directory, false, &budget)
	assert.NotEmpty(t, revision.Problem)
	assert.Empty(t, revision.DirtySHA256)
}

func TestProvenanceRecordsLinkedWorkerToolchain(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	var provenance Provenance
	require.NoError(t, provenance.recordWorker(executable))
	assert.Equal(t, runtime.Version(), provenance.WorkerModules[filepath.Base(executable)]["go"])
}

func TestOfflineGraphRetainsLinkedWorkerVersionAndLocalSources(t *testing.T) {
	directory := t.TempDir()
	repository := filepath.Join(directory, "app")
	dependency := filepath.Join(directory, "dep")
	optional := filepath.Join(directory, "optional")
	initialise := exec.CommandContext(t.Context(), "git", "init", "-q", directory)
	output, err := initialise.CombinedOutput()
	require.NoError(t, err, "git init: %s", output)
	writeFixture(t, filepath.Join(directory, ".gitignore"), "/app/\n/dep/\n/optional/\n")
	for _, path := range []string{repository, dependency, optional} {
		require.NoError(t, os.Mkdir(path, 0o700))
	}
	writeFixture(t, filepath.Join(dependency, "go.mod"), "module receipt.test/dep\n\ngo 1.27.0\n")
	writeFixture(t, filepath.Join(dependency, "dep.go"), "package dep\nvar ID = 1\n")
	writeFixture(t, filepath.Join(repository, "go.mod"), "module receipt.test/tool\n\ngo 1.27.0\nrequire receipt.test/dep v1.2.3\nreplace receipt.test/dep => ../dep\n")
	writeFixture(t, filepath.Join(repository, "main.go"), "package main\nimport \"receipt.test/dep\"\nfunc main(){println(dep.ID)}\n")
	binary := filepath.Join(directory, "selected-worker")
	build := exec.CommandContext(t.Context(), "go", "build", "-p=1", "-buildvcs=false", "-mod=readonly", "-o", binary, ".")
	build.Dir = repository
	build.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	output, err = build.CombinedOutput()
	require.NoError(t, err, "fixture compile: %s", output)

	writeFixture(t, filepath.Join(optional, "go.mod"), "module receipt.test/optional\n\ngo 1.27.0\nrequire missing.invalid/offline-fixture v0.1.0\n")
	workspace := filepath.Join(directory, "go.work")
	writeFixture(t, workspace, "go 1.27.0\nuse (\n./app\n./optional\n)\n")
	t.Setenv("GOWORK", workspace)
	provenance, err := RecordProvenance(t.Context(), repository)
	require.NoError(t, err)
	require.NotEmpty(t, provenance.ResolutionProblem, "did not exercise offline fallback")
	require.Equal(t, "linked-binaries", provenance.ResolutionSource)
	require.NoError(t, provenance.recordWorker(binary))
	version := provenance.WorkerModules["selected-worker"]["receipt.test/dep"]
	assert.Equal(t, "v1.2.3 => ../dep", strings.TrimSuffix(version, " (devel)"))
	assert.Equal(t, version, provenance.Resolved["receipt.test/dep"])
	assert.Equal(t, "receipt.test/dep", provenance.LocalModules[dependency])
	assert.NotEmpty(t, provenance.Sources[dependency].DirtySHA256)
	assert.Empty(t, provenance.Sources[dependency].Problem)
	assert.NotContains(t, provenance.Resolved, "missing.invalid/offline-fixture", "unlinked optional module reported as linked")
	require.NoError(t, provenance.recordWorker(binary))
	assert.Len(t, provenance.WorkerModules, 1, "recorded duplicate worker metadata")
}

func writeFixture(t *testing.T, path, contents string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
}
