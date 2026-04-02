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

package inspector_domain

// This file provides a high-performance, concurrent parser for transforming
// a map of source code into a map of Abstract Syntax Trees (ASTs).

import (
	"context"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"
)

// parseSourceContents parses all source files into ASTs using a pool of
// workers.
//
// This is the core logic for the default builderSourceParser implementation.
//
// When the source contents map is empty, returns an empty map without error.
//
// Takes sourceContents (map[string][]byte) which maps file paths to their
// contents.
// Takes maxWorkers (int) which sets the maximum number of parsing workers.
//
// Returns map[string]*goast.File which maps file paths to their parsed ASTs.
// Returns error when any file fails to parse.
func parseSourceContents(sourceContents map[string][]byte, maxWorkers int) (map[string]*goast.File, error) {
	if len(sourceContents) == 0 {
		return make(map[string]*goast.File), nil
	}

	allScriptBlocks := make(map[string]*goast.File)
	mu := &sync.Mutex{}
	g, _ := errgroup.WithContext(context.Background())

	jobs := make(chan string, len(sourceContents))
	for path := range sourceContents {
		jobs <- path
	}
	close(jobs)

	numWorkers := determineWorkerCount(maxWorkers, len(sourceContents))

	for range numWorkers {
		g.Go(func() error {
			return parsingWorker(jobs, sourceContents, allScriptBlocks, mu)
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("parsing source contents: %w", err)
	}

	return allScriptBlocks, nil
}

// determineWorkerCount works out how many workers to use for a pool.
//
// Takes maxWorkersConfig (int) which sets the maximum workers, or 0 for no
// limit.
// Takes numJobs (int) which is the total number of jobs to process.
//
// Returns int which is the worker count, limited by CPU cores, config, and the
// number of jobs.
func determineWorkerCount(maxWorkersConfig, numJobs int) int {
	numWorkers := runtime.NumCPU()

	if maxWorkersConfig > 0 && numWorkers > maxWorkersConfig {
		numWorkers = maxWorkersConfig
	}

	if numWorkers > numJobs {
		numWorkers = numJobs
	}

	if numWorkers == 0 && numJobs > 0 {
		numWorkers = 1
	}

	return numWorkers
}

// parsingWorker processes file paths from a jobs channel and parses each file
// into an AST. It pulls paths from the channel, parses them, and writes the
// resulting AST to the shared results map until the channel is empty.
//
// Takes jobs (<-chan string) which provides file paths to parse.
// Takes sourceContents (map[string][]byte) which holds the source code for
// each file path.
// Takes allScriptBlocks (map[string]*goast.File) which stores the parsed AST
// for each file.
// Takes mu (*sync.Mutex) which protects concurrent writes to allScriptBlocks.
//
// Returns error when a file fails to parse.
//
// Concurrent writes to allScriptBlocks are protected by the provided mutex.
func parsingWorker(
	jobs <-chan string,
	sourceContents map[string][]byte,
	allScriptBlocks map[string]*goast.File,
	mu *sync.Mutex,
) error {
	fset := token.NewFileSet()

	for path := range jobs {
		content, ok := sourceContents[path]
		if !ok {
			continue
		}

		file, err := parser.ParseFile(fset, path, content, parser.AllErrors)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		mu.Lock()
		allScriptBlocks[path] = file
		mu.Unlock()
	}
	return nil
}
