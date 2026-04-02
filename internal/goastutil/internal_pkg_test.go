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

package goastutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsInternalPackage(t *testing.T) {
	testCases := []struct {
		name        string
		packagePath string
		expected    bool
	}{
		{
			name:        "contains internal in middle",
			packagePath: "github.com/foo/bar/internal/util",
			expected:    true,
		},
		{
			name:        "ends with internal",
			packagePath: "github.com/foo/bar/internal",
			expected:    true,
		},
		{
			name:        "is just internal",
			packagePath: "internal",
			expected:    true,
		},
		{
			name:        "starts with internal",
			packagePath: "internal/foo",
			expected:    true,
		},
		{
			name:        "stdlib internal",
			packagePath: "crypto/internal/boring",
			expected:    true,
		},
		{
			name:        "no internal component",
			packagePath: "github.com/foo/bar/pkg/util",
			expected:    false,
		},
		{
			name:        "internal as substring of name",
			packagePath: "github.com/foo/internals/util",
			expected:    false,
		},
		{
			name:        "empty path",
			packagePath: "",
			expected:    false,
		},
		{
			name:        "simple stdlib package",
			packagePath: "fmt",
			expected:    false,
		},
		{
			name:        "nested stdlib package",
			packagePath: "encoding/json",
			expected:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsInternalPackage(tc.packagePath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCanAccessInternalPackage(t *testing.T) {
	testCases := []struct {
		name           string
		userModulePath string
		targetPath     string
		expected       bool
	}{

		{
			name:           "non-internal package",
			userModulePath: "github.com/foo/bar",
			targetPath:     "github.com/foo/bar/pkg/util",
			expected:       true,
		},
		{
			name:           "stdlib package",
			userModulePath: "nd-estates-website",
			targetPath:     "fmt",
			expected:       true,
		},
		{
			name:           "external package",
			userModulePath: "nd-estates-website",
			targetPath:     "piko.sh/piko/pkg/util",
			expected:       true,
		},

		{
			name:           "own internal package exact match",
			userModulePath: "github.com/foo/bar",
			targetPath:     "github.com/foo/bar/internal/util",
			expected:       true,
		},
		{
			name:           "own internal package nested",
			userModulePath: "github.com/foo/bar",
			targetPath:     "github.com/foo/bar/internal/deep/nested",
			expected:       true,
		},
		{
			name:           "own internal package simple module",
			userModulePath: "nd-estates-website",
			targetPath:     "nd-estates-website/internal/helpers",
			expected:       true,
		},

		{
			name:           "external module internal",
			userModulePath: "github.com/foo/bar",
			targetPath:     "github.com/other/pkg/internal/util",
			expected:       false,
		},
		{
			name:           "piko internal from nd-estates",
			userModulePath: "nd-estates-website",
			targetPath:     "piko.sh/piko/internal/ast",
			expected:       false,
		},
		{
			name:           "sibling module internal",
			userModulePath: "github.com/foo/bar",
			targetPath:     "github.com/foo/baz/internal/util",
			expected:       false,
		},

		{
			name:           "stdlib internal crypto",
			userModulePath: "nd-estates-website",
			targetPath:     "crypto/internal/boring",
			expected:       false,
		},
		{
			name:           "stdlib internal runtime",
			userModulePath: "github.com/foo/bar",
			targetPath:     "runtime/internal/sys",
			expected:       false,
		},
		{
			name:           "stdlib internal reflect",
			userModulePath: "myapp",
			targetPath:     "reflect/internal/example",
			expected:       false,
		},

		{
			name:           "root internal package",
			userModulePath: "myapp",
			targetPath:     "internal",
			expected:       false,
		},
		{
			name:           "root internal subpackage",
			userModulePath: "myapp",
			targetPath:     "internal/foo",
			expected:       false,
		},

		{
			name:           "empty module path with internal target",
			userModulePath: "",
			targetPath:     "github.com/foo/internal/bar",
			expected:       false,
		},
		{
			name:           "empty module path with non-internal target",
			userModulePath: "",
			targetPath:     "github.com/foo/bar",
			expected:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CanAccessInternalPackage(tc.userModulePath, tc.targetPath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestShouldIncludeInternalPackage(t *testing.T) {
	testCases := []struct {
		name           string
		userModulePath string
		targetPath     string
		hasModule      bool
		expected       bool
	}{

		{
			name:           "non-internal package",
			userModulePath: "github.com/foo/bar",
			targetPath:     "github.com/foo/bar/pkg/util",
			hasModule:      true,
			expected:       true,
		},
		{
			name:           "stdlib package",
			userModulePath: "nd-estates-website",
			targetPath:     "fmt",
			hasModule:      false,
			expected:       true,
		},

		{
			name:           "own internal package",
			userModulePath: "github.com/foo/bar",
			targetPath:     "github.com/foo/bar/internal/util",
			hasModule:      true,
			expected:       true,
		},

		{
			name:           "third-party internal with module - included for type aliases",
			userModulePath: "github.com/foo/bar",
			targetPath:     "github.com/other/pkg/internal/util",
			hasModule:      true,
			expected:       true,
		},
		{
			name:           "piko internal from user project - included for type aliases",
			userModulePath: "nd-estates-website",
			targetPath:     "piko.sh/piko/internal/templater/templater_dto",
			hasModule:      true,
			expected:       true,
		},

		{
			name:           "stdlib internal crypto - filtered",
			userModulePath: "nd-estates-website",
			targetPath:     "crypto/internal/boring",
			hasModule:      false,
			expected:       false,
		},
		{
			name:           "stdlib internal runtime - filtered",
			userModulePath: "github.com/foo/bar",
			targetPath:     "runtime/internal/sys",
			hasModule:      false,
			expected:       false,
		},

		{
			name:           "root internal package",
			userModulePath: "myapp",
			targetPath:     "internal",
			hasModule:      false,
			expected:       false,
		},
		{
			name:           "root internal subpackage",
			userModulePath: "myapp",
			targetPath:     "internal/foo",
			hasModule:      false,
			expected:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ShouldIncludeInternalPackage(tc.userModulePath, tc.targetPath, tc.hasModule)
			assert.Equal(t, tc.expected, result)
		})
	}
}
