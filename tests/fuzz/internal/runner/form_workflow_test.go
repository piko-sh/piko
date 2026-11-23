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

package runner

import (
	"crypto/sha256"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"

	"piko.sh/piko/tests/fuzz/internal/casefile"
)

func TestWorkflowDependenciesFollowEngineAndProviderRequirements(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "piko")
	for path, source := range map[string]string{
		"piko/wdk/db/db_engine_sqlite/go.mod":    "module piko.sh/piko/wdk/db/db_engine_sqlite\nrequire seagull.sh/seagull v0.7.0 // indirect\n",
		"piko/wdk/db/db_provider_seagull/go.mod": "module piko.sh/piko/wdk/db/db_provider_seagull\nrequire (\n seagull.sh/seagull v0.6.0\n modernc.org/sqlite v1.77.0\n)\n",
	} {
		path = filepath.Join(root, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
		require.NoError(t, os.WriteFile(path, []byte(source), 0600))
	}
	app, err := modfile.Parse("go.mod", []byte("module example.org/pikofuzzcase\n"), nil)
	require.NoError(t, err)
	require.NoError(t, stageWorkflowModules(repo, app))
	required := map[string]string{}
	for _, requirement := range app.Require {
		required[requirement.Mod.Path] = requirement.Mod.Version
	}
	assert.Equal(t, "v1.77.0", required["modernc.org/sqlite"], "staged SQLite requirement")
	assert.Equal(t, "v0.7.0", required["seagull.sh/seagull"], "staged Seagull requirement follows the highest pin")
	assert.Empty(t, app.Replace, "Seagull must come from its published release")
}

func TestFormWorkflowPrivateWorkspaceUsesPinnedSeagull(t *testing.T) {
	repo, err := filepath.Abs("../../../..")
	require.NoError(t, err)
	pins, err := workflowPins(repo)
	require.NoError(t, err)
	original, err := os.ReadFile(filepath.Join(repo, "go.work"))
	require.NoError(t, err)
	app, workspaceFile, err := stage(repo, t.TempDir(), &casefile.Case{Target: "form-workflow"})
	require.NoError(t, err)
	command := exec.CommandContext(t.Context(), "go", "list", "-mod=readonly", "-m", "-f", "{{.Version}} {{.Replace}}", "seagull.sh/seagull")
	command.Dir = app
	command.Env = append(os.Environ(), "GOWORK="+workspaceFile, "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "private module selection: %s", output)
	require.Equal(t, pins[seagullModule]+" <nil>", strings.TrimSpace(string(output)), "private module selection")
	require.NoError(t, prepareWorkflowModule(t.Context(), app, []string{"GOWORK=" + workspaceFile, "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local"}))
	standalone := exec.CommandContext(t.Context(), "go", "list", "-mod=readonly", "-deps",
		"piko.sh/piko/wdk/db/db_engine_sqlite", "piko.sh/piko/wdk/markdown/markdown_provider_goldmark", "modernc.org/sqlite")
	standalone.Dir = app
	standalone.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local")
	output, err = standalone.CombinedOutput()
	require.NoError(t, err, "compiled-host inspector dependency graph: %s", output)
	after, err := os.ReadFile(filepath.Join(repo, "go.work"))
	require.NoError(t, err)
	require.Equal(t, original, after, "repository workspace changed")
	for name, source := range map[string][]byte{"generator": []byte(workflowSQLGenerator), "host": []byte(workflowDBHost), "interp": []byte(workflowRegistry)} {
		_, err := parser.ParseFile(token.NewFileSet(), name+".go", source, parser.AllErrors)
		require.NoError(t, err, name)
	}
}

func TestWorkflowHostFixtureEmissionBaseline(t *testing.T) {
	for name, fixture := range map[string]struct{ source, digest string }{
		"workflow-sql": {source: workflowSQLGenerator, digest: "64708a78f7582e97ff5ebe0712b11b88378e9ab66d77c4b22fb19127bcd7a712"},
		"workflow-db":  {source: workflowDBHost, digest: "0aea73a328fc7350e3d8be8bf8a981f0d3290cec080398ab61e6b84aa3837a3b"},
	} {
		assert.Equal(t, fixture.digest, fmt.Sprintf("%x", sha256.Sum256([]byte(fixture.source))), name)
	}
}
