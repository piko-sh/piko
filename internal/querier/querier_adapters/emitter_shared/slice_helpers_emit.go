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
	"go/ast"
	"go/token"
	"strconv"

	"piko.sh/piko/internal/goastutil"
)

const (
	// identSliceQuery is the generated name of the query string parameter.
	identSliceQuery = "query"

	// identSliceSpecs is the generated name of the expansion-spec slice parameter.
	identSliceSpecs = "specs"

	// identSliceSorted is the generated name of the sorted copy of the spec slice.
	identSliceSorted = "sorted"

	// identSliceSpec is the generated loop-variable name for a single expansion spec.
	identSliceSpec = "spec"

	// identSliceRemap is the generated name of the placeholder remap map.
	identSliceRemap = "remap"

	// identSlicePos is the generated name of the running placeholder position counter.
	identSlicePos = "pos"

	// identSliceOccurrences is the generated name of the scanned occurrence slice.
	identSliceOccurrences = "occurrences"

	// identSliceOccurrence is the generated loop-variable name for one occurrence.
	identSliceOccurrence = "occ"

	// identSliceMapping is the generated name of a single remap-map entry.
	identSliceMapping = "m"

	// identSliceBuilder is the generated name of the outer strings.Builder.
	identSliceBuilder = "b"

	// identSliceInnerBuilder is the generated name of the inner strings.Builder.
	identSliceInnerBuilder = "sb"

	// identSliceReplacement is the generated name of the per-occurrence replacement string.
	identSliceReplacement = "replacement"

	// identSliceReplStart is the generated name of the replacement start offset.
	identSliceReplStart = "replStart"

	// identSliceReplEnd is the generated name of the replacement end offset.
	identSliceReplEnd = "replEnd"

	// identSlicePrevEnd is the generated name of the previous-occurrence end offset.
	identSlicePrevEnd = "prevEnd"

	// identSliceStart is the generated name of the occurrence start offset.
	identSliceStart = "start"

	// identSliceInParens is the generated name of the parenthesised-wrapping flag.
	identSliceInParens = "inParens"

	// identSliceCount is the generated name of the per-mapping element count field.
	identSliceCount = "count"

	// identSliceIndex is the generated name of the scan-loop cursor index.
	identSliceIndex = "i"

	// identSliceNumStart is the generated name of the placeholder digit-run start offset.
	identSliceNumStart = "numStart"

	// identSliceNumber is the generated name of the parsed placeholder number.
	identSliceNumber = "n"

	// identSliceBlockClosed is the generated name of the block-comment closed flag.
	identSliceBlockClosed = "closed"

	// identBuiltinFalse is the Go false literal rendered as an identifier.
	identBuiltinFalse = "false"

	// identBuiltinTrue is the Go true literal rendered as an identifier.
	identBuiltinTrue = "true"

	// identStringsPackage is the name of the standard-library strings package.
	identStringsPackage = "strings"

	// methodWriteString is the strings.Builder WriteString method name.
	methodWriteString = "WriteString"

	// methodWriteByte is the strings.Builder WriteByte method name.
	methodWriteByte = "WriteByte"

	// builtinLen is the name of the len built-in function.
	builtinLen = "len"

	// identCompareLeft is the generated name of the left operand in the sort comparator.
	identCompareLeft = "a"

	// identCompareRight is the generated name of the right operand in the sort comparator.
	identCompareRight = "b"

	// typeSliceExpansionSpec is the generated name of the expansion-spec struct type.
	typeSliceExpansionSpec = "pikoSliceExpansionSpec"

	// bytesPerExpandedPlaceholder is the estimated byte cost of one expanded placeholder
	// slot (marker, a short ordinal, and the comma separator) used to size the
	// strings.Builder Grow hint. An over- or under-estimate only affects how many
	// reallocations the builder performs, never correctness.
	bytesPerExpandedPlaceholder = 4

	// expandedRunParensBytes accounts for the opening and closing parentheses wrapping an
	// expanded placeholder run when sizing the inner builder's Grow hint.
	expandedRunParensBytes = 2
)

// buildSliceExpansionSpecStruct builds the pikoSliceExpansionSpec type carrying the
// original placeholder index and its expanded element count.
//
// Returns ast.Decl which is the generated struct type declaration.
func buildSliceExpansionSpecStruct() ast.Decl {
	return goastutil.GenDeclType(typeSliceExpansionSpec, goastutil.StructType(
		goastutil.Field(identPlaceholderField, goastutil.CachedIdent(IdentInt)),
		goastutil.Field("Count", goastutil.CachedIdent(IdentInt)),
	))
}

// buildExpandSlicePlaceholdersFunc builds pikoExpandSlicePlaceholders, which renumbers
// the numbered `?N` placeholders in a query so a slice parameter expands into the right
// number of bind slots while keeping every other placeholder index contiguous.
//
// The running pos counter, seeded at 1 and advanced by every expanded slot, is the
// highest placeholder index assigned, so pos minus 1 is the total bind count compared
// against pikoMaxBindVariables; exceeding the cap returns errPikoTooManyBindVariables
// rather than a statement the driver would reject.
//
// Takes marker (rune) which is the engine's positional placeholder marker scanned in the
// source SQL and written into each expanded slot ('?' for sqlite, mysql, clickhouse; '$'
// for the postgres family).
// Takes useNumberedPlaceholders (bool) which selects the per-placeholder bind suffix:
// numbered engines emit strconv.Itoa(index) while anonymous-marker engines emit the empty
// string so the expansion stays valid bare-? syntax.
//
// Returns ast.Decl which is the generated function declaration.
func buildExpandSlicePlaceholdersFunc(marker rune, useNumberedPlaceholders bool) ast.Decl {
	const approximateStatementCount = 16
	statements := make([]ast.Stmt, 0, approximateStatementCount)
	statements = append(statements, sliceEmptySpecsGuard())
	statements = append(statements, sliceSortSpecs()...)
	statements = append(statements, sliceBuildRemap()...)
	statements = append(statements, sliceCapGuard())
	statements = append(statements, sliceScanOccurrences(marker)...)
	statements = append(statements, sliceNoOccurrencesGuard())
	statements = append(statements, sliceRewriteLoop(marker, useNumberedPlaceholders)...)
	statements = append(statements,
		goastutil.ExprStmt(goastutil.CallExpr(
			goastutil.SelectorExpr(identSliceBuilder, methodWriteString),
			sliceFrom(goastutil.CachedIdent(identSliceQuery), goastutil.CachedIdent(identSlicePrevEnd)),
		)),
		goastutil.ReturnStmt(
			goastutil.CallExpr(goastutil.SelectorExpr(identSliceBuilder, "String")),
			goastutil.CachedIdent(IdentNil),
		),
	)
	body := &ast.BlockStmt{List: statements}
	return goastutil.FuncDecl(
		"pikoExpandSlicePlaceholders",
		goastutil.FieldList(
			goastutil.Field(identSliceQuery, goastutil.CachedIdent(IdentString)),
			goastutil.Field(identSliceSpecs, &ast.ArrayType{Elt: goastutil.CachedIdent(typeSliceExpansionSpec)}),
		),
		goastutil.FieldList(
			goastutil.Field("", goastutil.CachedIdent(IdentString)),
			goastutil.Field("", goastutil.CachedIdent(IdentError)),
		),
		body,
	)
}

// sliceEmptySpecsGuard builds `if len(specs) == 0 { return query, nil }`.
//
// Returns ast.Stmt which is the generated guard statement.
func sliceEmptySpecsGuard() ast.Stmt {
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{
			X:  goastutil.CallExpr(goastutil.CachedIdent(builtinLen), goastutil.CachedIdent(identSliceSpecs)),
			Op: token.EQL,
			Y:  goastutil.IntLit(0),
		},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(identSliceQuery), goastutil.CachedIdent(IdentNil))),
	)
}

// sliceSortSpecs builds the copy-and-sort preamble that orders the specs by ascending
// original placeholder index before remapping.
//
// Returns []ast.Stmt which are the generated copy and sort statements.
func sliceSortSpecs() []ast.Stmt {
	makeSorted := goastutil.DefineStmt(identSliceSorted, goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		&ast.ArrayType{Elt: goastutil.CachedIdent(typeSliceExpansionSpec)},
		goastutil.CallExpr(goastutil.CachedIdent(builtinLen), goastutil.CachedIdent(identSliceSpecs)),
	))
	copyStmt := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.CachedIdent("copy"), goastutil.CachedIdent(identSliceSorted), goastutil.CachedIdent(identSliceSpecs),
	))
	cmpFunc := goastutil.FuncLit(
		goastutil.FuncType(
			goastutil.FieldList(&ast.Field{
				Names: []*ast.Ident{goastutil.CachedIdent(identCompareLeft), goastutil.CachedIdent(identCompareRight)},
				Type:  goastutil.CachedIdent(typeSliceExpansionSpec),
			}),
			goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(IdentInt))),
		),
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CallExpr(
			goastutil.SelectorExpr("cmp", "Compare"),
			goastutil.SelectorExpr(identCompareLeft, identPlaceholderField),
			goastutil.SelectorExpr(identCompareRight, identPlaceholderField),
		))),
	)
	sortStmt := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr("slices", "SortFunc"),
		goastutil.CachedIdent(identSliceSorted),
		cmpFunc,
	))
	return []ast.Stmt{makeSorted, copyStmt, sortStmt}
}

// sliceBuildRemap builds the local `mapping` type plus the remap map that maps each
// original placeholder index to its new contiguous start and element count, advancing the
// shared pos counter as it goes.
//
// Returns []ast.Stmt which are the generated type, map, and loop statements.
func sliceBuildRemap() []ast.Stmt {
	mappingType := goastutil.GenDeclType("mapping", goastutil.StructType(
		goastutil.Field("newStart", goastutil.CachedIdent(IdentInt)),
		goastutil.Field(identSliceCount, goastutil.CachedIdent(IdentInt)),
	))
	remapAssign := goastutil.DefineStmt(identSliceRemap, goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		goastutil.MapType(goastutil.CachedIdent(IdentInt), goastutil.CachedIdent("mapping")),
		goastutil.CallExpr(goastutil.CachedIdent(builtinLen), goastutil.CachedIdent(identSliceSorted)),
	))
	posAssign := goastutil.DefineStmt(identSlicePos, goastutil.IntLit(1))
	loop := &ast.RangeStmt{
		Key:   goastutil.CachedIdent("_"),
		Value: goastutil.CachedIdent(identSliceSpec),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent(identSliceSorted),
		Body: goastutil.BlockStmt(
			goastutil.AssignStmt(
				goastutil.IndexExpr(goastutil.CachedIdent(identSliceRemap), goastutil.SelectorExpr(identSliceSpec, identPlaceholderField)),
				goastutil.CompositeLit(
					goastutil.CachedIdent("mapping"),
					goastutil.KeyValueIdent("newStart", goastutil.CachedIdent(identSlicePos)),
					goastutil.KeyValueIdent(identSliceCount, goastutil.SelectorExpr(identSliceSpec, "Count")),
				),
			),
			goastutil.IfStmt(
				nil,
				greaterThanZero(goastutil.SelectorExpr(identSliceSpec, "Count")),
				goastutil.BlockStmt(&ast.AssignStmt{
					Lhs: []ast.Expr{goastutil.CachedIdent(identSlicePos)},
					Tok: token.ADD_ASSIGN,
					Rhs: []ast.Expr{goastutil.SelectorExpr(identSliceSpec, "Count")},
				}),
			),
		),
	}
	return []ast.Stmt{&ast.DeclStmt{Decl: mappingType}, remapAssign, posAssign, loop}
}

// sliceCapGuard builds the bind-variable cap guard: `if totalBindCount := pos - 1;
// totalBindCount > pikoMaxBindVariables { ... }`.
//
// Returns ast.Stmt which is the generated guard statement.
func sliceCapGuard() ast.Stmt {
	return &ast.IfStmt{
		Init: goastutil.DefineStmt("totalBindCount", &ast.BinaryExpr{
			X: goastutil.CachedIdent(identSlicePos), Op: token.SUB, Y: goastutil.IntLit(1),
		}),
		Cond: &ast.BinaryExpr{
			X: goastutil.CachedIdent("totalBindCount"), Op: token.GTR, Y: goastutil.CachedIdent("pikoMaxBindVariables"),
		},
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(
			goastutil.StrLit(""),
			goastutil.CallExpr(
				goastutil.SelectorExpr("fmt", "Errorf"),
				goastutil.StrLit("piko: expanded query of %d bind variables exceeds the limit of %d: %w"),
				goastutil.CachedIdent("totalBindCount"),
				goastutil.CachedIdent("pikoMaxBindVariables"),
				goastutil.CachedIdent("errPikoTooManyBindVariables"),
			),
		)),
	}
}

// sliceScanOccurrences builds the local `occurrence` type and the scan loop.
//
// The loop records every numbered `?N` placeholder, its byte span, its parsed number, and
// whether it is wrapped directly in parentheses. It skips single-quoted string literals
// and `--` and `/* ... */` comments before testing for placeholders so embedded `?`
// characters do not pollute the occurrence list, mirroring the offline
// PlaceholderOccurrenceOrder scanner so both paths agree on which `?N` tokens are real.
//
// Takes marker (rune) which is the placeholder marker the scan loop recognises.
//
// Returns []ast.Stmt which are the generated type, variable, and loop statements.
func sliceScanOccurrences(marker rune) []ast.Stmt {
	occurrenceType := goastutil.GenDeclType("occurrence", goastutil.StructType(
		goastutil.Field(identSliceStart, goastutil.CachedIdent(IdentInt)),
		goastutil.Field("end", goastutil.CachedIdent(IdentInt)),
		goastutil.Field("originalNum", goastutil.CachedIdent(IdentInt)),
		goastutil.Field(identSliceInParens, goastutil.CachedIdent(IdentBool)),
	))
	occurrencesVar := goastutil.VarDecl(identSliceOccurrences, &ast.ArrayType{Elt: goastutil.CachedIdent("occurrence")})
	indexInit := goastutil.DefineStmt(identSliceIndex, goastutil.IntLit(0))
	loop := &ast.ForStmt{
		Cond: &ast.BinaryExpr{
			X: goastutil.CachedIdent(identSliceIndex), Op: token.LSS, Y: goastutil.CallExpr(goastutil.CachedIdent(builtinLen), goastutil.CachedIdent(identSliceQuery)),
		},
		Body: goastutil.BlockStmt(sliceScanIfStmt(marker)),
	}
	return []ast.Stmt{&ast.DeclStmt{Decl: occurrenceType}, occurrencesVar, indexInit, loop}
}

// sliceScanIfStmt builds the body of the occurrence scan loop.
//
// Each branch inspects the byte under the cursor: a single-quoted string literal or a
// `--` and `/* ... */` comment is skipped wholesale, a numbered `?N` placeholder is
// recorded, and any other byte advances the cursor by one. The skip branches keep
// embedded `?` characters out of the occurrence list, matching the offline scanner so the
// placeholder count and the flattened argument order stay in step.
//
// Takes marker (rune) which is the placeholder marker recognised at the cursor.
//
// Returns ast.Stmt which is the generated branch statement.
func sliceScanIfStmt(marker rune) ast.Stmt {
	iPlusOne := &ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.ADD, Y: goastutil.IntLit(1)}
	placeholderCond := chainAnd(
		&ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.EQL, Y: runeLit(marker)},
		&ast.BinaryExpr{X: iPlusOne, Op: token.LSS, Y: sliceQueryLen()},
		&ast.BinaryExpr{X: sliceQueryByteAt(iPlusOne), Op: token.GEQ, Y: runeLit('1')},
		&ast.BinaryExpr{X: sliceQueryByteAt(iPlusOne), Op: token.LEQ, Y: runeLit('9')},
	)
	placeholderBlock := goastutil.BlockStmt(
		goastutil.DefineStmt(identSliceStart, goastutil.CachedIdent(identSliceIndex)),
		&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC},
		goastutil.DefineStmt(identSliceNumStart, goastutil.CachedIdent(identSliceIndex)),
		sliceDigitLoop(),
		sliceParseNumber(),
		sliceInParensAssign(),
		sliceAppendOccurrence(),
	)
	placeholderIf := &ast.IfStmt{
		Cond: placeholderCond,
		Body: placeholderBlock,
		Else: goastutil.BlockStmt(&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC}),
	}
	blockCommentIf := &ast.IfStmt{
		Cond: sliceTwoByteAt('/', '*'),
		Body: sliceSkipBlockCommentBlock(),
		Else: placeholderIf,
	}
	lineCommentIf := &ast.IfStmt{
		Cond: sliceTwoByteAt('-', '-'),
		Body: sliceSkipLineCommentBlock(),
		Else: blockCommentIf,
	}
	return &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.EQL, Y: runeLit('\'')},
		Body: sliceSkipStringLiteralBlock(),
		Else: lineCommentIf,
	}
}

// sliceTwoByteAt builds the `query[i] == first && i+1 < len(query) && query[i+1] ==
// second` guard used to recognise the two-character `--` and `/*` comment openers in the
// scan loop.
//
// Takes first (rune) which is the first byte of the two-character opener.
// Takes second (rune) which is the second byte of the two-character opener.
//
// Returns ast.Expr which is the generated guard expression.
func sliceTwoByteAt(first, second rune) ast.Expr {
	iPlusOne := &ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.ADD, Y: goastutil.IntLit(1)}
	return chainAnd(
		&ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.EQL, Y: runeLit(first)},
		&ast.BinaryExpr{X: iPlusOne, Op: token.LSS, Y: sliceQueryLen()},
		&ast.BinaryExpr{X: sliceQueryByteAt(iPlusOne), Op: token.EQL, Y: runeLit(second)},
	)
}

// sliceSkipStringLiteralBlock builds the branch that advances the cursor past a
// single-quoted string literal, treating a doubled single quote as an embedded escaped
// quote rather than the terminator. This mirrors the offline skipSQLStringLiteral helper.
//
// Returns *ast.BlockStmt which is the generated skip block.
func sliceSkipStringLiteralBlock() *ast.BlockStmt {
	closeQuoteIf := &ast.IfStmt{
		Cond: &ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.EQL, Y: runeLit('\'')},
		Body: goastutil.BlockStmt(
			&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC},
			&ast.IfStmt{
				Cond: chainAnd(
					&ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.LSS, Y: sliceQueryLen()},
					&ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.EQL, Y: runeLit('\'')},
				),
				Body: goastutil.BlockStmt(
					&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC},
					&ast.BranchStmt{Tok: token.CONTINUE},
				),
			},
			&ast.BranchStmt{Tok: token.BREAK},
		),
	}
	innerLoop := &ast.ForStmt{
		Cond: &ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.LSS, Y: sliceQueryLen()},
		Body: goastutil.BlockStmt(
			closeQuoteIf,
			&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC},
		),
	}
	return goastutil.BlockStmt(
		&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC},
		innerLoop,
	)
}

// sliceSkipLineCommentBlock builds the branch that advances the cursor to the end of a
// `--` line comment and then past the terminating newline, mirroring the offline
// skipSQLLineComment helper.
//
// Returns *ast.BlockStmt which is the generated skip block.
func sliceSkipLineCommentBlock() *ast.BlockStmt {
	scanLoop := &ast.ForStmt{
		Cond: chainAnd(
			&ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.LSS, Y: sliceQueryLen()},
			&ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.NEQ, Y: runeLit('\n')},
		),
		Body: goastutil.BlockStmt(&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC}),
	}
	advancePastNewline := goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.LSS, Y: sliceQueryLen()},
		goastutil.BlockStmt(&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC}),
	)
	return goastutil.BlockStmt(scanLoop, advancePastNewline)
}

// sliceSkipBlockCommentBlock builds the branch that advances the cursor past a `/* ...
// */` block comment.
//
// A closed flag distinguishes a found terminator from an unterminated comment, which
// jumps the cursor to the end of the query exactly as the offline skipSQLBlockComment
// helper does.
//
// Returns *ast.BlockStmt which is the generated skip block.
func sliceSkipBlockCommentBlock() *ast.BlockStmt {
	advanceOpener := &ast.AssignStmt{
		Lhs: []ast.Expr{goastutil.CachedIdent(identSliceIndex)},
		Tok: token.ADD_ASSIGN,
		Rhs: []ast.Expr{goastutil.IntLit(2)},
	}
	closedInit := goastutil.DefineStmt(identSliceBlockClosed, goastutil.CachedIdent(identBuiltinFalse))
	iPlusOne := &ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.ADD, Y: goastutil.IntLit(1)}
	terminatorIf := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X:  &ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.EQL, Y: runeLit('*')},
			Op: token.LAND,
			Y:  &ast.BinaryExpr{X: sliceQueryByteAt(iPlusOne), Op: token.EQL, Y: runeLit('/')},
		},
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent(identSliceIndex)},
				Tok: token.ADD_ASSIGN,
				Rhs: []ast.Expr{goastutil.IntLit(2)},
			},
			goastutil.AssignStmt(goastutil.CachedIdent(identSliceBlockClosed), goastutil.CachedIdent(identBuiltinTrue)),
			&ast.BranchStmt{Tok: token.BREAK},
		),
	}
	scanLoop := &ast.ForStmt{
		Cond: &ast.BinaryExpr{X: iPlusOne, Op: token.LSS, Y: sliceQueryLen()},
		Body: goastutil.BlockStmt(
			terminatorIf,
			&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC},
		),
	}
	unterminatedGuard := goastutil.IfStmt(
		nil,
		&ast.UnaryExpr{Op: token.NOT, X: goastutil.CachedIdent(identSliceBlockClosed)},
		goastutil.BlockStmt(goastutil.AssignStmt(goastutil.CachedIdent(identSliceIndex), sliceQueryLen())),
	)
	return goastutil.BlockStmt(advanceOpener, closedInit, scanLoop, unterminatedGuard)
}

// sliceQueryByteAt builds the `query[index]` byte access used throughout the placeholder
// scanner.
//
// Takes index (ast.Expr) which is the index expression to subscript with.
//
// Returns ast.Expr which is the generated byte-access expression.
func sliceQueryByteAt(index ast.Expr) ast.Expr {
	return goastutil.IndexExpr(goastutil.CachedIdent(identSliceQuery), index)
}

// sliceQueryLen builds the `len(query)` expression.
//
// Returns ast.Expr which is the generated length expression.
func sliceQueryLen() ast.Expr {
	return goastutil.CallExpr(goastutil.CachedIdent(builtinLen), goastutil.CachedIdent(identSliceQuery))
}

// sliceDigitLoop builds the `for i < len(query) && query[i] >= '0' && query[i] <= '9' {
// i++ }` loop that consumes the placeholder's digit run.
//
// Returns ast.Stmt which is the generated digit-consuming loop.
func sliceDigitLoop() ast.Stmt {
	return &ast.ForStmt{
		Cond: chainAnd(
			&ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.LSS, Y: sliceQueryLen()},
			&ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.GEQ, Y: runeLit('0')},
			&ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.LEQ, Y: runeLit('9')},
		),
		Body: goastutil.BlockStmt(&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceIndex), Tok: token.INC}),
	}
}

// sliceParseNumber builds `n, _ := strconv.Atoi(query[numStart:i])`.
//
// The Atoi error is discarded because the input is the static SQL constant emitted by
// BuildSQLConstant: its `?N` placeholder numbers are produced by the analyser and always
// fit in int, so the digit run between numStart and i is a short, well-formed decimal.
// The generated code parses its own generated SQL, never untrusted text, so an
// unparseable or overflowing number cannot arise at runtime.
//
// Returns ast.Stmt which is the generated parse statement.
func sliceParseNumber() ast.Stmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{goastutil.CachedIdent(identSliceNumber), goastutil.CachedIdent("_")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{goastutil.CallExpr(
			goastutil.SelectorExpr("strconv", "Atoi"),
			sliceRange(goastutil.CachedIdent(identSliceQuery), goastutil.CachedIdent(identSliceNumStart), goastutil.CachedIdent(identSliceIndex)),
		)},
	}
}

// sliceInParensAssign builds the `inParens := start > 0 && query[start-1] == '(' && i <
// len(query) && query[i] == ')'` assignment that records whether the placeholder is
// wrapped directly in parentheses.
//
// Returns ast.Stmt which is the generated assignment statement.
func sliceInParensAssign() ast.Stmt {
	return goastutil.DefineStmt(identSliceInParens, chainAnd(
		greaterThanZero(goastutil.CachedIdent(identSliceStart)),
		&ast.BinaryExpr{
			X:  sliceQueryByteAt(&ast.BinaryExpr{X: goastutil.CachedIdent(identSliceStart), Op: token.SUB, Y: goastutil.IntLit(1)}),
			Op: token.EQL,
			Y:  runeLit('('),
		},
		&ast.BinaryExpr{X: goastutil.CachedIdent(identSliceIndex), Op: token.LSS, Y: sliceQueryLen()},
		&ast.BinaryExpr{X: sliceQueryByteAt(goastutil.CachedIdent(identSliceIndex)), Op: token.EQL, Y: runeLit(')')},
	))
}

// sliceAppendOccurrence builds the `occurrences = append(occurrences, occurrence{...})`
// statement that records the scanned placeholder span.
//
// Returns ast.Stmt which is the generated append statement.
func sliceAppendOccurrence() ast.Stmt {
	return goastutil.AssignStmt(
		goastutil.CachedIdent(identSliceOccurrences),
		goastutil.CallExpr(
			goastutil.CachedIdent("append"),
			goastutil.CachedIdent(identSliceOccurrences),
			goastutil.CompositeLit(
				goastutil.CachedIdent("occurrence"),
				goastutil.KeyValueIdent(identSliceStart, goastutil.CachedIdent(identSliceStart)),
				goastutil.KeyValueIdent("end", goastutil.CachedIdent(identSliceIndex)),
				goastutil.KeyValueIdent("originalNum", goastutil.CachedIdent(identSliceNumber)),
				goastutil.KeyValueIdent(identSliceInParens, goastutil.CachedIdent(identSliceInParens)),
			),
		),
	)
}

// sliceNoOccurrencesGuard builds `if len(occurrences) == 0 { return query, nil }`.
//
// Returns ast.Stmt which is the generated guard statement.
func sliceNoOccurrencesGuard() ast.Stmt {
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{
			X:  goastutil.CallExpr(goastutil.CachedIdent(builtinLen), goastutil.CachedIdent(identSliceOccurrences)),
			Op: token.EQL,
			Y:  goastutil.IntLit(0),
		},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(identSliceQuery), goastutil.CachedIdent(IdentNil))),
	)
}

// sliceRewriteLoop builds the strings.Builder preamble plus the rewrite loop that
// replaces each recorded occurrence with its expanded placeholder run.
//
// Takes marker (rune) which is written ahead of each rewritten placeholder slot.
// Takes useNumberedPlaceholders (bool) which selects numbered or anonymous bind suffixes
// for the rewritten placeholders.
//
// Returns []ast.Stmt which are the generated builder and loop statements.
func sliceRewriteLoop(marker rune, useNumberedPlaceholders bool) []ast.Stmt {
	builderVar := goastutil.VarDecl(identSliceBuilder, goastutil.SelectorExpr(identStringsPackage, "Builder"))
	grow := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr(identSliceBuilder, "Grow"),
		&ast.BinaryExpr{
			X:  goastutil.CallExpr(goastutil.CachedIdent(builtinLen), goastutil.CachedIdent(identSliceQuery)),
			Op: token.ADD,
			Y: &ast.BinaryExpr{
				X:  goastutil.CallExpr(goastutil.CachedIdent(builtinLen), goastutil.CachedIdent(identSliceOccurrences)),
				Op: token.MUL,
				Y:  goastutil.IntLit(bytesPerExpandedPlaceholder),
			},
		},
	))
	prevEnd := goastutil.DefineStmt(identSlicePrevEnd, goastutil.IntLit(0))
	loop := &ast.RangeStmt{
		Key:   goastutil.CachedIdent("_"),
		Value: goastutil.CachedIdent(identSliceOccurrence),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent(identSliceOccurrences),
		Body:  sliceRewriteBody(marker, useNumberedPlaceholders),
	}
	return []ast.Stmt{builderVar, grow, prevEnd, loop}
}

// sliceRewriteBody builds the body of the rewrite loop: it looks up the remap entry,
// computes the replacement (NULL sentinel, expanded run, or single renumbered marker),
// and writes the gap plus the replacement into the builder.
//
// Takes marker (rune) which is written ahead of each rewritten placeholder slot.
// Takes useNumberedPlaceholders (bool) which selects numbered or anonymous bind suffixes
// for the rewritten placeholders.
//
// Returns *ast.BlockStmt which is the generated loop body.
func sliceRewriteBody(marker rune, useNumberedPlaceholders bool) *ast.BlockStmt {
	lookupAssign := &ast.AssignStmt{
		Lhs: []ast.Expr{goastutil.CachedIdent(identSliceMapping), goastutil.CachedIdent("ok")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{goastutil.IndexExpr(goastutil.CachedIdent(identSliceRemap), goastutil.SelectorExpr(identSliceOccurrence, "originalNum"))},
	}
	lookupGuard := goastutil.IfStmt(
		nil,
		&ast.UnaryExpr{Op: token.NOT, X: goastutil.CachedIdent("ok")},
		goastutil.BlockStmt(&ast.BranchStmt{Tok: token.CONTINUE}),
	)
	replStart := goastutil.DefineStmt(identSliceReplStart, goastutil.SelectorExpr(identSliceOccurrence, identSliceStart))
	replEnd := goastutil.DefineStmt(identSliceReplEnd, goastutil.SelectorExpr(identSliceOccurrence, "end"))
	replacementVar := goastutil.VarDecl(identSliceReplacement, goastutil.CachedIdent(IdentString))

	switchStmt := &ast.SwitchStmt{
		Body: goastutil.BlockStmt(
			sliceNullCase(),
			sliceExpandedRunCase(marker, useNumberedPlaceholders),
			sliceDefaultCase(marker, useNumberedPlaceholders),
		),
	}
	writeGap := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr(identSliceBuilder, methodWriteString),
		sliceRange(goastutil.CachedIdent(identSliceQuery), goastutil.CachedIdent(identSlicePrevEnd), goastutil.CachedIdent(identSliceReplStart)),
	))
	writeReplacement := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr(identSliceBuilder, methodWriteString), goastutil.CachedIdent(identSliceReplacement),
	))
	advance := goastutil.AssignStmt(goastutil.CachedIdent(identSlicePrevEnd), goastutil.CachedIdent(identSliceReplEnd))
	return goastutil.BlockStmt(
		lookupAssign,
		lookupGuard,
		replStart,
		replEnd,
		replacementVar,
		switchStmt,
		writeGap,
		writeReplacement,
		advance,
	)
}

// sliceNullCase builds the `case m.count == 0 && occ.inParens:` branch that replaces an
// empty parenthesised slot with the (NULL) sentinel.
//
// Returns *ast.CaseClause which is the generated case clause.
func sliceNullCase() *ast.CaseClause {
	return &ast.CaseClause{
		List: []ast.Expr{&ast.BinaryExpr{
			X:  &ast.BinaryExpr{X: goastutil.SelectorExpr(identSliceMapping, identSliceCount), Op: token.EQL, Y: goastutil.IntLit(0)},
			Op: token.LAND,
			Y:  goastutil.SelectorExpr(identSliceOccurrence, identSliceInParens),
		}},
		Body: []ast.Stmt{
			goastutil.AssignStmt(goastutil.CachedIdent(identSliceReplacement), goastutil.StrLit("(NULL)")),
			&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceReplStart), Tok: token.DEC},
			&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceReplEnd), Tok: token.INC},
		},
	}
}

// sliceExpandedRunCase builds the `case m.count > 1 && occ.inParens:` branch that
// materialises a parenthesised run of placeholders for a multi-element slice.
//
// Takes marker (rune) which is written ahead of each expanded placeholder.
// Takes useNumberedPlaceholders (bool) which selects numbered or anonymous bind suffixes
// for the expanded placeholders.
//
// Returns *ast.CaseClause which is the generated case clause.
func sliceExpandedRunCase(marker rune, useNumberedPlaceholders bool) *ast.CaseClause {
	sbVar := goastutil.VarDecl(identSliceInnerBuilder, goastutil.SelectorExpr(identStringsPackage, "Builder"))
	grow := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr(identSliceInnerBuilder, "Grow"),
		&ast.BinaryExpr{
			X:  &ast.BinaryExpr{X: goastutil.SelectorExpr(identSliceMapping, identSliceCount), Op: token.MUL, Y: goastutil.IntLit(bytesPerExpandedPlaceholder)},
			Op: token.ADD,
			Y:  goastutil.IntLit(expandedRunParensBytes),
		},
	))
	openParen := goastutil.ExprStmt(goastutil.CallExpr(goastutil.SelectorExpr(identSliceInnerBuilder, methodWriteByte), runeLit('(')))
	innerLoop := &ast.RangeStmt{
		Key: goastutil.CachedIdent("j"),
		Tok: token.DEFINE,
		X:   goastutil.SelectorExpr(identSliceMapping, identSliceCount),
		Body: goastutil.BlockStmt(
			goastutil.IfStmt(
				nil,
				greaterThanZero(goastutil.CachedIdent("j")),
				goastutil.BlockStmt(goastutil.ExprStmt(goastutil.CallExpr(goastutil.SelectorExpr(identSliceInnerBuilder, methodWriteByte), runeLit(',')))),
			),
			goastutil.ExprStmt(goastutil.CallExpr(goastutil.SelectorExpr(identSliceInnerBuilder, methodWriteByte), runeLit(marker))),
			goastutil.ExprStmt(goastutil.CallExpr(goastutil.SelectorExpr(identSliceInnerBuilder, methodWriteString), sliceLoopSuffixExpr(useNumberedPlaceholders))),
		),
	}
	closeParen := goastutil.ExprStmt(goastutil.CallExpr(goastutil.SelectorExpr(identSliceInnerBuilder, methodWriteByte), runeLit(')')))
	assignReplacement := goastutil.AssignStmt(
		goastutil.CachedIdent(identSliceReplacement),
		goastutil.CallExpr(goastutil.SelectorExpr(identSliceInnerBuilder, "String")),
	)
	return &ast.CaseClause{
		List: []ast.Expr{&ast.BinaryExpr{
			X:  &ast.BinaryExpr{X: goastutil.SelectorExpr(identSliceMapping, identSliceCount), Op: token.GTR, Y: goastutil.IntLit(1)},
			Op: token.LAND,
			Y:  goastutil.SelectorExpr(identSliceOccurrence, identSliceInParens),
		}},
		Body: []ast.Stmt{
			sbVar, grow, openParen, innerLoop, closeParen, assignReplacement,
			&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceReplStart), Tok: token.DEC},
			&ast.IncDecStmt{X: goastutil.CachedIdent(identSliceReplEnd), Tok: token.INC},
		},
	}
}

// sliceDefaultCase builds the `default:` branch that emits a single renumbered
// placeholder for the common one-slot case.
//
// Takes marker (rune) which prefixes the single emitted placeholder.
// Takes useNumberedPlaceholders (bool) which selects numbered or anonymous bind suffixes
// for the emitted placeholder.
//
// Returns *ast.CaseClause which is the generated default case clause.
func sliceDefaultCase(marker rune, useNumberedPlaceholders bool) *ast.CaseClause {
	return &ast.CaseClause{
		Body: []ast.Stmt{goastutil.AssignStmt(
			goastutil.CachedIdent(identSliceReplacement),
			concatExpr(goastutil.StrLit(string(marker)), sliceDefaultSuffixExpr(useNumberedPlaceholders)),
		)},
	}
}

// sliceLoopSuffixExpr returns the per-element bind suffix written inside the expanded
// run: strconv.Itoa(m.newStart + j) for numbered engines, or the empty string for
// anonymous-marker engines.
//
// Takes useNumberedPlaceholders (bool) which selects the numbered or empty suffix.
//
// Returns ast.Expr which is the generated suffix expression.
func sliceLoopSuffixExpr(useNumberedPlaceholders bool) ast.Expr {
	if useNumberedPlaceholders {
		return goastutil.CallExpr(
			goastutil.SelectorExpr("strconv", "Itoa"),
			&ast.BinaryExpr{X: goastutil.SelectorExpr(identSliceMapping, "newStart"), Op: token.ADD, Y: goastutil.CachedIdent("j")},
		)
	}
	return goastutil.StrLit("")
}

// sliceDefaultSuffixExpr returns the single-slot bind suffix: strconv.Itoa(m.newStart)
// for numbered engines, or the empty string for anonymous-marker engines.
//
// Takes useNumberedPlaceholders (bool) which selects the numbered or empty suffix.
//
// Returns ast.Expr which is the generated suffix expression.
func sliceDefaultSuffixExpr(useNumberedPlaceholders bool) ast.Expr {
	if useNumberedPlaceholders {
		return goastutil.CallExpr(
			goastutil.SelectorExpr("strconv", "Itoa"),
			goastutil.SelectorExpr(identSliceMapping, "newStart"),
		)
	}
	return goastutil.StrLit("")
}

// chainAnd combines the supplied conditions with the logical-and operator into a
// left-associative chain.
//
// Takes parts (...ast.Expr) which are the conditions to combine, with at least one
// element.
//
// Returns ast.Expr which is the generated left-associative and-chain.
func chainAnd(parts ...ast.Expr) ast.Expr {
	expr := parts[0]
	for _, next := range parts[1:] {
		expr = &ast.BinaryExpr{X: expr, Op: token.LAND, Y: next}
	}
	return expr
}

// runeLit builds a rune literal (a single-quoted character) for the byte comparisons in
// the placeholder scanner. strconv.QuoteRune supplies the Go escape form so characters
// such as the single quote and newline render as valid literals rather than malformed
// source.
//
// Takes r (rune) which is the character to render as a literal.
//
// Returns ast.Expr which is the generated rune literal.
func runeLit(r rune) ast.Expr {
	return &ast.BasicLit{Kind: token.CHAR, Value: strconv.QuoteRune(r)}
}
