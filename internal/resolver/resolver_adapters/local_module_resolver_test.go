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

package resolver_adapters

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnit_findGoMod(t *testing.T) {
	tempRoot := t.TempDir()
	projectRoot := filepath.Join(tempRoot, "myproject")
	nestedDir := filepath.Join(projectRoot, "cmd", "server")
	siblingDir := filepath.Join(tempRoot, "siblingproject")

	require.NoError(t, os.MkdirAll(nestedDir, 0755))
	require.NoError(t, os.MkdirAll(siblingDir, 0755))

	goModPath := filepath.Join(projectRoot, "go.mod")
	err := os.WriteFile(goModPath, []byte("module myproject"), 0644)
	require.NoError(t, err)

	decoyDir := filepath.Join(siblingDir, "go.mod")
	require.NoError(t, os.Mkdir(decoyDir, 0755))

	testCases := []struct {
		expectedErrIs error
		name          string
		startPath     string
		expectedPath  string
		expectErr     bool
	}{
		{
			name:         "Should find go.mod when starting from a deeply nested directory",
			startPath:    nestedDir,
			expectedPath: goModPath,
			expectErr:    false,
		},
		{
			name:         "Should find go.mod when starting from the root directory",
			startPath:    projectRoot,
			expectedPath: goModPath,
			expectErr:    false,
		},
		{
			name:         "Should return empty string when no go.mod is in the parent path",
			startPath:    siblingDir,
			expectedPath: "",
			expectErr:    false,
		},
		{
			name:         "Should ignore a directory named go.mod",
			startPath:    decoyDir,
			expectedPath: "",
			expectErr:    false,
		},
		{
			name:          "Should return an error for a non-existent start path",
			startPath:     filepath.Join(tempRoot, "nonexistent"),
			expectedPath:  "",
			expectErr:     true,
			expectedErrIs: os.ErrNotExist,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			foundPath, err := findGoMod(tc.startPath)

			if tc.expectErr {
				require.Error(t, err)
				if tc.expectedErrIs != nil {
					assert.ErrorIs(t, err, tc.expectedErrIs)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedPath, foundPath)
			}
		})
	}
}

func TestUnit_readModuleName(t *testing.T) {
	createTempGoMod := func(t *testing.T, content string) string {
		t.Helper()
		directory := t.TempDir()
		goModPath := filepath.Join(directory, "go.mod")
		require.NoError(t, os.WriteFile(goModPath, []byte(content), 0644))
		return goModPath
	}

	testCases := []struct {
		name           string
		goModContent   string
		expectedModule string
		errContains    string
		expectErr      bool
	}{
		{
			name:           "Standard module line",
			goModContent:   "module my/project/name\n\ngo 1.25\n",
			expectedModule: "my/project/name",
			expectErr:      false,
		},
		{
			name:           "Module line with extra whitespace",
			goModContent:   "\t module    my/project/name   \n",
			expectedModule: "my/project/name",
			expectErr:      false,
		},
		{
			name:           "Module line with comments before",
			goModContent:   "# This is a comment\n// Another comment\nmodule myproject\n",
			expectedModule: "myproject",
			expectErr:      false,
		},
		{
			name:         "No module line",
			goModContent: "go 1.25\n\nrequire github.com/stretchr/testify v1.8.0\n",
			expectErr:    true,
			errContains:  "no 'module' line found",
		},
		{
			name:         "Empty file",
			goModContent: "",
			expectErr:    true,
			errContains:  "no 'module' line found",
		},
		{
			name:         "Typo in module directive",
			goModContent: "modul myproject\ngo 1.25\n",
			expectErr:    true,
			errContains:  "no 'module' line found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			goModPath := createTempGoMod(t, tc.goModContent)

			moduleName, err := readModuleName(goModPath, nil)

			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedModule, moduleName)
			}
		})
	}

	t.Run("Non-existent file", func(t *testing.T) {
		nonExistentPath := filepath.Join(t.TempDir(), "go.mod")
		_, err := readModuleName(nonExistentPath, nil)
		require.Error(t, err)
	})
}

func TestUnit_ResolveCSSPath(t *testing.T) {
	tempRoot := t.TempDir()
	projectRoot := filepath.Join(tempRoot, "testproject")
	stylesDir := filepath.Join(projectRoot, "styles")
	nestedDir := filepath.Join(projectRoot, "components", "button")

	require.NoError(t, os.MkdirAll(stylesDir, 0755))
	require.NoError(t, os.MkdirAll(nestedDir, 0755))

	goModPath := filepath.Join(projectRoot, "go.mod")
	err := os.WriteFile(goModPath, []byte("module testproject"), 0644)
	require.NoError(t, err)

	globalCSS := filepath.Join(stylesDir, "global.css")
	err = os.WriteFile(globalCSS, []byte("body { margin: 0; }"), 0644)
	require.NoError(t, err)

	themeCSS := filepath.Join(stylesDir, "theme.css")
	err = os.WriteFile(themeCSS, []byte(":root { --color: blue; }"), 0644)
	require.NoError(t, err)

	buttonCSS := filepath.Join(nestedDir, "button.css")
	err = os.WriteFile(buttonCSS, []byte(".button { padding: 10px; }"), 0644)
	require.NoError(t, err)

	nonCSSFile := filepath.Join(stylesDir, "config.json")
	err = os.WriteFile(nonCSSFile, []byte("{}"), 0644)
	require.NoError(t, err)

	resolver := NewLocalModuleResolver(projectRoot)
	ctx := context.Background()

	err = resolver.DetectLocalModule(ctx)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		importPath    string
		containingDir string
		expectedPath  string
		errContains   string
		expectErr     bool
	}{
		{
			name:          "Module-absolute path - root level CSS",
			importPath:    "testproject/styles/global.css",
			containingDir: projectRoot,
			expectedPath:  globalCSS,
			expectErr:     false,
		},
		{
			name:          "Module-absolute path - theme CSS",
			importPath:    "testproject/styles/theme.css",
			containingDir: projectRoot,
			expectedPath:  themeCSS,
			expectErr:     false,
		},
		{
			name:          "Module-absolute path - nested component CSS",
			importPath:    "testproject/components/button/button.css",
			containingDir: projectRoot,
			expectedPath:  buttonCSS,
			expectErr:     false,
		},
		{
			name:          "Relative path - current directory",
			importPath:    "./global.css",
			containingDir: stylesDir,
			expectedPath:  globalCSS,
			expectErr:     false,
		},
		{
			name:          "Relative path - parent directory",
			importPath:    "../styles/theme.css",
			containingDir: nestedDir,
			expectedPath:  filepath.Join(nestedDir, "..", "styles", "theme.css"),
			expectErr:     false,
		},
		{
			name:          "Relative path - nested navigation",
			importPath:    "./theme.css",
			containingDir: stylesDir,
			expectedPath:  themeCSS,
			expectErr:     false,
		},
		{
			name:          "Relative path - complex navigation",
			importPath:    "../../styles/global.css",
			containingDir: filepath.Join(nestedDir, "subdir"),
			expectedPath:  filepath.Join(nestedDir, "subdir", "..", "..", "styles", "global.css"),
			expectErr:     false,
		},

		{
			name:          "Invalid path - not module-absolute or relative",
			importPath:    "styles/global.css",
			containingDir: projectRoot,
			expectErr:     true,
			errContains:   "invalid CSS import path",
		},
		{
			name:          "Invalid path - absolute filesystem path",
			importPath:    "/absolute/path/styles.css",
			containingDir: projectRoot,
			expectErr:     true,
			errContains:   "invalid CSS import path",
		},
		{
			name:          "Invalid path - empty import path",
			importPath:    "",
			containingDir: projectRoot,
			expectErr:     true,
			errContains:   "invalid CSS import path",
		},
		{
			name:          "Non-CSS file - module-absolute path",
			importPath:    "testproject/styles/config.json",
			containingDir: projectRoot,
			expectErr:     true,
			errContains:   "resolved path is not a .css file",
		},
		{
			name:          "Non-CSS file - relative path",
			importPath:    "./config.json",
			containingDir: stylesDir,
			expectErr:     true,
			errContains:   "resolved path is not a .css file",
		},
		{
			name:          "No extension - module-absolute",
			importPath:    "testproject/styles/global",
			containingDir: projectRoot,
			expectErr:     true,
			errContains:   "resolved path is not a .css file",
		},
		{
			name:          "No extension - relative",
			importPath:    "./global",
			containingDir: stylesDir,
			expectErr:     true,
			errContains:   "resolved path is not a .css file",
		},
		{
			name:          "Wrong module name in path",
			importPath:    "wrongproject/styles/global.css",
			containingDir: projectRoot,
			expectErr:     true,
			errContains:   "invalid CSS import path",
		},
		{
			name:          "CSS extension with different case",
			importPath:    "testproject/styles/GLOBAL.CSS",
			containingDir: projectRoot,
			expectedPath:  filepath.Join(projectRoot, "styles", "GLOBAL.CSS"),
			expectErr:     false,
		},
		{
			name:          "Complex relative path with redundant components",
			importPath:    "./subdir/../theme.css",
			containingDir: stylesDir,
			expectedPath:  themeCSS,
			expectErr:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resolvedPath, err := resolver.ResolveCSSPath(ctx, tc.importPath, tc.containingDir)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				assert.Empty(t, resolvedPath)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedPath, resolvedPath)
			}
		})
	}
}

func TestUnit_ResolveCSSPath_WithoutModuleDetection(t *testing.T) {
	tempRoot := t.TempDir()
	resolver := NewLocalModuleResolver(tempRoot)
	ctx := context.Background()

	t.Run("Should fail when module info is missing", func(t *testing.T) {
		_, err := resolver.ResolveCSSPath(ctx, "someproject/styles/theme.css", tempRoot)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "module information is missing")
	})

	t.Run("Relative paths should fail without module detection", func(t *testing.T) {
		cssFile := filepath.Join(tempRoot, "test.css")
		err := os.WriteFile(cssFile, []byte("body {}"), 0644)
		require.NoError(t, err)

		_, err = resolver.ResolveCSSPath(ctx, "./test.css", tempRoot)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "module information is missing")
	})
}

func TestUnit_ResolveCSSPath_EdgeCases(t *testing.T) {
	tempRoot := t.TempDir()
	projectRoot := filepath.Join(tempRoot, "edgeproject")
	require.NoError(t, os.MkdirAll(projectRoot, 0755))

	goModPath := filepath.Join(projectRoot, "go.mod")
	err := os.WriteFile(goModPath, []byte("module edgeproject"), 0644)
	require.NoError(t, err)

	resolver := NewLocalModuleResolver(projectRoot)
	ctx := context.Background()
	err = resolver.DetectLocalModule(ctx)
	require.NoError(t, err)

	t.Run("Empty containing directory", func(t *testing.T) {
		resolvedPath, err := resolver.ResolveCSSPath(ctx, "./styles.css", "")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join("", "styles.css"), resolvedPath)
	})

	t.Run("Containing directory doesn't exist", func(t *testing.T) {
		nonExistentDir := filepath.Join(tempRoot, "nonexistent")
		resolvedPath, err := resolver.ResolveCSSPath(ctx, "./styles.css", nonExistentDir)
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(nonExistentDir, "styles.css"), resolvedPath)
	})

	t.Run("Very long path components", func(t *testing.T) {
		longName := strings.Repeat("a", 100)
		importPath := fmt.Sprintf("edgeproject/styles/%s.css", longName)
		resolvedPath, err := resolver.ResolveCSSPath(ctx, importPath, projectRoot)
		require.NoError(t, err)
		expectedPath := filepath.Join(projectRoot, "styles", longName+".css")
		assert.Equal(t, expectedPath, resolvedPath)
	})

	t.Run("Path with special characters", func(t *testing.T) {
		specialDir := filepath.Join(projectRoot, "styles-with_dots.and-dashes")
		require.NoError(t, os.MkdirAll(specialDir, 0755))

		specialCSS := filepath.Join(specialDir, "special-file_name.css")
		err := os.WriteFile(specialCSS, []byte("/* special */"), 0644)
		require.NoError(t, err)

		resolvedPath, err := resolver.ResolveCSSPath(ctx, "edgeproject/styles-with_dots.and-dashes/special-file_name.css", projectRoot)
		require.NoError(t, err)
		assert.Equal(t, specialCSS, resolvedPath)
	})
}

func TestConvertEntryPointPathToManifestKey(t *testing.T) {
	tests := []struct {
		name           string
		moduleName     string
		entryPointPath string
		expectedKey    string
	}{
		{
			name:           "simple module name with standard page",
			moduleName:     "myapp",
			entryPointPath: "myapp/pages/index.pk",
			expectedKey:    "pages/index.pk",
		},
		{
			name:           "hyphenated module name",
			moduleName:     "my-example-website",
			entryPointPath: "my-example-website/pages/index.pk",
			expectedKey:    "pages/index.pk",
		},
		{
			name:           "github-style module name",
			moduleName:     "github.com/my-org/my-app",
			entryPointPath: "github.com/my-org/my-app/pages/index.pk",
			expectedKey:    "pages/index.pk",
		},
		{
			name:           "gitlab-style module name",
			moduleName:     "gitlab.com/company/team/project",
			entryPointPath: "gitlab.com/company/team/project/pages/blog/post.pk",
			expectedKey:    "pages/blog/post.pk",
		},
		{
			name:           "nested page path",
			moduleName:     "myblog",
			entryPointPath: "myblog/pages/posts/2024/hello-world.pk",
			expectedKey:    "pages/posts/2024/hello-world.pk",
		},
		{
			name:           "components directory",
			moduleName:     "myapp",
			entryPointPath: "myapp/components/button.pk",
			expectedKey:    "components/button.pk",
		},
		{
			name:           "path already relative (no module prefix)",
			moduleName:     "myapp",
			entryPointPath: "pages/index.pk",
			expectedKey:    "pages/index.pk",
		},
		{
			name:           "wrong module prefix",
			moduleName:     "myapp",
			entryPointPath: "otherapp/pages/index.pk",
			expectedKey:    "otherapp/pages/index.pk",
		},
		{
			name:           "empty string",
			moduleName:     "myapp",
			entryPointPath: "",
			expectedKey:    "",
		},
		{
			name:           "single segment path (no slashes)",
			moduleName:     "myapp",
			entryPointPath: "standalone.pk",
			expectedKey:    "standalone.pk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := &LocalModuleResolver{
				moduleName: tt.moduleName,
			}

			result := resolver.ConvertEntryPointPathToManifestKey(tt.entryPointPath)
			assert.Equal(t, tt.expectedKey, result)
		})
	}
}

func TestConvertEntryPointPathToManifestKey_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name           string
		goModContent   string
		entryPointPath string
		expectedKey    string
	}{
		{
			name:           "Real project: my-example-website",
			goModContent:   "module my-example-website\n",
			entryPointPath: "my-example-website/pages/index.pk",
			expectedKey:    "pages/index.pk",
		},
		{
			name:           "GitHub-hosted project",
			goModContent:   "module piko.sh/piko\n",
			entryPointPath: "piko.sh/piko/pages/docs.pk",
			expectedKey:    "pages/docs.pk",
		},
		{
			name:           "Private GitLab project",
			goModContent:   "module gitlab.company.com/backend/user-service\n",
			entryPointPath: "gitlab.company.com/backend/user-service/components/auth.pk",
			expectedKey:    "components/auth.pk",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tempRoot := t.TempDir()
			projectRoot := filepath.Join(tempRoot, "testproject")
			require.NoError(t, os.MkdirAll(projectRoot, 0755))

			goModPath := filepath.Join(projectRoot, "go.mod")
			err := os.WriteFile(goModPath, []byte(scenario.goModContent), 0644)
			require.NoError(t, err)

			resolver := NewLocalModuleResolver(projectRoot)
			ctx := context.Background()
			err = resolver.DetectLocalModule(ctx)
			require.NoError(t, err)

			result := resolver.ConvertEntryPointPathToManifestKey(scenario.entryPointPath)
			assert.Equal(t, scenario.expectedKey, result)
		})
	}
}

func TestConvertEntryPointPathToManifestKey_DocumentsArchitecturalBoundary(t *testing.T) {
	buildTimeFormat := "github.com/my-org/my-piko-app/pages/index.pk"
	runtimeFormat := "pages/index.pk"

	resolver := &LocalModuleResolver{
		moduleName: "github.com/my-org/my-piko-app",
	}

	result := resolver.ConvertEntryPointPathToManifestKey(buildTimeFormat)
	assert.Equal(t, runtimeFormat, result)
}
