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

package assetpath

import (
	"path/filepath"
	"strings"
)

const (
	// DefaultServePath is the default URL path prefix for serving compiled
	// assets. Consumers with config access may use
	// bootstrap.ServerConfig.Paths.ArtefactServePath instead.
	DefaultServePath = "/_piko/assets"

	// ModuleAliasPrefix is the @/ prefix used for module-relative asset paths.
	ModuleAliasPrefix = "@/"
)

// NeedsTransform reports whether a source path requires asset pipeline
// transformation. Returns false for empty strings, absolute URLs (http://,
// https://), data URIs, protocol-relative URLs (//), absolute paths (/), and
// paths already prefixed with servePath.
//
// Takes src (string) which is the source path to check.
// Takes servePath (string) which is the asset serve path prefix to detect
// already-transformed paths.
//
// Returns bool which is true if the path needs asset pipeline transformation.
func NeedsTransform(src, servePath string) bool {
	if src == "" {
		return false
	}
	if strings.HasPrefix(src, servePath) {
		return false
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return false
	}
	if strings.HasPrefix(src, "//") {
		return false
	}
	if strings.HasPrefix(src, "data:") {
		return false
	}
	if strings.HasPrefix(src, "/") {
		return false
	}
	return true
}

// NeedsCleaning reports whether a path requires filepath.Clean. A path needs
// cleaning if it contains "./", "..", "//", or ends with "/".
//
// Takes src (string) which is the path to check.
//
// Returns bool which is true if the path has patterns that need cleaning.
func NeedsCleaning(src string) bool {
	return strings.Contains(src, "./") ||
		strings.Contains(src, "..") ||
		strings.Contains(src, "//") ||
		strings.HasSuffix(src, "/")
}

// ResolveModuleAlias resolves the @/ path alias by replacing it with
// moduleName + "/". If src does not start with @/ or moduleName is empty, src
// is returned unchanged.
//
// Takes src (string) which is the source path that may contain @/ prefix.
// Takes moduleName (string) which is the Go module name for alias resolution.
//
// Returns string which is the resolved path with @/ replaced by module name.
func ResolveModuleAlias(src, moduleName string) string {
	if !strings.HasPrefix(src, ModuleAliasPrefix) {
		return src
	}
	if moduleName == "" {
		return src
	}
	return moduleName + "/" + strings.TrimPrefix(src, ModuleAliasPrefix)
}

// Transform applies the full asset path transformation pipeline: checks
// whether transformation is needed, resolves the @/ module alias, cleans the
// path if necessary, and prepends the serve path. Returns src unchanged if no
// transformation is needed.
//
// Takes src (string) which is the original source path.
// Takes moduleName (string) which is the Go module name for @/ alias
// resolution.
// Takes servePath (string) which is the URL path prefix for serving assets.
//
// Returns string which is the transformed source path.
func Transform(src, moduleName, servePath string) string {
	if !NeedsTransform(src, servePath) {
		return src
	}

	resolved := ResolveModuleAlias(src, moduleName)

	if NeedsCleaning(resolved) {
		resolved = filepath.Clean(resolved)
	}

	return servePath + "/" + resolved
}

// AppendTransformed appends a transformed asset source path to buffer
// using a zero-allocation buffer-based API for hot render paths.
//
// It does not perform module alias resolution because the render
// layer receives already-resolved paths.
//
// Takes buffer ([]byte) which is the buffer to append to.
// Takes src (string) which is the asset source path to transform.
// Takes servePath (string) which is the URL path prefix for serving assets.
//
// Returns []byte which is the buffer with the transformed source appended.
func AppendTransformed(buffer []byte, src, servePath string) []byte {
	if !NeedsTransform(src, servePath) {
		return append(buffer, src...)
	}

	buffer = append(buffer, servePath...)
	buffer = append(buffer, '/')

	if NeedsCleaning(src) {
		buffer = append(buffer, filepath.Clean(src)...)
	} else {
		buffer = append(buffer, src...)
	}

	return buffer
}
