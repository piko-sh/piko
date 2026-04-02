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

package bootstrap

// This file orchestrates the command-line build process for a Piko project.
// It translates a CLI command into a series of calls to the core services,
// handling discovery, compilation, and artefact writing.

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_adapters"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/lifecycle/lifecycle_adapters"
	"piko.sh/piko/internal/lifecycle/lifecycle_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_go_sql"
	"piko.sh/piko/internal/querier/querier_adapters/migration_sql"
	"piko.sh/piko/internal/querier/querier_domain"
	"piko.sh/piko/internal/querier/querier_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// GenerateModeManifest is the generate mode that creates a manifest file.
	GenerateModeManifest = "manifest"

	// GenerateModeAll is the run mode that creates all build outputs.
	GenerateModeAll = "all"

	// GenerateModeSQL is the generate mode that runs the querier code
	// generator against all registered databases, producing typed Go query
	// methods from SQL files.
	GenerateModeSQL = "sql"

	// GenerateModeAssets is the generate mode that runs annotation to discover
	// template-derived asset requirements (image sizes, densities, formats),
	// then builds static assets. Code emission and formatting are skipped.
	GenerateModeAssets = "assets"

	// minHTTPStatusCode is the lowest valid HTTP status code.
	minHTTPStatusCode = 100

	// maxHTTPStatusCode is the highest valid HTTP status code.
	maxHTTPStatusCode = 599

	// generatedFilePerms is the file permission for generated source files.
	generatedFilePerms = 0o600

	// buildGCPercent is the GOGC percentage used during builds to reduce GC
	// pressure at the cost of higher peak memory.
	buildGCPercent = 500
)

// errNoPagesFound is a checkable error for when no entry points are found.
var errNoPagesFound = errors.New("no pages found in configured directory")

// buildOperation holds the state and logic for a single build run.
type buildOperation struct {
	// container provides access to services through dependency injection.
	container *Container

	// manifest holds the generated build manifest for writing to disk.
	manifest *generator_dto.Manifest

	// runMode specifies which build mode to use, such as "all" for full builds.
	runMode string

	// entryPoints holds the pages and emails found during discovery.
	entryPoints []annotator_dto.EntryPoint

	// artefacts holds the generated Go source files.
	artefacts []*generator_dto.GeneratedArtefact

	// sqlQueryCount tracks how many SQL queries had Go code generated.
	sqlQueryCount int
}

// walkEntryContext holds the state needed to process a single file system entry
// during a directory walk.
type walkEntryContext struct {
	// sourceRoot is the absolute path to the source folder being walked.
	sourceRoot string

	// moduleName is the Go module name used to build relative paths.
	moduleName string

	// isPotentiallyPage indicates whether this entry could be shown as a page.
	isPotentiallyPage bool

	// isPotentiallyEmail indicates whether this entry may be an email template.
	isPotentiallyEmail bool

	// isPotentiallyPdf indicates whether this entry may be a PDF template.
	isPotentiallyPdf bool

	// isE2EOnly indicates whether this path is used only for end-to-end tests.
	isE2EOnly bool
}

// execute runs the build steps in order.
//
// The exact steps depend on runMode:
//   - GenerateModeSQL: generates SQL only, then returns.
//   - GenerateModeAssets: initialises the orchestrator, discovers entry points,
//     runs annotation (no code emission), feeds asset requirements to the
//     pipeline, then builds static assets.
//   - GenerateModeAll: generates SQL, initialises the orchestrator, discovers
//     entry points, runs annotation, then runs code emission and asset building
//     in parallel.
//   - GenerateModeManifest: discovers entry points, generates and writes
//     artefacts (no SQL, no assets).
//
// Returns error when any step fails.
func (op *buildOperation) execute(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	l.Notice("Starting Piko project build...", logger_domain.String("mode", op.runMode))

	if op.runMode == GenerateModeSQL {
		return op.generateSQL(ctx)
	}

	if err := op.prepareForDiscovery(ctx); err != nil {
		return err
	}

	found, err := op.discoverEntryPointsWithFallback(ctx, l)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	switch op.runMode {
	case GenerateModeAssets:
		if err := op.runAnnotationAndBuildAssets(ctx); err != nil {
			return fmt.Errorf("annotation and asset build: %w", err)
		}
		return nil
	case GenerateModeAll:
		return op.runAnnotationEmitAndBuildAssets(ctx)
	default:

		return op.generateAndWriteArtefacts(ctx)
	}
}

// prepareForDiscovery runs any mode-specific setup that must happen before
// entry-point discovery. For GenerateModeAll it generates SQL first; for modes
// that need the asset pipeline it initialises the orchestrator.
//
// Returns error when SQL generation or orchestrator initialisation fails.
func (op *buildOperation) prepareForDiscovery(ctx context.Context) error {
	if op.runMode == GenerateModeAll {
		if err := op.generateSQL(ctx); err != nil {
			return fmt.Errorf("generating SQL: %w", err)
		}
	}

	if op.runMode == GenerateModeAll || op.runMode == GenerateModeAssets {
		if err := op.initialiseOrchestratorEarly(ctx); err != nil {
			return fmt.Errorf("initialising orchestrator early: %w", err)
		}
	}

	return nil
}

// discoverEntryPointsWithFallback discovers entry points and handles the
// "no pages found" case. It returns found=true when pages were discovered and
// the caller should continue; found=false means the build is complete.
//
// Takes l (logger_domain.Logger) which provides structured logging.
//
// Returns found (bool) which is true when pages were discovered and the caller
// should continue.
// Returns err (error) when discovery fails for a reason other than no pages
// found.
func (op *buildOperation) discoverEntryPointsWithFallback(ctx context.Context, l logger_domain.Logger) (found bool, err error) {
	if err := op.discoverEntryPoints(ctx); err != nil {
		if !errors.Is(err, errNoPagesFound) {
			return false, fmt.Errorf("failed during discovery phase: %w", err)
		}
		if op.runMode == GenerateModeAssets {
			return false, op.buildStaticAssets(ctx)
		}
		l.Notice("No pages found. Build is complete with nothing to do.")
		return false, nil
	}
	l.Internal("Pages found for compilation", logger_domain.Int("page_count", len(op.entryPoints)))
	return true, nil
}

// generateAndWriteArtefacts runs code generation, writes the output files and
// manifest, and prints the build summary.
//
// Returns error when any generation or write step fails.
func (op *buildOperation) generateAndWriteArtefacts(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	genStart := time.Now()
	if err := op.runGeneration(ctx); err != nil {
		return fmt.Errorf("running generation: %w", err)
	}
	genDuration := time.Since(genStart)

	if err := op.writeArtefacts(ctx); err != nil {
		return fmt.Errorf("writing artefacts: %w", err)
	}

	if err := op.writeManifest(ctx); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	genSummary := lifecycle_adapters.FormatGeneratorSummary(&lifecycle_adapters.GeneratorResult{
		Pages:      len(op.manifest.Pages),
		Partials:   len(op.manifest.Partials),
		Emails:     len(op.manifest.Emails),
		PDFs:       len(op.manifest.Pdfs),
		SQLQueries: op.sqlQueryCount,
		Artefacts:  len(op.artefacts),
		Duration:   genDuration,
	})
	_, _ = fmt.Fprint(os.Stderr, genSummary)

	l.Internal("Build successful",
		logger_domain.Int("artefact_count", len(op.artefacts)),
		logger_domain.String("dist_dir", distDirName))

	return nil
}

// initialiseOrchestratorEarly initialises the orchestrator service
// early in the build process.
//
// This is critical for 'all' mode because the orchestrator's event bridge must
// be subscribed to artefact events before any artefacts are created during
// generation. Without this, events published during runGeneration would have
// no subscribers.
//
// Returns error when the orchestrator service fails to initialise.
func (op *buildOperation) initialiseOrchestratorEarly(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Initialising orchestrator early for static asset processing...")
	_, err := op.container.GetOrchestratorService()
	if err != nil {
		return fmt.Errorf("failed to initialise orchestrator service: %w", err)
	}
	l.Internal("Orchestrator subscriptions established, ready for artefact events")
	return nil
}

// runAnnotationAndBuildAssets runs annotation to discover template-derived
// asset requirements, feeds them to the asset pipeline, then builds all static
// assets. This is the core of GenerateModeAssets: it gives the asset pipeline
// the FinalAssetManifest (image sizes, densities, formats extracted from
// templates) without running the expensive code emission and formatting steps.
//
// Returns error when annotation or asset building fails.
func (op *buildOperation) runAnnotationAndBuildAssets(ctx context.Context) error {
	generatorService, err := op.container.GetGeneratorService()
	if err != nil {
		return fmt.Errorf("getting generator service: %w", err)
	}

	annotationResult, err := generatorService.AnnotateProject(ctx, op.entryPoints)
	if err != nil {
		return fmt.Errorf("running annotation: %w", err)
	}

	if err := op.processAnnotationForAssets(ctx, annotationResult); err != nil {
		return fmt.Errorf("processing annotation for assets: %w", err)
	}

	return op.buildStaticAssets(ctx)
}

// processAnnotationForAssets feeds the annotation result's FinalAssetManifest
// to the asset pipeline so template-derived transformation profiles (image
// sizes, densities, formats) are registered before the filesystem walk.
//
// Takes annotationResult (*annotator_dto.ProjectAnnotationResult) which
// contains the FinalAssetManifest from template analysis.
//
// Returns error when the registry service cannot be obtained or processing
// fails.
func (op *buildOperation) processAnnotationForAssets(
	ctx context.Context,
	annotationResult *annotator_dto.ProjectAnnotationResult,
) error {
	registryService, err := op.container.GetRegistryService()
	if err != nil {
		return fmt.Errorf("getting registry service: %w", err)
	}

	pipeline := lifecycle_domain.NewAssetPipelineOrchestrator(registryService, op.container.GetAssetsConfig())
	return pipeline.ProcessBuildResult(ctx, annotationResult)
}

// runAnnotationEmitAndBuildAssets runs annotation first, then fans out code
// emission (with artefact writing) and asset building in parallel. This is
// the GenerateModeAll fast path: the expensive code formatting in EmitProject
// runs concurrently with image processing and other asset tasks.
//
// Returns error when annotation, emission, or asset building fails.
func (op *buildOperation) runAnnotationEmitAndBuildAssets(ctx context.Context) error {
	ctx, l := logger_domain.From(ctx, log)

	generatorService, err := op.container.GetGeneratorService()
	if err != nil {
		return fmt.Errorf("getting generator service: %w", err)
	}

	totalStart := time.Now()

	annotationResult, err := generatorService.AnnotateProject(ctx, op.entryPoints)
	if err != nil {
		return fmt.Errorf("running annotation: %w", err)
	}
	annotationDuration := time.Since(totalStart)

	var genDuration time.Duration
	var buildResult *lifecycle_domain.BuildResult

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var legErr error
		genDuration, legErr = op.runEmissionLeg(gctx, generatorService, annotationResult, l)
		return legErr
	})

	g.Go(func() error {
		var legErr error
		buildResult, legErr = op.runAssetLeg(gctx, annotationResult)
		return legErr
	})

	if waitErr := g.Wait(); waitErr != nil {
		return waitErr
	}
	totalDuration := time.Since(totalStart)

	summary := lifecycle_adapters.FormatCombinedSummary(
		&lifecycle_adapters.GeneratorResult{
			Pages:      len(op.manifest.Pages),
			Partials:   len(op.manifest.Partials),
			Emails:     len(op.manifest.Emails),
			PDFs:       len(op.manifest.Pdfs),
			SQLQueries: op.sqlQueryCount,
			Artefacts:  len(op.artefacts),
			Duration:   genDuration,
		},
		buildResult,
		annotationDuration,
		totalDuration,
	)
	_, _ = fmt.Fprint(os.Stderr, summary)

	return nil
}

// runEmissionLeg runs code emission, artefact writes, and manifest writing.
// It is the first leg of the parallel GenerateModeAll build.
//
// Takes generatorService which drives code emission.
// Takes annotationResult which contains the pre-computed annotation output.
// Takes l which carries the contextual logger.
//
// Returns the time taken for the emission phase and any error.
func (op *buildOperation) runEmissionLeg(
	ctx context.Context,
	generatorService generator_domain.GeneratorService,
	annotationResult *annotator_dto.ProjectAnnotationResult,
	l logger_domain.Logger,
) (time.Duration, error) {
	genStart := time.Now()
	artefacts, manifest, err := generatorService.EmitProject(ctx, annotationResult)
	if err != nil {
		return 0, fmt.Errorf("emitting project: %w", err)
	}
	genDuration := time.Since(genStart)

	op.artefacts = artefacts
	op.manifest = manifest

	if err := op.writeArtefacts(ctx); err != nil {
		return 0, fmt.Errorf("writing artefacts: %w", err)
	}
	if err := op.writeManifest(ctx); err != nil {
		return 0, fmt.Errorf("writing manifest: %w", err)
	}

	l.Internal("Emission and write complete",
		logger_domain.Int("artefact_count", len(op.artefacts)),
		logger_domain.String("dist_dir", distDirName))
	return genDuration, nil
}

// runAssetLeg processes annotation-derived asset profiles and then runs the
// static asset build. It is the second leg of the parallel GenerateModeAll
// build.
//
// Takes annotationResult which contains the FinalAssetManifest.
//
// Returns the build result and any error.
func (op *buildOperation) runAssetLeg(
	ctx context.Context,
	annotationResult *annotator_dto.ProjectAnnotationResult,
) (*lifecycle_domain.BuildResult, error) {
	if err := op.processAnnotationForAssets(ctx, annotationResult); err != nil {
		return nil, fmt.Errorf("processing annotation for assets: %w", err)
	}
	return op.runStaticAssetBuild(ctx)
}

// discoverEntryPoints walks the configured pages, partials, and emails
// directories to find all .pk files that should be part of the build. It
// distinguishes between pages (routable entry points) and partials, and
// respects the public/private naming convention (any file or directory
// prefixed with '_' is private).
//
// When E2EMode is enabled, also discovers pages and partials from the E2E
// directory. E2E entries are marked with IsE2EOnly=true and will override
// production entries with the same relative path.
//
// Returns error when directory discovery fails or no public entry points are
// found.
func (op *buildOperation) discoverEntryPoints(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	serverConfig := op.container.config.ServerConfig
	pagesDir := deref(serverConfig.Paths.PagesSourceDir, "pages")
	partialsDir := deref(serverConfig.Paths.PartialsSourceDir, "partials")
	emailsDir := deref(serverConfig.Paths.EmailsSourceDir, "emails")
	pdfsDir := deref(serverConfig.Paths.PdfsSourceDir, "pdfs")

	if err := op.discoverDirectory(ctx, pagesDir, true, false, false, false); err != nil {
		return fmt.Errorf("discovering pages directory: %w", err)
	}
	if err := op.discoverDirectory(ctx, partialsDir, false, false, false, false); err != nil {
		return fmt.Errorf("discovering partials directory: %w", err)
	}
	if err := op.discoverDirectory(ctx, emailsDir, false, true, false, false); err != nil {
		return fmt.Errorf("discovering emails directory: %w", err)
	}
	if err := op.discoverDirectory(ctx, pdfsDir, false, false, true, false); err != nil {
		return fmt.Errorf("discovering pdfs directory: %w", err)
	}

	if deref(serverConfig.Build.E2EMode, false) {
		l.Notice("E2E mode enabled - including test pages and partials from e2e/ directory")
		e2eDir := deref(serverConfig.Paths.E2ESourceDir, "e2e")

		e2ePagesDir := filepath.Join(e2eDir, "pages")
		if err := op.discoverDirectory(ctx, e2ePagesDir, true, false, false, true); err != nil {
			return fmt.Errorf("discovering E2E pages directory: %w", err)
		}
		e2ePartialsDir := filepath.Join(e2eDir, "partials")
		if err := op.discoverDirectory(ctx, e2ePartialsDir, false, false, false, true); err != nil {
			return fmt.Errorf("discovering E2E partials directory: %w", err)
		}
	}

	if !op.hasPublicEntryPoint() {
		return errNoPagesFound
	}
	return nil
}

// discoverDirectory scans a source directory and appends EntryPoint objects
// to the operation's list.
//
// Takes ctx (context.Context) which carries cancellation and logging context.
// Takes sourceDir (string) which specifies the directory path to scan.
// Takes isPotentiallyPage (bool) which indicates if entries may be pages.
// Takes isPotentiallyEmail (bool) which indicates if entries may be emails.
// Takes isPotentiallyPdf (bool) which indicates if entries may be PDF templates.
// Takes isE2EOnly (bool) which indicates if entries are E2E test-only.
//
// Returns error when the directory cannot be checked, the resolver cannot be
// obtained, or walking the directory fails.
func (op *buildOperation) discoverDirectory(ctx context.Context, sourceDir string, isPotentiallyPage, isPotentiallyEmail, isPotentiallyPdf, isE2EOnly bool) error {
	serverConfig := op.container.config.ServerConfig
	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	sourceRoot := filepath.Join(baseDir, sourceDir)

	exists, err := op.checkSourceDirectory(ctx, sourceDir, sourceRoot, isPotentiallyPage, isPotentiallyEmail, isPotentiallyPdf, isE2EOnly)
	if err != nil {
		return fmt.Errorf("checking source directory %q: %w", sourceDir, err)
	}
	if !exists {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before discovering %q: %w", sourceDir, err)
	}

	resolver, err := op.container.GetResolver()
	if err != nil {
		return fmt.Errorf("getting resolver for directory discovery: %w", err)
	}

	wctx := walkEntryContext{
		sourceRoot:         sourceRoot,
		moduleName:         resolver.GetModuleName(),
		isPotentiallyPage:  isPotentiallyPage,
		isPotentiallyEmail: isPotentiallyEmail,
		isPotentiallyPdf:   isPotentiallyPdf,
		isE2EOnly:          isE2EOnly,
	}
	return filepath.WalkDir(sourceRoot, func(absPath string, d os.DirEntry, walkErr error) error {
		return op.processWalkEntry(ctx, absPath, d, walkErr, wctx)
	})
}

// checkSourceDirectory checks if the source directory exists and decides
// whether to process it.
//
// Takes ctx (context.Context) which carries the logging context.
// Takes sourceDir (string) which is the relative source
// directory name.
// Takes sourceRoot (string) which is the absolute path to the
// source directory.
// Takes isPotentiallyPage (bool) which is true if this is a pages directory.
// Takes isPotentiallyEmail (bool) which is true if this is an emails directory.
// Takes isPotentiallyPdf (bool) which is true if this is a PDFs directory.
// Takes isE2EOnly (bool) which is true if this is an E2E-only directory.
//
// Returns bool which is true if the directory exists and should be processed.
// Returns error when the pages directory is missing or sandbox creation fails.
func (op *buildOperation) checkSourceDirectory(ctx context.Context, sourceDir, sourceRoot string, isPotentiallyPage, isPotentiallyEmail, isPotentiallyPdf, isE2EOnly bool) (bool, error) {
	_, l := logger_domain.From(ctx, log)

	serverConfig := op.container.config.ServerConfig
	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	baseSandbox, sandboxErr := op.container.createSandbox("build-source-check", baseDir, safedisk.ModeReadOnly)
	if sandboxErr != nil {
		return false, fmt.Errorf("failed to create sandbox for base directory '%s': %w", baseDir, sandboxErr)
	}
	_, statErr := baseSandbox.Stat(sourceDir)
	_ = baseSandbox.Close()

	if !errors.Is(statErr, fs.ErrNotExist) {
		return true, nil
	}

	if isE2EOnly {
		l.Internal("E2E source directory not found, skipping.", logger_domain.String("path", sourceRoot))
		return false, nil
	}

	if isPotentiallyPage {
		return false, fmt.Errorf("pages source directory '%s' does not exist", sourceRoot)
	}

	dirType := "Partials"
	if isPotentiallyEmail {
		dirType = "Emails"
	} else if isPotentiallyPdf {
		dirType = "PDFs"
	}
	l.Internal("Source directory not found, skipping",
		logger_domain.String("dir_type", dirType),
		logger_domain.String("path", sourceRoot))
	return false, nil
}

// processWalkEntry handles a single entry during directory walking.
//
// Takes ctx (context.Context) which carries cancellation and logging context.
// Takes absPath (string) which is the absolute path to the entry.
// Takes d (os.DirEntry) which provides entry metadata.
// Takes walkErr (error) which is any error from the walk operation.
// Takes wctx (walkEntryContext) which provides source context for entry
// processing.
//
// Returns error when the context is cancelled, walkErr is non-nil, or the
// relative path cannot be computed.
func (op *buildOperation) processWalkEntry(
	ctx context.Context,
	absPath string,
	d os.DirEntry,
	walkErr error,
	wctx walkEntryContext,
) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled during directory walk: %w", err)
	}
	if walkErr != nil {
		return fmt.Errorf("walking directory entry %q: %w", absPath, walkErr)
	}

	if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".pk") {
		return nil
	}

	serverConfig := op.container.config.ServerConfig
	baseDir := deref(serverConfig.Paths.BaseDir, ".")
	relPathToBase, err := filepath.Rel(baseDir, absPath)
	if err != nil {
		return fmt.Errorf("computing relative path for %q: %w", absPath, err)
	}
	pikoPath := filepath.ToSlash(filepath.Join(wctx.moduleName, relPathToBase))

	errResult := isErrorPage(d.Name())

	if !errResult.isErrorPage && strings.HasPrefix(d.Name(), "!") {
		return fmt.Errorf(
			"invalid error page filename %q: files starting with '!' must follow the error page convention. "+
				"Valid formats: !NNN.pk (e.g., !404.pk), !NNN-NNN.pk (e.g., !400-499.pk), or !error.pk (catch-all). "+
				"Only error pages may start with '!'",
			d.Name(),
		)
	}

	op.entryPoints = append(op.entryPoints, annotator_dto.EntryPoint{
		Path:               pikoPath,
		IsPage:             wctx.isPotentiallyPage && !errResult.isErrorPage,
		IsPublic:           wctx.isPotentiallyPage,
		IsEmail:            wctx.isPotentiallyEmail,
		IsPdf:              wctx.isPotentiallyPdf,
		IsE2EOnly:          wctx.isE2EOnly,
		IsErrorPage:        errResult.isErrorPage,
		ErrorStatusCode:    errResult.statusCode,
		ErrorStatusCodeMin: errResult.rangeMin,
		ErrorStatusCodeMax: errResult.rangeMax,
		IsCatchAllError:    errResult.isCatchAll,
	})
	return nil
}

// errorPageResult holds the parsed result of an error page filename check.
type errorPageResult struct {
	// statusCode is the exact HTTP status code (e.g., 404). Zero for catch-all
	// and range error pages.
	statusCode int

	// rangeMin is the lower bound of a range error page (e.g., 400 for
	// !400-499.pk). Zero when the page is not a range.
	rangeMin int

	// rangeMax is the upper bound of a range error page (e.g., 499 for
	// !400-499.pk). Zero when the page is not a range.
	rangeMax int

	// isErrorPage is true when the filename matches a valid error page pattern.
	isErrorPage bool

	// isCatchAll is true for !error.pk pages that handle all status codes.
	isCatchAll bool
}

// hasPublicEntryPoint checks if any discovered entry point is public.
//
// Returns bool which is true if at least one entry point is a page or public.
func (op *buildOperation) hasPublicEntryPoint() bool {
	for _, ep := range op.entryPoints {
		if ep.IsPage || ep.IsPublic {
			return true
		}
	}
	return false
}

// runGeneration gets the generator service and runs the main compilation
// pipeline.
//
// Returns error when the generator service cannot be obtained or generation
// fails.
func (op *buildOperation) runGeneration(ctx context.Context) error {
	generatorService, err := op.container.GetGeneratorService()
	if err != nil {
		return fmt.Errorf("failed to get generator service for build: %w", err)
	}

	artefacts, manifest, err := generatorService.GenerateProject(ctx, op.entryPoints)
	if err != nil {
		return fmt.Errorf("generating project: %w", err)
	}

	op.artefacts = artefacts
	op.manifest = manifest
	return nil
}

// writeArtefacts writes all generated Go files to disk.
//
// Returns error when sandbox creation fails or a file cannot be written.
func (op *buildOperation) writeArtefacts(ctx context.Context) error {
	serverConfig := op.container.config.ServerConfig
	baseDir := deref(serverConfig.Paths.BaseDir, ".")

	factory, err := op.container.GetSandboxFactory()
	if err != nil {
		return fmt.Errorf("failed to get sandbox factory: %w", err)
	}
	outputDir := filepath.Join(baseDir, distDirName)
	outputSandbox, err := factory.Create("build-output", outputDir, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("failed to create output sandbox: %w", err)
	}

	fsWriter := generator_adapters.NewFSWriter(outputSandbox)
	for _, artefact := range op.artefacts {
		if err := fsWriter.WriteFile(ctx, artefact.SuggestedPath, artefact.Content); err != nil {
			return fmt.Errorf("failed to write artefact %s: %w", artefact.SuggestedPath, err)
		}
	}
	return nil
}

// writeManifest writes the final project manifest file to the dist folder.
//
// Returns error when sandbox creation fails or the manifest cannot be written.
func (op *buildOperation) writeManifest(ctx context.Context) error {
	serverConfig := op.container.config.ServerConfig
	baseDir := deref(serverConfig.Paths.BaseDir, ".")

	factory, err := op.container.GetSandboxFactory()
	if err != nil {
		return fmt.Errorf("failed to get sandbox factory: %w", err)
	}
	distDir := filepath.Join(baseDir, distDirName)
	outputSandbox, err := factory.Create("manifest-output", distDir, safedisk.ModeReadWrite)
	if err != nil {
		return fmt.Errorf("failed to create output sandbox: %w", err)
	}

	manifestEmitter, err := createManifestEmitterFromConfig(outputSandbox)
	if err != nil {
		return fmt.Errorf("creating manifest emitter: %w", err)
	}

	manifestFilename := manifestFilenameBinary
	manifestPath := filepath.Join(distDir, manifestFilename)

	if err := manifestEmitter.EmitCode(ctx, op.manifest, manifestPath); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	_, l := logger_domain.From(ctx, log)
	l.Internal("Wrote manifest", logger_domain.String("path", manifestPath))
	return nil
}

// buildStaticAssets runs the static asset build process and prints the
// summary to stderr.
//
// Returns error when a required service cannot be obtained or the build fails.
func (op *buildOperation) buildStaticAssets(ctx context.Context) error {
	result, err := op.runStaticAssetBuild(ctx)
	if err != nil {
		return err
	}

	summary := lifecycle_adapters.FormatBuildSummary(result)
	_, _ = fmt.Fprint(os.Stderr, summary)
	return nil
}

// runStaticAssetBuild runs the static asset build process, returning the
// result without printing a summary. The caller is responsible for printing
// or combining the result with other summaries.
//
// Returns *lifecycle_domain.BuildResult which holds task counts and failure
// details.
// Returns error when a required service cannot be obtained or the build fails.
func (op *buildOperation) runStaticAssetBuild(ctx context.Context) (*lifecycle_domain.BuildResult, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Building static assets...")

	registryService, err := op.container.GetRegistryService()
	if err != nil {
		return nil, fmt.Errorf("failed to get registry service for asset build: %w", err)
	}
	orchestratorService, err := op.container.GetOrchestratorService()
	if err != nil {
		return nil, fmt.Errorf("failed to get orchestrator service for asset build: %w", err)
	}
	renderer := op.container.GetRenderer()
	resolver, err := op.container.GetResolver()
	if err != nil {
		return nil, fmt.Errorf("failed to get resolver for asset build: %w", err)
	}

	bridge := op.container.GetArtefactBridge()
	eventBus := op.container.GetEventBus()

	buildFactory, buildFactoryErr := op.container.GetSandboxFactory()
	if buildFactoryErr != nil {
		return nil, fmt.Errorf("getting sandbox factory for static build service: %w", buildFactoryErr)
	}

	buildPathsConfig := op.buildLifecyclePathsConfig()
	buildService := lifecycle_adapters.NewBuildService(
		op.container.config, buildPathsConfig, registryService,
		orchestratorService, bridge, eventBus, renderer, resolver,
		op.container.externalComponents, buildFactory,
	)
	result, err := buildService.RunBuild(ctx)
	if err != nil {
		return nil, fmt.Errorf("static asset build failed: %w", err)
	}

	if result.HasFailures() {
		orchestratorService.Stop()
		return result, fmt.Errorf("build completed with %d failed task(s)", result.TotalFailed)
	}

	orchestratorService.Stop()
	return result, nil
}

// buildLifecyclePathsConfig constructs a LifecyclePathsConfig by dereferencing
// the pointer fields from the server configuration.
//
// Returns lifecycle_domain.LifecyclePathsConfig which holds the resolved path
// values for file system operations.
func (op *buildOperation) buildLifecyclePathsConfig() lifecycle_domain.LifecyclePathsConfig {
	paths := op.container.config.ServerConfig.Paths
	return lifecycle_domain.LifecyclePathsConfig{
		BaseDir:             deref(paths.BaseDir, "."),
		PagesSourceDir:      deref(paths.PagesSourceDir, "pages"),
		PartialsSourceDir:   deref(paths.PartialsSourceDir, "partials"),
		ComponentsSourceDir: deref(paths.ComponentsSourceDir, "components"),
		EmailsSourceDir:     deref(paths.EmailsSourceDir, "emails"),
		AssetsSourceDir:     deref(paths.AssetsSourceDir, "lib"),
		I18nSourceDir:       deref(paths.I18nSourceDir, "locales"),
	}
}

// BuildProject runs the build process for the piko generate command.
// It creates and runs a build operation with the given settings.
//
// Takes runMode (string) which sets how the build should run.
// Takes c (*Container) which holds the dependencies needed for the build.
//
// Returns error when the build fails to run.
func BuildProject(
	ctx context.Context,
	runMode string,
	c *Container,
) error {
	if cleanup := c.StartGeneratorProfiling(); cleanup != nil {
		defer cleanup()
	}

	previousGOGC := debug.SetGCPercent(buildGCPercent)
	debug.SetMemoryLimit(512 << 20)
	defer func() {
		debug.SetGCPercent(previousGOGC)
		debug.SetMemoryLimit(math.MaxInt64)
	}()

	operation := &buildOperation{
		runMode:   runMode,
		container: c,
	}
	return operation.execute(ctx)
}

// isErrorPage checks whether a filename follows the error page
// convention.
//
// Supported patterns:
//   - !NNN.pk -- exact status code (e.g., !404.pk, !500.pk)
//   - !NNN-NNN.pk -- status code range (e.g., !400-499.pk)
//   - !error.pk -- catch-all for any error status code
//
// Status codes must be valid HTTP codes in the range 100-599. For ranges,
// the minimum must not exceed the maximum.
//
// Takes filename (string) which is the base filename to check (e.g., "!404.pk").
//
// Returns errorPageResult which describes the error page type, or a zero-value
// result when the filename is not an error page.
func isErrorPage(filename string) errorPageResult {
	name := strings.TrimSuffix(filename, ".pk")
	if name == filename {
		return errorPageResult{}
	}
	if !strings.HasPrefix(name, "!") {
		return errorPageResult{}
	}
	codeString := name[1:]

	if codeString == "error" {
		return errorPageResult{isErrorPage: true, isCatchAll: true}
	}

	if parts := strings.SplitN(codeString, "-", 2); len(parts) == 2 {
		minCode, errMin := strconv.Atoi(parts[0])
		maxCode, errMax := strconv.Atoi(parts[1])
		if errMin == nil && errMax == nil &&
			minCode >= minHTTPStatusCode && minCode <= maxHTTPStatusCode &&
			maxCode >= minHTTPStatusCode && maxCode <= maxHTTPStatusCode &&
			minCode <= maxCode {
			return errorPageResult{isErrorPage: true, rangeMin: minCode, rangeMax: maxCode}
		}
		return errorPageResult{}
	}

	code, err := strconv.Atoi(codeString)
	if err != nil {
		return errorPageResult{}
	}
	if code < minHTTPStatusCode || code > maxHTTPStatusCode {
		return errorPageResult{}
	}
	return errorPageResult{isErrorPage: true, statusCode: code}
}

// generateSQL runs the querier code generator against all registered databases
// that have query files configured. For each database with QueryFS set, it
// builds a schema catalogue from the migration files, analyses the SQL queries,
// and writes typed Go code to the output directory.
//
// Returns error when any database's code generation fails.
func (op *buildOperation) generateSQL(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	if len(op.container.dbRegistrations) == 0 {
		l.Internal("No databases registered, skipping SQL generation")
		return nil
	}

	generatedCount := 0

	for name, reg := range op.container.dbRegistrations {
		if reg.QueryFS == nil || reg.EngineConfig.Engine == nil {
			continue
		}

		queryCount, err := op.generateSQLForDatabase(ctx, name, reg)
		if err != nil {
			return err
		}
		op.sqlQueryCount += queryCount
		generatedCount++
	}

	if generatedCount > 0 {
		l.Notice("SQL code generation complete",
			logger_domain.Int("databases", generatedCount))
	}

	return nil
}

// generateSQLForDatabase generates typed Go code for a single database
// registration.
//
// Takes name (string) which identifies the database registration.
// Takes reg (*DatabaseRegistration) which provides the database configuration.
//
// Returns int which is the number of query methods generated.
// Returns error when code generation or file writing fails.
func (op *buildOperation) generateSQLForDatabase(ctx context.Context, name string, reg *DatabaseRegistration) (int, error) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Generating SQL code", logger_domain.String("database", name))

	service, serviceErr := querier_domain.NewQuerierService(querier_domain.QuerierPorts{
		Engine:     reg.EngineConfig.Engine,
		Emitter:    emitter_go_sql.NewSQLEmitter(),
		FileReader: migration_sql.NewFSFileReader(reg.QueryFS),
	})
	if serviceErr != nil {
		return 0, fmt.Errorf("creating querier service for %q: %w", name, serviceErr)
	}

	config := &querier_dto.DatabaseConfig{
		MigrationDirectory: stringOrDefault(reg.MigrationDirectory, "migrations"),
		QueryDirectory:     stringOrDefault(reg.QueryDirectory, "queries"),
	}

	result, genErr := service.GenerateDatabase(ctx, stringOrDefault(reg.GeneratedPackageName, "generated"), config, "")
	if genErr != nil {
		return 0, fmt.Errorf("generating code for database %q: %w", name, genErr)
	}

	if err := checkSQLDiagnostics(l, name, result.Diagnostics); err != nil {
		return 0, err
	}

	outputDir := stringOrDefault(reg.GeneratedOutputDirectory, "db/generated")

	factory, factoryErr := op.container.GetSandboxFactory()
	if factoryErr != nil {
		return 0, fmt.Errorf("getting sandbox factory for database %q: %w", name, factoryErr)
	}

	if err := writeSQLGeneratedFiles(factory, outputDir, name, result.Files); err != nil {
		return 0, err
	}

	queryCount := 0
	for _, file := range result.Files {
		if strings.HasSuffix(file.Name, ".sql.go") {
			queryCount += strings.Count(string(file.Content), "func (queries *Queries)")
		}
	}

	l.Internal("SQL code generated",
		logger_domain.String("database", name),
		logger_domain.Int("queries", queryCount),
		logger_domain.String("output", outputDir))

	return queryCount, nil
}

// checkSQLDiagnostics scans diagnostics for errors and logs warnings.
//
// Takes l (logger_domain.Logger) which is the logger for warning output.
// Takes name (string) which identifies the database for error messages.
// Takes diagnostics ([]querier_dto.SourceError) which holds the diagnostics to check.
//
// Returns error when a diagnostic with error severity is found.
func checkSQLDiagnostics(l logger_domain.Logger, name string, diagnostics []querier_dto.SourceError) error {
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == querier_dto.SeverityError {
			return fmt.Errorf("SQL generation error in %q: %s:%d: %s", name, diagnostic.Filename, diagnostic.Line, diagnostic.Message)
		}
		l.Warn("SQL generation warning",
			logger_domain.String("database", name),
			logger_domain.String("file", diagnostic.Filename),
			logger_domain.String("message", diagnostic.Message))
	}
	return nil
}

// writeSQLGeneratedFiles writes the generated Go files to the output directory
// using safedisk for safe file operations.
//
// Takes outputDir (string) which is the directory to write files into.
// Takes name (string) which identifies the database for error messages.
// Takes files ([]querier_dto.GeneratedFile) which holds the files to write.
//
// Returns error when directory creation or file writing fails.
func writeSQLGeneratedFiles(factory safedisk.Factory, outputDir, name string, files []querier_dto.GeneratedFile) error {
	sandbox, sandboxErr := factory.Create("sql-output-"+name, outputDir, safedisk.ModeReadWrite)
	if sandboxErr != nil {
		return fmt.Errorf("creating output sandbox for database %q: %w", name, sandboxErr)
	}
	defer func() { _ = sandbox.Close() }()

	for _, file := range files {
		if len(file.Content) == 0 {
			continue
		}
		if writeErr := sandbox.WriteFile(file.Name, file.Content, generatedFilePerms); writeErr != nil {
			return fmt.Errorf("writing generated file %q for database %q: %w", file.Name, name, writeErr)
		}
	}

	return nil
}

// stringOrDefault returns value if non-empty, otherwise fallback.
//
// Takes value (string) which is the preferred value.
// Takes fallback (string) which is used when value is empty.
//
// Returns string which is value when non-empty, otherwise fallback.
func stringOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
