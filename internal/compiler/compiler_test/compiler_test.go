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

package compiler_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/compiler/compiler_domain"
)

var update = flag.Bool("update", false, "update golden files")

func TestCompiler_GoldenFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compiler golden file tests in short mode")
	}

	testdataRoot := "testdata"
	testDirs, err := os.ReadDir(testdataRoot)
	require.NoError(t, err)

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}
		testName := entry.Name()
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			testDir := filepath.Join(testdataRoot, testName)
			pkcPath := filepath.Join(testDir, "component.pkc")
			goldenJSPath := filepath.Join(testDir, "golden.js")

			goldenScaffoldPath := filepath.Join(testDir, "golden.scaffold.html")

			pkcContent, err := os.ReadFile(pkcPath)
			require.NoError(t, err, "failed to read component.pkc")

			compilerService := compiler_domain.NewCompilerOrchestrator(nil, nil)
			artefact, err := compilerService.CompileSFCBytes(context.Background(), pkcPath, pkcContent)
			require.NoError(t, err, "SFC compilation failed")
			require.NotNil(t, artefact)
			require.NotEmpty(t, artefact.Files)

			mainJSFile := artefact.BaseJSPath
			actualJS := artefact.Files[mainJSFile]
			actualScaffoldHTML := artefact.ScaffoldHTML

			if *update {
				t.Logf("Updating golden files for %s", testName)
				require.NoError(t, os.WriteFile(goldenJSPath, []byte(actualJS), 0644))
				require.NoError(t, os.WriteFile(goldenScaffoldPath, []byte(actualScaffoldHTML), 0644))
				return
			}

			expectedJS, err := os.ReadFile(goldenJSPath)
			require.NoError(t, err, "failed to read golden.js")
			assert.Equal(t, string(expectedJS), actualJS, "Generated JavaScript does not match golden.js")

			expectedScaffoldHTML, err := os.ReadFile(goldenScaffoldPath)
			if !os.IsNotExist(err) {
				require.NoError(t, err, "failed to read golden.scaffold.html")
				assert.Equal(t, string(expectedScaffoldHTML), actualScaffoldHTML, "Generated Scaffold HTML does not match golden.scaffold.html")
			}
		})
	}
}
