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

package builder_test

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"piko.sh/piko/internal/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

var updateGoldenFiles = flag.Bool("update", false, "Update golden files")

type testCase struct {
	Name string
	Path string
}

type TestSpec struct {
	Description        string          `json:"description"`
	ModuleName         string          `json:"moduleName"`
	ErrorContains      string          `json:"errorContains"`
	GoldenFileIncludes []string        `json:"goldenFileIncludes"`
	Assertions         []JSONAssertion `json:"assertions"`
	ShouldError        bool            `json:"shouldError"`
}

type JSONAssertion struct {
	Expect      any    `json:"expect"`
	Description string `json:"description"`
	Select      string `json:"select"`
}

func runBuilderTestCase(t *testing.T, tc testCase) {

	specPath := filepath.Join(tc.Path, "testspec.json")
	specBytes, err := os.ReadFile(specPath)
	require.NoError(t, err, "Failed to read testspec.json")
	var spec TestSpec
	require.NoError(t, json.Unmarshal(specBytes, &spec), "Failed to unmarshal testspec.json")

	srcDir := filepath.Join(tc.Path, "src")
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err, "Failed to get absolute path for src directory")

	sourceContents := getAllGoSource(t, absSrcDir)

	config := inspector_dto.Config{
		BaseDir:    absSrcDir,
		ModuleName: spec.ModuleName,
	}

	builder := inspector_domain.NewTypeBuilder(config)

	err = builder.Build(context.Background(), sourceContents, map[string]string{})

	if spec.ShouldError {
		require.Error(t, err, "Expected builder.Build() to fail")
		if spec.ErrorContains != "" {
			assert.Contains(t, err.Error(), spec.ErrorContains)
		}
		return
	}
	require.NoError(t, err, "builder.Build() failed unexpectedly")

	goldenDir := filepath.Join(tc.Path, "golden")
	require.NoError(t, os.MkdirAll(goldenDir, 0755))

	testdataRoot, err := filepath.Abs(filepath.Dir(tc.Path))
	require.NoError(t, err)

	modCache := resolveModCachePath()
	goroot := resolveGorootPath()

	dumpOpts := inspector_domain.DumpOptions{
		SanitisePathPrefix:     testdataRoot,
		SanitiseModCachePrefix: modCache,
		SanitiseGorootPrefix:   goroot,
		FilterPackagePrefixes:  spec.GoldenFileIncludes,
	}

	dumpOpts.Format = inspector_domain.DumpFormatReadable
	readableDump, err := builder.DumpTypeData(dumpOpts)
	require.NoError(t, err)
	checkGoldenFile(t, tc.Name, filepath.Join(goldenDir, "dump.txt"), readableDump)

	dumpOpts.Format = inspector_domain.DumpFormatJSON
	jsonDump, err := builder.DumpTypeData(dumpOpts)
	require.NoError(t, err)
	checkGoldenFile(t, tc.Name, filepath.Join(goldenDir, "dump.json"), jsonDump)

	fullSanitisedJSON, err := builder.DumpTypeData(inspector_domain.DumpOptions{
		Format:                 inspector_domain.DumpFormatJSON,
		SanitisePathPrefix:     testdataRoot,
		SanitiseModCachePrefix: modCache,
		SanitiseGorootPrefix:   goroot,
	})
	require.NoError(t, err)

	var jsonData any
	require.NoError(t, json.Unmarshal([]byte(fullSanitisedJSON), &jsonData))

	for _, assertion := range spec.Assertions {
		t.Run(assertion.Description, func(t *testing.T) {
			assertOnJSON(t, jsonData, assertion)
		})
	}
}

func checkGoldenFile(t *testing.T, testName, goldenPath, actualContent string) {

	if !strings.HasSuffix(actualContent, "\n") {
		actualContent += "\n"
	}

	if *updateGoldenFiles {
		t.Logf("Updating golden file: %s", goldenPath)
		require.NoError(t, os.WriteFile(goldenPath, []byte(actualContent), 0644))
		return
	}

	expectedContent, err := os.ReadFile(goldenPath)
	if os.IsNotExist(err) {
		t.Fatalf("Golden file not found: %s. Run with -update to generate it.", goldenPath)
	}
	require.NoError(t, err)
	assert.Equal(t, string(expectedContent), actualContent, "Golden file mismatch for %s", goldenPath)
}

func resolveGorootPath() string {
	out, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func resolveModCachePath() string {
	if v := os.Getenv("GOMODCACHE"); v != "" {
		return v
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		gopath = filepath.Join(home, "go")
	}
	return filepath.Join(gopath, "pkg", "mod")
}

func getAllGoSource(t *testing.T, srcDir string) map[string][]byte {
	sources := make(map[string][]byte)
	absSrcDir, err := filepath.Abs(srcDir)
	require.NoError(t, err)

	err = filepath.Walk(absSrcDir, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			content, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			sources[path] = content
		}
		return nil
	})
	require.NoError(t, err)
	return sources
}

func assertOnJSON(t *testing.T, jsonData any, assertion JSONAssertion) {
	segments := strings.Split(assertion.Select, ".")
	var current = jsonData

	if len(segments) > 1 && segments[0] == "packages" {
		packagesObj, ok := jsonData.(map[string]any)
		if !ok {
			t.Fatalf("Top-level JSON data is not an object for selector '%s'", assertion.Select)
		}
		packagesMap, ok := packagesObj["packages"].(map[string]any)
		if !ok {
			t.Fatalf("JSON data does not contain a 'packages' map for selector '%s'", assertion.Select)
		}

		foundPackage := false

		for i := len(segments); i > 1; i-- {

			pkgKey := strings.Join(segments[1:i], ".")

			if pkgData, keyExists := packagesMap[pkgKey]; keyExists {

				current = pkgData

				segments = segments[i:]
				foundPackage = true
				break
			}
		}

		if !foundPackage {

			t.Fatalf("Could not find a matching package for selector '%s'", assertion.Select)
		}
	}

	for i, seg := range segments {
		if current == nil {
			t.Fatalf("Cannot select '%s': path is nil at segment #%d ('%s') for selector '%s'", seg, i, seg, assertion.Select)
		}

		if currentMap, ok := current.(map[string]any); ok {

			current, ok = currentMap[seg]
			if !ok {
				t.Fatalf("Cannot select '%s': key not found at segment #%d ('%s') for selector '%s'", seg, i, seg, assertion.Select)
			}
			continue
		}

		if currentArray, ok := current.([]any); ok {

			index, err := strconv.Atoi(seg)
			if err != nil {
				t.Fatalf("Cannot select '%s': expected an integer index for an array at segment #%d, but got '%s' for selector '%s'", seg, i, seg, assertion.Select)
			}
			if index < 0 || index >= len(currentArray) {
				t.Fatalf("Cannot select '%s': index %d is out of bounds for array of length %d at segment #%d for selector '%s'", seg, index, len(currentArray), i, assertion.Select)
			}
			current = currentArray[index]
			continue
		}

		t.Fatalf("Cannot select '%s': expected a JSON object or array at segment #%d, but got %T for selector '%s'", seg, i, current, assertion.Select)
	}

	if expectedNum, ok := assertion.Expect.(float64); ok {

		if actualNum, ok := current.(float64); ok {
			assert.InDelta(t, expectedNum, actualNum, 0.001, "Value mismatch for selector '%s'", assertion.Select)
			return
		}
	}

	assert.Equal(t, assertion.Expect, current, "Value mismatch for selector '%s'", assertion.Select)
}
