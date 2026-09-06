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

package compiler_domain

import (
	"fmt"

	"piko.sh/piko/internal/esbuild/ast"
	"piko.sh/piko/internal/esbuild/js_ast"
)

// importStatementSpan is the source range of one top-level import statement.
type importStatementSpan struct {
	// start is the offset of the statement's import keyword.
	start int

	// end is the offset just past the module path, or just past the closing brace of the
	// import attributes clause when the statement has one. A trailing semicolon is not part
	// of the span; the statement parses the same without it.
	end int
}

// importStatementLocator finds the top-level statement that owns each module path in a
// script.
type importStatementLocator struct {
	// imports maps the source offset of each top-level import's module path to the span of
	// its statement.
	imports map[int32]importStatementSpan

	// reexports holds the module path offsets of top-level export-from statements.
	reexports map[int32]struct{}
}

// locateImportStatements finds where every top-level import statement in a script starts
// and ends.
//
// The script is parsed a second time with unused imports preserved, so imports that the
// main parse trims (those used only by the template) are still found. Both ends of each
// statement come from this one parse, using the statement's own location as the start and
// the end of its module path or attributes clause as the end. The word "import" inside a
// binding name, comment or string can never be mistaken for the keyword, and a statement
// spanning several lines is never cut short.
//
// Takes source (string) which is the exact script text the main parse read.
//
// Returns importStatementLocator which maps each import path offset to its statement.
// Returns error when the script cannot be parsed, or when the parser reports an import
// whose record it does not hold.
func locateImportStatements(source string) (importStatementLocator, error) {
	locator := importStatementLocator{
		imports:   make(map[int32]importStatementSpan),
		reexports: make(map[int32]struct{}),
	}
	if source == "" {
		return locator, nil
	}

	options := importPreservingParserOptions()
	tree, err := parseTypeScript(source, "import-locator.ts", false, &options)
	if err != nil {
		return importStatementLocator{}, fmt.Errorf("locating import statements: %w", err)
	}

	for partIndex := range tree.Parts {
		for _, statement := range tree.Parts[partIndex].Stmts {
			if err := locator.addStatement(tree.ImportRecords, statement); err != nil {
				return importStatementLocator{}, err
			}
		}
	}
	return locator, nil
}

// addStatement records where a top-level statement that names a module path sits.
//
// Takes records ([]ast.ImportRecord) which are the import records of the locating parse.
// Takes statement (js_ast.Stmt) which is the top-level statement to record.
//
// Returns error when the statement names an import record the parse does not hold.
func (l importStatementLocator) addStatement(records []ast.ImportRecord, statement js_ast.Stmt) error {
	switch s := statement.Data.(type) {
	case *js_ast.SImport:
		record, err := importRecordAt(records, s.ImportRecordIndex, statement.Loc.Start)
		if err != nil {
			return err
		}
		l.imports[record.Range.Loc.Start] = importStatementSpan{
			start: int(statement.Loc.Start),
			end:   importStatementEnd(record),
		}
	case *js_ast.SExportFrom:
		return l.addReexport(records, s.ImportRecordIndex, statement.Loc.Start)
	case *js_ast.SExportStar:
		return l.addReexport(records, s.ImportRecordIndex, statement.Loc.Start)
	}
	return nil
}

// addReexport records the module path of a top-level export-from statement.
//
// Takes records ([]ast.ImportRecord) which are the import records of the locating parse.
// Takes recordIndex (uint32) which is the statement's import record index.
// Takes statementStart (int32) which is the statement's offset, for error reporting.
//
// Returns error when the record index is outside records.
func (l importStatementLocator) addReexport(records []ast.ImportRecord, recordIndex uint32, statementStart int32) error {
	record, err := importRecordAt(records, recordIndex, statementStart)
	if err != nil {
		return err
	}
	l.reexports[record.Range.Loc.Start] = struct{}{}
	return nil
}

// statementSpan returns the span of the top-level import statement owning a module path.
//
// Takes pathStart (int32) which is the source offset of the module path's opening quote.
//
// Returns importStatementSpan which is the owning statement's span.
// Returns error when no top-level import owns the path. An export-from statement yields
// errReexportUnsupported; an import inside a declare module block or another nested
// context yields errNestedImportUnsupported.
func (l importStatementLocator) statementSpan(pathStart int32) (importStatementSpan, error) {
	if span, ok := l.imports[pathStart]; ok {
		return span, nil
	}
	if _, ok := l.reexports[pathStart]; ok {
		return importStatementSpan{}, errReexportUnsupported
	}
	return importStatementSpan{}, errNestedImportUnsupported
}

// importRecordAt returns the import record a statement names.
//
// Takes records ([]ast.ImportRecord) which are the import records of the parse.
// Takes recordIndex (uint32) which is the index the statement names.
// Takes statementStart (int32) which is the statement's offset, for error reporting.
//
// Returns ast.ImportRecord which is the named record.
// Returns error when recordIndex is outside records.
func importRecordAt(records []ast.ImportRecord, recordIndex uint32, statementStart int32) (ast.ImportRecord, error) {
	if int(recordIndex) >= len(records) {
		return ast.ImportRecord{}, fmt.Errorf("statement at offset %d names import record %d of %d: %w",
			statementStart, recordIndex, len(records), errImportNotRecovered)
	}
	return records[recordIndex], nil
}

// importStatementEnd returns the offset just past the last token of an import statement
// other than its optional semicolon.
//
// Takes record (ast.ImportRecord) which is the statement's import record.
//
// Returns int which is the end of the module path, or of the attributes clause when the
// statement has one.
func importStatementEnd(record ast.ImportRecord) int {
	end := int(record.Range.Loc.Start) + int(record.Range.Len)
	if record.AssertOrWith != nil {
		end = max(end, int(record.AssertOrWith.InnerCloseBraceLoc.Start)+1)
	}
	return end
}
