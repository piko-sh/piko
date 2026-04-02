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

import "strings"

// IsInternalPackage checks if a package path contains an internal component.
// In Go, a directory named "internal" is special: packages within it can only
// be imported by packages rooted at the parent of the internal directory.
//
// Uses a single-pass algorithm: one strings.Index call followed by boundary
// checks, rather than multiple string operations.
//
// Takes packagePath (string) which is the package path to check.
//
// Returns bool which is true if the path contains "internal" as a path
// component (e.g., "/internal/", "/internal", "internal/", or exactly
// "internal").
func IsInternalPackage(packagePath string) bool {
	const target = "internal"
	const targetLen = 8

	index := strings.Index(packagePath, target)
	if index == -1 {
		return false
	}

	if index != 0 && packagePath[index-1] != '/' {
		return false
	}

	end := index + targetLen
	return end == len(packagePath) || packagePath[end] == '/'
}

// CanAccessInternalPackage checks if a module can access an internal package
// based on path analysis alone.
//
// This function handles two cases:
//   - User's own internal packages -> accessible (direct imports allowed)
//   - Stdlib internal packages -> filtered (truly unreachable, e.g.
//     crypto/internal)
//
// For third-party internal packages, use ShouldIncludeInternalPackage which
// takes a hasModule parameter for more reliable detection.
//
// Takes userModulePath (string) which is the user's module path (e.g.
// "nd-estates-website" or "github.com/foo/bar").
// Takes targetPath (string) which is the package path being checked.
//
// Returns bool which is true if targetPath is the user's own internal package.
func CanAccessInternalPackage(userModulePath, targetPath string) bool {
	if !IsInternalPackage(targetPath) {
		return true
	}

	requiredPrefix, _, found := strings.Cut(targetPath, "/internal")
	if !found {
		return false
	}

	return strings.HasPrefix(userModulePath, requiredPrefix+"/") ||
		userModulePath == requiredPrefix
}

// ShouldIncludeInternalPackage checks if an internal package should be used
// in type analysis. This is the main filter used during package discovery.
//
// The rules are:
//   - Non-internal packages are always included.
//   - The user's own internal packages are included.
//   - Third-party internal packages are included for type alias resolution.
//   - Standard library internal packages are excluded.
//
// Third-party libraries often have public types that are aliases to internal
// types. For example, piko.RequestData may be an alias to an internal type.
// Without including those internal packages, method resolution would fail.
//
// Takes userModulePath (string) which is the module path of the user's code.
// Takes targetPath (string) which is the package path to check.
// Takes hasModule (bool) which is true for third-party packages with a go.mod,
// or false for standard library packages.
//
// Returns bool which is true if the package should be included.
func ShouldIncludeInternalPackage(userModulePath, targetPath string, hasModule bool) bool {
	if !IsInternalPackage(targetPath) {
		return true
	}

	requiredPrefix, _, found := strings.Cut(targetPath, "/internal")
	if !found {
		return false
	}

	if strings.HasPrefix(userModulePath, requiredPrefix+"/") || userModulePath == requiredPrefix {
		return true
	}

	return hasModule
}
