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

const (
	// vimProbeFileName is the name vim writes when probing whether a directory is writable;
	// it is created and immediately removed and must never be treated as a source file.
	vimProbeFileName = "4913"

	// sedTempPrefix prefixes the six random characters in a GNU sed temporary file name.
	sedTempPrefix = "sed"

	// sedTempRandomLength is the number of random characters after sedTempPrefix.
	sedTempRandomLength = 6

	// jetBrainsTempMarker appears in the names JetBrains IDEs use for safe-write temporary
	// and backup files, such as "logo.svg___jb_tmp___" and "logo.svg___jb_old___".
	jetBrainsTempMarker = "___jb_"

	// jetBrainsTempSuffix ends JetBrains safe-write temporary and backup file names.
	jetBrainsTempSuffix = "___"
)

// IsTemporaryFile reports editor temporary and backup paths ignored by discovery and
// watching.
//
// Takes path (string) which is the file path or name to classify.
//
// Returns bool which is true when the path is a temporary or backup file.
func IsTemporaryFile(path string) bool {
	base := filepath.Base(path)
	if base == vimProbeFileName || isSedTempFileName(base) {
		return true
	}
	if strings.HasPrefix(base, ".#") || (len(base) > 2 && strings.HasPrefix(base, "#") && strings.HasSuffix(base, "#")) {
		return true
	}
	if strings.Contains(base, jetBrainsTempMarker) && strings.HasSuffix(base, jetBrainsTempSuffix) {
		return true
	}
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, "~") ||
		strings.HasSuffix(lower, ".swp") ||
		strings.HasSuffix(lower, ".tmp")
}

// isSedTempFileName recognises "sed" followed by six ASCII letters or digits without an
// extension.
//
// At least one suffix character must be uppercase or a digit to avoid matching ordinary
// lowercase words.
//
// Takes base (string) which is the file name without its directory.
//
// Returns bool which is true when the name matches.
func isSedTempFileName(base string) bool {
	suffix, found := strings.CutPrefix(base, sedTempPrefix)
	if !found || len(suffix) != sedTempRandomLength {
		return false
	}
	hasUpperOrDigit := false
	for _, character := range suffix {
		isLower := character >= 'a' && character <= 'z'
		isUpperOrDigit := (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9')
		if !isLower && !isUpperOrDigit {
			return false
		}
		hasUpperOrDigit = hasUpperOrDigit || isUpperOrDigit
	}
	return hasUpperOrDigit
}

// isRelevantFileForProcessing checks if a file should be processed by the Piko build
// system based on its extension and directory location.
//
// Acts as the single source of truth used by both the initial file seeder and the live
// file watcher.
//
// Takes relPath (string) which is the path to the file relative to the project root.
// Takes paths (*LifecyclePathsConfig) which specifies the source directories.
//
// Returns bool which is true if the file matches the processing rules.
func isRelevantFileForProcessing(relPath string, paths *LifecyclePathsConfig) bool {
	if IsTemporaryFile(relPath) {
		return false
	}

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

// isCoreSourceFile checks if a file change should trigger a full project rebuild (handled
// by the coordinator) rather than a simple asset update (handled by the orchestrator).
//
// Takes relPath (string) which is the path to the file being checked.
// Takes paths (*LifecyclePathsConfig) which provides the source folder paths.
//
// Returns bool which is true if the file is a core source file that needs a full rebuild.
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

// hasPrefix checks whether a path starts with a given directory prefix. It adds a
// trailing slash to the prefix if needed, so it matches full directory names rather than
// partial strings.
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
