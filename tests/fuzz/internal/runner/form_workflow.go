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
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
	"piko.sh/piko/tests/fuzz/internal/fixturego"
)

const (
	// seagullModule is the Seagull module path.
	seagullModule = "seagull.sh/seagull"

	// sqliteModule is the module path of the pure-Go SQLite driver.
	sqliteModule = "modernc.org/sqlite"

	// workflowRegistry registers the generated database package with the interpreted host.
	workflowRegistry = `package main
import (
	"reflect"
	_ "example.org/pikofuzzcase/dist"
	"example.org/pikofuzzcase/pkg/workflowdata"
)
func init() {
	extraSymbols=map[string]map[string]reflect.Value{
		"example.org/pikofuzzcase/pkg/workflowdata": {
			"Task":reflect.ValueOf((*workflowdata.Task)(nil)),
			"List":reflect.ValueOf(workflowdata.List),
		},
	}
}
`
)

var (
	// workflowSQLGenerator is the formatted Go source staged in the generated project.
	workflowSQLGenerator = fixturego.Fixed("hosts/sql/main.go")

	// workflowDBHost is the formatted Go source staged in the generated project.
	workflowDBHost = fixturego.Fixed("hosts/db/database.go")
)

// workflowRequirement is one module requirement added to the workflow application.
type workflowRequirement struct {
	// path is the required module path.
	path string

	// version is the required module version.
	version string
}

// stageWorkflowModules adds the workflow's required modules to its application module.
//
// Seagull and SQLite are required at the versions Piko's SQLite engine and Seagull
// provider pin, so the workflow builds against the releases Piko ships with.
//
// Takes repo (string) which is the repository source directory.
// Takes appMod (*modfile.File) which is the private application's module definition.
//
// Returns error which reports an unreadable pin or a module edit failure, or nil on
// success.
func stageWorkflowModules(repo string, appMod *modfile.File) error {
	pins, err := workflowPins(repo)
	if err != nil {
		return err
	}
	for _, dependency := range []string{seagullModule, sqliteModule} {
		if pins[dependency] == "" {
			return fmt.Errorf("form-workflow: SQLite engine and Seagull provider have no requirement for %s", dependency)
		}
	}
	requirements := []workflowRequirement{
		{path: "piko.sh/piko/wdk/db/db_engine_sqlite", version: localModuleVersion},
		{path: "piko.sh/piko/wdk/markdown/markdown_provider_goldmark", version: localModuleVersion},
		{path: seagullModule, version: pins[seagullModule]},
		{path: sqliteModule, version: pins[sqliteModule]},
	}
	for _, requirement := range requirements {
		if err := appMod.AddRequire(requirement.path, requirement.version); err != nil {
			return err
		}
	}
	return nil
}

// workflowPins resolves the Seagull and SQLite versions the engine and provider require.
//
// Both manifests can constrain the same dependency, so Go's highest required version rule
// applies when the engine's and provider's pins differ.
//
// Takes repo (string) which is the repository source directory.
//
// Returns map[string]string which maps each pinned module to its highest version.
// Returns error which reports an unreadable or invalid manifest.
func workflowPins(repo string) (map[string]string, error) {
	pins := map[string]string{seagullModule: "", sqliteModule: ""}
	for _, name := range []string{"db_engine_sqlite", "db_provider_seagull"} {
		relative := filepath.Join("wdk", "db", name, "go.mod")
		path := filepath.Join(repo, relative)
		source, err := readBoundedFile(repo, relative, stagedFileMax)
		if err != nil {
			return nil, err
		}
		module, err := modfile.ParseLax(path, source, nil)
		if err != nil {
			return nil, err
		}
		for _, requirement := range module.Require {
			if previous, wanted := pins[requirement.Mod.Path]; wanted && semver.Compare(requirement.Mod.Version, previous) > 0 {
				pins[requirement.Mod.Path] = requirement.Mod.Version
			}
		}
	}
	return pins, nil
}

// prepareWorkflowModule generates the workflow's private database package.
//
// Takes app (string) which is the private generated application directory.
// Takes env ([]string) which supplies the private process environment.
//
// Returns error which reports private-module setup or SQL-generation failure, or nil on
// success.
func prepareWorkflowModule(ctx context.Context, app string, env []string) error {
	_, err := command(ctx, app, append(env, "GOWORK=off"), "workflow-module", "go", "list", "-mod=mod", "-deps",
		"piko.sh/piko/wdk/db/db_engine_sqlite", "piko.sh/piko/wdk/markdown/markdown_provider_goldmark", sqliteModule)
	return err
}

// generateWorkflowSQL runs the SQL generator for the private workflow application.
//
// The generator binary is cached in the workers directory and installed atomically, so a
// concurrent or interrupted build never leaves a partial executable at the cached path.
//
// Takes app (string) which is the private generated application directory.
// Takes work (string) which is the invocation-private working directory.
// Takes workers (string) which is the invocation-private worker executable directory.
// Takes env ([]string) which supplies the private process environment.
//
// Returns error which reports private-module setup or SQL-generation failure, or nil on
// success.
func generateWorkflowSQL(ctx context.Context, app, work, workers string, env []string) error {
	if err := prepareWorkflowModule(ctx, app, env); err != nil {
		return err
	}
	source := filepath.Join(work, "workflow-sql.go")
	if err := os.WriteFile(source, []byte(workflowSQLGenerator), privateFileMode); err != nil {
		return err
	}
	worker, err := buildCachedBinary(ctx, app, env, "workflow-sql-build", filepath.Join(workers, "form-sql-worker"), source)
	if err != nil {
		return err
	}
	_, err = command(ctx, app, env, "workflow-sql", worker)
	return err
}
