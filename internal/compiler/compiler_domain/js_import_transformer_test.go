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
		{
			name:             "@/ path without extension gets .js appended",
			moduleName:       "github.com/user/project",
			importPath:       "@/lib/svg-animations",
			wantPath:         "/_piko/assets/github.com/user/project/lib/svg-animations.js",
			wantDependency:   true,
			wantResolvedPath: "github.com/user/project/lib/svg-animations.js",
			wantServedPath:   "/_piko/assets/github.com/user/project/lib/svg-animations.js",
		},
		{
			name:             "@/ path with .ts extension gets .js",
			moduleName:       "github.com/user/project",
			importPath:       "@/lib/svg-animations.ts",
			wantPath:         "/_piko/assets/github.com/user/project/lib/svg-animations.js",
			wantDependency:   true,
			wantResolvedPath: "github.com/user/project/lib/svg-animations.js",
			wantServedPath:   "/_piko/assets/github.com/user/project/lib/svg-animations.js",
		},
		{
			name:             "@/ path with .css extension stays unchanged",
			moduleName:       "github.com/user/project",
			importPath:       "@/styles/theme.css",
			wantPath:         "/_piko/assets/github.com/user/project/styles/theme.css",
			wantDependency:   true,
			wantResolvedPath: "github.com/user/project/styles/theme.css",
			wantServedPath:   "/_piko/assets/github.com/user/project/styles/theme.css",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			gotPath, gotDep := TransformJSImportPath(ctx, tt.importPath, tt.moduleName)

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
