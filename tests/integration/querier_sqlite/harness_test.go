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

package querier_sqlite_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/emitter_go"
	"piko.sh/piko/wdk/db/db_engine_sqlite"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

const runnerModuleName = "querier_test_runner"

type testSpec struct {
	Description     string                             `json:"description"`
	CustomFunctions []querier_dto.CustomFunctionConfig `json:"customFunctions,omitempty"`
	TypeOverrides   []querier_dto.TypeOverride         `json:"typeOverrides,omitempty"`
}

type realFileReader struct{}

func (*realFileReader) ReadFile(_ context.Context, filePath string) ([]byte, error) {
	return os.ReadFile(filePath)
}

func (*realFileReader) ReadDir(_ context.Context, directory string) ([]os.DirEntry, error) {
	return os.ReadDir(directory)
}

func runTestCase(t *testing.T, testCaseDirectory string) {
	t.Helper()

	spec := loadTestSpec(t, testCaseDirectory)
	t.Log(spec.Description)

	generatedFiles := generateCode(t, testCaseDirectory, spec)
	tempDirectory := t.TempDir()

	writeGeneratedFiles(t, tempDirectory, generatedFiles)
	copyRunnerSource(t, testCaseDirectory, tempDirectory)
	writeRunnerGoMod(t, tempDirectory, generatedFiles)
	tidyModules(t, tempDirectory)
	buildRunner(t, tempDirectory)

	output := executeRunner(t, tempDirectory)
	goldenPath := filepath.Join(testCaseDirectory, "golden", "output.json")
	assertGoldenJSON(t, goldenPath, output)
}

func generateCode(t *testing.T, testCaseDirectory string, spec testSpec) []querier_dto.GeneratedFile {
	t.Helper()
	ctx := context.Background()

	engine := db_engine_sqlite.NewSQLiteEngine()
	emitter := emitter_go.NewGoEmitter()

	migrationDirectory, err := filepath.Abs(filepath.Join(testCaseDirectory, "migrations"))
	require.NoError(t, err)
	queryDirectory, err := filepath.Abs(filepath.Join(testCaseDirectory, "queries"))
	require.NoError(t, err)

	service, serviceError := querier_domain.NewQuerierService(querier_domain.QuerierPorts{
		Engine:     engine,
		Emitter:    emitter,
		FileReader: &realFileReader{},
	})
	require.NoError(t, serviceError)

	databaseConfig := &querier_dto.DatabaseConfig{
		MigrationDirectory: migrationDirectory,
		QueryDirectory:     queryDirectory,
		CustomFunctions:    spec.CustomFunctions,
		TypeOverrides:      spec.TypeOverrides,
	}

	result, generateError := service.GenerateDatabase(ctx, "db", databaseConfig, "")
	require.NoError(t, generateError)
	require.NotNil(t, result)

	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Severity == querier_dto.SeverityError {
			t.Fatalf("generation produced error diagnostic: %s", diagnostic.Message)
		}
	}

	require.NotEmpty(t, result.Files, "expected generated files")
	return result.Files
}

func writeGeneratedFiles(t *testing.T, tempDirectory string, files []querier_dto.GeneratedFile) {
	t.Helper()

	databasePackageDirectory := filepath.Join(tempDirectory, "db")
	require.NoError(t, os.MkdirAll(databasePackageDirectory, 0o755))

	for _, file := range files {
		filePath := filepath.Join(databasePackageDirectory, file.Name)
		require.NoError(t, os.WriteFile(filePath, file.Content, 0o644))
	}
}

func copyRunnerSource(t *testing.T, testCaseDirectory string, tempDirectory string) {
	t.Helper()

	sourcePath := filepath.Join(testCaseDirectory, "runner.go")
	content, err := os.ReadFile(sourcePath)
	require.NoError(t, err, "reading runner.go")

	destinationPath := filepath.Join(tempDirectory, "main.go")
	require.NoError(t, os.WriteFile(destinationPath, content, 0o644))
}

func writeRunnerGoMod(t *testing.T, tempDirectory string, files []querier_dto.GeneratedFile) {
	t.Helper()

	goModContent := "module " + runnerModuleName + "\n\ngo 1.26.1\n\nrequire modernc.org/sqlite v1.42.2\n"

	if generatedCodeImportsPiko(files) {
		projectRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
		require.NoError(t, err)
		goModContent += "\nrequire piko.sh/piko v0.0.0\n\nreplace piko.sh/piko => " + projectRoot + "\n"
	}

	require.NoError(t, os.WriteFile(filepath.Join(tempDirectory, "go.mod"), []byte(goModContent), 0o644))
}

func generatedCodeImportsPiko(files []querier_dto.GeneratedFile) bool {
	for _, file := range files {
		if bytes.Contains(file.Content, []byte("piko.sh/piko/")) {
			return true
		}
	}
	return false
}

func tidyModules(t *testing.T, tempDirectory string) {
	t.Helper()

	command := exec.Command("go", "mod", "tidy")
	command.Dir = tempDirectory
	command.Env = append(os.Environ(), "GOWORK=off")

	output, err := command.CombinedOutput()
	require.NoError(t, err, "go mod tidy failed:\n%s", string(output))
}

func buildRunner(t *testing.T, tempDirectory string) {
	t.Helper()

	binaryName := "runner"
	if runtime.GOOS == "windows" {
		binaryName = "runner.exe"
	}

	command := exec.Command("go", "build", "-o", binaryName, ".")
	command.Dir = tempDirectory
	command.Env = append(os.Environ(), "GOWORK=off")

	output, err := command.CombinedOutput()
	require.NoError(t, err, "go build failed:\n%s", string(output))
}

func executeRunner(t *testing.T, tempDirectory string) []byte {
	t.Helper()

	binaryName := "runner"
	if runtime.GOOS == "windows" {
		binaryName = "runner.exe"
	}

	binaryPath := filepath.Join(tempDirectory, binaryName)
	command := exec.Command(binaryPath)
	command.Dir = tempDirectory

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	require.NoError(t, err, "runner failed:\nstderr: %s\nstdout: %s", stderr.String(), stdout.String())

	return stdout.Bytes()
}

func loadTestSpec(t *testing.T, testCaseDirectory string) testSpec {
	t.Helper()

	specPath := filepath.Join(testCaseDirectory, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "reading testspec.json")

	var spec testSpec
	require.NoError(t, json.Unmarshal(specBytes, &spec), "parsing testspec.json")
	return spec
}

func assertGoldenJSON(t *testing.T, goldenPath string, actual []byte) {
	t.Helper()

	prettyActual := prettyPrintJSON(t, actual)

	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
		require.NoError(t, os.WriteFile(goldenPath, prettyActual, 0o644))
		t.Logf("updated golden file: %s", goldenPath)
		return
	}

	expectedBytes, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("golden file not found at %s (run with -update to generate)", goldenPath)
	}
	require.NoError(t, err)

	assert.JSONEq(t, string(expectedBytes), string(prettyActual), "golden file mismatch: %s", goldenPath)
}

func prettyPrintJSON(t *testing.T, data []byte) []byte {
	t.Helper()

	var parsed any
	require.NoError(t, json.Unmarshal(data, &parsed), "invalid JSON output from runner")

	pretty, err := json.MarshalIndent(parsed, "", "  ")
	require.NoError(t, err)

	return append(pretty, '\n')
}
