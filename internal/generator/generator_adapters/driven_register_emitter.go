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
	"fmt"
	goast "go/ast"
	"go/printer"
	"go/token"
	"slices"
	"strconv"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

const (
	// tabWidth is the number of spaces for each indent level in formatted Go code.
	tabWidth = 4
)

// RegisterEmitter is a driven adapter responsible for generating the single
// Go file that imports all compiled component packages for their side effects,
// ensuring they are linked into the final server binary.
type RegisterEmitter struct {
	// fsWriter writes the generated output to the file system.
	fsWriter generator_domain.FSWriterPort
}

var _ generator_domain.RegisterEmitterPort = (*RegisterEmitter)(nil)

// NewRegisterEmitter creates a new registry emitter.
//
// Takes fsWriter (generator_domain.FSWriterPort) which handles file system
// write operations.
//
// Returns *RegisterEmitter which is ready to emit registry entries.
func NewRegisterEmitter(fsWriter generator_domain.FSWriterPort) *RegisterEmitter {
	return &RegisterEmitter{fsWriter: fsWriter}
}

// Generate creates the register file content for the given package paths.
//
// Takes allPackagePaths ([]string) which lists the canonical Go package paths
// for the components that need to be included in the build.
//
// Returns []byte which contains the formatted register file content.
// Returns error when the file content cannot be formatted.
func (*RegisterEmitter) Generate(_ context.Context, allPackagePaths []string) ([]byte, error) {
	fileAST := createRegisterFileAST(allPackagePaths)

	return formatRegisterFile(fileAST)
}

// Emit creates and writes the register file (e.g., piko_register.go).
//
// Takes outputPath (string) which specifies the destination file path.
// Takes allPackagePaths ([]string) which lists the Go package paths for the
// components to include in the build.
//
// Returns error when generation fails or the file cannot be written.
func (e *RegisterEmitter) Emit(ctx context.Context, outputPath string, allPackagePaths []string) error {
	content, err := e.Generate(ctx, allPackagePaths)
	if err != nil {
		return fmt.Errorf("generating register file content: %w", err)
	}
	if err := e.fsWriter.WriteFile(ctx, outputPath, content); err != nil {
		return fmt.Errorf("writing register file to %q: %w", outputPath, err)
	}
	return nil
}

// createRegisterFileAST builds a Go syntax tree for the register file.
//
// Takes allPackagePaths ([]string) which lists the import paths to include.
//
// Returns *goast.File which contains the syntax tree with a grouped import
// block.
func createRegisterFileAST(allPackagePaths []string) *goast.File {
	importSpecs := createImportSpecs(allPackagePaths)
	importDecl := &goast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: 1,
		Specs:  importSpecs,
		Doc:    nil,
		TokPos: 0,
		Rparen: 0,
	}

	return &goast.File{
		Name:       goast.NewIdent("dist"),
		Decls:      []goast.Decl{importDecl},
		Doc:        nil,
		Package:    0,
		FileStart:  0,
		FileEnd:    0,
		Scope:      nil,
		Imports:    nil,
		Unresolved: nil,
		Comments:   nil,
		GoVersion:  "",
	}
}

// createImportSpecs creates import specifications for the given package paths.
//
// Takes allPackagePaths ([]string) which lists the packages to import.
//
// Returns []goast.Spec which contains side-effect import specs for each path.
func createImportSpecs(allPackagePaths []string) []goast.Spec {
	slices.Sort(allPackagePaths)
	importSpecs := make([]goast.Spec, len(allPackagePaths))
	for i, packagePath := range allPackagePaths {
		importSpecs[i] = &goast.ImportSpec{
			Name: goast.NewIdent("_"),
			Path: &goast.BasicLit{
				Kind:     token.STRING,
				Value:    strconv.Quote(packagePath),
				ValuePos: 0,
			},
			Doc:     nil,
			Comment: nil,
			EndPos:  0,
		}
	}
	return importSpecs
}

// formatRegisterFile formats a Go AST into source code with header comments.
//
// Takes fileAST (*goast.File) which is the parsed Go AST to format.
//
// Returns []byte which contains the formatted Go source code with headers.
// Returns error when the AST cannot be formatted.
func formatRegisterFile(fileAST *goast.File) ([]byte, error) {
	var buffer bytes.Buffer
	_, _ = buffer.WriteString(generator_dto.AnalysisBuildConstraint)
	_, _ = buffer.WriteString("// Code generated by Piko - DO NOT EDIT.\n")
	_, _ = buffer.WriteString("// This file imports all compiled component packages to ensure they are included\n")
	_, _ = buffer.WriteString("// in the final binary and their init() functions are executed.\n\n")

	printerConfig := printer.Config{
		Mode:     printer.TabIndent | printer.UseSpaces,
		Tabwidth: tabWidth,
		Indent:   0,
	}
	fset := token.NewFileSet()
	if err := printerConfig.Fprint(&buffer, fset, fileAST); err != nil {
		return nil, fmt.Errorf("failed to format register aggregator file: %w", err)
	}

	return buffer.Bytes(), nil
}
