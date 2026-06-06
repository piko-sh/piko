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

package emitter_shared

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"piko.sh/piko/internal/goastutil"
)

const (
	// identRuntimeColumn names the generated column identifier.
	identRuntimeColumn = "column"

	// identRuntimeOperator names the generated operator identifier.
	identRuntimeOperator = "operator"

	// identRuntimeValue names the generated value identifier.
	identRuntimeValue = "value"

	// identRuntimeParamCount names the generated parameter-count identifier.
	identRuntimeParamCount = "paramCount"

	// identRuntimeTrimmed names the generated trimmed-column local.
	identRuntimeTrimmed = "trimmed"

	// identRuntimeInner names the generated inner-expression local.
	identRuntimeInner = "inner"

	// identRuntimeElements names the generated flattened-elements local.
	identRuntimeElements = "elements"

	// identRuntimeElementsLen names the generated element-count local.
	identRuntimeElementsLen = "elementsLen"

	// identRuntimeIndex names the generated loop-index local.
	identRuntimeIndex = "i"

	// identReflectValue names the reflect.Value local the reflect fallback in the emitted
	// pikoReflectSlice and pikoMultiValueLen helpers uses.
	identReflectValue = "rv"

	// identReflectPre names the pre-asserted []any local in the reflect fallback.
	identReflectPre = "pre"

	// identReflectOK names the type-assertion ok local in the reflect fallback.
	identReflectOK = "ok"

	// identStringsPkg names the strings package selector used by the emitted helpers.
	identStringsPkg = "strings"

	// methodTrimSpace names the strings.TrimSpace method called by the emitted helpers.
	methodTrimSpace = "TrimSpace"

	// builtinLength names the len builtin called by the emitted helpers.
	builtinLength = "len"

	// clauseSeparator is the single space joining clause tokens in the emitted helpers.
	clauseSeparator = " "

	// maxColumnExpressionLength bounds the column-expression string handed to the emitted
	// validator regex. RE2 is linear-time, but the bare-identifier alternation tolerates
	// arbitrary repetition of JSON arrow segments, so the developer-friendly cap keeps
	// pathological inputs from costing CPU before the allow-list check rejects them.
	maxColumnExpressionLength = 256
)

// helperLineAllocator hands out monotonically increasing source line positions from a
// synthetic token.File so positioned AST nodes (the allow-list map literals) render as
// gofmt-aligned vertical literals.
//
// The standard go/printer keeps a composite literal on a single line unless its elements
// sit on distinct source lines, and a positionless AST therefore collapses to one line
// that gofmt will not re-break. Allocating a real token.File with positions on separate
// lines lets the printer reproduce the canonical vertical form FormatFileWithAST cannot,
// because FormatFileWithAST renders from a position-free file set.
type helperLineAllocator struct {
	// file is the synthetic token.File whose registered line offsets back the positions
	// handed out by next.
	file *token.File

	// fileSet owns file and is passed to the printer so the allocated positions resolve to
	// their intended lines.
	fileSet *token.FileSet

	// nextLine is the 1-based line number returned by the following call to next.
	nextLine int
}

// newHelperLineAllocator builds a line allocator backed by a synthetic file large enough
// to register one position per line for any helper file the emitter produces.
//
// Returns *helperLineAllocator which is the initialised line allocator.
func newHelperLineAllocator() *helperLineAllocator {
	const syntheticFileSize = 1 << 20
	const syntheticLineStride = 64
	fileSet := token.NewFileSet()
	file := fileSet.AddFile("piko_runtime_helper_synthetic.go", -1, syntheticFileSize)
	for offset := syntheticLineStride; offset < syntheticFileSize; offset += syntheticLineStride {
		file.AddLine(offset)
	}
	return &helperLineAllocator{file: file, fileSet: fileSet, nextLine: 1}
}

// next returns the position at the start of the next unused synthetic line.
//
// Returns token.Pos which is the position at the start of the next synthetic line.
func (allocator *helperLineAllocator) next() token.Pos {
	position := allocator.file.LineStart(allocator.nextLine)
	allocator.nextLine++
	return position
}

// formatHelperFile renders the helper declarations through the same goimports / gofmt
// pipeline as FormatFileWithAST, but using the line allocator's file set so the
// positioned allow-list maps keep their vertical, aligned layout.
//
// Takes packageName which is the generated package name.
// Takes tracker which holds the imports to apply.
// Takes declarations which are the top-level declarations to render.
// Takes lines which provides the synthetic file set carrying the map positions.
//
// Returns []byte which is the GeneratedFileHeader-prefixed gofmt-canonical source,
// matching the shape of FormatFileWithAST output.
// Returns error when formatting fails.
func formatHelperFile(
	packageName string,
	tracker *ImportTracker,
	declarations []ast.Decl,
	lines *helperLineAllocator,
) ([]byte, error) {
	file := &ast.File{
		Name:  ast.NewIdent(packageName),
		Decls: declarations,
	}
	tracker.ApplyImports(lines.fileSet, file)

	formatted, formatError := goastutil.FormatAST(lines.fileSet, file)
	if formatError != nil {
		return nil, fmt.Errorf("formatting Go AST: %w", formatError)
	}

	var result bytes.Buffer
	result.Grow(len(GeneratedFileHeader) + len(formatted))
	_, _ = result.WriteString(GeneratedFileHeader)
	_, _ = result.Write(formatted)
	return result.Bytes(), nil
}

// buildMaxColumnExpressionLengthConst builds the pikoMaxColumnExpressionLength const
// declaration that bounds the column-expression string handed to the validator regex.
//
// Returns ast.Decl which is the const declaration.
func buildMaxColumnExpressionLengthConst() ast.Decl {
	return &ast.GenDecl{
		Doc: docComment(
			"// pikoMaxColumnExpressionLength bounds the column-expression string the",
			"// validator regex evaluates. RE2 is linear-time but the bare-identifier",
			"// alternation tolerates arbitrary repetition of JSON arrow segments, so",
			"// a developer-friendly cap prevents pathological inputs from costing",
			"// CPU before the allow-list check rejects them.",
		),
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent("pikoMaxColumnExpressionLength")},
				Values: []ast.Expr{goastutil.IntLit(maxColumnExpressionLength)},
			},
		},
	}
}

// buildRuntimeBuilderSentinelsVar emits the package-level sentinel errors for the
// out-of-range runtime-builder inputs.
//
// The generated runtime-builder chainable methods store these on builder.pendingError
// when a caller supplies a column, operator, or sort direction outside the validated
// allow-list. Deferring the sentinel to the All, One, or Count terminal keeps the
// chainable .Where and .OrderBy calls panic-free, so a bad caller value surfaces as a
// returned error rather than crashing the process, which in a framework would take down
// every consumer of the generated code.
//
// Returns ast.Decl which is the var declaration of the sentinel errors.
func buildRuntimeBuilderSentinelsVar() ast.Decl {
	sentinel := func(name, message string) ast.Spec {
		return &ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent(name)},
			Values: []ast.Expr{goastutil.CallExpr(goastutil.SelectorExpr("errors", "New"), goastutil.StrLit(message))},
		}
	}
	return &ast.GenDecl{
		Tok:    token.VAR,
		Lparen: 1,
		Specs: []ast.Spec{
			sentinel("errPikoUnknownColumn", "piko: unknown column in runtime query filter"),
			sentinel("errPikoUnknownOperator", "piko: unknown operator in runtime query filter"),
			sentinel("errPikoUnknownDirection", "piko: unknown sort direction in runtime query order"),
		},
	}
}

// buildAllowedOperatorsVar builds the pikoAllowedOperators allow-list. The entries are
// emitted in the same order as the source template so the generated map literal stays
// stable across runs.
//
// The PostgreSQL JSONB existence operators (?, ?|, ?&) are only included on engines whose
// driver binds by numbered placeholder ($N). On an anonymous-marker engine (MySQL or
// SQLite) the bind placeholder is itself "?", so allowing "?" as an operator would let a
// runtime predicate's literal "?" be miscounted as a bind site and corrupt the argument
// ordering.
//
// Takes lines (*helperLineAllocator) which supplies the synthetic line positions for the
// map literal.
// Takes useNumberedPlaceholders (bool) which enables the JSONB existence operators on
// numbered-placeholder engines.
//
// Returns ast.Decl which is the var declaration of the allow-list.
func buildAllowedOperatorsVar(lines *helperLineAllocator, useNumberedPlaceholders bool) ast.Decl {
	operators := []string{
		"=", "!=", "<>", "<", ">", "<=", ">=",
		"LIKE", "ILIKE", "IS NULL", "IS NOT NULL",
		"IN", "NOT IN",
	}
	if useNumberedPlaceholders {
		operators = append(operators, "?", "?|", "?&")
	}
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent("pikoAllowedOperators")},
				Values: []ast.Expr{boolAllowListLiteral(operators, lines)},
			},
		},
	}
}

// buildAllowedDirectionsVar builds the pikoAllowedDirections allow-list consulted after
// OrderBy() normalises the supplied direction. Only the upper-cased forms are stored
// because pikoNormaliseDirection folds the caller input to upper case before the lookup.
//
// Takes lines (*helperLineAllocator) which supplies the synthetic line positions for the
// map literal.
//
// Returns ast.Decl which is the var declaration of the allow-list.
func buildAllowedDirectionsVar(lines *helperLineAllocator) ast.Decl {
	directions := []string{
		"ASC", "DESC",
		"ASC NULLS FIRST", "ASC NULLS LAST",
		"DESC NULLS FIRST", "DESC NULLS LAST",
	}
	return &ast.GenDecl{
		Doc: docComment(
			"// pikoAllowedDirections is the exact-match allow-list consulted by every",
			"// runtime-builder OrderBy() call after the supplied direction has been",
			"// trimmed and uppercased. Storing only the upper-cased forms keeps the",
			"// allow-list small while letting callers pass mixed-case shorthand like",
			`// "Asc" or "desc nulls last" without surfacing a runtime panic.`,
		),
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{ast.NewIdent("pikoAllowedDirections")},
				Values: []ast.Expr{boolAllowListLiteral(directions, lines)},
			},
		},
	}
}

// boolAllowListLiteral builds a map[string]bool composite literal whose keys are the
// supplied strings in order, each mapped to the true identifier.
//
// Each key carries a distinct source line position drawn from lines, and the braces
// straddle the element block, so the standard printer renders the map vertically (one
// entry per line) rather than collapsing it onto a single line. Fresh *ast.BasicLit key
// nodes are built here rather than reusing the shared goastutil.StrLit cache because the
// position assignment mutates the node and the cached literals are shared across the
// whole emit pass.
//
// Takes keys ([]string) which are the allow-list entries in emission order.
// Takes lines (*helperLineAllocator) which supplies the per-key line positions.
//
// Returns ast.Expr which is the vertically formatted map literal.
func boolAllowListLiteral(keys []string, lines *helperLineAllocator) ast.Expr {
	lbrace := lines.next()
	elements := make([]ast.Expr, 0, len(keys))
	for _, key := range keys {
		keyLiteral := &ast.BasicLit{
			Kind:     token.STRING,
			Value:    strconv.Quote(key),
			ValuePos: lines.next(),
		}
		elements = append(elements, goastutil.KeyValueExpr(keyLiteral, goastutil.BoolIdent(true)))
	}
	return &ast.CompositeLit{
		Type:   goastutil.MapType(goastutil.CachedIdent(IdentString), goastutil.CachedIdent(IdentBool)),
		Lbrace: lbrace,
		Elts:   elements,
		Rbrace: lines.next(),
	}
}

// buildNormaliseDirectionFunc builds pikoNormaliseDirection, which trims and upper-cases
// a direction string before the allow-list lookup.
//
// Returns ast.Decl which is the function declaration.
func buildNormaliseDirectionFunc() ast.Decl {
	body := goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CallExpr(
		goastutil.SelectorExpr(identStringsPkg, "ToUpper"),
		goastutil.CallExpr(goastutil.SelectorExpr(identStringsPkg, methodTrimSpace), goastutil.CachedIdent("direction")),
	)))
	decl := goastutil.FuncDecl(
		"pikoNormaliseDirection",
		goastutil.FieldList(goastutil.Field("direction", goastutil.CachedIdent(IdentString))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentString))),
		body,
	)
	decl.Doc = docComment(
		"// pikoNormaliseDirection trims surrounding whitespace and folds the",
		"// caller-supplied direction string to upper case before the allow-list",
		"// lookup. Centralised so the validator and the eventual emit path agree",
		"// on the canonical form.",
	)
	return decl
}

// buildColumnExpressionRegexVar builds the pikoColumnExpressionRegex package variable
// assigned regexp.MustCompile(<pattern>). The pattern is emitted as a backtick raw-string
// literal because it contains backslash escape sequences the validator must preserve
// verbatim.
//
// Returns ast.Decl which is the var declaration.
func buildColumnExpressionRegexVar() ast.Decl {
	rawPattern := &ast.BasicLit{Kind: token.STRING, Value: "`" + runtimeColumnExpressionPattern + "`"}
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{ast.NewIdent("pikoColumnExpressionRegex")},
				Values: []ast.Expr{goastutil.CallExpr(
					goastutil.SelectorExpr("regexp", "MustCompile"),
					rawPattern,
				)},
			},
		},
	}
}

// buildValidColumnExpressionFunc builds pikoValidColumnExpression, which trims the
// column, rejects an over-length input, and defers to the regex match.
//
// Returns ast.Decl which is the function declaration.
func buildValidColumnExpressionFunc() ast.Decl {
	trimmed := goastutil.DefineStmt(identRuntimeTrimmed, goastutil.CallExpr(
		goastutil.SelectorExpr(identStringsPkg, methodTrimSpace), goastutil.CachedIdent(identRuntimeColumn),
	))
	lengthGuard := goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{
			X:  goastutil.CallExpr(goastutil.CachedIdent(builtinLength), goastutil.CachedIdent(identRuntimeTrimmed)),
			Op: token.GTR,
			Y:  goastutil.CachedIdent("pikoMaxColumnExpressionLength"),
		},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.BoolIdent(false))),
	)
	matchReturn := goastutil.ReturnStmt(goastutil.CallExpr(
		goastutil.SelectorExpr("pikoColumnExpressionRegex", "MatchString"),
		goastutil.CachedIdent(identRuntimeTrimmed),
	))
	return goastutil.FuncDecl(
		"pikoValidColumnExpression",
		goastutil.FieldList(goastutil.Field(identRuntimeColumn, goastutil.CachedIdent(IdentString))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentBool))),
		goastutil.BlockStmt(trimmed, lengthGuard, matchReturn),
	)
}

// buildExtractColumnRootFunc builds pikoExtractColumnRoot, which peels the validated cast
// wrappers off a column expression so the downstream allow-list check targets the inner
// root identifier.
//
// The body strips wrappers in three stages. A standard CAST(<bare> AS <type>) wrapper is
// removed first; the regex has already proved the shape, so the only work is finding the
// inner bare expression by preferring the last " as " separator and falling back to the
// last ")" when the cast keyword appears without an " as " segment. A parenthesised
// postgres cast wrapper (bare)::type is removed next; the trailing ")::type" suffix is
// the regex's responsibility, so here the closing ")::" marker locates the bare form to
// feed into the JSON and arrow extraction. A json_extract(<root>, '<path>') call or a
// JSON arrow path then yields the leading root identifier.
//
// Returns ast.Decl which is the function declaration.
func buildExtractColumnRootFunc() ast.Decl {
	body := goastutil.BlockStmt(
		goastutil.DefineStmt(identRuntimeTrimmed, goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, methodTrimSpace), goastutil.CachedIdent(identRuntimeColumn),
		)),
		extractStripStandardCast(),
		extractStripPostgresCast(),
		extractJSONExtractRoot(),
		extractArrowRoot(),
		goastutil.ReturnStmt(goastutil.CachedIdent(identRuntimeTrimmed)),
	)
	return goastutil.FuncDecl(
		"pikoExtractColumnRoot",
		goastutil.FieldList(goastutil.Field(identRuntimeColumn, goastutil.CachedIdent(IdentString))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentString))),
		body,
	)
}

// extractStripStandardCast builds the `if lower := strings.ToLower(trimmed);
// strings.HasPrefix(lower, "cast(") { ... }` block that removes a standard CAST(<bare> AS
// <type>) envelope from trimmed.
//
// Returns ast.Stmt which is the wrapper-stripping if block.
func extractStripStandardCast() ast.Stmt {
	innerAssign := goastutil.DefineStmt(identRuntimeInner, sliceFrom(
		goastutil.CachedIdent(identRuntimeTrimmed),
		goastutil.CallExpr(goastutil.CachedIdent(builtinLength), goastutil.StrLit("cast(")),
	))
	asIf := &ast.IfStmt{
		Init: goastutil.DefineStmt("asIndex", goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "LastIndex"),
			goastutil.CallExpr(goastutil.SelectorExpr(identStringsPkg, "ToLower"), goastutil.CachedIdent(identRuntimeInner)),
			goastutil.StrLit(" as "),
		)),
		Cond: greaterThanZero(goastutil.CachedIdent("asIndex")),
		Body: goastutil.BlockStmt(goastutil.AssignStmt(
			goastutil.CachedIdent(identRuntimeInner),
			sliceTo(goastutil.CachedIdent(identRuntimeInner), goastutil.CachedIdent("asIndex")),
		)),
		Else: &ast.IfStmt{
			Init: goastutil.DefineStmt("closeIndex", goastutil.CallExpr(
				goastutil.SelectorExpr(identStringsPkg, "LastIndex"),
				goastutil.CachedIdent(identRuntimeInner),
				goastutil.StrLit(")"),
			)),
			Cond: greaterThanZero(goastutil.CachedIdent("closeIndex")),
			Body: goastutil.BlockStmt(goastutil.AssignStmt(
				goastutil.CachedIdent(identRuntimeInner),
				sliceTo(goastutil.CachedIdent(identRuntimeInner), goastutil.CachedIdent("closeIndex")),
			)),
		},
	}
	trimAssign := goastutil.AssignStmt(
		goastutil.CachedIdent(identRuntimeTrimmed),
		goastutil.CallExpr(goastutil.SelectorExpr(identStringsPkg, methodTrimSpace), goastutil.CachedIdent(identRuntimeInner)),
	)
	return &ast.IfStmt{
		Init: goastutil.DefineStmt("lower", goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "ToLower"), goastutil.CachedIdent(identRuntimeTrimmed),
		)),
		Cond: goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "HasPrefix"),
			goastutil.CachedIdent("lower"),
			goastutil.StrLit("cast("),
		),
		Body: goastutil.BlockStmt(innerAssign, asIf, trimAssign),
	}
}

// extractStripPostgresCast builds the `if strings.HasPrefix(trimmed, "(") { ... }` block
// that removes a parenthesised (bare)::type postgres cast.
//
// Returns ast.Stmt which is the wrapper-stripping if block.
func extractStripPostgresCast() ast.Stmt {
	castIf := &ast.IfStmt{
		Init: goastutil.DefineStmt("castIndex", goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "LastIndex"),
			goastutil.CachedIdent(identRuntimeTrimmed),
			goastutil.StrLit(")::"),
		)),
		Cond: greaterThanZero(goastutil.CachedIdent("castIndex")),
		Body: goastutil.BlockStmt(goastutil.AssignStmt(
			goastutil.CachedIdent(identRuntimeTrimmed),
			goastutil.CallExpr(
				goastutil.SelectorExpr(identStringsPkg, methodTrimSpace),
				sliceRange(goastutil.CachedIdent(identRuntimeTrimmed), goastutil.IntLit(1), goastutil.CachedIdent("castIndex")),
			),
		)),
	}
	return goastutil.IfStmt(
		nil,
		goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "HasPrefix"),
			goastutil.CachedIdent(identRuntimeTrimmed),
			goastutil.StrLit("("),
		),
		goastutil.BlockStmt(castIf),
	)
}

// extractJSONExtractRoot builds the `if strings.HasPrefix(strings.ToLower( trimmed),
// "json_extract(") { ... }` block that returns the leading root identifier of a
// json_extract call.
//
// Returns ast.Stmt which is the root-extraction if block.
func extractJSONExtractRoot() ast.Stmt {
	innerAssign := goastutil.DefineStmt(identRuntimeInner, sliceFrom(
		goastutil.CachedIdent(identRuntimeTrimmed),
		goastutil.CallExpr(goastutil.CachedIdent(builtinLength), goastutil.StrLit("json_extract(")),
	))
	commaIf := &ast.IfStmt{
		Init: goastutil.DefineStmt("commaIndex", goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "Index"),
			goastutil.CachedIdent(identRuntimeInner),
			goastutil.StrLit(","),
		)),
		Cond: greaterThanZero(goastutil.CachedIdent("commaIndex")),
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, methodTrimSpace),
			sliceTo(goastutil.CachedIdent(identRuntimeInner), goastutil.CachedIdent("commaIndex")),
		))),
	}
	return goastutil.IfStmt(
		nil,
		goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "HasPrefix"),
			goastutil.CallExpr(goastutil.SelectorExpr(identStringsPkg, "ToLower"), goastutil.CachedIdent(identRuntimeTrimmed)),
			goastutil.StrLit("json_extract("),
		),
		goastutil.BlockStmt(innerAssign, commaIf),
	)
}

// extractArrowRoot builds the `if arrowIndex := strings.Index(trimmed, "->"); arrowIndex
// > 0 { ... }` block that returns the root identifier of a JSON arrow path.
//
// Returns ast.Stmt which is the root-extraction if block.
func extractArrowRoot() ast.Stmt {
	return &ast.IfStmt{
		Init: goastutil.DefineStmt("arrowIndex", goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "Index"),
			goastutil.CachedIdent(identRuntimeTrimmed),
			goastutil.StrLit("->"),
		)),
		Cond: greaterThanZero(goastutil.CachedIdent("arrowIndex")),
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, methodTrimSpace),
			sliceTo(goastutil.CachedIdent(identRuntimeTrimmed), goastutil.CachedIdent("arrowIndex")),
		))),
	}
}

// buildIsUnaryOperatorFunc builds pikoIsUnaryOperator, the IS NULL and IS NOT NULL
// classifier used to skip placeholder emission for unary predicates.
//
// Returns ast.Decl which is the function declaration.
func buildIsUnaryOperatorFunc() ast.Decl {
	decl := goastutil.FuncDecl(
		"pikoIsUnaryOperator",
		goastutil.FieldList(goastutil.Field("op", goastutil.CachedIdent(IdentString))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentBool))),
		operatorClassifierBody("IS NULL", "IS NOT NULL"),
	)
	decl.Doc = docComment(
		"// pikoIsUnaryOperator reports whether op is one of the operators that take no",
		"// right-hand value (IS NULL / IS NOT NULL). The dispatcher uses this to skip",
		"// the placeholder emission and the whereArgs append for unary predicates.",
	)
	return decl
}

// buildIsMultiOperatorFunc builds pikoIsMultiOperator, the IN and NOT IN classifier used
// to fan a slice value out into per-element placeholders.
//
// Returns ast.Decl which is the function declaration.
func buildIsMultiOperatorFunc() ast.Decl {
	decl := goastutil.FuncDecl(
		"pikoIsMultiOperator",
		goastutil.FieldList(goastutil.Field("op", goastutil.CachedIdent(IdentString))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentBool))),
		operatorClassifierBody("IN", "NOT IN"),
	)
	decl.Doc = docComment(
		"// pikoIsMultiOperator reports whether op consumes a slice value (IN / NOT IN).",
		"// The dispatcher uses this to fan out each slice element into its own",
		"// placeholder so the wire form ends up as IN ($1, $2, $3) regardless of the",
		"// element count.",
	)
	return decl
}

// operatorClassifierBody builds the shared body of the operator classifiers: a switch on
// the trimmed, upper-cased op that returns true for the supplied match labels and false
// otherwise.
//
// Takes matches (...string) which are the operator labels that classify as true.
//
// Returns *ast.BlockStmt which is the classifier function body.
func operatorClassifierBody(matches ...string) *ast.BlockStmt {
	caseLabels := make([]ast.Expr, 0, len(matches))
	for _, match := range matches {
		caseLabels = append(caseLabels, goastutil.StrLit(match))
	}
	switchStmt := &ast.SwitchStmt{
		Tag: goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "ToUpper"),
			goastutil.CallExpr(goastutil.SelectorExpr(identStringsPkg, methodTrimSpace), goastutil.CachedIdent("op")),
		),
		Body: goastutil.BlockStmt(&ast.CaseClause{
			List: caseLabels,
			Body: []ast.Stmt{goastutil.ReturnStmt(goastutil.BoolIdent(true))},
		}),
	}
	return goastutil.BlockStmt(switchStmt, goastutil.ReturnStmt(goastutil.BoolIdent(false)))
}

// reflectValueOfRuntimeValue builds `rv := reflect.ValueOf(value)`, shared by the reflect
// fallback in pikoReflectSlice and pikoMultiValueLen.
//
// Returns ast.Stmt which is the reflect.ValueOf define statement.
func reflectValueOfRuntimeValue() ast.Stmt {
	return goastutil.DefineStmt(identReflectValue, goastutil.CallExpr(
		goastutil.SelectorExpr("reflect", "ValueOf"), goastutil.CachedIdent(identRuntimeValue),
	))
}

// reflectNotSliceOrArrayCond builds the condition `rv.Kind() != reflect.Slice &&
// rv.Kind() != reflect.Array`.
//
// Returns ast.Expr which is the boolean condition expression.
func reflectNotSliceOrArrayCond() ast.Expr {
	return &ast.BinaryExpr{
		X:  &ast.BinaryExpr{X: goastutil.CallExpr(goastutil.SelectorExpr(identReflectValue, "Kind")), Op: token.NEQ, Y: goastutil.SelectorExpr("reflect", "Slice")},
		Op: token.LAND,
		Y:  &ast.BinaryExpr{X: goastutil.CallExpr(goastutil.SelectorExpr(identReflectValue, "Kind")), Op: token.NEQ, Y: goastutil.SelectorExpr("reflect", "Array")},
	}
}

// reflectSliceFuncDoc returns the godoc attached to the emitted pikoReflectSlice helper.
//
// Returns *ast.CommentGroup which is the doc comment for pikoReflectSlice.
func reflectSliceFuncDoc() *ast.CommentGroup {
	return docComment(
		"// pikoReflectSlice flattens any slice / array into []any so the multi-value",
		"// expander can drive a uniform append loop. A non-slice value is wrapped in a",
		`// one-element slice so callers can still write .Where(col, "IN", "single")`,
		"// without panicking, getting the same single-placeholder behaviour as the",
		"// binary path. A []byte is treated as a single BLOB value (one placeholder),",
		"// not fanned out per byte. A nil value yields nil, treated by the caller as an",
		"// empty set (which short-circuits to the 0=1 / 1=1 sentinel).",
	)
}

// buildReflectSliceFunc builds pikoReflectSlice, which flattens an arbitrary slice or
// array value into []any so the multi-value expander drives a uniform append loop. A
// non-slice value is wrapped in a one-element slice and a nil value yields nil.
//
// Returns ast.Decl which is the function declaration.
func buildReflectSliceFunc() ast.Decl {
	nilGuard := goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{X: goastutil.CachedIdent(identRuntimeValue), Op: token.EQL, Y: goastutil.CachedIdent(IdentNil)},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(IdentNil))),
	)
	preAssertion := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent(identReflectPre), goastutil.CachedIdent(identReflectOK)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{goastutil.TypeAssertExpr(
				goastutil.CachedIdent(identRuntimeValue),
				&ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)},
			)},
		},
		Cond: goastutil.CachedIdent(identReflectOK),
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(identReflectPre))),
	}

	byteSliceAssertion := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent("blob"), goastutil.CachedIdent(identReflectOK)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{goastutil.TypeAssertExpr(
				goastutil.CachedIdent(identRuntimeValue),
				&ast.ArrayType{Elt: goastutil.CachedIdent("byte")},
			)},
		},
		Cond: goastutil.CachedIdent(identReflectOK),
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CompositeLit(
			&ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)},
			goastutil.CachedIdent("blob"),
		))),
	}
	kindGuard := goastutil.IfStmt(
		nil,
		reflectNotSliceOrArrayCond(),
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CompositeLit(
			&ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)},
			goastutil.CachedIdent(identRuntimeValue),
		))),
	)
	body := goastutil.BlockStmt(
		nilGuard,
		preAssertion,
		byteSliceAssertion,
		reflectValueOfRuntimeValue(),
		kindGuard,
		reflectSliceMakeOut(),
		reflectSliceCopyLoop(),
		goastutil.ReturnStmt(goastutil.CachedIdent("out")),
	)
	decl := goastutil.FuncDecl(
		"pikoReflectSlice",
		goastutil.FieldList(goastutil.Field(identRuntimeValue, goastutil.CachedIdent(IdentAny))),
		goastutil.FieldList(goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)})),
		body,
	)
	decl.Doc = reflectSliceFuncDoc()
	return decl
}

// buildMultiValueLenFunc builds pikoMultiValueLen, which reports how many bind values an
// IN or NOT IN value would expand to without materialising or boxing them.
//
// It mirrors the length semantics of pikoReflectSlice, where nil yields 0, []any yields
// its len, []byte yields 1 as a single BLOB, any other slice or array yields its reflect
// Len, and a non-slice yields 1, so the dispatcher can reject an over-cap list before the
// O(n) flatten-and-box pass runs. Without this a large user-influenced slice forces a
// redundant full allocation and per-element boxing only to be rejected immediately
// afterwards.
//
// Returns ast.Decl which is the function declaration.
func buildMultiValueLenFunc() ast.Decl {
	nilGuard := goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{X: goastutil.CachedIdent(identRuntimeValue), Op: token.EQL, Y: goastutil.CachedIdent(IdentNil)},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.IntLit(0))),
	)
	preAssertion := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent(identReflectPre), goastutil.CachedIdent(identReflectOK)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{goastutil.TypeAssertExpr(
				goastutil.CachedIdent(identRuntimeValue),
				&ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)},
			)},
		},
		Cond: goastutil.CachedIdent(identReflectOK),
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(
			goastutil.CallExpr(goastutil.CachedIdent(builtinLength), goastutil.CachedIdent(identReflectPre)),
		)),
	}
	byteSliceAssertion := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent("_"), goastutil.CachedIdent(identReflectOK)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{goastutil.TypeAssertExpr(
				goastutil.CachedIdent(identRuntimeValue),
				&ast.ArrayType{Elt: goastutil.CachedIdent("byte")},
			)},
		},
		Cond: goastutil.CachedIdent(identReflectOK),
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.IntLit(1))),
	}
	kindGuard := goastutil.IfStmt(
		nil,
		reflectNotSliceOrArrayCond(),
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.IntLit(1))),
	)
	body := goastutil.BlockStmt(
		nilGuard,
		preAssertion,
		byteSliceAssertion,
		reflectValueOfRuntimeValue(),
		kindGuard,
		goastutil.ReturnStmt(goastutil.CallExpr(goastutil.SelectorExpr(identReflectValue, "Len"))),
	)
	decl := goastutil.FuncDecl(
		"pikoMultiValueLen",
		goastutil.FieldList(goastutil.Field(identRuntimeValue, goastutil.CachedIdent(IdentAny))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentInt))),
		body,
	)
	decl.Doc = docComment(
		"// pikoMultiValueLen reports how many bind values an IN / NOT IN value expands to",
		"// without materialising or boxing them, mirroring pikoReflectSlice's length",
		"// semantics. The dispatcher uses it to reject an over-limit list before the",
		"// flatten-and-box pass, so a pathologically large slice is bounded at the cap",
		"// check rather than being fully allocated first.",
	)
	return decl
}

// reflectSliceMakeOut builds `out := make([]any, rv.Len())`, the destination slice the
// reflect copy loop fills.
//
// Returns ast.Stmt which is the make-and-define statement.
func reflectSliceMakeOut() ast.Stmt {
	return goastutil.DefineStmt("out", goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		&ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)},
		goastutil.CallExpr(goastutil.SelectorExpr(identReflectValue, "Len")),
	))
}

// reflectSliceCopyLoop builds the `for i := 0; i < rv.Len(); i++ { out[i] =
// rv.Index(i).Interface() }` loop that materialises each reflected element.
//
// Returns ast.Stmt which is the copy loop statement.
func reflectSliceCopyLoop() ast.Stmt {
	return &ast.ForStmt{
		Init: goastutil.DefineStmt(identRuntimeIndex, goastutil.IntLit(0)),
		Cond: &ast.BinaryExpr{X: goastutil.CachedIdent(identRuntimeIndex), Op: token.LSS, Y: goastutil.CallExpr(goastutil.SelectorExpr(identReflectValue, "Len"))},
		Post: &ast.IncDecStmt{X: goastutil.CachedIdent(identRuntimeIndex), Tok: token.INC},
		Body: goastutil.BlockStmt(goastutil.AssignStmt(
			goastutil.IndexExpr(goastutil.CachedIdent("out"), goastutil.CachedIdent(identRuntimeIndex)),
			goastutil.CallExpr(&ast.SelectorExpr{
				X:   goastutil.CallExpr(goastutil.SelectorExpr(identReflectValue, "Index"), goastutil.CachedIdent(identRuntimeIndex)),
				Sel: ast.NewIdent("Interface"),
			}),
		)),
	}
}

// buildBuildWhereFragmentFunc builds pikoBuildWhereFragment, the single dispatcher the
// generated .Where method calls after validation. On engines that ignore paramCount the
// per-element and binary increment statements collapse to a `_ = paramCount` blank
// assignment so the hot path never pays for arithmetic that does not feed the placeholder
// body.
//
// Takes useNumberedPlaceholders (bool) which selects between the `paramCount++` increment
// (numbered engines) and the `_ = paramCount` blank assignment (anonymous-marker
// engines).
//
// Returns ast.Decl which is the function declaration.
func buildBuildWhereFragmentFunc(useNumberedPlaceholders bool) ast.Decl {
	unaryGuard := goastutil.IfStmt(
		nil,
		goastutil.CallExpr(goastutil.CachedIdent("pikoIsUnaryOperator"), goastutil.CachedIdent(identRuntimeOperator)),
		goastutil.BlockStmt(goastutil.ReturnStmt(
			concatColumnOperator(),
			goastutil.CachedIdent(IdentNil),
			goastutil.IntLit(0),
			goastutil.CachedIdent(IdentNil),
		)),
	)
	multiGuard := buildMultiOperatorBranch(useNumberedPlaceholders)
	binaryReturn := goastutil.ReturnStmt(
		binaryClauseExpr(),
		goastutil.CompositeLit(&ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)}, goastutil.CachedIdent(identRuntimeValue)),
		goastutil.IntLit(1),
		goastutil.CachedIdent(IdentNil),
	)
	body := goastutil.BlockStmt(
		unaryGuard,
		multiGuard,
		paramCountIncrement(useNumberedPlaceholders),
		binaryReturn,
	)
	decl := goastutil.FuncDecl(
		"pikoBuildWhereFragment",
		buildWhereFragmentParams(),
		buildWhereFragmentResults(),
		body,
	)
	decl.Doc = buildWhereFragmentDoc()
	return decl
}

// buildWhereFragmentParams builds the `(column, operator string, value any, paramCount
// int)` parameter list of the dispatcher.
//
// Returns *ast.FieldList which is the dispatcher's parameter list.
func buildWhereFragmentParams() *ast.FieldList {
	return goastutil.FieldList(
		&ast.Field{
			Names: []*ast.Ident{goastutil.CachedIdent(identRuntimeColumn), goastutil.CachedIdent(identRuntimeOperator)},
			Type:  goastutil.CachedIdent(IdentString),
		},
		goastutil.Field(identRuntimeValue, goastutil.CachedIdent(IdentAny)),
		goastutil.Field(identRuntimeParamCount, goastutil.CachedIdent(IdentInt)),
	)
}

// buildWhereFragmentResults builds the `(string, []any, int, error)` result list of the
// dispatcher.
//
// Returns *ast.FieldList which is the dispatcher's result list.
func buildWhereFragmentResults() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field("", goastutil.CachedIdent(IdentString)),
		goastutil.Field("", &ast.ArrayType{Elt: goastutil.CachedIdent(IdentAny)}),
		goastutil.Field("", goastutil.CachedIdent(IdentInt)),
		goastutil.Field("", goastutil.CachedIdent("error")),
	)
}

// buildWhereFragmentDoc returns the godoc comment group attached to the emitted
// pikoBuildWhereFragment dispatcher.
//
// Returns *ast.CommentGroup which is the doc comment for pikoBuildWhereFragment.
func buildWhereFragmentDoc() *ast.CommentGroup {
	return docComment(
		"// pikoBuildWhereFragment is the single dispatcher the generated .Where method",
		"// calls after column / operator validation succeeds. It returns the clause to",
		"// append (with the engine's bind placeholder already in place), the arguments",
		"// to append to whereArgs in the same order, the number of parameter slots",
		"// consumed so the builder can advance parameterCount, and an error.",
		"//",
		"// The three operator classes are handled here so the per-query .Where method",
		"// stays a small validation + assignment block:",
		`//   - Unary (IS NULL, IS NOT NULL): emit "col op" with no placeholder.`,
		`//   - Multi (IN, NOT IN): expand the slice value into "col op (<ph>, <ph>, ...)"`,
		"//     where <ph> is the per-engine bind placeholder. The slice length is bounded",
		"//     by pikoMaxBindVariables; exceeding it returns errPikoTooManyBindVariables",
		"//     wrapped with the column name and the limit because the driver would",
		"//     otherwise either silently truncate or emit a hard wire-protocol error.",
		"//     The deferred error is surfaced by the All / One / Count terminal so the",
		"//     chainable .Where call itself stays panic-free.",
		"//     Empty input short-circuits to 0=1 for IN and 1=1 for NOT IN so the",
		"//     parent query keeps producing the mathematically correct result set",
		`//     without a SQL syntax error from "IN ()".`,
		`//   - Binary (everything else): emit "col op <ph>" with one placeholder.`,
		"//",
		"// Engines whose driver only accepts anonymous question-mark markers do not",
		"// consume the paramCount value when forming a placeholder, so the increment is skipped",
		"// on that branch to avoid dead arithmetic on a hot path. The returned slot",
		"// count still reflects the number of placeholders emitted because the",
		"// builder uses it to keep its parameterCount counter monotonic for the",
		"// next .Where() call.",
	)
}

// buildMultiOperatorBranch builds the `if pikoIsMultiOperator(operator) { ... }` block
// that expands an IN or NOT IN slice value into per-element placeholders,
// short-circuiting the empty case and enforcing the bind-variable cap.
//
// Takes useNumberedPlaceholders (bool) which selects numbered placeholder increments over
// the anonymous-marker form.
//
// Returns ast.Stmt which is the multi-operator if block.
func buildMultiOperatorBranch(useNumberedPlaceholders bool) ast.Stmt {
	elementsLenAssign := goastutil.DefineStmt(identRuntimeElementsLen, goastutil.CallExpr(
		goastutil.CachedIdent("pikoMultiValueLen"), goastutil.CachedIdent(identRuntimeValue),
	))
	elementsAssign := goastutil.DefineStmt(identRuntimeElements, goastutil.CallExpr(
		goastutil.CachedIdent("pikoReflectSlice"), goastutil.CachedIdent(identRuntimeValue),
	))
	placeholdersAssign := goastutil.DefineStmt("placeholders", goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		&ast.ArrayType{Elt: goastutil.CachedIdent(IdentString)},
		multiElementsLen(),
	))
	multiReturn := goastutil.ReturnStmt(
		multiClauseExpr(),
		goastutil.CachedIdent(identRuntimeElements),
		multiElementsLen(),
		goastutil.CachedIdent(IdentNil),
	)

	return goastutil.IfStmt(
		nil,
		goastutil.CallExpr(goastutil.CachedIdent("pikoIsMultiOperator"), goastutil.CachedIdent(identRuntimeOperator)),
		goastutil.BlockStmt(
			elementsLenAssign,
			multiEmptyGuard(),
			multiCapGuard(),
			elementsAssign,
			placeholdersAssign,
			multiPlaceholderLoop(useNumberedPlaceholders),
			multiReturn,
		),
	)
}

// multiElementsLen builds the `len(elements)` expression reused across the multi-operator
// branch.
//
// Returns ast.Expr which is the len(elements) call expression.
func multiElementsLen() ast.Expr {
	return goastutil.CallExpr(goastutil.CachedIdent(builtinLength), goastutil.CachedIdent(identRuntimeElements))
}

// multiEmptyGuard builds the empty-slice short-circuit: NOT IN over an empty set yields
// 1=1 (match all) and IN over an empty set yields 0=1 (match none), keeping the parent
// query syntactically valid without an empty "IN ()".
//
// Returns ast.Stmt which is the empty-slice guard if block.
func multiEmptyGuard() ast.Stmt {
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{X: goastutil.CachedIdent(identRuntimeElementsLen), Op: token.EQL, Y: goastutil.IntLit(0)},
		goastutil.BlockStmt(
			goastutil.IfStmt(
				nil,
				goastutil.CallExpr(
					goastutil.SelectorExpr(identStringsPkg, "EqualFold"),
					goastutil.CallExpr(goastutil.SelectorExpr(identStringsPkg, methodTrimSpace), goastutil.CachedIdent(identRuntimeOperator)),
					goastutil.StrLit("NOT IN"),
				),
				goastutil.BlockStmt(goastutil.ReturnStmt(
					goastutil.StrLit("1=1"), goastutil.CachedIdent(IdentNil), goastutil.IntLit(0), goastutil.CachedIdent(IdentNil),
				)),
			),
			goastutil.ReturnStmt(
				goastutil.StrLit("0=1"), goastutil.CachedIdent(IdentNil), goastutil.IntLit(0), goastutil.CachedIdent(IdentNil),
			),
		),
	)
}

// multiCapGuard builds the bind-variable cap guard that returns the wrapped
// errPikoTooManyBindVariables sentinel when the slice would exceed the limit.
//
// Returns ast.Stmt which is the cap-guard if block.
func multiCapGuard() ast.Stmt {
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{X: goastutil.CachedIdent(identRuntimeElementsLen), Op: token.GTR, Y: goastutil.CachedIdent("pikoMaxBindVariables")},
		goastutil.BlockStmt(goastutil.ReturnStmt(
			goastutil.StrLit(""),
			goastutil.CachedIdent(IdentNil),
			goastutil.IntLit(0),
			goastutil.CallExpr(
				goastutil.SelectorExpr("fmt", "Errorf"),
				goastutil.StrLit("piko: column %q IN/NOT IN list of %d exceeds the limit of %d: %w"),
				goastutil.CachedIdent(identRuntimeColumn),
				goastutil.CachedIdent(identRuntimeElementsLen),
				goastutil.CachedIdent("pikoMaxBindVariables"),
				goastutil.CachedIdent("errPikoTooManyBindVariables"),
			),
		)),
	)
}

// multiPlaceholderLoop builds the range loop that advances the parameter count (on
// numbered engines) and fills each per-element bind placeholder.
//
// Takes useNumberedPlaceholders (bool) which selects numbered placeholder increments over
// the anonymous-marker form.
//
// Returns ast.Stmt which is the placeholder-filling range loop.
func multiPlaceholderLoop(useNumberedPlaceholders bool) ast.Stmt {
	return &ast.RangeStmt{
		Key: goastutil.CachedIdent(identRuntimeIndex),
		Tok: token.DEFINE,
		X:   goastutil.CachedIdent(identRuntimeElements),
		Body: goastutil.BlockStmt(
			paramCountIncrement(useNumberedPlaceholders),
			goastutil.AssignStmt(
				goastutil.IndexExpr(goastutil.CachedIdent("placeholders"), goastutil.CachedIdent(identRuntimeIndex)),
				goastutil.CallExpr(goastutil.CachedIdent("pikoBuildBindPlaceholder"), goastutil.CachedIdent(identRuntimeParamCount)),
			),
		),
	}
}

// paramCountIncrement returns `paramCount++` for numbered engines or the `_ = paramCount`
// blank assignment for anonymous-marker engines, keeping the value live without
// arithmetic on the hot path.
//
// Takes useNumberedPlaceholders (bool) which selects the increment over the blank
// assignment.
//
// Returns ast.Stmt which is the increment or blank-assignment statement.
func paramCountIncrement(useNumberedPlaceholders bool) ast.Stmt {
	if useNumberedPlaceholders {
		return &ast.IncDecStmt{X: goastutil.CachedIdent(identRuntimeParamCount), Tok: token.INC}
	}
	return goastutil.AssignStmt(goastutil.CachedIdent("_"), goastutil.CachedIdent(identRuntimeParamCount))
}

// concatColumnOperator builds the `column + clauseSeparator + operator` expression used
// by the unary branch.
//
// Returns ast.Expr which is the concatenated clause expression.
func concatColumnOperator() ast.Expr {
	return concatExpr(
		goastutil.CachedIdent(identRuntimeColumn),
		goastutil.StrLit(clauseSeparator),
		goastutil.CachedIdent(identRuntimeOperator),
	)
}

// binaryClauseExpr builds the `column + clauseSeparator + operator + clauseSeparator +
// pikoBuildBindPlaceholder(paramCount)` expression for the binary branch.
//
// Returns ast.Expr which is the concatenated clause expression.
func binaryClauseExpr() ast.Expr {
	return concatExpr(
		goastutil.CachedIdent(identRuntimeColumn),
		goastutil.StrLit(clauseSeparator),
		goastutil.CachedIdent(identRuntimeOperator),
		goastutil.StrLit(clauseSeparator),
		goastutil.CallExpr(goastutil.CachedIdent("pikoBuildBindPlaceholder"), goastutil.CachedIdent(identRuntimeParamCount)),
	)
}

// multiClauseExpr builds the `column + clauseSeparator + operator + " (" +
// strings.Join(placeholders, ", ") + ")"` expression for the multi branch.
//
// Returns ast.Expr which is the concatenated clause expression.
func multiClauseExpr() ast.Expr {
	return concatExpr(
		goastutil.CachedIdent(identRuntimeColumn),
		goastutil.StrLit(clauseSeparator),
		goastutil.CachedIdent(identRuntimeOperator),
		goastutil.StrLit(" ("),
		goastutil.CallExpr(
			goastutil.SelectorExpr(identStringsPkg, "Join"),
			goastutil.CachedIdent("placeholders"),
			goastutil.StrLit(", "),
		),
		goastutil.StrLit(")"),
	)
}

// buildBuildBindPlaceholderFunc builds pikoBuildBindPlaceholder. Numbered engines return
// "$" + strconv.Itoa(paramCount); anonymous-marker engines discard the count via a `_ =
// paramCount` blank assignment and return the bare "?" marker.
//
// Takes useNumberedPlaceholders (bool) which selects the numbered ($N) body over the bare
// (?) body.
//
// Returns ast.Decl which is the function declaration.
func buildBuildBindPlaceholderFunc(useNumberedPlaceholders bool) ast.Decl {
	var body *ast.BlockStmt
	if useNumberedPlaceholders {
		body = goastutil.BlockStmt(goastutil.ReturnStmt(concatExpr(
			goastutil.StrLit("$"),
			goastutil.CallExpr(goastutil.SelectorExpr("strconv", "Itoa"), goastutil.CachedIdent(identRuntimeParamCount)),
		)))
	} else {
		body = goastutil.BlockStmt(
			goastutil.AssignStmt(goastutil.CachedIdent("_"), goastutil.CachedIdent(identRuntimeParamCount)),
			goastutil.ReturnStmt(goastutil.StrLit("?")),
		)
	}
	decl := goastutil.FuncDecl(
		"pikoBuildBindPlaceholder",
		goastutil.FieldList(goastutil.Field(identRuntimeParamCount, goastutil.CachedIdent(IdentInt))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentString))),
		body,
	)
	decl.Doc = docComment(
		"// pikoBuildBindPlaceholder returns the wire-protocol placeholder for a single",
		"// bind slot. Engines that bind by index ($N) substitute the count; engines",
		"// that only accept anonymous markers (?) discard the count and emit the bare",
		"// placeholder. Generated once per package so every dynamic builder shares the",
		"// same convention.",
	)
	return decl
}

// concatExpr joins the supplied expressions with the `+` binary operator into a
// left-associative chain.
//
// Takes parts (...ast.Expr) which are the expressions to join, in order.
//
// Returns ast.Expr which is the left-associative concatenation chain.
func concatExpr(parts ...ast.Expr) ast.Expr {
	expr := parts[0]
	for _, next := range parts[1:] {
		expr = &ast.BinaryExpr{X: expr, Op: token.ADD, Y: next}
	}
	return expr
}

// greaterThanZero builds the `<expr> > 0` comparison shared by the column root
// extractor's wrapper-stripping guards.
//
// Takes expr (ast.Expr) which is the left-hand operand of the comparison.
//
// Returns ast.Expr which is the `expr > 0` comparison expression.
func greaterThanZero(expr ast.Expr) ast.Expr {
	return &ast.BinaryExpr{X: expr, Op: token.GTR, Y: goastutil.IntLit(0)}
}

// sliceFrom builds the `expr[low:]` slice expression.
//
// Takes expr (ast.Expr) which is the slice operand.
// Takes low (ast.Expr) which is the low bound.
//
// Returns ast.Expr which is the `expr[low:]` slice expression.
func sliceFrom(expr, low ast.Expr) ast.Expr {
	return &ast.SliceExpr{X: expr, Low: low}
}

// sliceTo builds the `expr[:high]` slice expression.
//
// Takes expr (ast.Expr) which is the slice operand.
// Takes high (ast.Expr) which is the high bound.
//
// Returns ast.Expr which is the `expr[:high]` slice expression.
func sliceTo(expr, high ast.Expr) ast.Expr {
	return &ast.SliceExpr{X: expr, High: high}
}

// sliceRange builds the `expr[low:high]` slice expression.
//
// Takes expr (ast.Expr) which is the slice operand.
// Takes low (ast.Expr) which is the low bound.
// Takes high (ast.Expr) which is the high bound.
//
// Returns ast.Expr which is the `expr[low:high]` slice expression.
func sliceRange(expr, low, high ast.Expr) ast.Expr {
	return &ast.SliceExpr{X: expr, Low: low, High: high}
}
