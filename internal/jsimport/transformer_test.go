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

package jsimport

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/logger"
)

func TestNormaliseExtension(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "no extension appends .js", input: "/_piko/assets/mod/lib/utils", expected: "/_piko/assets/mod/lib/utils.js"},
		{name: ".ts replaced with .js", input: "/_piko/assets/mod/lib/utils.ts", expected: "/_piko/assets/mod/lib/utils.js"},
		{name: ".js unchanged", input: "/_piko/assets/mod/lib/utils.js", expected: "/_piko/assets/mod/lib/utils.js"},
		{name: ".css unchanged", input: "/_piko/assets/mod/styles/theme.css", expected: "/_piko/assets/mod/styles/theme.css"},
		{name: ".svg unchanged", input: "/_piko/assets/mod/icons/arrow.svg", expected: "/_piko/assets/mod/icons/arrow.svg"},
		{name: "path with dots in directory", input: "/_piko/assets/v2.0/lib/utils", expected: "/_piko/assets/v2.0/lib/utils.js"},
		{name: "empty string", input: "", expected: ".js"},
		{name: "relative path no ext", input: "./utils", expected: "./utils.js"},
		{name: "relative path .ts", input: "./utils.ts", expected: "./utils.js"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, NormaliseExtension(tc.input))
		})
	}
}

func TestIsTransformable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		importPath string
		want       bool
	}{
		{name: "@/ path is transformable", importPath: "@/lib/utils.js", want: true},
		{name: "@/ path with deep nesting", importPath: "@/a/b/c/d/e.js", want: true},
		{name: "relative path is not transformable", importPath: "./local.js", want: false},
		{name: "absolute path is not transformable", importPath: "/absolute/path.js", want: false},
		{name: "external URL is not transformable", importPath: "https://example.com/lib.js", want: false},
		{name: "bare module specifier is not transformable", importPath: "lodash", want: false},
		{name: "@ without slash is not transformable", importPath: "@scope/package", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, IsTransformable(tc.importPath))
		})
	}
}

func TestResolveModuleAlias(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		importPath string
		moduleName string
		expected   string
	}{
		{
			name:       "extensionless path gets .js",
			importPath: "@/lib/utils",
			moduleName: "github.com/user/project",
			expected:   "/_piko/assets/github.com/user/project/lib/utils.js",
		},
		{
			name:       ".ts extension becomes .js",
			importPath: "@/lib/utils.ts",
			moduleName: "github.com/user/project",
			expected:   "/_piko/assets/github.com/user/project/lib/utils.js",
		},
		{
			name:       ".js extension unchanged",
			importPath: "@/lib/utils.js",
			moduleName: "github.com/user/project",
			expected:   "/_piko/assets/github.com/user/project/lib/utils.js",
		},
		{
			name:       ".css extension preserved",
			importPath: "@/styles/theme.css",
			moduleName: "github.com/user/project",
			expected:   "/_piko/assets/github.com/user/project/styles/theme.css",
		},
		{
			name:       "deeply nested path",
			importPath: "@/lib/utils/deep/module",
			moduleName: "github.com/org/repo",
			expected:   "/_piko/assets/github.com/org/repo/lib/utils/deep/module.js",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, ResolveModuleAlias(tc.importPath, tc.moduleName))
		})
	}
}

func TestResolveModulePath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		importPath string
		moduleName string
		expected   string
	}{
		{
			name:       "extensionless path gets .js",
			importPath: "@/lib/utils",
			moduleName: "github.com/user/project",
			expected:   "github.com/user/project/lib/utils.js",
		},
		{
			name:       ".ts becomes .js",
			importPath: "@/lib/utils.ts",
			moduleName: "github.com/user/project",
			expected:   "github.com/user/project/lib/utils.js",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, ResolveModulePath(tc.importPath, tc.moduleName))
		})
	}
}

func TestRewriteImportRecords(t *testing.T) {
	t.Parallel()

	t.Run("rewrites .ts to .js", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "./utils.ts"}},
		}
		RewriteImportRecords(records, "")
		assert.Equal(t, "./utils.js", records[0].Path.Text)
	})

	t.Run("leaves .js unchanged", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "./utils.js"}},
		}
		RewriteImportRecords(records, "")
		assert.Equal(t, "./utils.js", records[0].Path.Text)
	})

	t.Run("leaves bare specifiers unchanged", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "lodash"}},
		}
		RewriteImportRecords(records, "")
		assert.Equal(t, "lodash", records[0].Path.Text)
	})

	t.Run("resolves @/ with module name", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "@/lib/utils"}},
		}
		RewriteImportRecords(records, "mymod")
		assert.Equal(t, "/_piko/assets/mymod/lib/utils.js", records[0].Path.Text)
	})

	t.Run("resolves @/ with .js extension and module name", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "@/lib/utils.js"}},
		}
		RewriteImportRecords(records, "mymod")
		assert.Equal(t, "/_piko/assets/mymod/lib/utils.js", records[0].Path.Text)
	})

	t.Run("resolves @/ with .ts extension and module name", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "@/lib/utils.ts"}},
		}
		RewriteImportRecords(records, "mymod")
		assert.Equal(t, "/_piko/assets/mymod/lib/utils.js", records[0].Path.Text)
	})

	t.Run("skips @/ without module name", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "@/lib/utils"}},
		}
		RewriteImportRecords(records, "")
		assert.Equal(t, "@/lib/utils", records[0].Path.Text)
	})

	t.Run("skips empty paths", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: ""}},
		}
		RewriteImportRecords(records, "mymod")
		assert.Equal(t, "", records[0].Path.Text)
	})

	t.Run("appends .js to extensionless relative import", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "./bar"}},
		}
		RewriteImportRecords(records, "")
		assert.Equal(t, "./bar.js", records[0].Path.Text)
	})

	t.Run("appends .js to extensionless parent relative import", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "../utils"}},
		}
		RewriteImportRecords(records, "")
		assert.Equal(t, "../utils.js", records[0].Path.Text)
	})

	t.Run("does not append .js to bare specifier without extension", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "lodash"}},
		}
		RewriteImportRecords(records, "")
		assert.Equal(t, "lodash", records[0].Path.Text)
	})

	t.Run("handles mixed records", func(t *testing.T) {
		t.Parallel()
		records := []ast.ImportRecord{
			{Path: logger.Path{Text: "@/lib/greeting"}},
			{Path: logger.Path{Text: "./formatter.ts"}},
			{Path: logger.Path{Text: "lodash"}},
			{Path: logger.Path{Text: ""}},
		}
		RewriteImportRecords(records, "mymod")
		assert.Equal(t, "/_piko/assets/mymod/lib/greeting.js", records[0].Path.Text)
		assert.Equal(t, "./formatter.js", records[1].Path.Text)
		assert.Equal(t, "lodash", records[2].Path.Text)
		assert.Equal(t, "", records[3].Path.Text)
	})
}
