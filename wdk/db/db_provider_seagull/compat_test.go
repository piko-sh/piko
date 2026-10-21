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

package db_provider_seagull

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/sqldb/sqldb_dto"

	"seagull.sh/seagull/sdk/dialect/dialect_registry"
)

const (
	seagullModule   = "seagull.sh/seagull"
	compatDirectory = "internal/dialect/dialecttest/testdata/piko_compat"
	compatManifest  = "manifest.json"
	compatFullList  = "full.txt"
	compatFullDir   = "full"
	compatInputsDir = "inputs"
)

type compatEmitter struct {
	name   string
	driver sqldb_dto.GeneratedDriver
}

type compatEngine struct {
	name     string
	emitters []compatEmitter
}

type compatSpec struct {
	CustomFunctions []sqldb_dto.CustomFunction `json:"customFunctions"`
	ShouldError     bool                       `json:"shouldError"`
}

type compatOutput struct {
	files       map[string][]byte
	diagnostics []byte
	err         string
}

type compatRecord struct {
	Files       map[string]string `json:"files,omitempty"`
	Diagnostics string            `json:"diagnostics,omitempty"`
	Error       string            `json:"error,omitempty"`
}

func compatEngines() []compatEngine {
	databaseSQL := compatEmitter{name: "database_sql", driver: sqldb_dto.DriverDatabaseSQL}
	pgx := compatEmitter{name: "pgx", driver: sqldb_dto.DriverPgx}
	native := compatEmitter{name: "clickhouse_native", driver: sqldb_dto.DriverClickHouseNative}
	return []compatEngine{
		{name: "postgres", emitters: []compatEmitter{databaseSQL, pgx}},
		{name: "cockroachdb", emitters: []compatEmitter{databaseSQL, pgx}},
		{name: "mysql", emitters: []compatEmitter{databaseSQL}},
		{name: "mariadb", emitters: []compatEmitter{databaseSQL}},
		{name: "sqlite", emitters: []compatEmitter{databaseSQL}},
		{name: "duckdb", emitters: []compatEmitter{databaseSQL}},
		{name: "clickhouse", emitters: []compatEmitter{databaseSQL, native}},
	}
}

func TestPikoCompatibilityCorpus(t *testing.T) {
	if testing.Short() {
		t.Skip("the compatibility corpus generates every engine case; skipped under -short")
	}

	root := seagullCompatRoot(t)
	fullCases := loadFullCaseList(t, root)
	manifest := make(map[string]compatRecord)

	for _, engine := range compatEngines() {
		caseDirectories := engineCaseDirectories(t, root, engine.name)
		require.NotEmptyf(t, caseDirectories, "the compatibility corpus holds no %s case", engine.name)
		for _, caseDirectory := range caseDirectories {
			caseName := filepath.Base(caseDirectory)
			for _, emitter := range engine.emitters {
				key := engine.name + "/" + caseName + "/" + emitter.name
				output := generateCompatCase(t, engine, emitter, caseDirectory)
				manifest[key] = recordOf(output)
				if slices.Contains(fullCases, engine.name+"/"+caseName) {
					checkFullCase(t, root, key, output)
				}
			}
		}
	}

	expected, err := os.ReadFile(filepath.Join(root, compatManifest))
	require.NoError(t, err)

	var expectedManifest map[string]compatRecord
	require.NoError(t, json.Unmarshal(expected, &expectedManifest))
	require.NotEmpty(t, expectedManifest, "the compatibility manifest records no case")
	for key, record := range expectedManifest {
		record.Files = withoutEmptyFiles(record.Files)
		actual, ok := manifest[key]
		require.Truef(t, ok, "compatibility case %s is no longer generated", key)
		require.Equalf(t, record, actual, "compatibility case %s differs from Seagull's recorded output", key)
	}
	require.Len(t, manifest, len(expectedManifest), "the set of compatibility cases changed")
}

func seagullCompatRoot(t *testing.T) string {
	t.Helper()

	output, err := exec.CommandContext(t.Context(), "go", "list", "-m", "-f", "{{.Dir}}", seagullModule).Output()
	require.NoError(t, err, "locating the %s module", seagullModule)
	moduleDirectory := strings.TrimSpace(string(output))
	require.NotEmpty(t, moduleDirectory, "the %s module has no directory; download it with go mod download", seagullModule)

	root := filepath.Join(moduleDirectory, filepath.FromSlash(compatDirectory))
	require.DirExists(t, root, "Seagull's Piko compatibility corpus")
	return root
}

func engineCaseDirectories(t *testing.T, root, engineName string) []string {
	t.Helper()

	engineRoot := filepath.Join(root, compatInputsDir, engineName)
	entries, err := os.ReadDir(engineRoot)
	require.NoError(t, err)

	directories := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		directory := filepath.Join(engineRoot, entry.Name())
		if _, statErr := os.Stat(filepath.Join(directory, "migrations")); statErr != nil {
			continue
		}
		directories = append(directories, directory)
	}
	return directories
}

func generateCompatCase(t *testing.T, engine compatEngine, emitter compatEmitter, caseDirectory string) compatOutput {
	t.Helper()

	spec := loadCompatSpec(t, caseDirectory)
	chosen, err := dialect_registry.New(engine.name, dialect_registry.Options{})
	require.NoError(t, err)

	result, generateErr := NewToolchain(chosen).Generate(context.Background(), sqldb_dto.GenerateRequest{
		Files:              os.DirFS(caseDirectory),
		PackageName:        "generated",
		MigrationDirectory: "migrations",
		QueryDirectory:     "queries",
		CustomFunctions:    spec.CustomFunctions,
		Driver:             emitter.driver,
	})

	output := compatOutput{files: make(map[string][]byte)}
	if generateErr != nil {
		output.err = relativiseError(generateErr.Error(), caseDirectory)
		return output
	}

	for _, file := range result.Files {
		output.files[file.Name] = file.Content
	}
	output.diagnostics = renderDiagnostics(t, result.Diagnostics, caseDirectory)
	return output
}

func relativiseError(message, caseDirectory string) string {
	absolute, err := filepath.Abs(caseDirectory)
	if err != nil {
		return message
	}
	return strings.ReplaceAll(message, absolute+string(filepath.Separator), "")
}

func renderDiagnostics(t *testing.T, diagnostics []sqldb_dto.SourceError, caseDirectory string) []byte {
	t.Helper()

	if len(diagnostics) == 0 {
		return nil
	}
	absolute, err := filepath.Abs(caseDirectory)
	require.NoError(t, err)

	encoded, err := json.MarshalIndent(diagnostics, "", "  ")
	require.NoError(t, err)
	return []byte(strings.ReplaceAll(string(encoded), absolute+string(filepath.Separator), ""))
}

func recordOf(output compatOutput) compatRecord {
	record := compatRecord{Error: output.err}
	if len(output.files) > 0 {
		record.Files = make(map[string]string, len(output.files))
		for name, content := range output.files {
			record.Files[name] = digest(content)
		}
	}
	if len(output.diagnostics) > 0 {
		record.Diagnostics = digest(output.diagnostics)
	}
	return record
}

func withoutEmptyFiles(files map[string]string) map[string]string {
	emptyDigest := digest(nil)
	for name, fileDigest := range files {
		if fileDigest == emptyDigest {
			delete(files, name)
		}
	}
	if len(files) == 0 {
		return nil
	}
	return files
}

func digest(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func checkFullCase(t *testing.T, root, key string, output compatOutput) {
	t.Helper()

	directory := filepath.Join(root, compatFullDir, filepath.FromSlash(key))
	expected := make(map[string][]byte, len(output.files)+2)
	for name, content := range output.files {
		expected[name+".golden"] = content
	}
	if len(output.diagnostics) > 0 {
		expected["diagnostics.json"] = output.diagnostics
	}
	if output.err != "" {
		expected["error.txt"] = []byte(output.err + "\n")
	}

	entries, err := os.ReadDir(directory)
	require.NoErrorf(t, err, "Seagull records no full compatibility case %s", key)
	recordedNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		info, infoErr := entry.Info()
		require.NoError(t, infoErr)
		if info.Size() > 0 {
			recordedNames = append(recordedNames, entry.Name())
		}
	}
	require.Lenf(t, recordedNames, len(expected), "full compatibility case %s has a different file set", key)
	for name, content := range expected {
		recorded, readErr := os.ReadFile(filepath.Join(directory, name))
		require.NoErrorf(t, readErr, "full compatibility case %s is missing %s", key, name)
		require.Equalf(t, string(recorded), string(content), "full compatibility case %s differs from Seagull's recorded %s", key, name)
	}
}

func loadFullCaseList(t *testing.T, root string) []string {
	t.Helper()

	file, err := os.Open(filepath.Join(root, compatFullList))
	require.NoError(t, err)
	defer func() { require.NoError(t, file.Close()) }()

	var cases []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cases = append(cases, line)
	}
	require.NoError(t, scanner.Err())
	return cases
}

func loadCompatSpec(t *testing.T, caseDirectory string) compatSpec {
	t.Helper()

	var spec compatSpec
	data, err := os.ReadFile(filepath.Join(caseDirectory, "testspec.json"))
	if os.IsNotExist(err) {
		return spec
	}
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &spec))
	return spec
}
