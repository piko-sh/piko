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

package inventory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"piko.sh/json"
)

var (
	// roots selects authored fixture roots while excluding generated copies.
	roots = []sourceRoot{
		{pattern: "examples/scenarios/*/src", group: "examples"},
		{pattern: "components", group: "shipped-components"},
		{pattern: "internal/compiler/compiler_test/testdata/*", group: "compiler"},
		{pattern: "internal/generator/generator_test/generator/testdata/*/src", group: "generator"},
		{pattern: "tests/integration/*/testdata/*/src", group: "integration"},
	}
)

// Corpus groups source inventories by authored project population.
type Corpus struct {
	// Populations groups source inventories by project population.
	Populations map[string]*Population
}

// sourceRoot selects a set of authored fixture directories.
type sourceRoot struct {
	// pattern is the glob that matches authored fixture directories.
	pattern string

	// group names the population the matched fixtures belong to.
	group string
}

// fixtureSpecification holds the failure expectations of an authored fixture.
type fixtureSpecification struct {
	// ShouldError reports that the fixture expects a runtime failure.
	ShouldError bool

	// ShouldGenerateError reports that the fixture expects a generation failure.
	ShouldGenerateError bool

	// ShouldCompileError reports that the fixture expects a compilation failure.
	ShouldCompileError bool
}

// Repository inventories the repository's authored source populations.
//
// Takes root (fs.FS) which exposes the source tree to inventory.
//
// Returns Corpus which groups source-feature and composition counts by authored
// population.
// Returns error which reports authored-source or specification access and decoding
// failure, or nil on success.
func Repository(ctx context.Context, root fs.FS) (Corpus, error) {
	result := Corpus{Populations: map[string]*Population{}}
	for _, set := range roots {
		directories, err := fs.Glob(root, set.pattern)
		if err != nil {
			return result, fmt.Errorf("matching %s: %w", set.pattern, err)
		}
		for _, base := range directories {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			if err := addFixture(ctx, root, result.Populations, set.group, base); err != nil {
				return result, err
			}
		}
	}
	return result, nil
}

// addFixture inventories one authored fixture directory into its population.
//
// Takes root (fs.FS) which exposes the source tree to inventory.
// Takes populations (map[string]*Population) which receives the fixture's project.
// Takes group (string) which names the fixture root's population.
// Takes base (string) which is the fixture's source directory.
//
// Returns error which reports authored-source or specification access and decoding
// failure, or nil on success.
func addFixture(ctx context.Context, root fs.FS, populations map[string]*Population, group, base string) error {
	info, err := fs.Stat(root, base)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}
	fixture := base
	if path.Base(base) == "src" {
		fixture = path.Dir(base)
	}
	negative, err := negativeFixture(root, fixture)
	if err != nil {
		return err
	}
	if negative {
		group += "-expected-error"
	}
	var current *project
	err = fs.WalkDir(root, base, func(name string, entry fs.DirEntry, walkErr error) error {
		source, ok, err := readFixtureSource(ctx, root, name, entry, walkErr)
		if err != nil || !ok {
			return err
		}
		if current == nil {
			population := populations[group]
			if population == nil {
				population = &Population{}
				populations[group] = population
			}
			current = population.project(base)
		}
		return current.file(ctx, strings.TrimPrefix(name, base+"/"), source)
	})
	if err != nil {
		return fmt.Errorf("%s: %w", base, err)
	}
	if current != nil {
		current.finish()
	}
	return nil
}

// readFixtureSource reads one walked fixture entry when it is authored source.
//
// Takes root (fs.FS) which exposes the source tree to inventory.
// Takes name (string) which is the walked entry's path.
// Takes entry (fs.DirEntry) which describes the walked entry.
// Takes walkErr (error) which is the walk's error for the entry.
//
// Returns string which holds the authored source.
// Returns bool which reports whether the entry is authored source to inventory.
// Returns error which reports a walk or read failure, cancellation, or fs.SkipDir for an
// excluded directory.
func readFixtureSource(ctx context.Context, root fs.FS, name string, entry fs.DirEntry, walkErr error) (string, bool, error) {
	if walkErr != nil {
		return "", false, walkErr
	}
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	if entry.IsDir() {
		if excludedDirectory(entry.Name()) {
			return "", false, fs.SkipDir
		}
		return "", false, nil
	}
	if !entry.Type().IsRegular() || !isSource(name) || strings.HasPrefix(entry.Name(), "golden.") || strings.HasPrefix(entry.Name(), "gen_") {
		return "", false, nil
	}
	source, err := fs.ReadFile(root, name)
	if err != nil {
		return "", false, err
	}
	if strings.HasSuffix(name, ".go") && generatedGo(source) {
		return "", false, nil
	}
	return string(source), true, nil
}

// excludedDirectory reports whether a fixture directory holds generated or hidden files.
//
// Takes name (string) which is the directory's base name.
//
// Returns bool which reports whether the walk skips the directory.
func excludedDirectory(name string) bool {
	switch name {
	case "dist", "vendor", "golden", "bin", "out", "node_modules", "testdata":
		return true
	}
	return strings.HasPrefix(name, ".")
}

// hasAuthoredFiles reports whether a directory holds any file outside generated or hidden
// directories.
//
// A directory left behind with only build output, such as bin or .piko, holds no authored
// files.
//
// Takes root (fs.FS) which exposes the source tree to inventory.
// Takes directory (string) which is the directory to search.
//
// Returns bool which is true once an authored file is found.
// Returns error which reports an unreadable directory, or nil on success.
func hasAuthoredFiles(root fs.FS, directory string) (bool, error) {
	found := false
	err := fs.WalkDir(root, directory, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if current != directory && excludedDirectory(entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		found = true
		return fs.SkipAll
	})
	if err != nil {
		return false, fmt.Errorf("searching %s for authored files: %w", directory, err)
	}
	return found, nil
}

// negativeFixture reports whether a fixture intentionally expects a failure.
//
// Takes root (fs.FS) which exposes the source tree to inventory.
// Takes fixture (string) which is the authored fixture directory.
//
// Returns bool which reports whether the fixture expects a deliberate failure.
// Returns error which reports authored-source or specification access and decoding
// failure, or nil on success.
func negativeFixture(root fs.FS, fixture string) (bool, error) {
	filename := path.Join(fixture, "testspec.json")
	data, err := fs.ReadFile(root, filename)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", filename, err)
	}
	var spec fixtureSpecification
	if err := json.Unmarshal(data, &spec); err != nil {
		return false, fmt.Errorf("%s: %w", filename, err)
	}
	return spec.ShouldError || spec.ShouldGenerateError || spec.ShouldCompileError, nil
}

// generatedGo reports whether the source has Go's generated-code marker.
//
// Takes source ([]byte) which holds the Go source bytes.
//
// Returns bool which reports whether the source contains the standard generated marker.
func generatedGo(source []byte) bool {
	for line := range bytes.SplitSeq(source, []byte("\n")) {
		if bytes.HasPrefix(line, []byte("package ")) {
			break
		}
		if bytes.HasPrefix(line, []byte("// Code generated ")) && bytes.HasSuffix(bytes.TrimSpace(line), []byte(" DO NOT EDIT.")) {
			return true
		}
	}
	return false
}
