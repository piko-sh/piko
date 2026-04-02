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

package annotator_domain

// Parses .pikoignore files and matches file paths against ignore patterns to exclude files from compilation.
// Supports gitignore-style patterns including wildcards, negation, and directory matching for flexible file filtering.

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"piko.sh/piko/wdk/safedisk"
)

// defaultIgnorePatterns is a list of common files and directories to ignore.
var defaultIgnorePatterns = []string{
	".git", ".svn", ".hg", ".bzr",

	"dist", "node_modules", "vendor",

	"__pycache__", ".venv", "venv", ".tox", ".eggs", "*.egg-info",
	"vendor/bundle", ".bundle",
	"_build", "deps",
	"target", "build", ".gradle",
	"bin", "obj",

	".idea", ".vscode", ".vs",
	"*.swp", "*.swo",

	".DS_Store", "Thumbs.db",

	"coverage", "logs",
}

// ignoreMatcher holds glob patterns and checks if file paths match them.
type ignoreMatcher struct {
	// patterns holds glob patterns to match against file names.
	patterns []string
}

// ignoreMatcherOption configures an ignoreMatcher during construction.
type ignoreMatcherOption func(*ignoreMatcherConfig)

// ignoreMatcherConfig holds configuration for creating an ignoreMatcher.
type ignoreMatcherConfig struct {
	// sandboxFactory creates sandboxes when no sandbox is directly injected.
	// When non-nil and sandbox is nil, this factory is used instead of
	// safedisk.NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// sandbox is an optional sandbox for testing file operations.
	sandbox safedisk.Sandbox
}

// Matches checks if a given path (file or directory name) matches any of the
// ignore patterns.
//
// Takes name (string) which is the file or directory name to check.
//
// Returns bool which is true if the name matches any pattern.
func (m *ignoreMatcher) Matches(name string) bool {
	for _, pattern := range m.patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}

// withIgnoreMatcherSandbox injects a sandbox for testing .pikoignore file
// parsing. The caller is responsible for closing the sandbox.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access.
//
// Returns ignoreMatcherOption which configures the matcher with the given
// sandbox.
func withIgnoreMatcherSandbox(sandbox safedisk.Sandbox) ignoreMatcherOption {
	return func(c *ignoreMatcherConfig) {
		c.sandbox = sandbox
	}
}

// newIgnoreMatcher creates a new matcher from several sources of patterns.
// It combines default patterns, patterns from a .pikoignore file, and
// user-set patterns.
//
// Takes baseDir (string) which is the directory containing the .pikoignore
// file.
// Takes configPatterns ([]string) which provides user-set ignore patterns.
// Takes opts (...ignoreMatcherOption) which provides optional configuration
// such as withIgnoreMatcherSandbox for testing.
//
// Returns *ignoreMatcher which is ready to match paths against all combined
// patterns.
func newIgnoreMatcher(baseDir string, configPatterns []string, opts ...ignoreMatcherOption) *ignoreMatcher {
	config := &ignoreMatcherConfig{}
	for _, opt := range opts {
		opt(config)
	}

	finalPatterns := make([]string, 0)

	finalPatterns = append(finalPatterns, defaultIgnorePatterns...)

	ignoreFilePath := filepath.Join(baseDir, ".pikoignore")
	if filePatterns, err := parseIgnoreFile(ignoreFilePath, config.sandbox, config.sandboxFactory); err == nil {
		finalPatterns = append(finalPatterns, filePatterns...)
	}

	finalPatterns = append(finalPatterns, configPatterns...)

	return &ignoreMatcher{patterns: finalPatterns}
}

// parseIgnoreFile reads a gitignore-style file and returns the patterns it
// contains. It skips empty lines and comment lines (those starting with #).
//
// When the file does not exist, returns nil without error.
//
// Takes path (string) which specifies the file to read.
// Takes injectedSandbox (safedisk.Sandbox) which is an optional sandbox for
// testing. When nil, a sandbox is created for the file's parent directory.
// Takes factory (safedisk.Factory) which is an optional factory for creating
// sandboxes. When non-nil and injectedSandbox is nil, this factory is used
// instead of safedisk.NewNoOpSandbox.
//
// Returns []string which contains the parsed patterns from the file.
// Returns error when the file cannot be read.
func parseIgnoreFile(path string, injectedSandbox safedisk.Sandbox, factory safedisk.Factory) ([]string, error) {
	cleanPath := filepath.Clean(path)
	fileName := filepath.Base(cleanPath)

	sandbox, sandboxOwned := resolveIgnoreSandbox(cleanPath, injectedSandbox, factory)
	if sandbox == nil {
		return nil, nil
	}
	if sandboxOwned {
		defer func() { _ = sandbox.Close() }()
	}

	file, err := sandbox.Open(fileName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening ignore file %q: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	var patterns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("scanning ignore file %q: %w", path, err)
	}
	return patterns, nil
}

// resolveIgnoreSandbox returns the sandbox to use for reading the ignore file,
// preferring an injected sandbox and falling back to a factory or no-op sandbox.
//
// Returns (nil, false) when sandbox creation fails, signalling the caller to
// skip the file silently.
//
// Takes cleanPath (string) which is the cleaned path used to derive the parent
// directory.
// Takes injectedSandbox (safedisk.Sandbox) which is an optional pre-built
// sandbox.
// Takes factory (safedisk.Factory) which creates sandboxes when
// injectedSandbox is nil.
//
// Returns safedisk.Sandbox which provides filesystem access, or nil on failure.
// Returns bool which is true when the caller owns (and must close) the sandbox.
func resolveIgnoreSandbox(cleanPath string, injectedSandbox safedisk.Sandbox, factory safedisk.Factory) (safedisk.Sandbox, bool) {
	if injectedSandbox != nil {
		return injectedSandbox, false
	}
	parentDir := filepath.Dir(cleanPath)
	if factory != nil {
		sandbox, err := factory.Create("pikoignore", parentDir, safedisk.ModeReadOnly)
		if err != nil {
			return nil, false
		}
		return sandbox, true
	}
	sandbox, err := safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadOnly)
	if err != nil {
		return nil, false
	}
	return sandbox, true
}
