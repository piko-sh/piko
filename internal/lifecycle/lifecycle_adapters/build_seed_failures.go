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

package lifecycle_adapters

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// seedStageWalk marks a seed failure while walking a source directory.
	seedStageWalk = "walk"

	// seedStageRead marks a seed failure while resolving or opening a source file.
	seedStageRead = "read"

	// seedStageUpsert marks a seed failure while upserting a source file into the registry.
	seedStageUpsert = "upsert"
)

// seedFailureLog collects the source files a build could not seed into the registry.
//
// The walk's file workers record into it concurrently, so every access holds mu.
type seedFailureLog struct {
	// failures holds the recorded failures in the order they happened.
	failures []lifecycle_domain.SeedFailure

	// mu guards failures.
	mu sync.Mutex
}

// add records one seed failure.
//
// Takes failure (lifecycle_domain.SeedFailure) which describes the failure.
//
// Safe for concurrent use.
func (s *seedFailureLog) add(failure lifecycle_domain.SeedFailure) {
	s.mu.Lock()
	s.failures = append(s.failures, failure)
	s.mu.Unlock()
}

// take returns the recorded failures and empties the log.
//
// Returns []lifecycle_domain.SeedFailure which holds the failures, or nil when none were
// recorded.
//
// Safe for concurrent use.
func (s *seedFailureLog) take() []lifecycle_domain.SeedFailure {
	s.mu.Lock()
	defer s.mu.Unlock()
	failures := s.failures
	s.failures = nil
	return failures
}

// recordSeedFailure records a source file or directory that could not be seeded.
//
// Missing paths are logged and ignored. Other failures cause the build to fail.
//
// Takes stage (string) which names the step that failed (seedStageWalk, seedStageRead or
// seedStageUpsert).
// Takes path (string) which is the file or directory that could not be seeded.
// Takes err (error) which is the cause.
func (bs *buildService) recordSeedFailure(ctx context.Context, stage, path string, err error) {
	_, l := logger_domain.From(ctx, log)
	if stage != seedStageUpsert && errors.Is(err, fs.ErrNotExist) {
		l.Trace("Skipping a source path that does not exist",
			logger_domain.String(fieldPath, path),
			logger_domain.String("stage", stage))
		return
	}

	l.Error("Failed to seed a source path into the registry",
		logger_domain.String(fieldPath, path),
		logger_domain.String("stage", stage),
		logger_domain.Error(err))
	bs.seedFailures.add(lifecycle_domain.SeedFailure{
		Path:  bs.projectRelativePath(path),
		Stage: stage,
		Error: err.Error(),
	})
}

// projectRelativePath returns a path relative to the project root when it lies inside it,
// for shorter failure reports.
//
// Takes path (string) which is the path to shorten.
//
// Returns string which is the slash-separated relative path, or path unchanged when it is
// already relative or lies outside the project root.
func (bs *buildService) projectRelativePath(path string) string {
	if bs.pathsConfig.BaseDir == "" || !filepath.IsAbs(path) {
		return path
	}
	baseDir, err := filepath.Abs(bs.pathsConfig.BaseDir)
	if err != nil {
		return path
	}
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.ToSlash(relPath)
}
