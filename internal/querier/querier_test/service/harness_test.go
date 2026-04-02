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

package service_test

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testSpec struct {
	Description             string               `json:"description"`
	Engine                  string               `json:"engine"`
	ShouldError             bool                 `json:"shouldError"`
	ExpectedDiagnosticCount *int                 `json:"expectedDiagnosticCount"`
	Diagnostics             []expectedDiagnostic `json:"diagnostics"`
}

type expectedDiagnostic struct {
	Severity        string `json:"severity"`
	Code            string `json:"code"`
	MessageContains string `json:"messageContains"`
	Filename        string `json:"filename"`
	OnLine          *int   `json:"onLine"`
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
	ctx := context.Background()

	spec := loadTestSpec(t, testCaseDirectory)
	engineConfig := loadEngineConfig(t, testCaseDirectory)

	engine := newMockEngine(engineConfig)
	emitter := &recordingCodeEmitter{}

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
	}

	result, generateError := service.GenerateDatabase(ctx, "test", databaseConfig, "")

	if spec.ShouldError {
		assert.Error(t, generateError, "expected GenerateDatabase to return an error")
		return
	}
	require.NoError(t, generateError)
	require.NotNil(t, result)

	catalogue := engine.recordedCatalogue
	if catalogue == nil && emitter.catalogue != nil {
		catalogue = emitter.catalogue
	}

	goldenDirectory := filepath.Join(testCaseDirectory, "golden")
	require.NoError(t, os.MkdirAll(goldenDirectory, 0o755))

	if catalogue != nil {
		catalogueJSON := serialiseDeterministic(t, catalogue)
		assertGoldenJSON(t, filepath.Join(goldenDirectory, "catalogue.json"), catalogueJSON)
	}

	if len(emitter.queries) > 0 {
		queriesJSON := serialiseDeterministic(t, emitter.queries)
		assertGoldenJSON(t, filepath.Join(goldenDirectory, "queries.json"), queriesJSON)
	}

	if len(result.Diagnostics) > 0 {
		diagnosticsJSON := serialiseDeterministic(t, result.Diagnostics)
		assertGoldenJSON(t, filepath.Join(goldenDirectory, "diagnostics.json"), diagnosticsJSON)
	}

	if spec.ExpectedDiagnosticCount != nil {
		assert.Equal(t, *spec.ExpectedDiagnosticCount, len(result.Diagnostics),
			"unexpected diagnostic count")
	}

	for _, expected := range spec.Diagnostics {
		found := false
		for _, actual := range result.Diagnostics {
			if matchesDiagnostic(actual, expected) {
				found = true
				break
			}
		}
		assert.True(t, found,
			"expected diagnostic not found: severity=%s code=%s message containing %q",
			expected.Severity, expected.Code, expected.MessageContains)
	}
}

func loadTestSpec(t *testing.T, testCaseDirectory string) testSpec {
	t.Helper()
	specPath := filepath.Join(testCaseDirectory, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "failed to read testspec.json")

	var spec testSpec
	require.NoError(t, json.Unmarshal(specBytes, &spec), "failed to parse testspec.json")
	return spec
}

func loadEngineConfig(t *testing.T, testCaseDirectory string) mockEngineConfig {
	t.Helper()
	enginePath := filepath.Join(testCaseDirectory, "engine.json")
	engineBytes, err := os.ReadFile(enginePath)
	require.NoError(t, err, "failed to read engine.json")

	var config mockEngineConfig
	require.NoError(t, json.Unmarshal(engineBytes, &config), "failed to parse engine.json")
	return config
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

func serialiseDeterministic(t *testing.T, value any) []byte {
	t.Helper()
	result, err := json.MarshalIndent(value, "", "  ")
	require.NoError(t, err, "failed to serialise to JSON")
	return append(result, '\n')
}

func matchesDiagnostic(actual querier_dto.SourceError, expected expectedDiagnostic) bool {
	if expected.Code != "" && actual.Code != expected.Code {
		return false
	}
	if expected.Severity != "" && !matchesSeverity(actual.Severity, expected.Severity) {
		return false
	}
	if expected.MessageContains != "" && !strings.Contains(actual.Message, expected.MessageContains) {
		return false
	}
	if expected.Filename != "" && !strings.HasSuffix(actual.Filename, expected.Filename) {
		return false
	}
	if expected.OnLine != nil && actual.Line != *expected.OnLine {
		return false
	}
	return true
}

func matchesSeverity(actual querier_dto.ErrorSeverity, expected string) bool {
	switch strings.ToLower(expected) {
	case "error":
		return actual == querier_dto.SeverityError
	case "warning":
		return actual == querier_dto.SeverityWarning
	case "hint":
		return actual == querier_dto.SeverityHint
	default:
		return false
	}
}
