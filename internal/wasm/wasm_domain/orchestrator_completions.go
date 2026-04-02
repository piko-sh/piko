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

package wasm_domain

import (
	"go/ast"
	"go/token"
	"regexp"
	"strings"
	"unicode"

	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/wasm/wasm_dto"
)

// completionKind represents the type of completion context.
type completionKind int

const (
	// completionKindScope indicates top-level scope completions, including
	// keywords, builtins, and packages.
	completionKindScope completionKind = iota

	// completionKindPackageMember indicates completions after a package name
	// (fmt.).
	completionKindPackageMember

	// completionKindFieldMethod indicates completions after a variable or
	// expression (e.g. user.).
	completionKindFieldMethod
)

var (
	// importSinglePattern matches single-line Go import statements.
	importSinglePattern = regexp.MustCompile(`import\s+(?:(\w+)\s+)?"([^"]+)"`)
	// importBlockPattern matches grouped Go import blocks.
	importBlockPattern = regexp.MustCompile(`import\s*\(\s*([\s\S]*?)\s*\)`)

	// importLinePattern matches individual import lines within a grouped block.
	importLinePattern = regexp.MustCompile(`(?:(\w+)\s+)?"([^"]+)"`)
	// goKeywords lists Go keywords for completions.
	goKeywords = []string{
		"func", "type", "struct", "interface", "map", "chan",
		"if", "else", "for", "range", "switch", "case", "default",
		"return", "break", "continue", "goto", "fallthrough",
		"var", "const", "package", "import",
		"defer", "go", "select",
	}
	// goBuiltinTypes lists Go builtin types for completions.
	goBuiltinTypes = []string{
		"bool", "byte", "complex64", "complex128",
		"error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64",
		"rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"any", "comparable",
	}
	// goBuiltinFuncs lists Go builtin functions for completions.
	goBuiltinFuncs = []string{
		"append", "cap", "clear", "close", "complex", "copy", "delete",
		"imag", "len", "make", "max", "min", "new", "panic", "print",
		"println", "real", "recover",
	}
)

// completionContext holds details about what type of code completion is needed.
type completionContext struct {
	// pkgAlias is the package alias (e.g. "fmt") used for PackageMember
	// completions.
	pkgAlias string

	// packagePath is the full package path for PackageMember completions.
	packagePath string

	// prefix is the partial text already typed, such as "Pri" in "fmt.Pri".
	prefix string

	// expressionType is the type of the expression for FieldMethod
	// completions (future).
	expressionType string

	// kind indicates what type of completion is needed at this position.
	kind completionKind
}

// getBasicCompletions builds a completion response containing keywords, builtin
// types, builtin functions, and standard library packages.
//
// Takes stdlibData (*inspector_dto.TypeData) which provides standard library
// type information for package completions.
//
// Returns *wasm_dto.CompletionResponse which contains all basic completion
// items.
func getBasicCompletions(stdlibData *inspector_dto.TypeData) *wasm_dto.CompletionResponse {
	items := make([]wasm_dto.CompletionItem, 0, defaultCompletionCapacity)
	items = appendKeywordCompletions(items)
	items = appendBuiltinTypeCompletions(items)
	items = appendBuiltinFuncCompletions(items)
	items = appendStdlibPackageCompletions(items, stdlibData)

	return &wasm_dto.CompletionResponse{
		Success: true,
		Items:   items,
		Error:   "",
	}
}

// appendKeywordCompletions adds Go keyword completions to the given slice.
//
// Takes items ([]wasm_dto.CompletionItem) which is the slice to add to.
//
// Returns []wasm_dto.CompletionItem which has all the original items plus Go
// keywords.
func appendKeywordCompletions(items []wasm_dto.CompletionItem) []wasm_dto.CompletionItem {
	for _, kw := range goKeywords {
		items = append(items, newCompletionItem(kw, "keyword", ""))
	}
	return items
}

// appendBuiltinTypeCompletions adds Go built-in type completions to the given
// slice.
//
// Takes items ([]wasm_dto.CompletionItem) which is the slice to add to.
//
// Returns []wasm_dto.CompletionItem which contains the original items plus
// built-in type completions.
func appendBuiltinTypeCompletions(items []wasm_dto.CompletionItem) []wasm_dto.CompletionItem {
	for _, bt := range goBuiltinTypes {
		items = append(items, newCompletionItem(bt, "type", ""))
	}
	return items
}

// appendBuiltinFuncCompletions adds Go builtin function completions to the
// given slice.
//
// Takes items ([]wasm_dto.CompletionItem) which is the slice to add to.
//
// Returns []wasm_dto.CompletionItem which contains the original items plus
// the builtin function completions.
func appendBuiltinFuncCompletions(items []wasm_dto.CompletionItem) []wasm_dto.CompletionItem {
	for _, builtinFunction := range goBuiltinFuncs {
		items = append(items, newCompletionItem(builtinFunction, "function", ""))
	}
	return items
}

// appendStdlibPackageCompletions adds standard library package completions to
// the given list.
//
// Takes items ([]wasm_dto.CompletionItem) which is the existing completion
// list.
// Takes stdlibData (*inspector_dto.TypeData) which provides standard library
// package data.
//
// Returns []wasm_dto.CompletionItem which contains the original items plus any
// standard library package completions.
func appendStdlibPackageCompletions(items []wasm_dto.CompletionItem, stdlibData *inspector_dto.TypeData) []wasm_dto.CompletionItem {
	if stdlibData == nil {
		return items
	}
	for packagePath := range stdlibData.Packages {
		items = append(items, newCompletionItem(packagePath, "module", "import"))
	}
	return items
}

// newCompletionItem creates a new completion item with the given label, kind,
// and detail.
//
// Takes label (string) which specifies the text shown in the completion list.
// Takes kind (string) which indicates the type of completion (keyword, type,
// function, or package).
// Takes detail (string) which provides additional information about the item.
//
// Returns wasm_dto.CompletionItem which is the configured completion item with
// empty documentation, insert text, and sort text fields.
func newCompletionItem(label, kind, detail string) wasm_dto.CompletionItem {
	return wasm_dto.CompletionItem{
		Label:         label,
		Kind:          kind,
		Detail:        detail,
		Documentation: "",
		InsertText:    "",
		SortText:      "",
	}
}

// analyseCompletionContext determines what kind of completion is needed at the
// cursor position.
//
// Takes source (string) which contains the full source code.
// Takes line (int) which specifies the one-based cursor line.
// Takes column (int) which specifies the one-based cursor column.
// Takes imports (map[string]string) which maps import aliases to package
// paths.
//
// Returns *completionContext which describes the completion kind and
// relevant identifiers.
func analyseCompletionContext(source string, line, column int, _ *ast.File, imports map[string]string) *completionContext {
	textBeforeCursor, ok := getTextBeforeCursor(source, line, column)
	if !ok {
		return newScopeContext()
	}

	dotIndex := strings.LastIndex(textBeforeCursor, ".")
	if dotIndex == -1 {
		return newScopeContext()
	}

	prefix := textBeforeCursor[dotIndex+1:]
	beforeDot := strings.TrimSpace(textBeforeCursor[:dotIndex])
	if beforeDot == "" {
		return newScopeContext()
	}

	identifier := extractLastIdentifier(beforeDot)
	if identifier == "" {
		return newScopeContext()
	}

	return buildCompletionContext(identifier, prefix, imports)
}

// newScopeContext returns a default scope completion context.
//
// Returns *completionContext which has empty fields and completionKindScope
// as the kind.
func newScopeContext() *completionContext {
	return &completionContext{
		pkgAlias:       "",
		packagePath:    "",
		prefix:         "",
		expressionType: "",
		kind:           completionKindScope,
	}
}

// getTextBeforeCursor extracts the text before the cursor position.
//
// Takes source (string) which contains the full source code text.
// Takes line (int) which specifies the one-based line number.
// Takes column (int) which specifies the one-based column position.
//
// Returns string which contains the text from the start of the line up to the
// cursor position.
// Returns bool which indicates whether the extraction was successful.
func getTextBeforeCursor(source string, line, column int) (string, bool) {
	lines := strings.Split(source, "\n")
	if line < 1 || line > len(lines) {
		return "", false
	}

	lineText := lines[line-1]
	column = max(1, min(column, len(lineText)+1))

	return lineText[:column-1], true
}

// extractLastIdentifier finds the last identifier in a string by scanning
// backwards from the end.
//
// Takes text (string) which contains the input to scan.
//
// Returns string which is the trailing identifier, or empty if none found.
func extractLastIdentifier(text string) string {
	identStart := len(text)
	for i := len(text) - 1; i >= 0; i-- {
		c := text[i]
		if !isIdentChar(c) {
			break
		}
		identStart = i
	}
	return text[identStart:]
}

// isIdentChar reports whether c is a valid identifier character.
//
// Takes c (byte) which is the character to check.
//
// Returns bool which is true if c is a letter, digit, or underscore.
func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// buildCompletionContext creates a completion context for the given
// identifier and prefix.
//
// Takes identifier (string) which specifies the type or package name.
// Takes prefix (string) which provides the partial text to complete.
// Takes imports (map[string]string) which maps import aliases to paths.
//
// Returns *completionContext which contains the resolved completion details.
func buildCompletionContext(identifier, prefix string, imports map[string]string) *completionContext {
	if packagePath, ok := imports[identifier]; ok {
		return &completionContext{
			pkgAlias:       identifier,
			packagePath:    packagePath,
			prefix:         prefix,
			expressionType: "",
			kind:           completionKindPackageMember,
		}
	}

	return &completionContext{
		pkgAlias:       "",
		packagePath:    "",
		prefix:         prefix,
		expressionType: identifier,
		kind:           completionKindFieldMethod,
	}
}

// getPackageMemberCompletions returns completions for exported types and
// functions from a package.
//
// Takes packagePath (string) which specifies the package import path to search.
// Takes prefix (string) which filters results to names starting with this text.
// Takes stdlibData (*inspector_dto.TypeData) which holds the type data to
// search.
//
// Returns []wasm_dto.CompletionItem which contains matching exported types and
// functions, or nil when stdlibData is nil or the package is not found.
func getPackageMemberCompletions(packagePath, prefix string, stdlibData *inspector_dto.TypeData) []wasm_dto.CompletionItem {
	if stdlibData == nil {
		return nil
	}

	inspectedPackage, ok := stdlibData.Packages[packagePath]
	if !ok {
		return nil
	}

	items := make([]wasm_dto.CompletionItem, 0, len(inspectedPackage.NamedTypes)+len(inspectedPackage.Funcs))
	prefixLower := strings.ToLower(prefix)

	for name, typ := range inspectedPackage.NamedTypes {
		if !isExported(name) {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), prefixLower) {
			continue
		}
		items = append(items, wasm_dto.CompletionItem{
			Label:         name,
			Kind:          "type",
			Detail:        typ.UnderlyingTypeString,
			Documentation: "",
			InsertText:    name,
			SortText:      "0" + name,
		})
	}

	for name, inspectedFunction := range inspectedPackage.Funcs {
		if !isExported(name) {
			continue
		}
		if prefix != "" && !strings.HasPrefix(strings.ToLower(name), prefixLower) {
			continue
		}
		items = append(items, wasm_dto.CompletionItem{
			Label:         name,
			Kind:          "function",
			Detail:        inspectedFunction.TypeString,
			Documentation: "",
			InsertText:    name,
			SortText:      "1" + name,
		})
	}

	return items
}

// findCompletionsAtPosition finds code completion suggestions for a position.
//
// When file is nil, imports are extracted directly from the source text.
//
// Takes file (*ast.File) which provides the parsed AST, or nil if not
// available.
// Takes source (string) which contains the raw source code.
// Takes line (int) which specifies the cursor line position.
// Takes column (int) which specifies the cursor column position.
// Takes stdlibData (*inspector_dto.TypeData) which provides standard library
// type information.
//
// Returns []wasm_dto.CompletionItem which contains matching completion
// suggestions for the context, or nil if none are found.
func findCompletionsAtPosition(file *ast.File, source string, line, column int, stdlibData *inspector_dto.TypeData) []wasm_dto.CompletionItem {
	var imports map[string]string
	if file != nil {
		imports = buildImportMap(file)
	} else {
		imports = extractImportsFromSource(source, stdlibData)
	}

	ctx := analyseCompletionContext(source, line, column, file, imports)

	switch ctx.kind {
	case completionKindPackageMember:
		return getPackageMemberCompletions(ctx.packagePath, ctx.prefix, stdlibData)

	case completionKindFieldMethod:
		return nil

	default:
		return getBasicCompletions(stdlibData).Items
	}
}

// extractImportsFromSource extracts import statements from source text using
// regex. This is used as a fallback when AST parsing fails.
//
// Takes source (string) which is the Go source code to parse for imports.
// Takes stdlibData (*inspector_dto.TypeData) which provides the standard
// library package data for validation.
//
// Returns map[string]string which maps import aliases to package paths for
// all standard library imports found in the source.
func extractImportsFromSource(source string, stdlibData *inspector_dto.TypeData) map[string]string {
	imports := make(map[string]string)
	if stdlibData == nil {
		return imports
	}

	matches := importSinglePattern.FindAllStringSubmatch(source, -1)

	for _, match := range matches {
		alias := match[1]
		packagePath := match[2]

		if alias == "" {
			parts := strings.Split(packagePath, pathSeparator)
			alias = parts[len(parts)-1]
		}

		if _, ok := stdlibData.Packages[packagePath]; ok {
			imports[alias] = packagePath
		}
	}

	blockMatches := importBlockPattern.FindAllStringSubmatch(source, -1)
	for _, blockMatch := range blockMatches {
		block := blockMatch[1]
		lineMatches := importLinePattern.FindAllStringSubmatch(block, -1)
		for _, lineMatch := range lineMatches {
			alias := lineMatch[1]
			packagePath := lineMatch[2]

			if alias == "" {
				parts := strings.Split(packagePath, pathSeparator)
				alias = parts[len(parts)-1]
			}

			if _, ok := stdlibData.Packages[packagePath]; ok {
				imports[alias] = packagePath
			}
		}
	}

	return imports
}

// isExported checks whether a name is an exported Go identifier.
//
// Takes name (string) which is the identifier to check.
//
// Returns bool which is true if name starts with an uppercase letter.
func isExported(name string) bool {
	if len(name) == 0 {
		return false
	}
	return unicode.IsUpper(rune(name[0]))
}

// lineColumnToOffset converts a 1-indexed line and column to a byte offset.
//
// Takes file (*token.File) which provides the source file position data.
// Takes line (int) which specifies the 1-indexed line number.
// Takes column (int) which specifies the 1-indexed column number.
//
// Returns int which is the byte offset within the file.
func lineColumnToOffset(file *token.File, line, column int) int {
	if file == nil {
		return 0
	}

	line = max(line, 1)
	line = min(line, file.LineCount())

	lineStart := file.LineStart(line)

	offset := int(lineStart) - file.Base() + (column - 1)
	offset = max(offset, 0)
	offset = min(offset, file.Size())

	return offset
}
