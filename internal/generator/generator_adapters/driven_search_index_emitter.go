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

package generator_adapters

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	goast "go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"io/fs"
	"path/filepath"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/search/search_domain"
	"piko.sh/piko/internal/search/search_schema/search_schema_gen"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// printerTabwidth is the tab width used by the Go printer.
	printerTabwidth = 4

	// goStringQuote wraps Go string literal values in generated AST nodes.
	goStringQuote = `"`
)

// DrivenSearchIndexEmitter implements SearchIndexEmitterPort.
//
// This adapter orchestrates the generation of search index binary artefacts:
//  1. Builds inverted indexes (Fast and/or Smart modes)
//  2. Writes binaries to dist/collections/{name}/search_{mode}.bin
//  3. Updates Go wrapper to embed and register search indexes
//
// Architecture:
//   - Uses IndexBuilderPort from search hexagon for index construction
//   - Uses FSWriterPort for file I/O
//   - Supports parallel generation of both Fast and Smart indexes
//   - All file operations are sandboxed to prevent path traversal attacks
type DrivenSearchIndexEmitter struct {
	// indexBuilder builds search index data for a collection.
	indexBuilder search_domain.IndexBuilderPort

	// fsWriter writes generated files to the filesystem.
	fsWriter generator_domain.FSWriterPort

	// sandbox provides safe file system access with path checking.
	sandbox safedisk.Sandbox

	// moduleName is the Go module name from go.mod.
	moduleName string

	// manifestFormat specifies the output format: "flatbuffers" or "json".
	manifestFormat string
}

// NewDrivenSearchIndexEmitter creates a new search index emitter instance.
//
// Takes indexBuilder (search_domain.IndexBuilderPort) which builds the search
// index entries.
// Takes fsWriter (generator_domain.FSWriterPort) which writes files to the
// output directory.
// Takes sandbox (safedisk.Sandbox) which provides safe file system access.
// Takes moduleName (string) which specifies the Go module being documented.
// Takes manifestFormat (string) which defines the output format for the index.
//
// Returns *DrivenSearchIndexEmitter which is configured and ready to emit
// search index files.
func NewDrivenSearchIndexEmitter(
	indexBuilder search_domain.IndexBuilderPort,
	fsWriter generator_domain.FSWriterPort,
	sandbox safedisk.Sandbox,
	moduleName string,
	manifestFormat string,
) *DrivenSearchIndexEmitter {
	return &DrivenSearchIndexEmitter{
		indexBuilder:   indexBuilder,
		fsWriter:       fsWriter,
		sandbox:        sandbox,
		moduleName:     moduleName,
		manifestFormat: manifestFormat,
	}
}

// EmitSearchIndex generates search index binaries and updates the collection
// wrapper.
//
// Takes collectionName (string) which identifies the target collection.
// Takes items ([]collection_dto.ContentItem) which provides the content to
// index.
// Takes outputDir (string) which specifies where to write generated files.
// Takes modes ([]string) which lists the search modes to generate (fast,
// smart).
//
// Returns error when the collection directory does not exist, an unknown mode
// is specified, or writing fails.
//
// Workflow:
//  1. Validate that collection directory exists
//  2. For each requested mode (Fast/Smart):
//     a. Build inverted index using IndexBuilder
//     b. Write binary to search_{mode}.bin
//  3. Update generated.go to embed and register search indexes
//
// May be called multiple times for different modes, or once with multiple modes
// to generate all indexes in a single call.
func (e *DrivenSearchIndexEmitter) EmitSearchIndex(
	ctx context.Context,
	collectionName string,
	items []collection_dto.ContentItem,
	outputDir string,
	modes []string,
) error {
	relOutputDir := e.sandbox.RelPath(outputDir)

	collectionDir := filepath.Join(relOutputDir, "collections", collectionName)

	if _, err := e.sandbox.Stat(collectionDir); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("collection directory does not exist: %s (run CollectionEmitter first)", collectionDir)
	}

	var fastIndexGenerated, smartIndexGenerated bool

	for _, mode := range modes {
		switch mode {
		case "fast":
			if err := e.emitModeIndex(ctx, collectionName, items, collectionDir, search_schema_gen.SearchModeFast); err != nil {
				return fmt.Errorf("failed to emit Fast mode index: %w", err)
			}
			fastIndexGenerated = true

		case "smart":
			if err := e.emitModeIndex(ctx, collectionName, items, collectionDir, search_schema_gen.SearchModeSmart); err != nil {
				return fmt.Errorf("failed to emit Smart mode index: %w", err)
			}
			smartIndexGenerated = true

		default:
			return fmt.Errorf("unknown search mode: %s (valid: fast, smart)", mode)
		}
	}

	goFilePath := filepath.Join(collectionDir, "generated.go")
	goCode, err := e.generateGoWrapper(collectionName, fastIndexGenerated, smartIndexGenerated)
	if err != nil {
		return fmt.Errorf("failed to generate Go wrapper for collection %q: %w", collectionName, err)
	}

	if err := e.fsWriter.WriteFile(ctx, goFilePath, goCode); err != nil {
		return fmt.Errorf("failed to write Go wrapper for collection %q: %w", collectionName, err)
	}

	return nil
}

// emitModeIndex builds and writes a single search index for a given mode.
//
// Takes collectionName (string) which names the collection to index.
// Takes items ([]collection_dto.ContentItem) which holds the content to index.
// Takes collectionDir (string) which gives the output folder path.
// Takes mode (search_schema_gen.SearchMode) which sets the indexing method.
//
// Returns error when the mode is not supported, index building fails, or the
// file cannot be written.
func (e *DrivenSearchIndexEmitter) emitModeIndex(
	ctx context.Context,
	collectionName string,
	items []collection_dto.ContentItem,
	collectionDir string,
	mode search_schema_gen.SearchMode,
) error {
	fileExt := ".bin"
	if e.manifestFormat == "json" {
		fileExt = ".json"
	}

	var fileName string
	switch mode {
	case search_schema_gen.SearchModeFast:
		fileName = "search_fast" + fileExt
	case search_schema_gen.SearchModeSmart:
		fileName = "search_smart" + fileExt
	default:
		return fmt.Errorf("unsupported search mode: %d", mode)
	}

	filePath := filepath.Join(collectionDir, fileName)

	config := search_domain.DefaultIndexBuildConfig()
	config.AnalysisMode = mode
	config.Format = e.manifestFormat

	indexData, err := e.indexBuilder.BuildIndex(ctx, collectionName, items, mode, config)
	if err != nil {
		return fmt.Errorf("failed to build %s index: %w", fileName, err)
	}

	if err := e.fsWriter.WriteFile(ctx, filePath, indexData); err != nil {
		return fmt.Errorf("failed to write %s: %w", fileName, err)
	}

	return nil
}

// generateGoWrapper creates Go source code for a collection and search index
// wrapper using AST-based code generation for deterministic output.
//
// The generated file:
//   - Uses //go:embed to embed data.bin and search_{mode}.bin files
//   - Registers all blobs with the runtime in init()
//   - Conditionally includes search indexes based on what was generated
//
// Takes collectionName (string) which specifies the package name for the
// generated code.
// Takes hasFast (bool) which indicates whether to include fast search index.
// Takes hasSmart (bool) which indicates whether to include smart search index.
//
// Returns ([]byte, error) containing the formatted Go source code or an error.
func (e *DrivenSearchIndexEmitter) generateGoWrapper(collectionName string, hasFast, hasSmart bool) ([]byte, error) {
	fileExt := ".bin"
	if e.manifestFormat == "json" {
		fileExt = ".json"
	}

	config := goWrapperConfig{
		packageName:    collectionName,
		collectionName: collectionName,
		fileExtension:  fileExt,
		hasFast:        hasFast,
		hasSmart:       hasSmart,
	}

	return buildGoWrapper(config)
}

// goWrapperConfig holds the settings for generating a Go wrapper file.
type goWrapperConfig struct {
	// packageName is the Go package name for the generated wrapper file.
	packageName string

	// collectionName is the name used to register the collection blob.
	collectionName string

	// fileExtension is the file suffix for generated embed files (e.g. ".gz").
	fileExtension string

	// hasFast indicates whether to generate a fast search index.
	hasFast bool

	// hasSmart indicates whether to generate a smart search index.
	hasSmart bool
}

// buildGoWrapper builds a complete Go AST file and formats it as bytes.
//
// Takes config (goWrapperConfig) which specifies the wrapper generation
// settings.
//
// Returns []byte which contains the formatted Go source code.
// Returns error when printing the AST to the buffer fails.
func buildGoWrapper(config goWrapperConfig) ([]byte, error) {
	fset := token.NewFileSet()
	file := buildGoWrapperFile(config)

	var buffer bytes.Buffer
	_, _ = buffer.WriteString(generator_dto.AnalysisBuildConstraint)
	_, _ = buffer.WriteString("// Code generated by Piko. DO NOT EDIT.\n")
	_, _ = buffer.WriteString("// This file embeds the collection data and search indexes, registering them with the runtime.\n\n")

	printerConfig := printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: printerTabwidth}
	if err := printerConfig.Fprint(&buffer, fset, file); err != nil {
		return nil, fmt.Errorf("printing Go AST for search index wrapper: %w", err)
	}

	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return buffer.Bytes(), nil
	}

	return formatted, nil
}

// buildGoWrapperFile builds a complete AST file for a Go wrapper.
//
// Takes config (goWrapperConfig) which specifies the package name and settings
// for imports, variables, and init function.
//
// Returns *goast.File which contains the full AST with imports, embed variable
// declarations, and an init function.
func buildGoWrapperFile(config goWrapperConfig) *goast.File {
	file := &goast.File{
		Name:  goast.NewIdent(config.packageName),
		Decls: []goast.Decl{},
	}

	file.Decls = append(file.Decls, buildGoWrapperImports())

	file.Decls = append(file.Decls, buildGoWrapperVarDecls(config)...)

	file.Decls = append(file.Decls, buildGoWrapperInitFunc(config))

	return file
}

// buildGoWrapperImports creates the import declaration block.
//
// Returns *goast.GenDecl which contains the embed and piko runtime imports.
func buildGoWrapperImports() *goast.GenDecl {
	specs := []goast.Spec{
		&goast.ImportSpec{
			Path: &goast.BasicLit{Kind: token.STRING, Value: `"context"`},
		},
		&goast.ImportSpec{
			Name: goast.NewIdent("_"),
			Path: &goast.BasicLit{Kind: token.STRING, Value: `"embed"`},
		},
		&goast.ImportSpec{
			Name: goast.NewIdent("pikoruntime"),
			Path: &goast.BasicLit{Kind: token.STRING, Value: `"piko.sh/piko/wdk/runtime"`},
		},
	}

	return &goast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: token.Pos(1),
		Specs:  specs,
	}
}

// buildGoWrapperVarDecls creates variable declarations with //go:embed comments.
//
// Takes config (goWrapperConfig) which specifies which search indices to
// include.
//
// Returns []goast.Decl which contains the embed variable declarations.
func buildGoWrapperVarDecls(config goWrapperConfig) []goast.Decl {
	var decls []goast.Decl

	decls = append(decls, buildEmbedVarDecl("data.bin", "collectionBlob"))

	if config.hasFast {
		decls = append(decls, buildEmbedVarDecl("search_fast"+config.fileExtension, "searchFastBlob"))
	}

	if config.hasSmart {
		decls = append(decls, buildEmbedVarDecl("search_smart"+config.fileExtension, "searchSmartBlob"))
	}

	return decls
}

// buildEmbedVarDecl creates a variable declaration with a //go:embed comment.
//
// Takes embedPath (string) which is the file path for the embed directive.
// Takes varName (string) which is the name of the variable to declare.
//
// Returns *goast.GenDecl which is the AST node for a []byte variable with the
// embed directive attached.
func buildEmbedVarDecl(embedPath, varName string) *goast.GenDecl {
	return &goast.GenDecl{
		Tok: token.VAR,
		Doc: &goast.CommentGroup{
			List: []*goast.Comment{
				{Text: "//go:embed " + embedPath},
			},
		},
		Specs: []goast.Spec{
			&goast.ValueSpec{
				Names: []*goast.Ident{goast.NewIdent(varName)},
				Type: &goast.ArrayType{
					Elt: goast.NewIdent("byte"),
				},
			},
		},
	}
}

// buildGoWrapperInitFunc creates the init() function that registers blobs
// with the runtime.
//
// Takes config (goWrapperConfig) which specifies the collection name and
// which search indices to register.
//
// Returns *goast.FuncDecl which is the AST node for the generated init
// function.
func buildGoWrapperInitFunc(config goWrapperConfig) *goast.FuncDecl {
	statements := []goast.Stmt{buildRegisterCollectionBlobStmt(config.collectionName)}

	if config.hasFast {
		statements = append(statements, buildRegisterSearchIndexStmt(config.collectionName, "fast", "searchFastBlob"))
	}
	if config.hasSmart {
		statements = append(statements, buildRegisterSearchIndexStmt(config.collectionName, "smart", "searchSmartBlob"))
	}

	return &goast.FuncDecl{
		Name: goast.NewIdent("init"),
		Type: &goast.FuncType{Params: &goast.FieldList{}},
		Body: &goast.BlockStmt{List: statements},
	}
}

// buildRegisterCollectionBlobStmt creates an AST statement that calls
// pikoruntime.RegisterStaticCollectionBlob with context.Background().
//
// Takes collectionName (string) which identifies the collection to register.
//
// Returns goast.Stmt which is the expression statement for the registration
// call.
func buildRegisterCollectionBlobStmt(collectionName string) goast.Stmt {
	return &goast.ExprStmt{
		X: &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent("pikoruntime"),
				Sel: goast.NewIdent("RegisterStaticCollectionBlob"),
			},
			Args: []goast.Expr{
				&goast.CallExpr{
					Fun: &goast.SelectorExpr{
						X:   goast.NewIdent("context"),
						Sel: goast.NewIdent("Background"),
					},
				},
				&goast.BasicLit{Kind: token.STRING, Value: goStringQuote + collectionName + goStringQuote},
				goast.NewIdent("collectionBlob"),
			},
		},
	}
}

// buildRegisterSearchIndexStmt creates an AST statement that calls
// pikoruntime.RegisterSearchIndex for a specific mode.
//
// Takes collectionName (string) which identifies the collection.
// Takes modeName (string) which is the search mode (e.g. "fast", "smart").
// Takes varName (string) which is the Go variable name holding the embedded
// blob.
//
// Returns goast.Stmt which is the expression statement for the registration
// call.
func buildRegisterSearchIndexStmt(collectionName, modeName, varName string) goast.Stmt {
	return &goast.ExprStmt{
		X: &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   goast.NewIdent("pikoruntime"),
				Sel: goast.NewIdent("RegisterSearchIndex"),
			},
			Args: []goast.Expr{
				&goast.BasicLit{Kind: token.STRING, Value: goStringQuote + collectionName + goStringQuote},
				&goast.BasicLit{Kind: token.STRING, Value: goStringQuote + modeName + goStringQuote},
				goast.NewIdent(varName),
			},
		},
	}
}
