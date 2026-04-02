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

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

// dotSeparator is the dot character used to separate parts in action names.
const dotSeparator = "."

// ActionDiscoverer scans the actions/ directory and extracts preliminary action
// metadata. It runs during Stage 1.6 of the annotator pipeline, using lightweight
// AST parsing to identify action structs without full type checking.
//
// Full type resolution (input/output types, capabilities) happens later in
// Stage 3.5 after packages.Load() has run.
type ActionDiscoverer struct {
	// resolver provides path resolution and module name lookup.
	resolver resolver_domain.ResolverPort

	// fsReader provides access to the file system.
	fsReader FSReaderPort

	// pathsConfig holds the path settings.
	pathsConfig AnnotatorPathsConfig

	// inMemoryMode skips file system operations for WASM or testing use.
	inMemoryMode bool
}

// ActionDiscovererOption configures an ActionDiscoverer.
type ActionDiscovererOption func(*ActionDiscoverer)

// NewActionDiscoverer creates a new ActionDiscoverer.
//
// Takes resolver (resolver_domain.ResolverPort) for path resolution.
// Takes fsReader (FSReaderPort) for file system access.
// Takes pathsConfig (AnnotatorPathsConfig) for path settings.
// Takes opts (...ActionDiscovererOption) for optional configuration.
//
// Returns *ActionDiscoverer ready for action discovery.
func NewActionDiscoverer(
	resolver resolver_domain.ResolverPort,
	fsReader FSReaderPort,
	pathsConfig AnnotatorPathsConfig,
	opts ...ActionDiscovererOption,
) *ActionDiscoverer {
	ad := &ActionDiscoverer{
		resolver:    resolver,
		fsReader:    fsReader,
		pathsConfig: pathsConfig,
	}
	for _, opt := range opts {
		opt(ad)
	}
	return ad
}

// Discover scans the actions/ directory and returns preliminary action
// metadata. It uses lightweight AST parsing to identify structs embedding
// piko.ActionMetadata, deferring full type resolution to Stage 3.5.
//
// Returns *annotator_dto.ActionManifest containing discovered action candidates.
// Returns []*ast_domain.Diagnostic with any warnings or errors encountered.
func (ad *ActionDiscoverer) Discover(
	ctx context.Context,
) (*annotator_dto.ActionManifest, []*ast_domain.Diagnostic) {
	ctx, l := logger_domain.From(ctx, log)
	manifest := annotator_dto.NewActionManifest()

	if ad.inMemoryMode {
		l.Internal("Skipping action discovery in in-memory mode")
		return manifest, nil
	}

	baseDir := ad.resolver.GetBaseDir()
	actionsDir := filepath.Join(baseDir, "actions")

	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		l.Internal("No actions directory found, skipping action discovery",
			logger_domain.String("path", actionsDir),
		)
		return manifest, nil
	}

	goFiles, diagnostic := ad.scanActionsDirectory(actionsDir)
	if diagnostic != nil {
		return manifest, []*ast_domain.Diagnostic{diagnostic}
	}

	l.Internal("Scanning actions directory",
		logger_domain.String("path", actionsDir),
		logger_domain.Int("fileCount", len(goFiles)),
	)

	diagnostics := ad.processActionFiles(ctx, goFiles, baseDir, actionsDir, manifest)

	l.Internal("Action discovery complete",
		logger_domain.Int("actionCount", len(manifest.Actions)),
	)

	return manifest, diagnostics
}

// scanActionsDirectory finds all Go files in the actions directory.
//
// Takes actionsDir (string) which is the path to the actions directory
// to scan.
//
// Returns []string which contains the paths to all found Go files.
// Returns *ast_domain.Diagnostic which reports an error when the
// directory cannot be scanned.
func (ad *ActionDiscoverer) scanActionsDirectory(actionsDir string) ([]string, *ast_domain.Diagnostic) {
	goFiles, err := ad.findGoFiles(actionsDir)
	if err != nil {
		return nil, ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			"Failed to scan actions directory: "+err.Error(),
			actionsDir, annotator_dto.CodeActionError, ast_domain.Location{}, "",
		)
	}
	return goFiles, nil
}

// processActionFiles parses Go files and adds discovered actions to the
// manifest.
//
// Takes goFiles ([]string) which lists the Go file paths to parse.
// Takes baseDir (string) which specifies the base directory for resolution.
// Takes actionsDir (string) which specifies the actions directory path.
// Takes manifest (*annotator_dto.ActionManifest) which receives discovered
// actions.
//
// Returns []*ast_domain.Diagnostic which contains any issues found during
// parsing.
func (ad *ActionDiscoverer) processActionFiles(
	ctx context.Context,
	goFiles []string,
	baseDir, actionsDir string,
	manifest *annotator_dto.ActionManifest,
) []*ast_domain.Diagnostic {
	var diagnostics []*ast_domain.Diagnostic
	moduleName := ad.resolver.GetModuleName()

	ctx, l := logger_domain.From(ctx, log)
	for _, filePath := range goFiles {
		candidates, fileDiags := ad.parseFile(ctx, filePath, baseDir, actionsDir, moduleName)
		diagnostics = append(diagnostics, fileDiags...)

		for _, candidate := range candidates {
			action := candidateToDefinition(candidate)
			manifest.AddAction(action)

			l.Internal("Discovered action candidate",
				logger_domain.String("name", action.Name),
				logger_domain.String("struct", action.StructName),
				logger_domain.String("file", action.FilePath),
			)
		}
	}

	return diagnostics
}

// findGoFiles recursively finds all .go files in the given directory.
//
// Takes directory (string) which specifies the root directory to search.
//
// Returns []string which contains the paths to all found .go files.
// Returns error when the directory cannot be walked.
func (*ActionDiscoverer) findGoFiles(directory string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(directory, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walking directory %q: %w", path, err)
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// parseFile parses a single Go file and extracts action candidates.
//
// Takes filePath (string) which is the path to the Go file to parse.
// Takes baseDir (string) which is the base directory for computing
// relative paths.
// Takes moduleName (string) which is the Go module name for building
// package paths.
//
// Returns []*annotator_dto.ActionCandidate which contains action
// candidates found in the file.
// Returns []*ast_domain.Diagnostic which contains any parse errors.
func (*ActionDiscoverer) parseFile(
	_ context.Context,
	filePath string,
	baseDir string,
	_ string,
	moduleName string,
) ([]*annotator_dto.ActionCandidate, []*ast_domain.Diagnostic) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		diagnostic := ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			"Failed to parse Go file: "+err.Error(),
			filePath, annotator_dto.CodeActionError, ast_domain.Location{}, "",
		)
		return nil, []*ast_domain.Diagnostic{diagnostic}
	}

	info := buildFilePathInfo(filePath, baseDir, moduleName)
	candidates := extractActionCandidates(fset, file, info)

	return candidates, nil
}

// filePathInfo holds path details for a Go file being processed.
type filePathInfo struct {
	// filePath is the absolute path to the source file.
	filePath string

	// relPath is the file path relative to the module root.
	relPath string

	// packagePath is the import path of the package containing this file.
	packagePath string
}

// WithActionDiscovererInMemoryMode sets the ActionDiscoverer to skip filesystem
// operations. Use this for WASM or testing contexts.
//
// Returns ActionDiscovererOption which configures in-memory mode when applied.
func WithActionDiscovererInMemoryMode() ActionDiscovererOption {
	return func(ad *ActionDiscoverer) {
		ad.inMemoryMode = true
	}
}

// candidateToDefinition converts an ActionCandidate to an ActionDefinition.
//
// Takes candidate (*annotator_dto.ActionCandidate) which provides the source
// data for the conversion.
//
// Returns annotator_dto.ActionDefinition which contains the converted action
// with HTTPMethod set to POST.
func candidateToDefinition(candidate *annotator_dto.ActionCandidate) annotator_dto.ActionDefinition {
	return annotator_dto.ActionDefinition{
		Name:           candidate.ActionName,
		TSFunctionName: candidate.TSFunctionName,
		FilePath:       candidate.RelativePath,
		PackagePath:    candidate.PackagePath,
		StructName:     candidate.StructName,
		PackageName:    candidate.PackageName,
		Description:    candidate.DocComment,
		HTTPMethod:     "POST",
		StructLine:     candidate.StructLine,
	}
}

// buildFilePathInfo computes path information for a Go file.
//
// Takes filePath (string) which is the absolute path to the Go file.
// Takes baseDir (string) which is the base directory for relative paths.
// Takes moduleName (string) which is the Go module name for package paths.
//
// Returns filePathInfo which contains the file path, relative path, and
// package path.
func buildFilePathInfo(filePath, baseDir, moduleName string) filePathInfo {
	relPath, _ := filepath.Rel(baseDir, filePath)
	relPath = filepath.ToSlash(relPath)

	pkgDir := filepath.Dir(filePath)
	pkgRelPath, _ := filepath.Rel(baseDir, pkgDir)
	pkgRelPath = filepath.ToSlash(pkgRelPath)
	packagePath := moduleName + "/" + pkgRelPath

	return filePathInfo{
		filePath:    filePath,
		relPath:     relPath,
		packagePath: packagePath,
	}
}

// extractActionCandidates finds action structs in a parsed Go file.
//
// Takes fset (*token.FileSet) which maps AST positions to source locations.
// Takes file (*ast.File) which is the parsed Go source file to search.
// Takes info (filePathInfo) which provides file path details for candidates.
//
// Returns []*annotator_dto.ActionCandidate which contains the found action
// struct candidates, or nil if none are found.
func extractActionCandidates(fset *token.FileSet, file *ast.File, info filePathInfo) []*annotator_dto.ActionCandidate {
	var candidates []*annotator_dto.ActionCandidate

	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			candidate := tryExtractActionCandidate(fset, file, genDecl, spec, info)
			if candidate != nil {
				candidates = append(candidates, candidate)
			}
		}
	}

	return candidates
}

// tryExtractActionCandidate attempts to extract an action candidate from a
// type spec.
//
// Takes fset (*token.FileSet) which maps AST positions to source locations.
// Takes file (*ast.File) which provides the parsed Go source file.
// Takes genDecl (*ast.GenDecl) which contains the generic declaration.
// Takes spec (ast.Spec) which is the specification to examine.
// Takes info (filePathInfo) which provides file path details.
//
// Returns *annotator_dto.ActionCandidate which is the extracted candidate, or
// nil if the spec is not a struct type embedding ActionMetadata.
func tryExtractActionCandidate(
	fset *token.FileSet,
	file *ast.File,
	genDecl *ast.GenDecl,
	spec ast.Spec,
	info filePathInfo,
) *annotator_dto.ActionCandidate {
	typeSpec, ok := spec.(*ast.TypeSpec)
	if !ok {
		return nil
	}

	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil
	}

	if !embedsActionMetadata(structType) {
		return nil
	}

	actionName := structNameToActionName(typeSpec.Name.Name, file.Name.Name)
	tsFunctionName := actionNameToTSFunction(actionName)
	docComment := extractStructDocComment(file, genDecl, typeSpec)
	structLine := fset.Position(typeSpec.Pos()).Line

	return &annotator_dto.ActionCandidate{
		FilePath:       info.filePath,
		RelativePath:   info.relPath,
		PackagePath:    info.packagePath,
		PackageName:    file.Name.Name,
		StructName:     typeSpec.Name.Name,
		ActionName:     actionName,
		TSFunctionName: tsFunctionName,
		DocComment:     docComment,
		StructLine:     structLine,
	}
}

// embedsActionMetadata checks if a struct embeds piko.ActionMetadata.
//
// Takes structType (*ast.StructType) which is the struct type to inspect.
//
// Returns bool which is true if the struct embeds piko.ActionMetadata.
func embedsActionMetadata(structType *ast.StructType) bool {
	for _, field := range structType.Fields.List {
		if len(field.Names) > 0 {
			continue
		}

		if selectorExpression, ok := field.Type.(*ast.SelectorExpr); ok {
			if identifier, ok := selectorExpression.X.(*ast.Ident); ok {
				if identifier.Name == "piko" && selectorExpression.Sel.Name == "ActionMetadata" {
					return true
				}
			}
		}
	}
	return false
}

// structNameToActionName converts a struct name to an action name.
//
// The action name is derived from the struct type, not the file path, giving
// developers explicit control over naming. Multiple actions per file are
// supported.
//
// Takes structName (string) which is the struct type name to convert.
// Takes packageName (string) which is the package to prefix the action with.
//
// Returns string which is the qualified action name in "package.Name" format.
func structNameToActionName(structName string, packageName string) string {
	name := strings.TrimSuffix(structName, "Action")
	return packageName + dotSeparator + name
}

// actionNameToTSFunction converts a dot-separated action name to a camelCase
// TypeScript function name.
//
// Takes actionName (string) which is the dot-separated action name to convert.
//
// Returns string which is the camelCase function name.
func actionNameToTSFunction(actionName string) string {
	parts := strings.Split(actionName, dotSeparator)
	if len(parts) == 0 {
		return actionName
	}

	var result strings.Builder
	result.WriteString(parts[0])

	for _, part := range parts[1:] {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			result.WriteString(part[1:])
		}
	}

	return result.String()
}

// kebabToCamelInSegments converts kebab-case to camelCase within
// dot-separated segments.
//
// Takes name (string) which is the dot-separated string to convert.
//
// Returns string which is the converted string with each segment in camelCase.
func kebabToCamelInSegments(name string) string {
	segments := strings.Split(name, dotSeparator)
	for i, segment := range segments {
		segments[i] = kebabToCamel(segment)
	}
	return strings.Join(segments, dotSeparator)
}

// kebabToCamel converts a kebab-case string to camelCase.
//
// Takes s (string) which is the kebab-case input to convert.
//
// Returns string which is the camelCase version of the input.
func kebabToCamel(s string) string {
	parts := strings.Split(s, "-")
	if len(parts) <= 1 {
		return s
	}

	var result strings.Builder
	result.WriteString(parts[0])
	for _, part := range parts[1:] {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]))
			result.WriteString(part[1:])
		}
	}
	return result.String()
}

// extractStructDocComment extracts the doc comment for a type declaration.
//
// Takes genDecl (*ast.GenDecl) which is the generic declaration that
// may hold a doc comment when it contains a single spec.
// Takes typeSpec (*ast.TypeSpec) which is the type specification whose
// own doc comment takes priority.
//
// Returns string which is the trimmed doc comment text, or empty if no
// comment is found.
func extractStructDocComment(_ *ast.File, genDecl *ast.GenDecl, typeSpec *ast.TypeSpec) string {
	if typeSpec.Doc != nil && typeSpec.Doc.Text() != "" {
		return strings.TrimSpace(typeSpec.Doc.Text())
	}

	if len(genDecl.Specs) == 1 && genDecl.Doc != nil {
		return strings.TrimSpace(genDecl.Doc.Text())
	}

	return ""
}
