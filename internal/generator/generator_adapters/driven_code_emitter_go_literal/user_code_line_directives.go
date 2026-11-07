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

package driven_code_emitter_go_literal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxStatementMappingDepth bounds how deeply nested blocks are walked when statements
	// are mapped to .pk lines.
	maxStatementMappingDepth = 512

	// lineDirectiveResetOverhead is the length of a reset directive without its path,
	// used to size the output buffer.
	lineDirectiveResetOverhead = len("//line :\n") + 10
)

var (
	// lineDirectivePrefix is the prefix of a compiler-honoured line directive once it has
	// been moved to column 1.
	lineDirectivePrefix = []byte("//line ")

	// errUserDeclarationsMisaligned is returned when user functions in the original and
	// rewritten script ASTs cannot be paired.
	errUserDeclarationsMisaligned = errors.New("user declarations in the original and rewritten script do not line up")

	// errLineDirectiveResetParse is returned when the generated source cannot be parsed to
	// place line directive resets.
	errLineDirectiveResetParse = errors.New("parsing generated code to place line directive resets")

	// errLineDirectiveResetPositions is returned when the parsed generated source has no
	// position table.
	errLineDirectiveResetPositions = errors.New("generated code has no position table for line directive resets")

	// errGeneratedFilePath is returned when the generated file path cannot be made absolute.
	errGeneratedFilePath = errors.New("resolving generated file path for line directives")
)

// userCodeStatementMapper attaches a //line directive to every statement of a user script
// function so that each statement maps to its own .pk line in DWARF.
type userCodeStatementMapper struct {
	// emitter formats the directives.
	emitter *emitter

	// originalFileSet resolves positions in the original script AST.
	originalFileSet *token.FileSet

	// directivePath is the absolute .pk path written into each directive.
	directivePath string

	// scriptStartLine is the .pk line on which the script content starts.
	scriptStartLine int

	// mismatchedStatements counts statement pairs that could not be matched, so their
	// nested statements were left unmapped.
	mismatchedStatements int

	// depthLimitReached records that nesting went past maxStatementMappingDepth.
	depthLimitReached bool
}

// mapDeclaration returns the declaration to emit for a rewritten user function, carrying
// per-statement directives when it has a known original.
//
// Takes rewritten (*goast.FuncDecl) which is the function from the rewritten script AST.
// Takes original (*goast.FuncDecl) which is the paired function from the original script
// AST, or nil when none was paired.
//
// Returns goast.Decl which is either a copy carrying directives or the input unchanged.
func (m *userCodeStatementMapper) mapDeclaration(rewritten, original *goast.FuncDecl) goast.Decl {
	if m == nil || rewritten == nil || original == nil || rewritten.Body == nil || original.Body == nil {
		return rewritten
	}

	mappedFunc := *rewritten
	mappedFunc.Body = m.mapBlock(original.Body, rewritten.Body, 0)
	return &mappedFunc
}

// reportIncompleteMapping logs a warning when some statements could not be mapped to
// their .pk lines.
func (m *userCodeStatementMapper) reportIncompleteMapping(ctx context.Context) {
	if m == nil || (m.mismatchedStatements == 0 && !m.depthLimitReached) {
		return
	}

	_, l := logger_domain.From(ctx, log)
	l.Warn("Some user statements could not be mapped to their .pk lines; breakpoints on them may not bind",
		logger_domain.String("sourcePath", m.directivePath),
		logger_domain.Int("mismatchedStatements", m.mismatchedStatements),
		logger_domain.Bool("depthLimitReached", m.depthLimitReached),
		logger_domain.Int("maxDepth", maxStatementMappingDepth),
	)
}

// mapBlock copies a block, inserting a directive before each statement.
//
// Takes original (*goast.BlockStmt) which is the block from the original AST.
// Takes rewritten (*goast.BlockStmt) which is the matching block from the rewritten AST.
// Takes depth (int) which is the nesting depth of the block.
//
// Returns *goast.BlockStmt which is a new block, or rewritten unchanged when the blocks
// cannot be paired.
func (m *userCodeStatementMapper) mapBlock(original, rewritten *goast.BlockStmt, depth int) *goast.BlockStmt {
	if original == nil || rewritten == nil {
		if original != rewritten {
			m.mismatchedStatements++
		}
		return rewritten
	}

	mappedBlock := *rewritten
	mappedBlock.List = m.mapStatementList(original.List, rewritten.List, depth+1)
	return &mappedBlock
}

// mapStatementList pairs two statement lists by index and interleaves directives.
//
// Takes originalList ([]goast.Stmt) which holds the original statements with real
// positions.
// Takes rewrittenList ([]goast.Stmt) which holds the statements that will be printed.
// Takes depth (int) which is the nesting depth of the list.
//
// Returns []goast.Stmt which is the new list, or rewrittenList unchanged when the lengths
// differ or the depth limit is reached.
func (m *userCodeStatementMapper) mapStatementList(originalList, rewrittenList []goast.Stmt, depth int) []goast.Stmt {
	if depth > maxStatementMappingDepth {
		m.depthLimitReached = true
		return rewrittenList
	}
	if len(originalList) != len(rewrittenList) {
		m.mismatchedStatements++
		return rewrittenList
	}

	mappedList := make([]goast.Stmt, 0, 2*len(rewrittenList))
	for index, rewrittenStatement := range rewrittenList {
		originalStatement := originalList[index]
		if pkLine := m.pkLine(originalStatement.Pos()); pkLine > 0 {
			mappedList = append(mappedList, m.directiveStatement(pkLine))
		}
		mappedList = append(mappedList, m.mapStatement(originalStatement, rewrittenStatement, depth))
	}
	return mappedList
}

// mapStatement descends into compound statements so nested statements are mapped too.
//
// Takes original (goast.Stmt) which is the statement from the original AST.
// Takes rewritten (goast.Stmt) which is the matching rewritten statement.
// Takes depth (int) which is the nesting depth of the statement.
//
// Returns goast.Stmt which is a copy with mapped children, or rewritten unchanged when it
// has no nested statements or the two statements differ in kind.
func (m *userCodeStatementMapper) mapStatement(original, rewritten goast.Stmt, depth int) goast.Stmt {
	switch rewrittenStatement := rewritten.(type) {
	case *goast.BlockStmt:
		if originalStatement, ok := pairedStatement[*goast.BlockStmt](m, original); ok {
			return m.mapBlock(originalStatement, rewrittenStatement, depth)
		}
	case *goast.IfStmt:
		if originalStatement, ok := pairedStatement[*goast.IfStmt](m, original); ok {
			return m.mapIfStatement(originalStatement, rewrittenStatement, depth)
		}
	case *goast.LabeledStmt:
		if originalStatement, ok := pairedStatement[*goast.LabeledStmt](m, original); ok {
			mappedStatement := *rewrittenStatement
			mappedStatement.Stmt = m.mapStatement(originalStatement.Stmt, rewrittenStatement.Stmt, depth)
			return &mappedStatement
		}
	default:
		return m.mapBodiedStatement(original, rewritten, depth)
	}
	return rewritten
}

// mapBodiedStatement maps the body of a loop, switch or select statement.
//
// Takes original (goast.Stmt) which is the statement from the original AST.
// Takes rewritten (goast.Stmt) which is the matching rewritten statement.
// Takes depth (int) which is the nesting depth of the statement.
//
// Returns goast.Stmt which is a copy with a mapped body, or rewritten unchanged when it is
// not a loop, switch or select statement or the two statements differ in kind.
func (m *userCodeStatementMapper) mapBodiedStatement(original, rewritten goast.Stmt, depth int) goast.Stmt {
	switch rewrittenStatement := rewritten.(type) {
	case *goast.ForStmt:
		if originalStatement, ok := pairedStatement[*goast.ForStmt](m, original); ok {
			mappedStatement := *rewrittenStatement
			mappedStatement.Body = m.mapBlock(originalStatement.Body, rewrittenStatement.Body, depth)
			return &mappedStatement
		}
	case *goast.RangeStmt:
		if originalStatement, ok := pairedStatement[*goast.RangeStmt](m, original); ok {
			mappedStatement := *rewrittenStatement
			mappedStatement.Body = m.mapBlock(originalStatement.Body, rewrittenStatement.Body, depth)
			return &mappedStatement
		}
	case *goast.SwitchStmt:
		if originalStatement, ok := pairedStatement[*goast.SwitchStmt](m, original); ok {
			mappedStatement := *rewrittenStatement
			mappedStatement.Body = m.mapClauseBlock(originalStatement.Body, rewrittenStatement.Body, depth)
			return &mappedStatement
		}
	case *goast.TypeSwitchStmt:
		if originalStatement, ok := pairedStatement[*goast.TypeSwitchStmt](m, original); ok {
			mappedStatement := *rewrittenStatement
			mappedStatement.Body = m.mapClauseBlock(originalStatement.Body, rewrittenStatement.Body, depth)
			return &mappedStatement
		}
	case *goast.SelectStmt:
		if originalStatement, ok := pairedStatement[*goast.SelectStmt](m, original); ok {
			mappedStatement := *rewrittenStatement
			mappedStatement.Body = m.mapClauseBlock(originalStatement.Body, rewrittenStatement.Body, depth)
			return &mappedStatement
		}
	}
	return rewritten
}

// mapIfStatement maps the body and else branch of an if statement.
//
// Takes original (*goast.IfStmt) which is the statement from the original AST.
// Takes rewritten (*goast.IfStmt) which is the matching rewritten statement.
// Takes depth (int) which is the nesting depth of the statement.
//
// Returns goast.Stmt which is a copy with mapped branches.
func (m *userCodeStatementMapper) mapIfStatement(original, rewritten *goast.IfStmt, depth int) goast.Stmt {
	mappedStatement := *rewritten
	mappedStatement.Body = m.mapBlock(original.Body, rewritten.Body, depth)
	switch {
	case rewritten.Else != nil && original.Else != nil:
		mappedStatement.Else = m.mapStatement(original.Else, rewritten.Else, depth)
	case rewritten.Else != nil || original.Else != nil:
		m.mismatchedStatements++
	}
	return &mappedStatement
}

// mapClauseBlock maps the bodies of the case or communication clauses in a switch or
// select block.
//
// Takes original (*goast.BlockStmt) which holds the original clauses.
// Takes rewritten (*goast.BlockStmt) which holds the matching rewritten clauses.
// Takes depth (int) which is the nesting depth of the block.
//
// Returns *goast.BlockStmt which is a new block of mapped clauses, or rewritten unchanged
// when the clause counts differ.
func (m *userCodeStatementMapper) mapClauseBlock(original, rewritten *goast.BlockStmt, depth int) *goast.BlockStmt {
	if original == nil || rewritten == nil || len(original.List) != len(rewritten.List) {
		m.mismatchedStatements++
		return rewritten
	}

	mappedBlock := *rewritten
	mappedBlock.List = make([]goast.Stmt, len(rewritten.List))
	for index, rewrittenClause := range rewritten.List {
		mappedBlock.List[index] = m.mapClause(original.List[index], rewrittenClause, depth+1)
	}
	return &mappedBlock
}

// mapClause maps the body of a single case or communication clause.
//
// Takes original (goast.Stmt) which is the original clause.
// Takes rewritten (goast.Stmt) which is the matching rewritten clause.
// Takes depth (int) which is the nesting depth of the clause.
//
// Returns goast.Stmt which is a copy with a mapped body, or rewritten unchanged when the
// clauses differ in kind.
func (m *userCodeStatementMapper) mapClause(original, rewritten goast.Stmt, depth int) goast.Stmt {
	switch rewrittenClause := rewritten.(type) {
	case *goast.CaseClause:
		if originalClause, ok := pairedStatement[*goast.CaseClause](m, original); ok {
			mappedClause := *rewrittenClause
			mappedClause.Body = m.mapStatementList(originalClause.Body, rewrittenClause.Body, depth)
			return &mappedClause
		}
	case *goast.CommClause:
		if originalClause, ok := pairedStatement[*goast.CommClause](m, original); ok {
			mappedClause := *rewrittenClause
			mappedClause.Body = m.mapStatementList(originalClause.Body, rewrittenClause.Body, depth)
			return &mappedClause
		}
	default:
		m.mismatchedStatements++
	}
	return rewritten
}

// pkLine converts a position in the original script AST to its .pk line.
//
// Takes position (token.Pos) which is a position in the original script AST.
//
// Returns int which is the .pk line, or 0 when the position is unknown.
func (m *userCodeStatementMapper) pkLine(position token.Pos) int {
	if !position.IsValid() {
		return 0
	}
	scriptLine := m.originalFileSet.Position(position).Line
	if scriptLine <= 0 {
		return 0
	}
	return m.scriptStartLine + scriptLine - 1
}

// directiveStatement builds the pseudo-statement that prints as a line directive.
//
// Takes pkLine (int) which is the .pk line the next statement maps to.
//
// Returns goast.Stmt which prints as the directive text on its own line.
func (m *userCodeStatementMapper) directiveStatement(pkLine int) goast.Stmt {
	return &goast.ExprStmt{
		X: &goast.BasicLit{
			Kind:  token.COMMENT,
			Value: m.emitter.formatLineDirective(m.directivePath, pkLine, 0),
		},
	}
}

// recordUserDeclLineDirective queues the line directive that precedes a user declaration
// in the formatted output.
//
// Takes declaration (goast.Decl) which is the user declaration being copied.
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the .pk path.
// Takes userDeclLines (map[string]int) which maps declaration names to .pk lines, or nil
// when source locations are unavailable.
func (em *emitter) recordUserDeclLineDirective(
	declaration goast.Decl,
	mainComponent *annotator_dto.VirtualComponent,
	userDeclLines map[string]int,
) {
	if userDeclLines == nil {
		return
	}

	name, sig := declNameAndSignature(declaration)
	pkLine, ok := userDeclLines[name]
	if !ok || sig == "" {
		return
	}

	directivePath := em.lineDirectivePath(mainComponent.Source.SourcePath)
	em.ctx.userCodeLineDirectives = append(em.ctx.userCodeLineDirectives, userCodeLineDirective{
		declSignature: sig,
		directive:     em.formatLineDirective(directivePath, pkLine, 0),
	})
}

// newUserCodeStatementMapper builds a statement mapper for a component's script.
//
// Takes component (*annotator_dto.VirtualComponent) which provides the original script
// AST, its FileSet and the script start line.
// Takes em (*emitter) which supplies the configuration and directive formatting.
//
// Returns *userCodeStatementMapper which is nil when compiler-honoured line directives
// are disabled or the component lacks source location data.
func newUserCodeStatementMapper(component *annotator_dto.VirtualComponent, em *emitter) *userCodeStatementMapper {
	if em == nil || !em.config.EnableDwarfLineDirectives || component == nil || component.Source == nil {
		return nil
	}

	script := component.Source.Script
	if script == nil || script.AST == nil || script.Fset == nil || script.ScriptStartLocation.Line <= 0 {
		return nil
	}

	return &userCodeStatementMapper{
		emitter:              em,
		originalFileSet:      script.Fset,
		directivePath:        em.lineDirectivePath(component.Source.SourcePath),
		scriptStartLine:      script.ScriptStartLocation.Line,
		mismatchedStatements: 0,
		depthLimitReached:    false,
	}
}

// resolveGeneratedFilePath works out the generated file path written into reset line
// directives.
//
// Takes request (generator_dto.GenerateRequest) which provides the output path and the
// line directive setting.
//
// Returns string which is the path, or empty when compiler-honoured line directives are
// off or no output path was given.
// Returns error when the output path cannot be made absolute.
func resolveGeneratedFilePath(ctx context.Context, request generator_dto.GenerateRequest) (string, error) {
	if !request.EnableDwarfLineDirectives {
		return "", nil
	}
	if request.OutputPath == "" {
		_, l := logger_domain.From(ctx, log)
		l.Warn("No output path for generated code; code after mapped .pk lines will report .pk positions",
			logger_domain.String("sourcePath", request.SourcePath))
		return "", nil
	}
	return generatedFilePathForDirectives(request.OutputPath, filepath.Abs)
}

// pairFunctionsForMapping pairs the rewritten user functions with their originals when
// statements are being mapped, logging a warning when some cannot be paired.
//
// Takes statementMapper (*userCodeStatementMapper) which is nil when statements are not
// being mapped.
// Takes component (*annotator_dto.VirtualComponent) which provides both script ASTs.
//
// Returns map[*goast.FuncDecl]*goast.FuncDecl which maps rewritten functions to their
// originals, or nil when statements are not being mapped.
func pairFunctionsForMapping(
	ctx context.Context,
	statementMapper *userCodeStatementMapper,
	component *annotator_dto.VirtualComponent,
) map[*goast.FuncDecl]*goast.FuncDecl {
	if statementMapper == nil {
		return nil
	}

	pairs, err := pairUserFunctions(component.Source.Script.AST, component.RewrittenScriptAST)
	if err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Warn("Some user functions could not be mapped to their .pk lines; breakpoints in them may not bind",
			logger_domain.String("sourcePath", statementMapper.directivePath),
			logger_domain.Error(err))
	}
	return pairs
}

// pairedStatement asserts that an original statement has the same kind as its rewritten
// partner, counting a mismatch on the mapper when it does not.
//
// Takes m (*userCodeStatementMapper) which counts mismatches.
// Takes original (goast.Stmt) which is the statement from the original AST.
//
// Returns T which is the typed original statement.
// Returns bool which is true when the kinds match.
func pairedStatement[T goast.Stmt](m *userCodeStatementMapper, original goast.Stmt) (T, bool) {
	typed, ok := original.(T)
	if !ok {
		m.mismatchedStatements++
	}
	return typed, ok
}

// pairUserFunctions pairs each rewritten function declaration with the original
// declaration it was printed from.
//
// Takes original (*goast.File) which is the original script AST.
// Takes rewritten (*goast.File) which is the rewritten script AST.
//
// Returns map[*goast.FuncDecl]*goast.FuncDecl which maps rewritten functions to their
// originals; it holds every pair that could be made, even when an error is returned.
// Returns error which wraps errUserDeclarationsMisaligned when an original function has
// no matching rewritten function.
func pairUserFunctions(original, rewritten *goast.File) (map[*goast.FuncDecl]*goast.FuncDecl, error) {
	pairs := make(map[*goast.FuncDecl]*goast.FuncDecl)
	if original == nil || rewritten == nil {
		return pairs, nil
	}

	originalDecls := nonImportDeclarations(original.Decls)
	rewrittenDecls := nonImportDeclarations(rewritten.Decls)
	offset := len(rewrittenDecls) - len(originalDecls)
	if offset < 0 {
		return pairs, fmt.Errorf("%w: %d original declarations but only %d rewritten",
			errUserDeclarationsMisaligned, len(originalDecls), len(rewrittenDecls))
	}

	var unpaired []string
	for index, originalDecl := range originalDecls {
		originalFunc, originalIsFunc := originalDecl.(*goast.FuncDecl)
		if !originalIsFunc {
			continue
		}
		rewrittenFunc, rewrittenIsFunc := rewrittenDecls[offset+index].(*goast.FuncDecl)
		if !rewrittenIsFunc || !sameFunctionShape(originalFunc, rewrittenFunc) {
			unpaired = append(unpaired, functionName(originalFunc))
			continue
		}
		pairs[rewrittenFunc] = originalFunc
	}

	if len(unpaired) > 0 {
		return pairs, fmt.Errorf("%w: no match for %s", errUserDeclarationsMisaligned, strings.Join(unpaired, ", "))
	}
	return pairs, nil
}

// sameFunctionShape reports whether two function declarations plausibly print the same
// source function.
//
// Takes first (*goast.FuncDecl) which is one declaration.
// Takes second (*goast.FuncDecl) which is the other declaration.
//
// Returns bool which is true when the names and receiver presence match.
func sameFunctionShape(first, second *goast.FuncDecl) bool {
	if first.Name == nil || second.Name == nil || first.Name.Name != second.Name.Name {
		return false
	}
	return (first.Recv == nil) == (second.Recv == nil)
}

// functionName returns the name of a function declaration for messages.
//
// Takes functionDecl (*goast.FuncDecl) which is the declaration to name.
//
// Returns string which is the name, or "<unnamed>" when it has none.
func functionName(functionDecl *goast.FuncDecl) string {
	if functionDecl.Name == nil {
		return "<unnamed>"
	}
	return functionDecl.Name.Name
}

// nonImportDeclarations filters out import declarations.
//
// Takes declarations ([]goast.Decl) which is the declaration list to filter.
//
// Returns []goast.Decl which holds the non-import declarations in order.
func nonImportDeclarations(declarations []goast.Decl) []goast.Decl {
	filtered := make([]goast.Decl, 0, len(declarations))
	for _, declaration := range declarations {
		if genDecl, isGen := declaration.(*goast.GenDecl); isGen && genDecl.Tok == token.IMPORT {
			continue
		}
		filtered = append(filtered, declaration)
	}
	return filtered
}

// generatedFilePathForDirectives turns the generated file's output path into the form
// written into reset line directives.
//
// Takes outputPath (string) which is the path the generated file is written to; it may be
// relative to the working directory.
// Takes absolute (func(string) (string, error)) which makes a path absolute.
//
// Returns string which is the absolute, slash-separated path, or empty when outputPath is
// empty.
// Returns error which wraps errGeneratedFilePath when the path cannot be made absolute.
func generatedFilePathForDirectives(outputPath string, absolute func(string) (string, error)) (string, error) {
	if outputPath == "" {
		return "", nil
	}

	absolutePath, err := absolute(outputPath)
	if err != nil {
		return "", fmt.Errorf("%w %q: %w", errGeneratedFilePath, outputPath, err)
	}

	return filepath.ToSlash(absolutePath), nil
}

// appendLineDirectiveResets restores the generated file's own positions after every
// top-level declaration that carries a line directive.
//
// Takes source ([]byte) which is the generated Go source with directives already at
// column 1.
// Takes generatedFilePath (string) which is the absolute, slash-separated path of the
// generated file.
//
// Returns []byte which is the source with reset directives inserted, or source unchanged
// when there is nothing to reset.
// Returns error when the source cannot be parsed.
func appendLineDirectiveResets(source []byte, generatedFilePath string) ([]byte, error) {
	if generatedFilePath == "" || !bytes.Contains(source, lineDirectivePrefix) {
		return source, nil
	}

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "", source, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errLineDirectiveResetParse, err)
	}

	tokenFile := fileSet.File(file.Package)
	if tokenFile == nil {
		return nil, errLineDirectiveResetPositions
	}

	lineStarts := tokenFile.Lines()
	resetAfterLines := declarationEndsNeedingReset(tokenFile, file.Decls, source, lineStarts)
	if len(resetAfterLines) == 0 {
		return source, nil
	}

	result := make([]byte, 0, len(source)+len(resetAfterLines)*(len(generatedFilePath)+lineDirectiveResetOverhead))
	copiedUpTo := 0
	for resetCount, endLine := range resetAfterLines {
		lineEnd := len(source)
		if endLine < len(lineStarts) {
			lineEnd = lineStarts[endLine]
		}
		result = append(result, source[copiedUpTo:lineEnd]...)
		copiedUpTo = lineEnd
		if len(result) > 0 && result[len(result)-1] != '\n' {
			result = append(result, '\n')
		}
		nextOutputLine := endLine + resetCount + 2
		result = fmt.Appendf(result, "//line %s:%d\n", generatedFilePath, nextOutputLine)
	}
	return append(result, source[copiedUpTo:]...), nil
}

// declarationEndsNeedingReset finds the last physical line of every top-level declaration
// whose span, including the lines between it and the previous declaration, holds a line
// directive.
//
// Takes tokenFile (*token.File) which resolves the parsed positions.
// Takes declarations ([]goast.Decl) which are the parsed top-level declarations in order.
// Takes source ([]byte) which is the parsed source.
// Takes lineStarts ([]int) which holds the byte offset of each line start.
//
// Returns []int which holds the 1-based physical end lines needing a reset, in increasing
// order.
func declarationEndsNeedingReset(tokenFile *token.File, declarations []goast.Decl, source []byte, lineStarts []int) []int {
	var endLines []int
	previousEndLine := 0
	for _, declaration := range declarations {
		endLine := tokenFile.PositionFor(declaration.End(), false).Line
		if endLine <= previousEndLine || endLine > len(lineStarts) {
			continue
		}
		for lineIndex := previousEndLine; lineIndex < endLine; lineIndex++ {
			if bytes.HasPrefix(source[lineStarts[lineIndex]:], lineDirectivePrefix) {
				endLines = append(endLines, endLine)
				break
			}
		}
		previousEndLine = endLine
	}
	return endLines
}
