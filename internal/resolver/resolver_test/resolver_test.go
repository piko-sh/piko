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

package resolver_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

type pathTest struct {
	importPath                 string
	expectedResolvedPathSuffix string
	expectErrOnResolve         bool
}

func TestLocalModuleResolver_Integration(t *testing.T) {
	testCases := []struct {
		name                  string
		projectDir            string
		startDir              string
		expectedErrOnInit     string
		expectedModuleName    string
		expectedBaseDirSuffix string
		pathsToTest           []pathTest
		expectErrOnInit       bool
	}{
		{
			name:                  "Standard Project - Starting from Root",
			projectDir:            "./testdata/01_standard_project",
			startDir:              ".",
			expectErrOnInit:       false,
			expectedModuleName:    "testproject_standard",
			expectedBaseDirSuffix: "testdata/01_standard_project",
			pathsToTest: []pathTest{
				{

					importPath:                 "testproject_standard/partials/card.pk",
					expectedResolvedPathSuffix: "testdata/01_standard_project/partials/card.pk",
				},
			},
		},
		{
			name:                  "Standard Project - Starting from Deeply Nested Directory",
			projectDir:            "./testdata/01_standard_project",
			startDir:              "./deeply/nested/dir",
			expectErrOnInit:       false,
			expectedModuleName:    "testproject_standard",
			expectedBaseDirSuffix: "testdata/01_standard_project",
			pathsToTest: []pathTest{
				{

					importPath:                 "testproject_standard/partials/card.pk",
					expectedResolvedPathSuffix: "testdata/01_standard_project/partials/card.pk",
				},
			},
		},
		{
			name:              "Malformed go.mod - Missing Module Directive",
			projectDir:        "./testdata/04_empty_gomod",
			startDir:          ".",
			expectErrOnInit:   true,
			expectedErrOnInit: "no 'module' line found",
		},
		{
			name:              "Malformed go.mod - Typo in Directive",
			projectDir:        "./testdata/03_malformed_gomod",
			startDir:          ".",
			expectErrOnInit:   true,
			expectedErrOnInit: "no 'module' line found",
		},
		{
			name:                  "Remote Path Resolution",
			projectDir:            "./testdata/01_standard_project",
			startDir:              ".",
			expectErrOnInit:       false,
			expectedModuleName:    "testproject_standard",
			expectedBaseDirSuffix: "testdata/01_standard_project",
			pathsToTest: []pathTest{
				{
					importPath:         "https://example.com/remote.pk",
					expectErrOnResolve: true,
				},
			},
		},
		{
			name:                  "Directory Name Collision",
			projectDir:            "./testdata/06_dir_collision/project",
			startDir:              ".",
			expectErrOnInit:       false,
			expectedModuleName:    "collision_project",
			expectedBaseDirSuffix: "testdata/06_dir_collision/project",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			projectAbsPath, err := filepath.Abs(tc.projectDir)
			require.NoError(t, err, "Failed to get absolute path for project directory")
			startAbsPath := filepath.Join(projectAbsPath, tc.startDir)

			var resolver resolver_domain.ResolverPort = resolver_adapters.NewLocalModuleResolver(startAbsPath)
			err = resolver.DetectLocalModule(ctx)

			if tc.expectErrOnInit {
				require.Error(t, err, "Expected an error during DetectLocalModule but got none")
				assert.Contains(t, err.Error(), tc.expectedErrOnInit, "Error message did not contain expected text")
				return
			}

			require.NoError(t, err, "Expected no error during DetectLocalModule but got one")
			assert.Equal(t, tc.expectedModuleName, resolver.GetModuleName(), "Module name does not match")

			cleanedSuffix := filepath.Clean(tc.expectedBaseDirSuffix)
			assert.True(t, strings.HasSuffix(resolver.GetBaseDir(), cleanedSuffix), "Base directory suffix does not match. Got: %s, Expected Suffix: %s", resolver.GetBaseDir(), cleanedSuffix)

			for _, pathTest := range tc.pathsToTest {

				resolvedPath, resolveErr := resolver.ResolvePKPath(ctx, pathTest.importPath, "")

				if pathTest.expectErrOnResolve {
					assert.Error(t, resolveErr, "Expected an error resolving path but got none for import: %s", pathTest.importPath)
					continue
				}

				require.NoError(t, resolveErr, "Got an unexpected error resolving path for import: %s", pathTest.importPath)
				cleanedPathSuffix := filepath.Clean(pathTest.expectedResolvedPathSuffix)
				assert.True(t, strings.HasSuffix(resolvedPath, cleanedPathSuffix), "Resolved path does not have the correct suffix. Got: %s, Expected Suffix: %s", resolvedPath, cleanedPathSuffix)
			}
		})
	}

	t.Run("Project with No go.mod - Path resolution requires go.mod", func(t *testing.T) {
		ctx := context.Background()

		tempDir := t.TempDir()
		projectDir := filepath.Join(tempDir, "myproject")
		err := os.Mkdir(projectDir, 0755)
		require.NoError(t, err)

		dummyFilePath := filepath.Join(projectDir, "page.pk")
		err = os.WriteFile(dummyFilePath, []byte("<template></template>"), 0644)
		require.NoError(t, err)

		resolver := resolver_adapters.NewLocalModuleResolver(projectDir)
		err = resolver.DetectLocalModule(ctx)

		require.NoError(t, err, "DetectLocalModule should not fail even if no go.mod is found")

		assert.Equal(t, "", resolver.GetModuleName(), "Module name should be empty")

		expectedBaseDir, err := filepath.Abs(projectDir)
		require.NoError(t, err)
		assert.Equal(t, expectedBaseDir, resolver.GetBaseDir(), "Base directory should default to the start directory")

		_, err = resolver.ResolvePKPath(ctx, "page.pk", "")
		require.Error(t, err, "Path resolution should fail without a go.mod")
		assert.Contains(t, err.Error(), "no local module detected")
	})

	t.Run("Relative paths are not supported", func(t *testing.T) {
		ctx := context.Background()
		projectDir := "./testdata/05_sibling_resolve/app"

		absStartDir, err := filepath.Abs(projectDir)
		require.NoError(t, err)

		resolver := resolver_adapters.NewLocalModuleResolver(absStartDir)
		err = resolver.DetectLocalModule(ctx)
		require.NoError(t, err)

		assert.Equal(t, "sibling_project", resolver.GetModuleName())

		_, err = resolver.ResolvePKPath(ctx, "../partials/header.pk", "")
		require.Error(t, err, "Relative paths should not be supported")
		assert.Contains(t, err.Error(), "invalid component import path")
	})
}
