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

//go:build integration

package querier_clickhouse_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_adapters/emitter_go"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/db/db_engine_clickhouse"
)

const runnerModuleName = "querier_clickhouse_runner"

type testSpec struct {
	Description     string                             `json:"description"`
	Skip            bool                               `json:"skip,omitempty"`
	SkipReason      string                             `json:"skipReason,omitempty"`
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

	if spec.Skip {
		reason := spec.SkipReason
		if reason == "" {
			reason = "fixture marked skip in testspec.json"
		}
		t.Skip(reason)
	}

	databaseName := createIsolatedDatabase(t)
	defer dropDatabase(t, databaseName)

	dsn := dsnForDatabase(databaseName)

	applyMigrations(t, testCaseDirectory, dsn)
	generatedFiles := generateCode(t, testCaseDirectory, spec)

	tempDirectory := t.TempDir()
	writeGeneratedFiles(t, tempDirectory, generatedFiles)
	copyRunnerSource(t, testCaseDirectory, tempDirectory)
	writeRunnerGoMod(t, tempDirectory)
	tidyModules(t, tempDirectory)
	buildRunner(t, tempDirectory)

	output := executeRunner(t, tempDirectory, dsn)
	goldenPath := filepath.Join(testCaseDirectory, "golden", "output.json")
	assertGoldenJSON(t, goldenPath, output)
}

func loadTestSpec(t *testing.T, testCaseDirectory string) testSpec {
	t.Helper()
	specPath := filepath.Join(testCaseDirectory, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err)
	var spec testSpec
	require.NoError(t, json.Unmarshal(specBytes, &spec))
	return spec
}

func createIsolatedDatabase(t *testing.T) string {
	t.Helper()
	suffix := randomSuffix(8)
	databaseName := "piko_test_" + suffix

	conn, err := sql.Open("clickhouse", testConnectionString)
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.ExecContext(context.Background(), "CREATE DATABASE "+databaseName)
	require.NoError(t, err)
	return databaseName
}

func dropDatabase(t *testing.T, databaseName string) {
	t.Helper()
	conn, err := sql.Open("clickhouse", testConnectionString)
	if err != nil {
		return
	}
	defer conn.Close()
	_, _ = conn.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+databaseName)
}

func dsnForDatabase(databaseName string) string {

	dsn := testConnectionString
	queryStart := strings.Index(dsn, "?")
	var query string
	if queryStart >= 0 {
		query = dsn[queryStart:]
		dsn = dsn[:queryStart]
	}

	if lastSlash := strings.LastIndex(dsn, "/"); lastSlash > strings.Index(dsn, "://")+2 {
		dsn = dsn[:lastSlash+1] + databaseName
	} else {
		dsn += "/" + databaseName
	}
	return dsn + query
}

func applyMigrations(t *testing.T, testCaseDirectory, dsn string) {
	t.Helper()

	migrationDirectory := filepath.Join(testCaseDirectory, "migrations")
	entries, err := os.ReadDir(migrationDirectory)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		require.NoError(t, err)
	}

	conn, err := sql.Open("clickhouse", dsn)
	require.NoError(t, err)
	defer conn.Close()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		body, readErr := os.ReadFile(filepath.Join(migrationDirectory, entry.Name()))
		require.NoError(t, readErr)
		for _, statement := range splitOnSemicolons(string(body)) {
			trimmed := strings.TrimSpace(statement)
			if trimmed == "" {
				continue
			}
			_, execErr := conn.ExecContext(context.Background(), trimmed)
			require.NoError(t, execErr, "applying %s: %s", entry.Name(), trimmed)
		}
	}
}

func splitOnSemicolons(input string) []string {
	var out []string
	var current strings.Builder
	inSingle := false
	inDouble := false
	escapeNext := false
	for _, character := range input {
		if escapeNext {
			current.WriteRune(character)
			escapeNext = false
			continue
		}
		switch {
		case character == '\\' && (inSingle || inDouble):
			current.WriteRune(character)
			escapeNext = true
		case character == '\'' && !inDouble:
			inSingle = !inSingle
			current.WriteRune(character)
		case character == '"' && !inSingle:
			inDouble = !inDouble
			current.WriteRune(character)
		case character == ';' && !inSingle && !inDouble:
			out = append(out, current.String())
			current.Reset()
		default:
			current.WriteRune(character)
		}
	}
	if remainder := strings.TrimSpace(current.String()); remainder != "" {
		out = append(out, remainder)
	}
	return out
}

func generateCode(t *testing.T, testCaseDirectory string, spec testSpec) []querier_dto.GeneratedFile {
	t.Helper()

	engine := db_engine_clickhouse.NewClickHouseEngine()
	emitter := emitter_go.NewGoEmitterForClickHouse()

	service, err := querier_domain.NewQuerierService(querier_domain.QuerierPorts{
		Engine:     engine,
		Emitter:    emitter,
		FileReader: &realFileReader{},
	})
	require.NoError(t, err)

	migrationDirectory, _ := filepath.Abs(filepath.Join(testCaseDirectory, "migrations"))
	queryDirectory, _ := filepath.Abs(filepath.Join(testCaseDirectory, "queries"))

	config := &querier_dto.DatabaseConfig{
		MigrationDirectory: migrationDirectory,
		QueryDirectory:     queryDirectory,
		CustomFunctions:    spec.CustomFunctions,
		TypeOverrides:      spec.TypeOverrides,
	}

	result, err := service.GenerateDatabase(context.Background(), "db", config)
	require.NoError(t, err)
	require.NotNil(t, result)
	return result.Files
}

func writeGeneratedFiles(t *testing.T, tempDirectory string, files []querier_dto.GeneratedFile) {
	t.Helper()
	dbDirectory := filepath.Join(tempDirectory, "db")
	require.NoError(t, os.MkdirAll(dbDirectory, 0o755))
	for _, file := range files {
		path := filepath.Join(dbDirectory, file.Name)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, file.Content, 0o644))
	}
}

func copyRunnerSource(t *testing.T, testCaseDirectory, tempDirectory string) {
	t.Helper()
	source := filepath.Join(testCaseDirectory, "runner.go")
	target := filepath.Join(tempDirectory, "main.go")
	body, err := os.ReadFile(source)
	require.NoError(t, err, "reading runner.go")
	require.NoError(t, os.WriteFile(target, body, 0o644))
}

func writeRunnerGoMod(t *testing.T, tempDirectory string) {
	t.Helper()

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)

	body := fmt.Sprintf(`module %s

go 1.24

require (
	github.com/ClickHouse/clickhouse-go/v2 v2.30.0
)

require piko.sh/piko v0.0.0
replace piko.sh/piko v0.0.0 => %s
`, runnerModuleName, repoRoot)
	require.NoError(t, os.WriteFile(filepath.Join(tempDirectory, "go.mod"), []byte(body), 0o644))
}

func tidyModules(t *testing.T, tempDirectory string) {
	t.Helper()
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tempDirectory
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "go mod tidy: %s", output)
}

func buildRunner(t *testing.T, tempDirectory string) {
	t.Helper()
	binaryPath := filepath.Join(tempDirectory, "runner")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = tempDirectory
	cmd.Env = append(os.Environ(), "GOWORK=off")
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "go build: %s", output)
}

func executeRunner(t *testing.T, tempDirectory, dsn string) []byte {
	t.Helper()
	cmd := exec.Command(filepath.Join(tempDirectory, "runner"))
	cmd.Env = append(os.Environ(), "DATABASE_URL="+dsn)
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "runner exit: %s", output)
	return output
}

func assertGoldenJSON(t *testing.T, goldenPath string, actual []byte) {
	t.Helper()
	if *updateGoldenFiles {
		require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o755))
		require.NoError(t, os.WriteFile(goldenPath, actual, 0o644))
		t.Logf("updated golden file: %s", goldenPath)
		return
	}
	expectedBytes, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("golden file not found at %s (run with -update to generate)", goldenPath)
	}
	require.NoError(t, err)
	assert.JSONEq(t, string(expectedBytes), string(actual),
		"golden file mismatch: %s", goldenPath)
}

func randomSuffix(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}
