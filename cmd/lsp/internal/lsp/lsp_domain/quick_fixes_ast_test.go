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

package lsp_domain

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestFindPropsStruct(t *testing.T) {
	testCases := []struct {
		name         string
		source       string
		expectFound  bool
		expectFields int
	}{
		{
			name: "props struct found",
			source: `package main

type Props struct {
	Title string
	Count int
}`,
			expectFound:  true,
			expectFields: 2,
		},
		{
			name: "no props struct",
			source: `package main

type State struct {
	Value string
}`,
			expectFound: false,
		},
		{
			name: "props not a struct",
			source: `package main

type Props = string`,
			expectFound: false,
		},
		{
			name: "empty props struct",
			source: `package main

type Props struct {}`,
			expectFound:  true,
			expectFields: 0,
		},
		{
			name: "multiple types with props",
			source: `package main

type State struct {
	X int
}

type Props struct {
	Name string
}

type Other struct {
	Y int
}`,
			expectFound:  true,
			expectFields: 1,
		},
		{
			name: "props in var block",
			source: `package main

var (
	x = 1
)

type Props struct {
	Value int
}`,
			expectFound:  true,
			expectFields: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tc.source, 0)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			typeSpec, structType, found := findPropsStruct(file)

			if found != tc.expectFound {
				t.Errorf("findPropsStruct() found = %v, want %v", found, tc.expectFound)
			}

			if tc.expectFound {
				if typeSpec == nil {
					t.Error("expected typeSpec to be non-nil")
				}
				if structType == nil {
					t.Error("expected structType to be non-nil")
				}
				if structType != nil && len(structType.Fields.List) != tc.expectFields {
					t.Errorf("expected %d fields, got %d", tc.expectFields, len(structType.Fields.List))
				}
			}
		})
	}
}

func TestFindFieldInStruct(t *testing.T) {
	source := `package main

type Props struct {
	Title string
	Count int
	Description string
}`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	_, structType, found := findPropsStruct(file)
	if !found {
		t.Fatal("expected to find Props struct")
	}

	testCases := []struct {
		name        string
		fieldName   string
		expectFound bool
		expectIndex int
	}{
		{
			name:        "find Title field",
			fieldName:   "Title",
			expectFound: true,
			expectIndex: 0,
		},
		{
			name:        "find Count field",
			fieldName:   "Count",
			expectFound: true,
			expectIndex: 1,
		},
		{
			name:        "find Description field",
			fieldName:   "Description",
			expectFound: true,
			expectIndex: 2,
		},
		{
			name:        "field not found",
			fieldName:   "NonExistent",
			expectFound: false,
			expectIndex: -1,
		},
		{
			name:        "case sensitive",
			fieldName:   "title",
			expectFound: false,
			expectIndex: -1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			field, index, found := findFieldInStruct(structType, tc.fieldName)

			if found != tc.expectFound {
				t.Errorf("findFieldInStruct() found = %v, want %v", found, tc.expectFound)
			}

			if index != tc.expectIndex {
				t.Errorf("findFieldInStruct() index = %d, want %d", index, tc.expectIndex)
			}

			if tc.expectFound && field == nil {
				t.Error("expected field to be non-nil when found")
			}
		})
	}
}

func TestFindFieldInStruct_MultipleNamesPerField(t *testing.T) {
	source := `package main

type Props struct {
	X, Y, Z int
	Name string
}`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	_, structType, found := findPropsStruct(file)
	if !found {
		t.Fatal("expected to find Props struct")
	}

	testCases := []struct {
		name        string
		fieldName   string
		expectFound bool
	}{
		{name: "find X", fieldName: "X", expectFound: true},
		{name: "find Y", fieldName: "Y", expectFound: true},
		{name: "find Z", fieldName: "Z", expectFound: true},
		{name: "find Name", fieldName: "Name", expectFound: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, found := findFieldInStruct(structType, tc.fieldName)
			if found != tc.expectFound {
				t.Errorf("findFieldInStruct(%q) found = %v, want %v", tc.fieldName, found, tc.expectFound)
			}
		})
	}
}

func TestAddCoerceTagToField(t *testing.T) {
	testCases := []struct {
		name        string
		source      string
		fieldName   string
		expectedTag string
	}{
		{
			name: "field without tag",
			source: `package main

type Props struct {
	Count int
}`,
			fieldName:   "Count",
			expectedTag: "`coerce:\"true\"`",
		},
		{
			name: "field with existing tag",
			source: `package main

type Props struct {
	Name string ` + "`json:\"name\"`" + `
}`,
			fieldName:   "Name",
			expectedTag: "`json:\"name\" coerce:\"true\"`",
		},
		{
			name: "field with empty tag",
			source: `package main

type Props struct {
	Value int ` + "``" + `
}`,
			fieldName:   "Value",
			expectedTag: "`coerce:\"true\"`",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tc.source, 0)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			_, structType, found := findPropsStruct(file)
			if !found {
				t.Fatal("expected to find Props struct")
			}

			field, _, found := findFieldInStruct(structType, tc.fieldName)
			if !found {
				t.Fatalf("expected to find field %s", tc.fieldName)
			}

			addCoerceTagToField(field)

			if field.Tag == nil {
				t.Fatal("expected tag to be non-nil after addCoerceTagToField")
			}

			if field.Tag.Value != tc.expectedTag {
				t.Errorf("tag = %q, want %q", field.Tag.Value, tc.expectedTag)
			}
		})
	}
}

func TestFindImportDecl(t *testing.T) {
	testCases := []struct {
		name        string
		source      string
		expectFound bool
	}{
		{
			name: "has import declaration",
			source: `package main

import "fmt"

func main() {}`,
			expectFound: true,
		},
		{
			name: "has grouped import declaration",
			source: `package main

import (
	"fmt"
	"strings"
)

func main() {}`,
			expectFound: true,
		},
		{
			name: "no import declaration",
			source: `package main

func main() {}`,
			expectFound: false,
		},
		{
			name:        "empty file",
			source:      `package main`,
			expectFound: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tc.source, 0)
			if err != nil {
				t.Fatalf("failed to parse source: %v", err)
			}

			genDecl, found := findImportDecl(file)

			if found != tc.expectFound {
				t.Errorf("findImportDecl() found = %v, want %v", found, tc.expectFound)
			}

			if tc.expectFound && genDecl == nil {
				t.Error("expected genDecl to be non-nil when found")
			}

			if tc.expectFound && genDecl != nil && genDecl.Tok != token.IMPORT {
				t.Errorf("expected genDecl.Tok = IMPORT, got %v", genDecl.Tok)
			}
		})
	}
}

func TestAddImportToAST_ExistingImportDecl(t *testing.T) {
	source := `package main

import "fmt"

func main() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	originalImportCount := countImportSpecs(file)

	addImportToAST(file, "", "strings")

	newImportCount := countImportSpecs(file)
	if newImportCount != originalImportCount+1 {
		t.Errorf("expected %d imports, got %d", originalImportCount+1, newImportCount)
	}

	if !hasImportPath(file, "strings") {
		t.Error("expected to find 'strings' import")
	}
}

func TestAddImportToAST_WithAlias(t *testing.T) {
	source := `package main

import "fmt"
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	addImportToAST(file, "str", "strings")

	if !hasImportWithAlias(file, "str", "strings") {
		t.Error("expected to find 'strings' import with alias 'str'")
	}
}

func TestAddImportToAST_NoExistingImportDecl(t *testing.T) {
	source := `package main

func main() {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, 0)
	if err != nil {
		t.Fatalf("failed to parse source: %v", err)
	}

	addImportToAST(file, "", "fmt")

	importDecl, found := findImportDecl(file)
	if !found {
		t.Fatal("expected import declaration to be created")
	}

	if len(importDecl.Specs) != 1 {
		t.Errorf("expected 1 import spec, got %d", len(importDecl.Specs))
	}

	if !hasImportPath(file, "fmt") {
		t.Error("expected to find 'fmt' import")
	}
}

func countImportSpecs(file *ast.File) int {
	count := 0
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		count += len(genDecl.Specs)
	}
	return count
}

func hasImportPath(file *ast.File, path string) bool {
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		for _, spec := range genDecl.Specs {
			importSpec, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}

			if importSpec.Path.Value == `"`+path+`"` {
				return true
			}
		}
	}
	return false
}

func hasImportWithAlias(file *ast.File, alias, path string) bool {
	for _, declaration := range file.Decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		for _, spec := range genDecl.Specs {
			importSpec, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			pathMatch := importSpec.Path.Value == `"`+path+`"`
			aliasMatch := importSpec.Name != nil && importSpec.Name.Name == alias
			if pathMatch && aliasMatch {
				return true
			}
		}
	}
	return false
}
