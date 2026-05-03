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

package driver_symbols_extract

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"

	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/interp/interp_adapters/driven_piko_symbols"
	"piko.sh/piko/internal/interp/interp_adapters/driven_system_symbols"
)

// maxPKFileSizeBytes caps the per-file byte count read during
// discovery. A single .pk file is typically well under 100 KiB; this
// cap guards against accidental or adversarial oversized inputs
// exhausting memory during the initial scan.
const maxPKFileSizeBytes = 4 * 1024 * 1024

// maxDiscoveredImports caps the deduplicated import set collected
// from .pk script blocks. The real-world limit is far lower; this is
// a defence-in-depth bound that still leaves generous headroom.
const maxDiscoveredImports = 10_000

// errTooManyImports is returned when the aggregate import set
// collected from .pk files exceeds maxDiscoveredImports. Real
// projects never come close; hitting this indicates an abusive or
// malformed input.
var errTooManyImports = errors.New("discovery exceeded import limit")

// errPKFileTooLarge is returned when a .pk file exceeds
// maxPKFileSizeBytes during discovery.
var errPKFileTooLarge = errors.New("pk file exceeds size limit")

// DiscoverOptions configures a Discover run. Zero values provide
// sensible defaults where possible.
type DiscoverOptions struct {
	// Root is the project root.
	//
	// Must be a valid Go module root (contain a go.mod file). Defaults
	// to "." when empty.
	Root string

	// SourceDirs lists directories under Root to scan for .pk files.
	// When empty, a conservative set of piko defaults is used
	// (pages, partials, components, emails, pdfs, pk, actions).
	SourceDirs []string

	// ExtraIgnored lists additional import paths to exclude from the
	// result on top of the stdlib, piko-native, component-reference,
	// and project-self filters.
	ExtraIgnored []string

	// BuildTags are forwarded to the downstream packages.Load call so
	// build-constrained files in the discovered packages are resolved
	// consistently with the caller's environment. Discovery itself
	// does not consult build tags.
	BuildTags []string
}

// DiscoverResult captures the output of a Discover run. Values are
// deterministic: slices are sorted and deduplicated.
type DiscoverResult struct {
	// RequiredImports lists third-party import paths referenced by
	// .pk script blocks or the project's Go sources, excluding
	// stdlib, piko-native, and explicitly ignored packages. This is
	// the set that belongs in piko-symbols.yaml.
	RequiredImports []string

	// SkippedCgo lists discovered packages that use cgo and therefore
	// cannot be interpreted. These are reported so the user knows not
	// to attempt registering them.
	SkippedCgo []string

	// GenericCandidates lists discovered packages that export generic
	// types. These need a manual generic: block in the manifest and
	// are flagged so the user can address them.
	GenericCandidates []string
}

// Discover walks a project, collects every Go import required by its
// .pk script blocks and Go sources, filters out stdlib and piko-native
// packages, and returns the remaining set as DiscoverResult.
//
// Takes ctx (context.Context) which carries cancellation and logging.
// Takes opts (DiscoverOptions) which configures the run.
//
// Returns DiscoverResult which holds the deduplicated, sorted lists
// of discovered imports.
// Returns error when the project cannot be walked or loaded.
func Discover(ctx context.Context, opts DiscoverOptions) (DiscoverResult, error) {
	if err := ctx.Err(); err != nil {
		return DiscoverResult{}, fmt.Errorf("discover: %w", err)
	}

	root := opts.Root
	if root == "" {
		root = "."
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return DiscoverResult{}, fmt.Errorf("resolving root %q: %w", opts.Root, err)
	}

	sourceDirs := opts.SourceDirs
	if len(sourceDirs) == 0 {
		sourceDirs = defaultPkSourceDirs()
	}

	pkImports, err := collectPkImports(ctx, root, sourceDirs)
	if err != nil {
		return DiscoverResult{}, err
	}

	ignored := buildIgnoreSet(opts.ExtraIgnored)

	candidates := filterImports(pkImports, ignored)
	if len(candidates) == 0 {
		return DiscoverResult{RequiredImports: nil}, nil
	}

	if err := ctx.Err(); err != nil {
		return DiscoverResult{}, fmt.Errorf("discover: %w", err)
	}

	resolved, cgo, generic := inspectDirectPackages(root, opts.BuildTags, candidates)

	return DiscoverResult{
		RequiredImports:   resolved,
		SkippedCgo:        cgo,
		GenericCandidates: generic,
	}, nil
}

// filterImports applies the component-reference and already-registered
// filters to the raw set of imports collected from .pk script blocks.
// Returns a sorted slice of candidate paths.
//
// Discover only filters paths piko already provides symbols for.
// Unregistered stdlib packages (for example os/exec, net/http) and
// unregistered piko packages (anything outside driven_piko_symbols)
// intentionally surface so the user can add them to the manifest -
// silently assuming "stdlib means registered" would hide real gaps
// that only materialise at dev-i runtime.
//
// Project-local imports are not filtered here: user code under the
// project module is consumed by the interpreter the same way as any
// third-party dependency and therefore needs symbol registration.
//
// Takes pkImports (map[string]struct{}) which is the raw import set.
// Takes ignored (map[string]struct{}) which holds the already-
// registered stdlib and piko paths, plus user-supplied exclusions.
//
// Returns []string of sorted, deduplicated candidate imports.
func filterImports(pkImports, ignored map[string]struct{}) []string {
	result := make(map[string]struct{}, len(pkImports))
	for path := range pkImports {
		if isComponentReference(path) {
			continue
		}
		if _, skip := ignored[path]; skip {
			continue
		}
		result[path] = struct{}{}
	}
	return sortedKeys(result)
}

// isComponentReference reports whether an import path is a piko
// component reference (ending in .pk) rather than a real Go package.
// .pk imports inside script blocks resolve at compile time to the
// compiled partial, not to a registered Go package.
//
// Takes path (string) which is the import path to classify.
//
// Returns bool which is true for piko .pk component references.
func isComponentReference(path string) bool {
	return strings.HasSuffix(path, ".pk")
}

// inspectDirectPackages runs packages.Load on exactly the direct
// imports collected from .pk files and classifies each result.
// No dependency walking - just the seam itself.
//
// Candidates that resolve to a real Go package with at least one .go
// file are kept. Candidates that don't resolve (virtual piko paths,
// missing modules) are silently dropped. Cgo and generic-exporting
// packages are surfaced alongside the resolved set so callers can
// warn users.
//
// Takes root (string) which is the absolute project root, used as
// packages.Config.Dir so go/packages resolves imports against the
// project module.
// Takes buildTags ([]string) which is forwarded to packages.Load.
// Takes candidates ([]string) which is the filtered candidate list.
//
// Returns three sorted slices: resolved Go packages, cgo packages,
// and generic-exporting packages. When packages.Load fails outright
// the candidates are returned as-is to avoid hiding real user
// inputs; callers can still validate via subsequent tooling.
func inspectDirectPackages(root string, buildTags, candidates []string) (resolved, cgo, generic []string) {
	if len(candidates) == 0 {
		return nil, nil, nil
	}

	config := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedImports |
			packages.NeedTypes,
		Dir: root,
	}
	if len(buildTags) > 0 {
		config.BuildFlags = []string{"-tags=" + strings.Join(buildTags, ",")}
	}

	loaded, err := packages.Load(config, candidates...)
	if err != nil {
		return candidates, nil, nil
	}

	loadedByPath := indexPackagesByPath(loaded)

	resolvedSet := make(map[string]struct{})
	cgoSet := make(map[string]struct{})
	genericSet := make(map[string]struct{})

	for _, path := range candidates {
		pkg, ok := loadedByPath[path]
		if !ok || !packageHasGoSource(pkg) {
			continue
		}
		resolvedSet[path] = struct{}{}
		if packageUsesCgo(pkg) {
			cgoSet[path] = struct{}{}
			continue
		}
		if packageExportsGenericType(pkg) {
			genericSet[path] = struct{}{}
		}
	}
	return sortedKeys(resolvedSet), sortedKeys(cgoSet), sortedKeys(genericSet)
}

// indexPackagesByPath returns a lookup table of loaded packages keyed
// by PkgPath so inspectDirectPackages can honour the caller's original
// ordering and detect missing entries deterministically.
//
// Takes loaded ([]*packages.Package) which is the packages.Load
// output.
//
// Returns a map keyed by PkgPath; nil entries and empty paths are
// skipped.
func indexPackagesByPath(loaded []*packages.Package) map[string]*packages.Package {
	result := make(map[string]*packages.Package, len(loaded))
	for _, pkg := range loaded {
		if pkg == nil || pkg.PkgPath == "" {
			continue
		}
		result[pkg.PkgPath] = pkg
	}
	return result
}

// packageHasGoSource reports whether a loaded package contains any
// Go source files. Virtual paths such as piko.sh/piko/docs (a docs
// directory with no Go code) surface in packages.Load output with an
// empty GoFiles list; they must be excluded from the registered set.
//
// Takes pkg (*packages.Package) which is the loaded package.
//
// Returns bool which is true when the package has at least one .go file.
func packageHasGoSource(pkg *packages.Package) bool {
	if pkg == nil {
		return false
	}
	return len(pkg.GoFiles) > 0 || len(pkg.CompiledGoFiles) > 0
}

// collectPkImports walks the given source directories under root,
// parses every .pk file via the annotator, extracts each script block
// as Go, and returns the union of import paths discovered.
//
// Takes ctx (context.Context) which carries cancellation and logging.
// Takes root (string) which is the absolute project root.
// Takes sourceDirs ([]string) which are directory names under root to
// scan (relative, not absolute).
//
// Returns map[string]struct{} which is a set keyed by import path,
// holding every import referenced by .pk script blocks.
// Returns error when file I/O or parsing fails.
func collectPkImports(ctx context.Context, root string, sourceDirs []string) (map[string]struct{}, error) {
	imports := make(map[string]struct{})

	for _, directoryName := range sourceDirs {
		directoryPath := filepath.Join(root, directoryName)
		if err := walkPkDirectory(ctx, directoryPath, imports); err != nil {
			return nil, err
		}
	}

	return imports, nil
}

// walkPkDirectory walks a single source directory under the project
// root and appends every script block's imports to the shared set.
// Missing or non-directory paths are treated as no-ops so callers can
// pass every conventional source dir without checking existence.
//
// Takes ctx (context.Context) which carries cancellation.
// Takes directoryPath (string) which is the absolute path to the
// directory to walk.
// Takes imports (map[string]struct{}) which accumulates the results.
//
// Returns error when I/O or parsing fails, or when discovery limits
// are exceeded.
func walkPkDirectory(ctx context.Context, directoryPath string, imports map[string]struct{}) error {
	info, err := os.Stat(directoryPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat %s: %w", directoryPath, err)
	}
	if !info.IsDir() {
		return nil
	}

	walkErr := filepath.WalkDir(directoryPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if !isCandidatePKFile(entry) {
			return nil
		}
		if len(imports) >= maxDiscoveredImports {
			return fmt.Errorf("%w: %d", errTooManyImports, len(imports))
		}
		return appendScriptImports(ctx, path, imports)
	})
	if walkErr != nil {
		return fmt.Errorf("walking %s: %w", directoryPath, walkErr)
	}
	return nil
}

// isCandidatePKFile reports whether a WalkDir entry is a .pk file that
// should participate in discovery. Directories, non-.pk files, and
// files whose name starts with "_" are excluded.
//
// Takes entry (fs.DirEntry) which is the current walk entry.
//
// Returns bool which is true when the entry should be parsed.
func isCandidatePKFile(entry fs.DirEntry) bool {
	if entry.IsDir() {
		return false
	}
	name := entry.Name()
	if strings.HasPrefix(name, "_") {
		return false
	}
	return strings.HasSuffix(name, ".pk")
}

// appendScriptImports parses a single .pk file, extracts its Go script
// block, parses that as Go, and adds every import spec it contains to
// the provided set.
//
// Takes ctx (context.Context) which carries cancellation and logging.
// Takes path (string) which is the absolute path to the .pk file.
// Takes imports (map[string]struct{}) which receives the discovered
// paths.
//
// Returns error when the file cannot be read or parsed.
func appendScriptImports(ctx context.Context, path string, imports map[string]struct{}) error {
	data, err := readBoundedFile(path, maxPKFileSizeBytes)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	_, sources, parseErr := annotator_domain.ParsePK(ctx, data, path)
	if sources.ScriptSource == "" {
		return nil
	}

	if parseErr != nil && !isAnnotatorSoftError(parseErr) {
		return fmt.Errorf("parsing %s: %w", path, parseErr)
	}

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, sources.ScriptSource, parser.ImportsOnly|parser.SkipObjectResolution)
	if err != nil {
		return fmt.Errorf("parsing Go script in %s: %w", path, err)
	}

	for _, spec := range file.Imports {
		value := strings.Trim(spec.Path.Value, `"`)
		if value == "" {
			continue
		}
		if len(imports) >= maxDiscoveredImports {
			return fmt.Errorf("%w: %d", errTooManyImports, len(imports))
		}
		imports[value] = struct{}{}
	}
	return nil
}

// readBoundedFile reads a file up to limit bytes and returns an error
// that wraps errPKFileTooLarge when the file would exceed the cap.
// io.LimitReader silently truncates; we explicitly surface the
// truncation so callers avoid acting on a partial file.
//
// Takes path (string) which is the file to read.
// Takes limit (int64) which is the maximum acceptable byte count.
//
// Returns the file bytes or an error when I/O fails or the file is
// too large.
func readBoundedFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path) //nolint:gosec // path is constrained by WalkDir under a caller-supplied project root
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var buffer bytes.Buffer
	reader := io.LimitReader(file, limit+1)
	if _, err := io.Copy(&buffer, reader); err != nil {
		return nil, err
	}
	if int64(buffer.Len()) > limit {
		return nil, fmt.Errorf("%w: %s (>%d bytes)", errPKFileTooLarge, path, limit)
	}
	return buffer.Bytes(), nil
}

// isAnnotatorSoftError reports whether an error from ParsePK should
// be tolerated during discovery.
//
// Delegates to annotator_domain.IsParseSoftError so the classification
// is driven by error-type matching rather than substring search on
// English error messages.
//
// Takes err (error) which is the annotator error to classify.
//
// Returns true when the error is a tolerable parse failure.
func isAnnotatorSoftError(err error) bool {
	return annotator_domain.IsParseSoftError(err)
}

// buildIgnoreSet assembles the complete set of import paths that must
// be excluded from the discovered list. This includes the Go standard
// library (registered via driven_system_symbols), piko-native
// packages (driven_piko_symbols), and any user-provided extras.
//
// Takes extras ([]string) which are caller-supplied ignores.
//
// Returns a set keyed by import path.
func buildIgnoreSet(extras []string) map[string]struct{} {
	ignored := make(map[string]struct{})
	for path := range driven_system_symbols.Symbols {
		ignored[path] = struct{}{}
	}
	for path := range driven_piko_symbols.Symbols {
		ignored[path] = struct{}{}
	}
	for _, path := range extras {
		ignored[path] = struct{}{}
	}
	return ignored
}

// packageUsesCgo reports whether any file in the loaded package
// imports the pseudo-package "C", which indicates cgo usage and
// therefore incompatibility with the interpreter.
//
// Takes pkg (*packages.Package) which is the loaded package.
//
// Returns bool which is true when cgo is used.
func packageUsesCgo(pkg *packages.Package) bool {
	for _, imp := range pkg.Imports {
		if imp != nil && imp.PkgPath == "C" {
			return true
		}
	}
	for _, goFile := range pkg.GoFiles {
		if strings.HasSuffix(goFile, ".cgo1.go") {
			return true
		}
	}
	return false
}

// packageExportsGenericType reports whether the package has any
// exported type declaration with type parameters. Such packages need
// manual generic: configuration in the manifest.
//
// Takes pkg (*packages.Package) which is the loaded package.
//
// Returns bool which is true when a generic exported type is present.
func packageExportsGenericType(pkg *packages.Package) bool {
	if pkg.Types == nil {
		return false
	}
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		typeName, isTypeName := object.(*types.TypeName)
		if !isTypeName || !typeName.Exported() {
			continue
		}
		named, isNamed := typeName.Type().(*types.Named)
		if !isNamed {
			continue
		}
		if named.TypeParams() != nil && named.TypeParams().Len() > 0 {
			return true
		}
	}
	return false
}

// sortedKeys returns the keys of a set as a sorted slice so callers
// receive deterministic output.
//
// Takes set (map[string]struct{}) which is the source set.
//
// Returns []string sorted ascending.
func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// defaultPkSourceDirs returns the set of project-relative directories
// that conventionally hold .pk files. The conservative default matches
// piko's path defaults so Discover works without explicit configuration.
//
// Returns []string which is the default directory list.
func defaultPkSourceDirs() []string {
	return []string{"pages", "partials", "components", "emails", "pdfs", "pk", "actions"}
}
