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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// StagedEnvironment manages a temporary staging directory for multi-stage
// tests.
type StagedEnvironment struct {
	// t is the test context for reporting failures and logging.
	t *testing.T

	// SrcDir is the original source directory.
	SrcDir string

	// StagedDir is the path to the staging folder where files are changed.
	StagedDir string

	// TestCaseDir is the parent directory that contains the test case.
	TestCaseDir string
}

// NewStagedEnvironment creates a new staged environment by copying base files
// from srcDir to a new staged directory.
//
// Takes t (*testing.T) which is the test context for reporting failures.
// Takes testCaseDir (string) which is the root directory containing source
// files.
//
// Returns *StagedEnvironment which provides access to the staged test files.
func NewStagedEnvironment(t *testing.T, testCaseDir string) *StagedEnvironment {
	t.Helper()

	srcDir := filepath.Join(testCaseDir, "src")
	stagedDir := filepath.Join(testCaseDir, "staged")

	if err := os.RemoveAll(stagedDir); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove existing staged directory: %v", err)
	}

	require.NoError(t, os.MkdirAll(stagedDir, fileModeDir))

	CopyBaseFiles(t, srcDir, stagedDir)

	return &StagedEnvironment{
		t:           t,
		SrcDir:      srcDir,
		StagedDir:   stagedDir,
		TestCaseDir: testCaseDir,
	}
}

// Cleanup removes the staged directory unless KeepStaged is set and the test
// failed.
func (env *StagedEnvironment) Cleanup() {
	if env.t.Failed() && KeepStaged != nil && *KeepStaged {
		env.t.Logf("Test failed, keeping staged directory for inspection: %s", env.StagedDir)
		return
	}
	if err := os.RemoveAll(env.StagedDir); err != nil {
		env.t.Logf("Warning: failed to cleanup staged directory: %v", err)
	}
}

// ApplyStagePatch applies all patch files for a given stage number.
// Patch files are named with a _N suffix (e.g., "main.pk_1" for stage 1).
//
// Takes stageNum (int) which specifies the stage number to apply patches for.
//
// Returns int which is the number of patches applied.
func (env *StagedEnvironment) ApplyStagePatch(stageNum int) int {
	env.t.Helper()

	patchSuffix := fmt.Sprintf("_%d", stageNum)
	patchCount := 0

	err := filepath.WalkDir(env.SrcDir, func(srcPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(srcPath, patchSuffix) {
			return nil
		}

		baseName := strings.TrimSuffix(filepath.Base(srcPath), patchSuffix)

		relDir, err := filepath.Rel(env.SrcDir, filepath.Dir(srcPath))
		if err != nil {
			return err
		}

		destDir := filepath.Join(env.StagedDir, relDir)
		destPath := filepath.Join(destDir, baseName)

		if err := os.MkdirAll(destDir, fileModeDir); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", destDir, err)
		}

		if err := CopyFile(srcPath, destPath); err != nil {
			return fmt.Errorf("failed to copy patch %s -> %s: %w", srcPath, destPath, err)
		}

		env.t.Logf("Applied patch: %s -> %s", filepath.Base(srcPath), baseName)
		patchCount++

		return nil
	})

	require.NoError(env.t, err, "Failed to apply stage patches")
	return patchCount
}

// CopyBaseFiles copies all files without a _N suffix from src to dest.
//
// Takes t (*testing.T) which is the test context for reporting failures.
// Takes srcDir (string) which is the source folder to copy from.
// Takes destDir (string) which is the target folder to copy to.
func CopyBaseFiles(t *testing.T, srcDir, destDir string) {
	t.Helper()

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && IsPatchFile(path) {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, fileModeDir)
		}

		return CopyFile(path, destPath)
	})

	require.NoError(t, err, "Failed to copy base files")
}

// IsPatchFile checks if a file path refers to a patch file.
//
// A patch file has a name ending with _N where N is a number.
//
// Takes path (string) which is the file path to check.
//
// Returns bool which is true if the file is a patch file.
func IsPatchFile(path string) bool {
	base := filepath.Base(path)
	parts := strings.Split(base, "_")
	if len(parts) < 2 {
		return false
	}
	lastPart := parts[len(parts)-1]
	_, err := strconv.Atoi(lastPart)
	return err == nil
}

// CopyFile copies a file from a source path to a destination path.
//
// Takes src (string) which is the path to the file to copy.
// Takes dst (string) which is the path where the file will be saved.
//
// Returns error when the source file cannot be opened, the destination
// folder cannot be created, the destination file cannot be created,
// the copy fails, or the file cannot be synced.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = srcFile.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), fileModeDir); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = dstFile.Close() }()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

// CopyDir recursively copies a directory from src to dest.
//
// Takes src (string) which is the source directory path to copy from.
// Takes dest (string) which is the destination directory path to copy to.
//
// Returns error when the directory cannot be walked or files cannot be copied.
func CopyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dest, relPath)
		if d.IsDir() {
			return os.MkdirAll(destPath, fileModeDir)
		}
		return CopyFile(path, destPath)
	})
}

// DiscoverStages scans a source directory to find all stage numbers.
//
// Takes srcDir (string) which specifies the directory to scan for stage files.
//
// Returns []int which contains the discovered stage numbers in ascending order.
// Returns error when the directory cannot be walked.
func DiscoverStages(srcDir string) ([]int, error) {
	stageSet := make(map[int]struct{})

	err := filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		base := filepath.Base(path)
		parts := strings.Split(base, "_")
		if len(parts) < 2 {
			return nil
		}

		lastPart := parts[len(parts)-1]
		stageNum, err := strconv.Atoi(lastPart)
		if err == nil {
			stageSet[stageNum] = struct{}{}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	stages := make([]int, 0, len(stageSet))
	for stage := range stageSet {
		stages = append(stages, stage)
	}

	for i := 0; i < len(stages); i++ {
		for j := i + 1; j < len(stages); j++ {
			if stages[i] > stages[j] {
				stages[i], stages[j] = stages[j], stages[i]
			}
		}
	}

	return stages, nil
}
