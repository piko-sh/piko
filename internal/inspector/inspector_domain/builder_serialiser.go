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

// This file contains the core business logic for encoding Go source code type
// information into the DTO format used by the querier.

import (
	"context"
	"fmt"
	"go/token"
	"go/types"
	"runtime"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

const (
	// initialMethodSetCacheSize is the starting capacity for the method set
	// cache in each encoder.
	initialMethodSetCacheSize = 256

	// initialPrimitiveGuardSize is the starting capacity for the primitive
	// recursion guard map in each encoder.
	initialPrimitiveGuardSize = 16
)

// processedPackageResult is a small struct to pass results from worker
// goroutines back to the main aggregator.
type processedPackageResult struct {
	// Package holds the processed package data for storage in the type data map.
	Package *inspector_dto.Package
}

var (
	// getTextMarshalerInterface returns a lazily initialised interface type
	// representing encoding.TextMarshaler.
	getTextMarshalerInterface = sync.OnceValue(func() *types.Interface {
		return getMarshalerInterface("MarshalText")
	})

	// getJSONMarshalerInterface returns a lazily initialised interface type
	// representing encoding/json.Marshaler.
	getJSONMarshalerInterface = sync.OnceValue(func() *types.Interface {
		return getMarshalerInterface("MarshalJSON")
	})

	// encoderPool reuses encoder instances to reduce allocation pressure during
	// type information serialisation.
	encoderPool = sync.Pool{
		New: func() any {
			return &encoder{
				methodSetCache: make(map[types.Type]*types.MethodSet, initialMethodSetCacheSize),
				primitiveGuard: make(map[types.Type]bool, initialPrimitiveGuardSize),
			}
		},
	}

	// pikoSpecialTypes is a lookup map for known Piko types that have
	// dedicated, high-performance formatters in the runtime.
	pikoSpecialTypes = map[string]bool{
		"piko.sh/piko/wdk/maths.Money":   true,
		"piko.sh/piko/wdk/maths.Decimal": true,
		"piko.sh/piko/wdk/maths.BigInt":  true,
	}
)

// encodingPipeline manages the state and execution of concurrent package
// processing. It encapsulates the complexity of the worker pool, allowing the
// main extractAndEncode function to remain a simple, high-level orchestrator.
type encodingPipeline struct {
	// ctx controls cancellation for worker goroutines.
	ctx context.Context

	// g coordinates worker goroutines and collects their errors.
	g *errgroup.Group

	// allPackages holds all parsed packages keyed by path for worker processing.
	allPackages map[string]*packages.Package

	// results is the channel that receives processed package data from workers.
	results chan processedPackageResult
}

// run starts workers simultaneously, feeds them jobs, and waits
// for completion.
//
// Returns error when any worker fails or the context is cancelled.
//
// Concurrent goroutines are spawned for NumCPU workers plus a feeder,
// coordinated by an errgroup.
func (p *encodingPipeline) run() error {
	jobs := make(chan *packages.Package, len(p.allPackages))

	numWorkers := max(1, min(runtime.NumCPU(), len(p.allPackages)))

	for range numWorkers {
		p.g.Go(func() error {
			return p.worker(jobs)
		})
	}

	p.g.Go(func() error {
		defer close(jobs)
		return p.feedJobs(jobs)
	})

	go func() {
		_ = p.g.Wait()
		close(p.results)
	}()

	return p.g.Wait()
}

// worker processes packages from a jobs channel and sends results to the
// results channel.
//
// Takes jobs (<-chan *packages.Package) which provides packages to process.
//
// Returns error when the context is cancelled.
func (p *encodingPipeline) worker(jobs <-chan *packages.Package) error {
	s := getEncoder(p.allPackages)
	defer putEncoder(s)

	for pkg := range jobs {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		default:
		}

		processedPackage := s.processSinglePackage(pkg)
		if processedPackage != nil {
			p.results <- processedPackageResult{Package: processedPackage}
		}
	}
	return nil
}

// feedJobs populates the jobs channel with all packages to be processed.
//
// Takes jobs (chan<- *packages.Package) which receives packages for processing.
//
// Returns error when the context is cancelled before all jobs are sent.
func (p *encodingPipeline) feedJobs(jobs chan<- *packages.Package) error {
	for _, pkg := range p.allPackages {
		select {
		case jobs <- pkg:
		case <-p.ctx.Done():
			return p.ctx.Err()
		}
	}
	return nil
}

// collectResults drains the results channel and gathers them into the final
// TypeData DTO.
//
// Returns *inspector_dto.TypeData which contains all packages and file mappings.
func (p *encodingPipeline) collectResults() *inspector_dto.TypeData {
	typeData := &inspector_dto.TypeData{
		Packages:      make(map[string]*inspector_dto.Package, len(p.allPackages)),
		FileToPackage: make(map[string]string),
	}
	for result := range p.results {
		typeData.Packages[result.Package.Path] = result.Package
		for filePath := range result.Package.FileImports {
			typeData.FileToPackage[filePath] = result.Package.Path
		}
	}
	return typeData
}

// ExtractAndEncodeForTest provides an exported wrapper for white-box
// testing of the encoding logic.
//
// Takes loadedPackages ([]*packages.Package) which contains the parsed Go packages
// to process.
// Takes moduleName (string) which is the user's module path for filtering
// internal packages from external dependencies. Pass empty string to disable
// filtering.
//
// Returns *inspector_dto.TypeData which holds the extracted type information.
// Returns error when extraction or encoding fails.
func ExtractAndEncodeForTest(loadedPackages []*packages.Package, moduleName string) (*inspector_dto.TypeData, error) {
	return extractAndEncode(loadedPackages, moduleName)
}

// getEncoder retrieves an encoder from the pool and sets it up for use.
//
// Takes allPackages (map[string]*packages.Package) which provides the parsed
// packages for cross-reference lookups.
//
// Returns *encoder which is ready for use with the given packages.
func getEncoder(allPackages map[string]*packages.Package) *encoder {
	s, ok := encoderPool.Get().(*encoder)
	if !ok {
		s = &encoder{
			methodSetCache: make(map[types.Type]*types.MethodSet, initialMethodSetCacheSize),
			primitiveGuard: make(map[types.Type]bool, initialPrimitiveGuardSize),
		}
	}
	s.allPackages = allPackages
	s.arena = newEncoderArena()
	return s
}

// putEncoder resets the encoder and returns it to the pool.
//
// Takes s (*encoder) which is the encoder to reset and return.
func putEncoder(s *encoder) {
	s.allPackages = nil
	s.arena = nil
	clear(s.methodSetCache)
	clear(s.primitiveGuard)
	encoderPool.Put(s)
}

// extractAndEncode converts live package data into an encodable DTO
// format. It orchestrates a concurrent encoding pipeline.
//
// Takes loadedPackages ([]*packages.Package) which provides the loaded Go packages
// to process.
// Takes moduleName (string) which is the user's module path for filtering
// internal packages from external dependencies.
//
// Returns *inspector_dto.TypeData which contains the encoded type
// information for all discovered packages.
// Returns error when the encoding pipeline fails to run.
func extractAndEncode(loadedPackages []*packages.Package, moduleName string) (*inspector_dto.TypeData, error) {
	allPackages := discoverAllPackages(loadedPackages, moduleName)
	if len(allPackages) == 0 {
		return &inspector_dto.TypeData{
			Packages:      make(map[string]*inspector_dto.Package),
			FileToPackage: make(map[string]string),
		}, nil
	}

	pipeline := newEncodingPipeline(allPackages)

	if err := pipeline.run(); err != nil {
		return nil, fmt.Errorf("running encoding pipeline: %w", err)
	}

	return pipeline.collectResults(), nil
}

// newEncodingPipeline creates and initialises a new encoding pipeline.
//
// Takes allPackages (map[string]*packages.Package) which provides the
// packages to process.
//
// Returns *encodingPipeline which is the initialised pipeline ready for use.
func newEncodingPipeline(allPackages map[string]*packages.Package) *encodingPipeline {
	g, ctx := errgroup.WithContext(context.Background())

	return &encodingPipeline{
		ctx:         ctx,
		g:           g,
		allPackages: allPackages,
		results:     make(chan processedPackageResult, len(allPackages)),
	}
}

// discoverAllPackages finds all unique packages by searching the dependency
// graph starting from the given packages. It skips internal packages from
// external modules that the user's code cannot import.
//
// Takes initialPackages ([]*packages.Package) which provides the starting packages
// from which to discover all dependencies.
// Takes moduleName (string) which is the user's module path for filtering
// internal packages. When empty, no filtering is done.
//
// Returns map[string]*packages.Package which maps package paths to their
// package objects for all discovered packages.
func discoverAllPackages(initialPackages []*packages.Package, moduleName string) map[string]*packages.Package {
	allPackages := make(map[string]*packages.Package, len(initialPackages)*2)
	seen := make(map[string]struct{}, len(initialPackages)*2)
	queue := make([]*packages.Package, 0, len(initialPackages))
	queue = append(queue, initialPackages...)

	for len(queue) > 0 {
		pkg := queue[0]
		queue = queue[1:]

		if _, ok := seen[pkg.PkgPath]; ok {
			continue
		}
		seen[pkg.PkgPath] = struct{}{}

		if moduleName != "" && !goastutil.ShouldIncludeInternalPackage(moduleName, pkg.PkgPath, pkg.Module != nil) {
			continue
		}

		allPackages[pkg.PkgPath] = pkg

		for _, importedPackage := range pkg.Imports {
			if _, ok := seen[importedPackage.PkgPath]; !ok {
				queue = append(queue, importedPackage)
			}
		}
	}
	return allPackages
}

// getMarshalerInterface constructs a types.Interface for a marshaler method.
//
// Takes methodName (string) which specifies the name of the marshal method.
//
// Returns *types.Interface which defines an interface with a single method
// that takes no arguments and returns ([]byte, error).
func getMarshalerInterface(methodName string) *types.Interface {
	byteSliceType := types.NewSlice(types.Typ[types.Uint8])
	errorType := types.Universe.Lookup("error").Type()
	marshalMethod := types.NewFunc(token.NoPos, nil, methodName,
		types.NewSignatureType(nil, nil, nil, nil,
			types.NewTuple(
				types.NewVar(token.NoPos, nil, "", byteSliceType),
				types.NewVar(token.NoPos, nil, "", errorType),
			), false),
	)
	return types.NewInterfaceType([]*types.Func{marshalMethod}, nil).Complete()
}
