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

package quickpackages

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/tools/go/gcexportdata"
	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

// overlayFileMode is the permission mode for temporary overlay files.
const overlayFileMode = 0o600

// exportReaderBufSize is the buffer size for the buffered reader wrapping
// export data files.
const exportReaderBufSize = 65536

// logKeyPackage is the structured logging key for package paths.
const logKeyPackage = "package"

// logKeyError is the structured logging key for error messages.
const logKeyError = "error"

// parseFileFunc is the signature for ParseFile callbacks used during loading.
type parseFileFunc = func(*token.FileSet, string, []byte) (*ast.File, error)

// Load loads and type-checks Go packages for compatibility with
// existing code. It optimises for the common case where only root
// packages need full TypesInfo and Syntax, while dependency packages
// only need their types.Package for type resolution.
//
// Takes cfg (*packages.Config) which provides loading configuration
// including directory, environment, build flags, overlay, and
// ParseFile callback.
// Takes patterns ([]string) which specifies the packages to load.
//
// Returns []*Package which contains the loaded root
// packages with their full dependency graph.
// Returns error when go list fails or packages contain
// errors.
func Load(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
	if cfg == nil {
		cfg = &packages.Config{}
	}
	ctx, l := logger_domain.From(cfg.Context, log)

	fset := cfg.Fset
	if fset == nil {
		fset = token.NewFileSet()
	}

	parseInitial := selectParseInitial(cfg.ParseFile)

	t0 := time.Now()

	overlayFile, cleanup, err := writeOverlayJSON(cfg.Overlay)
	if err != nil {
		return nil, fmt.Errorf("quickpackages: writing overlay: %w", err)
	}
	defer cleanup()

	listed, err := runGoList(ctx, cfg, overlayFile, patterns)
	if err != nil {
		return nil, fmt.Errorf("quickpackages: go list: %w", err)
	}

	driverDur := time.Since(t0)
	t1 := time.Now()

	pkgMap, roots, err := buildPackageGraph(listed, fset, parseInitial, parseDep)
	if err != nil {
		return nil, fmt.Errorf("quickpackages: building graph: %w", err)
	}

	sizes := types.SizesFor("gc", goarch(cfg.Env))

	if err := parseAndTypeCheck(ctx, pkgMap, fset, sizes, cfg.Overlay); err != nil {
		return nil, fmt.Errorf("quickpackages: type-checking: %w", err)
	}

	refineDur := time.Since(t1)
	logLoadStats(l, driverDur, refineDur, pkgMap)

	return roots, nil
}

// logLoadStats logs summary statistics after a successful load.
//
// Takes l (logger_domain.Logger) which is the logger instance.
// Takes driverDur (time.Duration) which is the time spent in go
// list.
// Takes refineDur (time.Duration) which is the time spent parsing
// and type-checking.
// Takes pkgMap (map[string]*loaderPkg) which contains all loaded
// packages.
func logLoadStats(l logger_domain.Logger, driverDur, refineDur time.Duration, pkgMap map[string]*loaderPkg) {
	exportCount := 0
	sourceCount := 0
	for _, lp := range pkgMap {
		if !lp.initial && lp.Syntax == nil && lp.Types != nil {
			exportCount++
		} else if lp.Types != nil {
			sourceCount++
		}
	}

	l.Internal("load complete",
		logger_domain.Duration("driver", driverDur),
		logger_domain.Duration("refine", refineDur),
		logger_domain.Duration("total", driverDur+refineDur),
		logger_domain.Int("packages", len(pkgMap)),
		logger_domain.Int("export", exportCount),
		logger_domain.Int("source", sourceCount),
	)
}

// selectParseInitial returns the ParseFile callback for root
// packages. If the caller provided one, it is used; otherwise a
// default is returned.
//
// Takes callerParseFile (parseFileFunc) which is the caller-supplied
// parse callback, or nil.
//
// Returns parseFileFunc which is the selected parse callback.
func selectParseInitial(callerParseFile parseFileFunc) parseFileFunc {
	if callerParseFile != nil {
		return callerParseFile
	}
	return func(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
		return parser.ParseFile(fset, filename, src, parser.AllErrors|parser.ParseComments|parser.SkipObjectResolution)
	}
}

// parseDep parses a dependency package source file using the leanest
// possible configuration.
//
// It uses no comments, no object resolution, and strips all function
// bodies except init() and generics. This is safe because the type
// checker runs with IgnoreFuncBodies=true for dep packages.
//
// Takes fset (*token.FileSet) which is the shared file set.
// Takes filename (string) which is the path to the source file.
// Takes src ([]byte) which is the file contents.
//
// Returns *File which is the parsed file with bodies
// stripped.
// Returns error when parsing fails.
func parseDep(fset *token.FileSet, filename string, src []byte) (*ast.File, error) {
	file, err := parser.ParseFile(fset, filename, src, parser.AllErrors|parser.SkipObjectResolution)
	if err != nil {
		return file, err
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name == "init" {
			continue
		}
		if fn.Type.TypeParams != nil && fn.Type.TypeParams.NumFields() > 0 {
			continue
		}
		fn.Body = nil
	}
	return file, nil
}

// Visit visits all packages in the import graph in depth-first post-order.
// It provides the same interface as packages.Visit for compatibility.
//
// Takes pkgs ([]*packages.Package) which are the root packages to start from.
// Takes pre (func(*packages.Package) bool) which is called before visiting
// imports. If pre returns false, imports are not visited.
// Takes post (func(*packages.Package)) which is called after visiting imports.
func Visit(pkgs []*packages.Package, pre func(*packages.Package) bool, post func(*packages.Package)) {
	packages.Visit(pkgs, pre, post)
}

// goListPkg is the JSON structure returned by go list -json -deps
// -export.
type goListPkg struct {
	// Module is the module containing this package.
	Module *packages.Module

	// Error is the error reported by go list, if any.
	Error *goListError

	// ImportMap maps source import paths to resolved import paths.
	ImportMap map[string]string

	// Dir is the absolute path to the package directory.
	Dir string

	// ImportPath is the canonical import path of the package.
	ImportPath string

	// Name is the package name declared in the source.
	Name string

	// Export is the path to compiled export data in the build
	// cache.
	Export string

	// GoFiles is the list of Go source file names.
	GoFiles []string

	// CompiledGoFiles is the list of compiled Go source file
	// names.
	CompiledGoFiles []string

	// Imports is the list of import paths used by the package.
	Imports []string

	// DepOnly is true if the package is only a dependency, not a
	// root.
	DepOnly bool
}

// goListError is the error structure from go list JSON.
type goListError struct {
	// Pos is the position string where the error occurred.
	Pos string

	// Err is the error message text.
	Err string
}

// loaderPkg extends packages.Package with loading state.
type loaderPkg struct {
	*packages.Package

	// parseFile is the parse callback for this package.
	parseFile parseFileFunc

	// importMap maps source import paths to resolved import
	// paths.
	importMap map[string]string

	// exportFile is the path to compiled export data, or empty if
	// unavailable.
	exportFile string

	// preds is the list of packages that import this one.
	preds []*loaderPkg

	// goFiles is the list of absolute paths to Go source files.
	goFiles []string

	// importPaths is the list of raw import paths from go list.
	importPaths []string

	// unfinishedDeps is the count of dependencies not yet
	// type-checked.
	unfinishedDeps atomic.Int32

	// processing guards against double-enqueue.
	processing atomic.Int32

	// initial is true if this is a root package, not DepOnly.
	initial bool
}

// overlayJSON is the JSON structure expected by go list -overlay.
type overlayJSON struct {
	// Replace maps logical file paths to temporary file paths.
	Replace map[string]string `json:"replace"`
}

// writeOverlayJSON writes the in-memory overlay to a temporary JSON file
// in the format expected by go list -overlay.
//
// Takes overlay (map[string][]byte) which maps file paths to their content.
//
// Returns the path to the overlay JSON file, a cleanup function, and any
// error.
func writeOverlayJSON(overlay map[string][]byte) (string, func(), error) {
	noop := func() {}
	if len(overlay) == 0 {
		return "", noop, nil
	}

	overlayFiles := collectOverlayFiles(overlay)
	if len(overlayFiles) == 0 {
		return "", noop, nil
	}

	tmpDir, err := os.MkdirTemp("", "quickpackages-overlay-*")
	if err != nil {
		return "", noop, fmt.Errorf("creating temp dir: %w", err)
	}

	sandbox, sandboxErr := safedisk.NewSandbox(tmpDir, safedisk.ModeReadWrite)
	if sandboxErr != nil {
		_ = os.RemoveAll(tmpDir)
		return "", noop, fmt.Errorf("creating overlay sandbox: %w", sandboxErr)
	}
	cleanupFn := func() {
		_ = sandbox.Close()
		_ = os.RemoveAll(tmpDir)
	}

	overlayPath, err := writeOverlayToDisk(sandbox, overlayFiles, overlay)
	if err != nil {
		cleanupFn()
		return "", noop, err
	}

	return overlayPath, cleanupFn, nil
}

// collectOverlayFiles returns all overlay paths that need to be
// written to the overlay JSON for go list.
//
// All entries are included because on-disk files may have different
// build tags than their overlay replacements (e.g. generated files
// with //go:build !piko_analysis that the overlay replaces with
// analysis-friendly versions).
//
// Takes overlay (map[string][]byte) which maps logical file paths to
// their content.
//
// Returns map[string]string mapping logical paths to content strings.
func collectOverlayFiles(overlay map[string][]byte) map[string]string {
	result := make(map[string]string, len(overlay))
	for logicalPath, content := range overlay {
		result[logicalPath] = string(content)
	}
	return result
}

// writeOverlayToDisk writes overlay files to a sandbox and produces
// the overlay JSON file for go list.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed file
// system access for writing overlay files.
// Takes newFiles (map[string]string) which maps logical paths to
// content.
// Takes overlay (map[string][]byte) which is the original overlay
// for raw byte content.
//
// Returns string which is the path to the overlay JSON file.
// Returns error when writing fails.
func writeOverlayToDisk(sandbox safedisk.Sandbox, newFiles map[string]string, overlay map[string][]byte) (string, error) {
	replace := make(map[string]string, len(newFiles))
	i := 0
	for logicalPath := range newFiles {
		content := overlay[logicalPath]
		base := filepath.Base(logicalPath)
		relPath := fmt.Sprintf("%d-%s", i, base)
		if err := sandbox.WriteFile(relPath, content, overlayFileMode); err != nil {
			return "", fmt.Errorf("writing overlay file %s: %w", base, err)
		}
		replace[logicalPath] = filepath.Join(sandbox.Root(), relPath)
		i++
	}

	jsonData, err := json.Marshal(overlayJSON{Replace: replace})
	if err != nil {
		return "", fmt.Errorf("marshalling overlay JSON: %w", err)
	}

	if err := sandbox.WriteFile("overlay.json", jsonData, overlayFileMode); err != nil {
		return "", fmt.Errorf("writing overlay JSON: %w", err)
	}

	return filepath.Join(sandbox.Root(), "overlay.json"), nil
}

// runGoList executes go list -json -deps and returns the parsed
// packages.
//
// Takes ctx (context.Context) for cancellation.
// Takes cfg (*packages.Config) for directory, env, and build flags.
// Takes overlayFile (string) which is the path to the overlay JSON
// file, or empty if no overlay.
// Takes patterns ([]string) which are the package patterns to list.
//
// Returns []goListPkg which contains all listed packages
// including dependencies.
// Returns error when the command fails.
func runGoList(ctx context.Context, cfg *packages.Config, overlayFile string, patterns []string) ([]goListPkg, error) {
	args := []string{"list", "-json=ImportPath,Name,Dir,GoFiles,Imports,ImportMap,DepOnly,Module,Error,Export", "-deps", "-export", "-e"}
	if overlayFile != "" {
		args = append(args, "-overlay="+overlayFile)
	}
	args = append(args, cfg.BuildFlags...)
	args = append(args, patterns...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = cfg.Dir
	cmd.Env = cfg.Env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list: %w\nstderr: %s", err, stderr.String())
	}

	var result []goListPkg
	dec := json.NewDecoder(&stdout)
	for dec.More() {
		var pkg goListPkg
		if err := dec.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("decoding go list output: %w", err)
		}
		result = append(result, pkg)
	}

	return result, nil
}

// buildPackageGraph creates packages.Package structs from go list output
// and wires up the import graph.
//
// Takes listed ([]goListPkg) which is the go list JSON output.
// Takes fset (*token.FileSet) which is the shared file set.
//
// Returns a map of all loader packages keyed by import path, the root
// packages slice, and any error.
func buildPackageGraph(
	listed []goListPkg,
	fset *token.FileSet,
	parseInitial, parseDepFn parseFileFunc,
) (map[string]*loaderPkg, []*packages.Package, error) {
	pkgMap, roots := createPackages(listed, fset, parseInitial, parseDepFn)
	wireImports(pkgMap)
	return pkgMap, roots, nil
}

// createPackages allocates all packages.Package and loaderPkg structs
// from go list output. It uses contiguous slab allocation for cache
// locality.
//
// Takes listed ([]goListPkg) which is the go list JSON output.
// Takes fset (*token.FileSet) which is the shared file set.
// Takes parseInitial (parseFileFunc) which is the parse callback for
// root packages.
// Takes parseDepFn (parseFileFunc) which is the parse callback for
// dependency packages.
//
// Returns map[string]*loaderPkg which contains all packages
// keyed by import path.
// Returns []*Package which contains only the root packages.
func createPackages(
	listed []goListPkg,
	fset *token.FileSet,
	parseInitial, parseDepFn parseFileFunc,
) (map[string]*loaderPkg, []*packages.Package) {
	pkgMap := make(map[string]*loaderPkg, len(listed))
	var roots []*packages.Package

	pkgSlab := make([]packages.Package, len(listed))
	lPkgSlab := make([]loaderPkg, len(listed))

	for i := range listed {
		lp := &listed[i]
		goFiles := make([]string, len(lp.GoFiles))
		for j, f := range lp.GoFiles {
			goFiles[j] = filepath.Join(lp.Dir, f)
		}

		pkg := &pkgSlab[i]
		pkg.ID = lp.ImportPath
		pkg.PkgPath = lp.ImportPath
		pkg.Name = lp.Name
		pkg.Fset = fset
		pkg.GoFiles = goFiles
		pkg.Module = lp.Module
		pkg.Imports = make(map[string]*packages.Package)

		if lp.Error != nil {
			pkg.Errors = append(pkg.Errors, packages.Error{
				Pos:  lp.Error.Pos,
				Msg:  lp.Error.Err,
				Kind: packages.ListError,
			})
		}

		lPkg := &lPkgSlab[i]
		lPkg.Package = pkg
		lPkg.goFiles = goFiles
		lPkg.importPaths = lp.Imports
		lPkg.importMap = lp.ImportMap
		lPkg.exportFile = lp.Export
		lPkg.initial = !lp.DepOnly
		if lPkg.initial {
			lPkg.parseFile = parseInitial
		} else {
			lPkg.parseFile = parseDepFn
		}
		pkgMap[lp.ImportPath] = lPkg

		if lPkg.initial {
			roots = append(roots, pkg)
		}
	}

	return pkgMap, roots
}

// wireImports resolves import paths and counts dependencies for each
// package.
//
// Takes pkgMap (map[string]*loaderPkg) which contains all packages
// keyed by import path.
func wireImports(pkgMap map[string]*loaderPkg) {
	for _, lp := range pkgMap {
		depCount := 0
		for _, impPath := range lp.importPaths {
			resolved := resolveImportPath(lp.importMap, impPath)
			dep, ok := pkgMap[resolved]
			if !ok {
				continue
			}
			lp.Imports[impPath] = dep.Package
			dep.preds = append(dep.preds, lp)
			depCount++
		}

		for srcPath, resolvedPath := range lp.importMap {
			if dep, ok := lp.Imports[resolvedPath]; ok {
				lp.Imports[srcPath] = dep
			}
		}
		lp.unfinishedDeps.Store(safeconv.IntToInt32(depCount))
	}
}

// resolveImportPath resolves an import path through the ImportMap,
// handling vendored stdlib packages where source says
// "golang.org/x/..." but the actual package is
// "vendor/golang.org/x/...".
//
// Takes importMap (map[string]string) which maps source paths to
// resolved paths.
// Takes impPath (string) which is the import path to resolve.
//
// Returns string which is the resolved import path.
func resolveImportPath(importMap map[string]string, impPath string) string {
	if importMap != nil {
		if mapped, ok := importMap[impPath]; ok {
			return mapped
		}
	}
	return impPath
}

// typeCheckState holds shared state for the parallel type-checking
// pass.
type typeCheckState struct {
	// exportBufPool recycles byte buffers used for reading export data
	// files. Each buffer is used for a single io.ReadAll then returned
	// to the pool after gcexportdata.Read finishes decoding.
	exportBufPool sync.Pool

	// sizes provides type size information for the target arch.
	sizes types.Sizes

	// readFile reads a file by path. Defaults to os.ReadFile;
	// injectable for testing.
	readFile func(string) ([]byte, error)

	// openFile opens a file by path. Defaults to os.Open;
	// injectable for testing.
	openFile func(string) (io.ReadCloser, error)

	// overlay maps file paths to in-memory content.
	overlay map[string][]byte

	// exportImports is shared across gcexportdata.Read calls.
	exportImports map[string]*types.Package

	// fset is the shared file set for all packages.
	fset *token.FileSet

	// cpuLimit bounds the number of concurrent type-check workers.
	cpuLimit chan struct{}

	// wg tracks in-flight goroutines.
	wg sync.WaitGroup

	// exportMu protects concurrent access to exportImports.
	exportMu sync.Mutex
}

// parseAndTypeCheck parses source files and type-checks all packages in
// dependency order using parallel workers.
//
// Takes ctx (context.Context) for cancellation.
// Takes pkgMap (map[string]*loaderPkg) which contains all packages.
// Takes fset (*token.FileSet) which is the shared file set.
// Takes sizes (types.Sizes) which provides type size information.
// Takes overlay (map[string][]byte) which provides in-memory file contents.
//
// Returns error when type-checking fails fatally.
func parseAndTypeCheck(
	ctx context.Context,
	pkgMap map[string]*loaderPkg,
	fset *token.FileSet,
	sizes types.Sizes,
	overlay map[string][]byte,
) error {
	exportImports := make(map[string]*types.Package, len(pkgMap))
	for pkgPath, lp := range pkgMap {
		exportImports[pkgPath] = types.NewPackage(pkgPath, lp.Name)
	}

	st := &typeCheckState{
		readFile: os.ReadFile,
		openFile: func(name string) (io.ReadCloser, error) {
			return os.Open(name) //nolint:gosec // trusted go list output
		},
		fset:          fset,
		sizes:         sizes,
		overlay:       overlay,
		exportImports: exportImports,
		cpuLimit:      make(chan struct{}, runtime.GOMAXPROCS(0)),
	}

	for _, lp := range pkgMap {
		if lp.unfinishedDeps.Load() == 0 {
			st.enqueue(ctx, lp)
		}
	}

	st.wg.Wait()
	return nil
}

// enqueue starts processing a package once all its deps are ready.
//
// Takes ctx (context.Context) for cancellation.
// Takes lp (*loaderPkg) which is the package to process.
func (st *typeCheckState) enqueue(ctx context.Context, lp *loaderPkg) {
	st.wg.Go(func() {
		if ctx.Err() != nil {
			return
		}

		if lp.processing.Add(1) != 1 {
			return
		}
		st.loadSinglePackage(ctx, lp)

		for _, pred := range lp.preds {
			if pred.unfinishedDeps.Add(-1) == 0 {
				st.enqueue(ctx, pred)
			}
		}
	})
}

// loadSinglePackage loads a single package, choosing the fastest
// path: export data for deps when available, or source parsing plus
// type-checking.
//
// Takes ctx (context.Context) for cancellation.
// Takes lp (*loaderPkg) which is the package to load.
func (st *typeCheckState) loadSinglePackage(ctx context.Context, lp *loaderPkg) {
	if ctx.Err() != nil {
		return
	}

	if lp.PkgPath == "unsafe" {
		lp.Types = types.Unsafe
		st.registerExportPkg(lp.PkgPath, types.Unsafe)
		return
	}

	if !lp.initial && lp.exportFile != "" {
		if st.loadFromExportData(ctx, lp) {
			return
		}
	}

	lp.Syntax = st.parsePackageFiles(ctx, lp)
	lp.Types = types.NewPackage(lp.PkgPath, lp.Name)
	st.typeCheckPackage(ctx, lp)
	st.registerExportPkg(lp.PkgPath, lp.Types)
}

// loadFromExportData reads pre-compiled type information from the
// build cache, avoiding source parsing and type-checking entirely.
//
// Takes lp (*loaderPkg) which is the package to load export data
// for.
//
// Returns bool which is true if export data was loaded successfully.
//
// Concurrency: safe for concurrent use; serialises access to the
// shared exportImports map via exportMu.
func (st *typeCheckState) loadFromExportData(ctx context.Context, lp *loaderPkg) bool {
	if ctx.Err() != nil {
		return false
	}

	_, l := logger_domain.From(ctx, log)

	f, err := st.openFile(lp.exportFile)
	if err != nil {
		l.Warn("failed to open export file",
			logger_domain.String(logKeyPackage, lp.PkgPath),
			logger_domain.String(logKeyError, err.Error()),
		)
		return false
	}

	r, err := gcexportdata.NewReader(bufio.NewReaderSize(f, exportReaderBufSize))
	if err != nil {
		_ = f.Close()
		l.Warn("failed to create export reader",
			logger_domain.String(logKeyPackage, lp.PkgPath),
			logger_domain.String(logKeyError, err.Error()),
		)
		return false
	}

	buf, ok := st.exportBufPool.Get().(*bytes.Buffer)
	if !ok || buf == nil {
		buf = bytes.NewBuffer(make([]byte, 0, 256*1024))
	} else {
		buf.Reset()
	}
	_, err = buf.ReadFrom(r)
	_ = f.Close()
	if err != nil {
		st.exportBufPool.Put(buf)
		l.Warn("failed to read export data bytes",
			logger_domain.String(logKeyPackage, lp.PkgPath),
			logger_domain.String(logKeyError, err.Error()),
		)
		return false
	}

	st.exportMu.Lock()
	pkg, err := gcexportdata.Read(bytes.NewReader(buf.Bytes()), st.fset, st.exportImports, lp.PkgPath)
	st.exportMu.Unlock()

	st.exportBufPool.Put(buf)

	if err != nil {
		l.Warn("failed to read export data",
			logger_domain.String(logKeyPackage, lp.PkgPath),
			logger_domain.String(logKeyError, err.Error()),
		)
		return false
	}

	lp.Types = pkg
	lp.TypesSizes = st.sizes
	return true
}

// registerExportPkg adds a loaded package to the shared export
// imports map so that gcexportdata.Read can find it when loading
// dependents.
//
// Takes pkgPath (string) which is the import path of the package.
// Takes pkg (*types.Package) which is the loaded type information.
//
// Concurrency: safe for concurrent use; serialises access to the
// shared exportImports map via exportMu.
func (st *typeCheckState) registerExportPkg(pkgPath string, pkg *types.Package) {
	st.exportMu.Lock()
	st.exportImports[pkgPath] = pkg
	st.exportMu.Unlock()
}

// parsePackageFiles reads and parses all Go source files for a
// package.
//
// Takes ctx (context.Context) for cancellation.
// Takes lp (*loaderPkg) which is the package whose files to parse.
//
// Returns []*File which contains the successfully parsed
// files.
func (st *typeCheckState) parsePackageFiles(ctx context.Context, lp *loaderPkg) []*ast.File {
	syntax := make([]*ast.File, 0, len(lp.goFiles))
	for _, filename := range lp.goFiles {
		if ctx.Err() != nil {
			return syntax
		}
		src, ok := st.overlay[filename]
		if !ok {
			var err error
			src, err = st.readFile(filename)
			if err != nil {
				lp.Errors = append(lp.Errors, packages.Error{
					Msg:  fmt.Sprintf("reading %s: %v", filename, err),
					Kind: packages.ParseError,
				})
				continue
			}
		}

		file, err := lp.parseFile(st.fset, filename, src)
		if err != nil {
			lp.Errors = append(lp.Errors, packages.Error{
				Msg:  fmt.Sprintf("parsing %s: %v", filename, err),
				Kind: packages.ParseError,
			})
			if file == nil {
				continue
			}
		}
		syntax = append(syntax, file)
	}
	return syntax
}

// typeCheckPackage runs the Go type checker on a single package.
//
// Takes ctx (context.Context) for cancellation.
// Takes lp (*loaderPkg) which is the package to type-check.
func (st *typeCheckState) typeCheckPackage(ctx context.Context, lp *loaderPkg) {
	if ctx.Err() != nil {
		return
	}

	importer := importerFunc(func(path string) (*types.Package, error) {
		if path == "unsafe" {
			return types.Unsafe, nil
		}
		dep := lp.Imports[path]
		if dep == nil {
			return nil, fmt.Errorf("quickpackages: no package for import %q", path)
		}
		if dep.Types != nil && dep.Types.Complete() {
			return dep.Types, nil
		}
		return nil, fmt.Errorf("quickpackages: package %q not yet type-checked", path)
	})

	info := newTypesInfo(lp.initial)

	tc := &types.Config{
		Importer:                 importer,
		IgnoreFuncBodies:         !lp.initial,
		DisableUnusedImportCheck: !lp.initial,
		Error: func(err error) {
			pe := packages.Error{
				Kind: packages.TypeError,
			}
			if te, ok := errors.AsType[types.Error](err); ok {
				pe.Pos = st.fset.Position(te.Pos).String()
				pe.Msg = te.Msg
			} else {
				pe.Msg = err.Error()
			}
			lp.Errors = append(lp.Errors, pe)
		},
		Sizes: st.sizes,
	}
	if lp.Module != nil && lp.Module.GoVersion != "" {
		tc.GoVersion = "go" + lp.Module.GoVersion
	}

	st.cpuLimit <- struct{}{}
	_ = types.NewChecker(tc, st.fset, lp.Types, info).Files(lp.Syntax)
	<-st.cpuLimit

	lp.TypesInfo = info
	lp.TypesSizes = st.sizes
}

// newTypesInfo returns a fully initialised types.Info for root
// packages, or nil for dependency packages.
//
// Takes initial (bool) which indicates whether this is a root
// package.
//
// Returns *Info which is the type information collector, or
// nil for deps.
func newTypesInfo(initial bool) *types.Info {
	if !initial {
		return nil
	}
	return &types.Info{
		Types:        make(map[ast.Expr]types.TypeAndValue),
		Defs:         make(map[*ast.Ident]types.Object),
		Uses:         make(map[*ast.Ident]types.Object),
		Implicits:    make(map[ast.Node]types.Object),
		Instances:    make(map[*ast.Ident]types.Instance),
		Scopes:       make(map[ast.Node]*types.Scope),
		Selections:   make(map[*ast.SelectorExpr]*types.Selection),
		FileVersions: make(map[*ast.File]string),
	}
}

// goarch extracts the GOARCH value from the environment, defaulting to
// the runtime architecture.
//
// Takes env ([]string) which is the environment variable list.
//
// Returns string which is the target architecture.
func goarch(env []string) string {
	for _, e := range env {
		if v, ok := strings.CutPrefix(e, "GOARCH="); ok {
			return v
		}
	}
	return runtime.GOARCH
}

// importerFunc implements types.Importer via a function value.
type importerFunc func(path string) (*types.Package, error)

// Import satisfies the types.Importer interface.
//
// Takes path (string) which is the import path to resolve.
//
// Returns *Package which contains the imported package
// types.
// Returns error when the import path cannot be resolved.
func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }
