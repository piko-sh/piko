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

// Provides helper functions for the annotator service including diagnostic
// collection, error handling, and result aggregation. Contains utility methods
// that support the main service operations during the compilation pipeline
// execution.

import (
	"cmp"
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/sync/errgroup"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

// annotationWorkerConfig holds settings for annotation workers.
type annotationWorkerConfig struct {
	// componentGraph holds the dependency graph used during analysis.
	componentGraph *annotator_dto.ComponentGraph

	// virtualModule is the virtual module being processed.
	virtualModule *annotator_dto.VirtualModule

	// typeResolver provides type information for annotations.
	typeResolver *TypeResolver

	// actions maps action names to their info providers.
	actions map[string]ActionInfoProvider

	// options holds the settings for annotation processing.
	options *annotationOptions
}

// annotationJob represents a single task for a worker to process.
type annotationJob struct {
	// ctx carries the session logger for this annotation job.
	ctx context.Context

	// vc is the virtual component to annotate.
	vc *annotator_dto.VirtualComponent
}

// annotationJobResult holds the outcome of processing an annotation job.
type annotationJobResult struct {
	// result holds the annotation output for this job; nil if the job failed.
	result *annotator_dto.AnnotationResult

	// diagnostics holds the lint issues found for this annotation job.
	diagnostics []*ast_domain.Diagnostic
}

// createAnnotationWorker creates a worker function for the annotation
// errgroup.
//
// Takes jobs (<-chan *annotationJob) which provides annotation jobs to process.
// Takes results (chan<- *annotationJobResult) which receives completed job
// results.
// Takes config (*annotationWorkerConfig) which sets worker behaviour.
//
// Returns func() error which processes jobs until the channel closes or the
// context is cancelled.
func (s *AnnotatorService) createAnnotationWorker(
	ctx context.Context,
	jobs <-chan *annotationJob,
	results chan<- *annotationJobResult,
	config *annotationWorkerConfig,
) func() error {
	return func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case job, ok := <-jobs:
				if !ok {
					return nil
				}
				result := s.processAnnotationJob(ctx, job, config)
				select {
				case results <- result:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}

// processAnnotationJob handles a single annotation job and returns the result.
//
// Takes job (*annotationJob) which specifies the task to process.
// Takes config (*annotationWorkerConfig) which provides the worker settings.
//
// Returns *annotationJobResult which contains the result or error details.
func (s *AnnotatorService) processAnnotationJob(
	_ context.Context,
	job *annotationJob,
	config *annotationWorkerConfig,
) *annotationJobResult {
	pipeline := &componentAnnotationPipeline{
		service:           s,
		vc:                job.vc,
		componentGraph:    config.componentGraph,
		virtualModule:     config.virtualModule,
		typeResolver:      config.typeResolver,
		actions:           config.actions,
		options:           config.options,
		diagnostics:       nil,
		componentRegistry: s.componentRegistry,
	}
	compResult, compDiags, err := pipeline.run(job.ctx)

	if err != nil {
		return s.createErrorJobResult(job.ctx, job.vc, err)
	}

	return &annotationJobResult{
		result:      compResult,
		diagnostics: compDiags,
	}
}

// createErrorJobResult creates a job result for a failed annotation.
//
// Takes ctx (context.Context) which carries the logger.
// Takes vc (*annotator_dto.VirtualComponent) which is the component that
// failed.
// Takes err (error) which is the error that occurred.
//
// Returns *annotationJobResult which holds the error details and a nil result.
func (*AnnotatorService) createErrorJobResult(
	ctx context.Context,
	vc *annotator_dto.VirtualComponent,
	err error,
) *annotationJobResult {
	_, l := logger_domain.From(ctx, log)
	errorDiag := ast_domain.NewDiagnosticWithCode(
		ast_domain.Error,
		fmt.Sprintf("Fatal error during annotation: %v", err),
		"",
		annotator_dto.CodeFatalAnnotationError,
		ast_domain.Location{Line: 1, Column: 1, Offset: 0},
		vc.Source.SourcePath,
	)
	l.Error("Component annotation failed, continuing with others",
		logger_domain.String("component", vc.Source.SourcePath),
		logger_domain.Error(err))

	return &annotationJobResult{
		result:      nil,
		diagnostics: []*ast_domain.Diagnostic{errorDiag},
	}
}

// prepareAnnotationJob creates an annotation job for a virtual component.
//
// Takes ctx (context.Context) which is the parent context for the job.
// Takes vc (*annotator_dto.VirtualComponent) which specifies the component to
// annotate.
//
// Returns *annotationJob which contains the component and a context carrying
// the session logger.
func (s *AnnotatorService) prepareAnnotationJob(ctx context.Context, vc *annotator_dto.VirtualComponent) *annotationJob {
	relPath, err := filepath.Rel(s.resolver.GetBaseDir(), vc.Source.SourcePath)
	if err != nil {
		relPath = strings.ReplaceAll(vc.Source.SourcePath, string(filepath.Separator), "_")
	}

	sessionLogger := s.logStore.StartSession(ctx, vc.Source.SourcePath, relPath)

	return &annotationJob{
		ctx: logger_domain.WithLogger(ctx, sessionLogger),
		vc:  vc,
	}
}

// runSrcsetAnnotation adds srcset attributes to images for responsive display.
//
// Takes ctx (context.Context) which carries the logger.
// Takes finalResult (*annotator_dto.ProjectAnnotationResult) which holds the
// component results to update with srcset attributes.
func (s *AnnotatorService) runSrcsetAnnotation(
	ctx context.Context,
	finalResult *annotator_dto.ProjectAnnotationResult,
) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("--- [STAGE 10/10] Starting: Srcset Annotation for Responsive Images ---")

	allAssetDeps := make([]*annotator_dto.StaticAssetDependency, 0)
	for _, result := range finalResult.ComponentResults {
		if result.AssetDependencies != nil {
			allAssetDeps = append(allAssetDeps, result.AssetDependencies...)
		}
	}

	for _, result := range finalResult.ComponentResults {
		if result.AnnotatedAST != nil {
			AnnotateSrcsetForWebImages(ctx, result.AnnotatedAST, allAssetDeps, s.pathsConfig)
		}
	}

	l.Internal("--- [STAGE 10/10] Finished: Srcset Annotation ---")
}

// runAnnotationWorkerPool starts workers that annotate components in parallel.
//
// Takes components (map[string]*annotator_dto.VirtualComponent) which are
// the components to annotate.
// Takes config (*annotationWorkerConfig) which sets worker behaviour.
//
// Returns <-chan *annotationJobResult which yields annotation results and
// closes when all workers finish.
// Returns error which is always nil (kept for future use).
//
// Spawns a goroutine that sends jobs to the workers, plus one
// goroutine per worker. An additional goroutine waits for all workers to
// finish and then closes the results channel. The spawned workers run until
// all jobs are processed.
func (s *AnnotatorService) runAnnotationWorkerPool(
	ctx context.Context,
	components map[string]*annotator_dto.VirtualComponent,
	config *annotationWorkerConfig,
) (<-chan *annotationJobResult, error) {
	g, gCtx := errgroup.WithContext(ctx)

	numComponents := len(components)
	jobs := make(chan *annotationJob, numComponents)
	results := make(chan *annotationJobResult, numComponents)

	numWorkers := calculateWorkerCount(numComponents)
	for range numWorkers {
		g.Go(s.createAnnotationWorker(gCtx, jobs, results, config))
	}

	go feedAnnotationJobs(gCtx, jobs, s.prepareAnnotationJob, components)

	go func() {
		_ = g.Wait()
		close(results)
	}()

	return results, nil
}

// buildActionsFromManifest converts an ActionManifest into a map of
// ActionInfoProvider for use in semantic analysis. This enables auto-discovery
// of actions without requiring callers to pass an explicit actions map.
//
// Takes manifest (*annotator_dto.ActionManifest) which contains discovered
// actions.
//
// Returns map[string]ActionInfoProvider which maps action names to their info.
func buildActionsFromManifest(manifest *annotator_dto.ActionManifest) map[string]ActionInfoProvider {
	if manifest == nil || len(manifest.Actions) == 0 {
		return nil
	}

	actions := make(map[string]ActionInfoProvider, len(manifest.Actions))
	for i := range manifest.Actions {
		action := &manifest.Actions[i]
		actions[action.Name] = action
	}
	return actions
}

// calculateWorkerCount finds the number of workers to use based on the CPU
// count and job count.
//
// Takes jobCount (int) which is the number of jobs to process.
//
// Returns int which is the worker count, always at least 1.
func calculateWorkerCount(jobCount int) int {
	return cmp.Or(min(runtime.NumCPU(), jobCount), 1)
}

// feedAnnotationJobs sends annotation jobs to a channel for processing.
//
// Takes jobs (chan<- *annotationJob) which receives the prepared annotation
// jobs.
// Takes prepareJob (func(...)) which creates an annotation job from a virtual
// component.
// Takes components (map[string]*annotator_dto.VirtualComponent) which holds
// the virtual components to process.
func feedAnnotationJobs(
	ctx context.Context,
	jobs chan<- *annotationJob,
	prepareJob func(context.Context, *annotator_dto.VirtualComponent) *annotationJob,
	components map[string]*annotator_dto.VirtualComponent,
) {
	defer close(jobs)
	for _, vc := range components {
		select {
		case jobs <- prepareJob(ctx, vc):
		case <-ctx.Done():
			return
		}
	}
}

// aggregateAnnotationResults collects results from the results channel and
// stores them in the final result.
//
// Takes resultsChan (<-chan *annotationJobResult) which provides annotation
// job results to collect.
// Takes finalResult (*annotator_dto.ProjectAnnotationResult) which receives
// the collected diagnostics and component results.
//
// Returns []error which contains serious errors found during collection.
func aggregateAnnotationResults(
	resultsChan <-chan *annotationJobResult,
	finalResult *annotator_dto.ProjectAnnotationResult,
) []error {
	severeErrors := make([]error, 0)

	for jobResult := range resultsChan {
		finalResult.AllDiagnostics = append(finalResult.AllDiagnostics, jobResult.diagnostics...)

		if jobResult.result == nil {
			continue
		}

		mainComponent, err := getMainComponent(jobResult.result)
		if err != nil {
			severeErrors = append(severeErrors, err)
			continue
		}
		finalResult.ComponentResults[mainComponent.HashedName] = jobResult.result
	}

	return severeErrors
}

// runAssetAggregation gathers assets from all components into a single
// manifest.
//
// Takes ctx (context.Context) which carries the logger.
// Takes finalResult (*annotator_dto.ProjectAnnotationResult) which holds
// component results and receives the final asset manifest.
func runAssetAggregation(
	ctx context.Context,
	finalResult *annotator_dto.ProjectAnnotationResult,
) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("--- [STAGE 9/9] Starting: Project-Wide Asset Aggregation ---")

	allResultsSlice := make([]*annotator_dto.AnnotationResult, 0, len(finalResult.ComponentResults))
	for _, result := range finalResult.ComponentResults {
		allResultsSlice = append(allResultsSlice, result)
	}

	finalResult.FinalAssetManifest = AggregateProjectAssets(allResultsSlice)
	l.Internal("--- [STAGE 9/9] Finished: Asset Aggregation ---",
		logger_domain.Int("unique_assets_found", len(finalResult.FinalAssetManifest)))
}

// handlePhase2Completion checks for errors after annotation and returns the
// final result.
//
// When fault tolerance is on, it logs a warning and returns partial results
// even if there are errors. When fault tolerance is off, it returns an error
// if there are any build errors.
//
// Takes ctx (context.Context) which carries the logger.
// Takes finalResult (*annotator_dto.ProjectAnnotationResult) which contains
// the annotation results to check.
// Takes logStore (*CompilationLogStore) which holds build log entries.
// Takes options (*annotationOptions) which controls fault tolerance behaviour.
//
// Returns *annotator_dto.ProjectAnnotationResult which is the processed
// result with duplicate diagnostics removed.
// Returns *CompilationLogStore which is the unchanged log store.
// Returns error when there are build errors and fault tolerance is off.
func handlePhase2Completion(
	ctx context.Context,
	finalResult *annotator_dto.ProjectAnnotationResult,
	logStore *CompilationLogStore,
	options *annotationOptions,
) (*annotator_dto.ProjectAnnotationResult, *CompilationLogStore, error) {
	ctx, l := logger_domain.From(ctx, log)
	finalResult.AllDiagnostics = ast_domain.DeduplicateDiagnostics(finalResult.AllDiagnostics)

	if ast_domain.HasErrors(finalResult.AllDiagnostics) {
		if options.faultTolerant {
			l.Warn("Compilation has errors, but returning partial results for LSP",
				logger_domain.Int("error_count", len(finalResult.AllDiagnostics)))
			return finalResult, logStore, nil
		}
		l.Error("Compilation failed with errors during per-component annotation.")
		return finalResult, logStore, NewSemanticError(finalResult.AllDiagnostics)
	}

	l.Internal("All project components annotated successfully.")
	return finalResult, logStore, nil
}
