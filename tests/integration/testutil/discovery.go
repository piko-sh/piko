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

package testutil

import (
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// TestCase holds information about a discovered test case.
type TestCase struct {
	// Name is the test case name (the directory name without its numeric prefix).
	Name string

	// Path is the full file path to the test case directory.
	Path string
}

// EntryPoint represents a template file to be compiled.
type EntryPoint struct {
	// Path is the Piko module path, such as "testmodule/pages/main.pk".
	Path string

	// IsPage indicates whether this entry point is a full page rather than a partial.
	IsPage bool

	// IsPublic indicates whether this entry point is publicly accessible.
	IsPublic bool
}

// DiscoverTestCases scans a testdata directory for test case directories.
// It expects directories named with a numeric prefix (e.g., "01_some_test").
//
// Takes testdataRoot (string) which specifies the path to the testdata
// directory to scan.
//
// Returns []TestCase which contains the discovered test cases sorted by their
// numeric prefix.
// Returns error when the directory cannot be read or a path cannot be resolved.
func DiscoverTestCases(testdataRoot string) ([]TestCase, error) {
	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		return nil, err
	}

	testCases := make([]TestCase, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) == 0 || name[0] < '0' || name[0] > '9' {
			continue
		}

		absPath, err := filepath.Abs(filepath.Join(testdataRoot, name))
		if err != nil {
			return nil, err
		}

		testCases = append(testCases, TestCase{
			Name: name,
			Path: absPath,
		})
	}

	slices.SortFunc(testCases, func(a, b TestCase) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return testCases, nil
}

// DiscoverEntryPoints scans a source directory for .pk template files.
// It returns entry points with their paths relative to the module root.
//
// Takes srcDir (string) which specifies the root source directory to scan.
// Takes moduleName (string) which identifies the module for path resolution.
// Takes pagesDir (string) which specifies the subdirectory containing pages.
// Takes partialsDir (string) which specifies the subdirectory containing
// partials.
//
// Returns []EntryPoint which contains discovered templates from both pages
// and partials directories.
// Returns error when a directory cannot be read.
func DiscoverEntryPoints(srcDir, moduleName, pagesDir, partialsDir string) ([]EntryPoint, error) {
	var entryPoints []EntryPoint

	pageEntries, err := discoverEntryPointsInDir(srcDir, moduleName, pagesDir, true)
	if err != nil {
		return nil, err
	}
	entryPoints = append(entryPoints, pageEntries...)

	partialEntries, err := discoverEntryPointsInDir(srcDir, moduleName, partialsDir, false)
	if err != nil {
		return nil, err
	}
	entryPoints = append(entryPoints, partialEntries...)

	return entryPoints, nil
}

// DiscoverAssetFiles scans a directory for asset files (SVG, PNG, etc.).
//
// Takes srcDir (string) which specifies the root directory to scan.
// Takes extensions ([]string) which lists the file extensions to match.
//
// Returns []string which contains the paths of all matching asset files.
// Returns error when the directory cannot be walked.
func DiscoverAssetFiles(srcDir string, extensions []string) ([]string, error) {
	var assets []string

	extSet := make(map[string]bool)
	for _, ext := range extensions {
		extSet[strings.ToLower(ext)] = true
	}

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if extSet[ext] {
			assets = append(assets, path)
		}
		return nil
	})

	return assets, err
}

// discoverEntryPointsInDir scans a specific subdirectory for .pk template files.
//
// Takes srcDir (string) which is the root source directory path.
// Takes moduleName (string) which identifies the module being scanned.
// Takes subDir (string) which specifies the subdirectory to scan within srcDir.
// Takes isPotentiallyPage (bool) which indicates if found entries may be pages.
//
// Returns []EntryPoint which contains the discovered template entry points.
// Returns error when the directory walk fails.
func discoverEntryPointsInDir(srcDir, moduleName, subDir string, isPotentiallyPage bool) ([]EntryPoint, error) {
	sourceRoot := filepath.Join(srcDir, subDir)
	if _, err := os.Stat(sourceRoot); os.IsNotExist(err) {
		return nil, nil
	}

	var entryPoints []EntryPoint
	err := filepath.WalkDir(sourceRoot, func(absPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".pk") {
			return nil
		}

		entry := buildEntryPoint(srcDir, moduleName, absPath, isPotentiallyPage)
		entryPoints = append(entryPoints, entry)
		return nil
	})

	return entryPoints, err
}

// buildEntryPoint creates an EntryPoint for a discovered .pk file.
//
// Takes srcDir (string) which is the source directory base path.
// Takes moduleName (string) which is the module name for the piko path.
// Takes absPath (string) which is the absolute path to the .pk file.
// Takes isPotentiallyPage (bool) which indicates if the file may be a page.
//
// Returns EntryPoint which contains the computed path and visibility flags.
func buildEntryPoint(srcDir, moduleName, absPath string, isPotentiallyPage bool) EntryPoint {
	relPathToBase, _ := filepath.Rel(srcDir, absPath)
	pikoPath := filepath.ToSlash(filepath.Join(moduleName, relPathToBase))

	return EntryPoint{
		Path:     pikoPath,
		IsPage:   isPotentiallyPage,
		IsPublic: isPotentiallyPage,
	}
}
