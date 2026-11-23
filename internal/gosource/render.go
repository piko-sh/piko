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

package gosource

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

var (
	// implementation holds this file's source, so the renderer's identity changes whenever
	// its behaviour can.
	//
	//go:embed render.go
	implementation []byte
)

// SourceSHA256 identifies the renderer in the fuzz generator's saved profile.
//
// Returns string which is the hex SHA-256 of this file's source.
func SourceSHA256() string {
	return fmt.Sprintf("%x", sha256.Sum256(implementation))
}

// Render parses a complete Go file, applies an edit and prints it as gofmt does.
//
// Each call owns its syntax tree. An edit can use positions in the file set to identify
// nodes and their comments. The output is parsed again so a partial or invalid edit is
// reported rather than written.
//
// Takes name (string) which names the file in errors.
// Takes src ([]byte) which is the complete Go source.
// Takes edit (func(fset *token.FileSet, file *ast.File) error) which changes the tree, or
// nil to reformat only.
//
// Returns []byte which is the formatted source.
// Returns error when parsing, the edit, printing or the final parse fails.
func Render(name string, src []byte, edit func(fset *token.FileSet, file *ast.File) error) ([]byte, error) {
	const flags = parser.ParseComments | parser.SkipObjectResolution
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, flags)
	if err != nil {
		return nil, fmt.Errorf("parse Go fixture %s: %w", name, err)
	}
	if edit != nil {
		if err := edit(fset, file); err != nil {
			return nil, fmt.Errorf("edit Go fixture %s: %w", name, err)
		}
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, fmt.Errorf("print Go fixture %s: %w", name, err)
	}

	finalSet := token.NewFileSet()
	result, err := parser.ParseFile(finalSet, name, buf.Bytes(), flags)
	if err != nil {
		return nil, fmt.Errorf("verify rendered Go fixture %s: %w", name, err)
	}
	buf.Reset()
	if err := format.Node(&buf, finalSet, result); err != nil {
		return nil, fmt.Errorf("format rendered Go fixture %s: %w", name, err)
	}
	return buf.Bytes(), nil
}

// RenderFixture renders a Go fixture whose imports of one module move to another.
//
// The fixture must import the module exactly once, so a harness cannot silently run
// against the wrong package.
//
// Takes name (string) which names the file in errors.
// Takes src ([]byte) which is the complete Go source.
// Takes fromModule (string) which is the module path the fixture imports.
// Takes toModule (string) which is the module path to import instead.
//
// Returns []byte which is the formatted source.
// Returns error when rendering fails or the fixture does not import fromModule exactly
// once.
func RenderFixture(name string, src []byte, fromModule, toModule string) ([]byte, error) {
	return Render(name, src, func(_ *token.FileSet, file *ast.File) error {
		count, err := ReplaceModuleImports(file, fromModule, toModule)
		if err != nil {
			return err
		}
		if count != 1 {
			return fmt.Errorf("expected one import of %s, found %d", fromModule, count)
		}
		return nil
	})
}

// ReplaceModuleImports relocates imports of a module and its subpackages to another
// module path. Aliases and unrelated modules are preserved.
//
// Takes file (*ast.File) which is the file to edit.
// Takes oldModule (string) which is the module path to replace.
// Takes newModule (string) which is the replacement module path.
//
// Returns int which is the number of imports changed, so callers can require the imports
// their fixture promises.
// Returns error when an import path is not a valid string literal.
func ReplaceModuleImports(file *ast.File, oldModule, newModule string) (int, error) {
	count := 0
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return count, fmt.Errorf("reading import path %s: %w", spec.Path.Value, err)
		}
		if path == oldModule {
			spec.Path.Value = strconv.Quote(newModule)
			count++
			continue
		}
		if subpackage, found := strings.CutPrefix(path, oldModule+"/"); found {
			spec.Path.Value = strconv.Quote(newModule + "/" + subpackage)
			count++
		}
	}
	return count, nil
}
