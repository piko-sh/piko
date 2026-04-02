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

package compiler_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransformJSImportPath(t *testing.T) {
	tests := []struct {
		name             string
		moduleName       string
		importPath       string
		wantPath         string
		wantResolvedPath string
		wantServedPath   string
		wantDependency   bool
	}{
		{
			name:           "non @/ path passes through unchanged",
			moduleName:     "github.com/user/project",
			importPath:     "some-external-module",
			wantPath:       "some-external-module",
			wantDependency: false,
		},
		{
			name:           "relative path passes through unchanged",
			moduleName:     "github.com/user/project",
			importPath:     "./local-file.js",
			wantPath:       "./local-file.js",
			wantDependency: false,
		},
		{
			name:           "external URL passes through unchanged",
			moduleName:     "github.com/user/project",
			importPath:     "https://cdn.example.com/lib.js",
			wantPath:       "https://cdn.example.com/lib.js",
			wantDependency: false,
		},
		{
			name:             "@/ path transforms to served URL",
			moduleName:       "github.com/user/project",
			importPath:       "@/scripts/lib.js",
			wantPath:         "/_piko/assets/github.com/user/project/scripts/lib.js",
			wantDependency:   true,
			wantResolvedPath: "github.com/user/project/scripts/lib.js",
			wantServedPath:   "/_piko/assets/github.com/user/project/scripts/lib.js",
		},
		{
			name:             "@/ path with nested directories transforms correctly",
			moduleName:       "github.com/org/repo",
			importPath:       "@/lib/utils/deep/module.js",
			wantPath:         "/_piko/assets/github.com/org/repo/lib/utils/deep/module.js",
			wantDependency:   true,
			wantResolvedPath: "github.com/org/repo/lib/utils/deep/module.js",
			wantServedPath:   "/_piko/assets/github.com/org/repo/lib/utils/deep/module.js",
		},
		{
			name:           "@/ path without module name returns original",
			moduleName:     "",
			importPath:     "@/scripts/lib.js",
			wantPath:       "@/scripts/lib.js",
			wantDependency: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.moduleName != "" {
				ctx = WithModuleName(ctx, tt.moduleName)
			}

			gotPath, gotDep := TransformJSImportPath(ctx, tt.importPath)

			assert.Equal(t, tt.wantPath, gotPath)

			if tt.wantDependency {
				assert.NotNil(t, gotDep)
				assert.Equal(t, tt.importPath, gotDep.OriginalPath)
				assert.Equal(t, tt.wantResolvedPath, gotDep.ResolvedPath)
				assert.Equal(t, tt.wantServedPath, gotDep.ServedPath)
			} else {
				assert.Nil(t, gotDep)
			}
		})
	}
}

func TestIsJSImportTransformable(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		want       bool
	}{
		{
			name:       "@/ path is transformable",
			importPath: "@/lib/utils.js",
			want:       true,
		},
		{
			name:       "@/ path with deep nesting is transformable",
			importPath: "@/a/b/c/d/e.js",
			want:       true,
		},
		{
			name:       "relative path is not transformable",
			importPath: "./local.js",
			want:       false,
		},
		{
			name:       "absolute path is not transformable",
			importPath: "/absolute/path.js",
			want:       false,
		},
		{
			name:       "external URL is not transformable",
			importPath: "https://example.com/lib.js",
			want:       false,
		},
		{
			name:       "bare module specifier is not transformable",
			importPath: "lodash",
			want:       false,
		},
		{
			name:       "@ without slash is not transformable",
			importPath: "@scope/package",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJSImportTransformable(tt.importPath)
			assert.Equal(t, tt.want, got)
		})
	}
}
