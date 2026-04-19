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

package lifecycle_domain

import (
	"path/filepath"
	"strings"
)

// isRelevantFileForProcessing checks if a file should be processed by the
// Piko build system based on its extension and directory location.
//
// Acts as the single source of truth used by both the initial file seeder and the
// live file watcher.
//
// Takes relPath (string) which is the path to the file relative to the project
// root.
// Takes paths (*LifecyclePathsConfig) which specifies the source directories.
//
// Returns bool which is true if the file matches the processing rules.
func isRelevantFileForProcessing(relPath string, paths *LifecyclePathsConfig) bool {
	ext := strings.ToLower(filepath.Ext(relPath))

	if ext == ".go" && !strings.HasSuffix(relPath, "_test.go") {
		return true
	}

	if paths.AssetsSourceDir != "" && hasPrefix(relPath, paths.AssetsSourceDir) {
		return true
	}

	switch ext {
	case ".pk":
		return (paths.PagesSourceDir != "" && hasPrefix(relPath, paths.PagesSourceDir)) ||
			(paths.PartialsSourceDir != "" && hasPrefix(relPath, paths.PartialsSourceDir))
	case ".pkc":
		return paths.ComponentsSourceDir != "" && hasPrefix(relPath, paths.ComponentsSourceDir)
	case ".json":
		return paths.I18nSourceDir != "" && hasPrefix(relPath, paths.I18nSourceDir)
	}

	return false
}

// isCoreSourceFile checks if a file change should trigger a full project
// rebuild (handled by the coordinator) rather than a simple asset update
// (handled by the orchestrator).
//
// Takes relPath (string) which is the path to the file being checked.
// Takes paths (*LifecyclePathsConfig) which provides the source folder paths.
//
// Returns bool which is true if the file is a core source file that needs a
// full rebuild.
func isCoreSourceFile(relPath string, paths *LifecyclePathsConfig) bool {
	ext := strings.ToLower(filepath.Ext(relPath))

	if ext == ".go" {
		return true
	}

	switch ext {
	case ".pk":
		return hasPrefix(relPath, paths.PagesSourceDir) || hasPrefix(relPath, paths.PartialsSourceDir)
	case ".pkc":
		return hasPrefix(relPath, paths.ComponentsSourceDir)
	case ".json":
		return hasPrefix(relPath, paths.I18nSourceDir)
	}

	return false
}

// hasPrefix checks whether a path starts with a given directory prefix.
// It adds a trailing slash to the prefix if needed, so it matches full
// directory names rather than partial strings.
//
// Takes path (string) which is the file path to check.
// Takes prefix (string) which is the directory prefix to match against.
//
// Returns bool which is true if the path is within the given directory.
func hasPrefix(path, prefix string) bool {
	if prefix == "" {
		return false
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return strings.HasPrefix(path, prefix)
}
