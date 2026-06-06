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

package emitter_go_sql

import (
	"fmt"
	"go/ast"
	"go/token"

	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/querier/querier_adapters/emitter_shared"
	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// pikoClickHouseFormatFunc names the exported value formatter in the generated package.
	pikoClickHouseFormatFunc = "pikoClickHouseFormat"

	// pikoClickHouseLiteralFunc names the exported composite-literal wrapper in the
	// generated package.
	pikoClickHouseLiteralFunc = "pikoClickHouseLiteral"

	// pikoClickHouseFormatDepthFunc names the internal depth-carrying format variant.
	//
	// The exported helper delegates to it with a depth of zero; every structural descent
	// (pointer element, slice element, map key or value) recurses with depth+1 so a
	// self-referential or deeply nested input cannot recurse without bound.
	pikoClickHouseFormatDepthFunc = "pikoClickHouseFormatDepth"

	// pikoClickHouseLiteralDepthFunc names the internal depth-carrying literal variant.
	//
	// The exported helper delegates to it with a depth of zero; every structural descent
	// (pointer element, slice element, map key or value) recurses with depth+1 so a
	// self-referential or deeply nested input cannot recurse without bound.
	pikoClickHouseLiteralDepthFunc = "pikoClickHouseLiteralDepth"

	// pikoClickHouseMaxDepthConst names the emitted recursion cap. Once a value has been
	// descended into more than this many times the formatter falls back to fmt.Sprint rather
	// than recursing further, bounding the work done on a cyclic or deeply nested input.
	pikoClickHouseMaxDepthConst = "pikoClickHouseMaxDepth"

	// pikoClickHouseMaxDepthValue is the cap baked into the generated helper. A flat value
	// (the common case) descends at most once, so this leaves ample headroom for
	// legitimately nested slices and maps while still terminating quickly on a cycle.
	pikoClickHouseMaxDepthValue = 32

	// identDepth is the depth-counter parameter name on the emitted depth variants.
	identDepth = "depth"

	// clickHouseFormatHelperFile is the file name of the generated helper.
	clickHouseFormatHelperFile = "clickhouse_format.go"

	// identValue is the Go identifier used for the `value any` parameter on the emitted
	// helper functions. Centralised so a rename in the generated helper only touches one
	// place.
	identValue = "value"

	// identString is the Go identifier for the built-in `string` type.
	identString = "string"

	// identV is the local variable name used for type-switch / reflect bindings inside the
	// emitted helpers.
	identV = "v"

	// identReflectValue is the local variable name bound to reflect.ValueOf(value) in the
	// emitted format and literal helpers before their reflect.Kind switch.
	identReflectValue = "rv"

	// identUTC is the local variable holding the UTC-normalised time value in the time.Time
	// case body. Both the time-component zero check and the Format calls run against it so
	// the date-only/date-time decision and the rendered text are computed in the same (UTC)
	// location.
	identUTC = "u"

	// identS is the local variable name used for the string-form destination in the emitted
	// helpers.
	identS = "s"

	// identFmt is the Go import path / package identifier used in the emitted fmt.* calls.
	identFmt = "fmt"

	// identReflect is the Go import path / package identifier used in the emitted reflect.*
	// calls.
	identReflect = "reflect"

	// identParts is the slice variable name used to accumulate serialised slice / map
	// members in the emitted helpers.
	identParts = "parts"

	// identStrings is the Go import path / package identifier used in the emitted strings.*
	// calls.
	identStrings = "strings"

	// identOk is the second-return-value name used in the emitted type-assertion idioms.
	identOk = "ok"

	// singleQuote is the SQL single-quote character used for literal wrapping in the
	// ClickHouse output stream.
	singleQuote = "'"
)

// renderClickHouseFormatHelper builds the ClickHouse parameter formatter runtime helper
// directly as a go/ast tree and renders it via the standard emitter pipeline.
//
// The helper exposes two unexported functions in the generated package.
// pikoClickHouseFormat(value any) string converts a Go value into the string form the
// ClickHouse `{name:Type}` parameter binder expects, while pikoClickHouseLiteral(value
// any) string wraps the value for embedding inside a composite literal, single-quoting
// string-shaped values and recursing every other shape through pikoClickHouseFormat.
//
// Takes packageName (string) which is the Go package name of the caller's generated
// package.
//
// Returns querier_dto.GeneratedFile which contains the rendered helper source.
// Returns error when AST formatting fails.
func renderClickHouseFormatHelper(packageName string) (querier_dto.GeneratedFile, error) {
	tracker := emitter_shared.NewImportTracker()
	tracker.AddImport(identFmt)
	tracker.AddImport(identReflect)
	tracker.AddImport(identStrings)
	tracker.AddImport("slices")
	tracker.AddImport("time")

	declarations := []ast.Decl{
		buildClickHouseMaxDepthConst(),
		buildClickHouseFormatFunc(),
		buildClickHouseFormatDepthFunc(),
		buildClickHouseLiteralFunc(),
		buildClickHouseLiteralDepthFunc(),
	}

	content, err := emitter_shared.FormatFileWithAST(packageName, tracker, declarations)
	if err != nil {
		return querier_dto.GeneratedFile{}, fmt.Errorf("clickhouse format helper: %w", err)
	}
	return querier_dto.GeneratedFile{
		Name:    clickHouseFormatHelperFile,
		Content: content,
	}, nil
}

// buildClickHouseMaxDepthConst emits the package-level recursion cap constant used by the
// depth-carrying format and literal helpers.
//
// Returns ast.Decl holding the const declaration.
func buildClickHouseMaxDepthConst() ast.Decl {
	return &ast.GenDecl{
		Doc: commentGroup([]string{
			"// pikoClickHouseMaxDepth bounds the structural recursion the value",
			"// formatter performs. A flat value descends at most once, so this leaves",
			"// ample room for nested slices and maps while still terminating quickly on",
			"// a self-referential or pathologically deep input.",
		}),
		Tok: token.CONST,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names:  []*ast.Ident{goastutil.CachedIdent(pikoClickHouseMaxDepthConst)},
				Values: []ast.Expr{goastutil.IntLit(pikoClickHouseMaxDepthValue)},
			},
		},
	}
}

// buildClickHouseFormatFunc constructs the AST for the exported pikoClickHouseFormat.
//
// It is a thin entry point that delegates to the depth-carrying variant with a starting
// depth of zero, so pikoClickHouseFormat(value any) string returns
// pikoClickHouseFormatDepth(value, 0).
//
// Returns *ast.FuncDecl holding the complete function declaration.
func buildClickHouseFormatFunc() *ast.FuncDecl {
	body := goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CallExpr(
		goastutil.CachedIdent(pikoClickHouseFormatDepthFunc),
		goastutil.CachedIdent(identValue),
		goastutil.IntLit(0),
	)))
	decl := goastutil.FuncDecl(
		pikoClickHouseFormatFunc,
		goastutil.FieldList(goastutil.Field(identValue, goastutil.CachedIdent(emitter_shared.IdentAny))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identString))),
		body,
	)
	decl.Doc = formatFuncDoc()
	return decl
}

// buildClickHouseFormatDepthFunc constructs the AST for pikoClickHouseFormatDepth, which
// holds the real formatting logic and carries the recursion depth.
//
// A nil value renders as the empty string and a depth past the cap falls back to
// fmt.Sprint. A type switch then handles time.Time with date or date-time formatting,
// returns a string and a byte slice converted to string directly, uses String() for an
// fmt.Stringer, and for a String() (string, error) value returns the result when the
// error is nil. Failing those, a reflect.Kind switch recurses on the element of a pointer
// or interface at depth+1, converts a byte slice to string while joining other slice or
// array elements as literals, and joins map key and value pairs at depth+1, with
// fmt.Sprint as the final fallback.
//
// Returns *ast.FuncDecl holding the complete function declaration.
func buildClickHouseFormatDepthFunc() *ast.FuncDecl {
	body := goastutil.BlockStmt(
		nilGuardReturnEmpty(identValue),
		depthGuardSprint(),
		buildValueTypeSwitch(),
		assignReflectValueOf(identReflectValue, identValue),
		buildReflectKindSwitch(identReflectValue),
		goastutil.ReturnStmt(goastutil.CallExpr(
			goastutil.SelectorExpr(identFmt, "Sprint"),
			goastutil.CachedIdent(identValue),
		)),
	)
	return goastutil.FuncDecl(
		pikoClickHouseFormatDepthFunc,
		depthParams(),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identString))),
		body,
	)
}

// depthParams builds the `(value any, depth int)` parameter list shared by the two
// depth-carrying helpers.
//
// Returns *ast.FieldList which is the shared parameter list.
func depthParams() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field(identValue, goastutil.CachedIdent(emitter_shared.IdentAny)),
		goastutil.Field(identDepth, goastutil.CachedIdent("int")),
	)
}

// depthGuardSprint emits the recursion cap on the format helper, namely `if depth >
// pikoClickHouseMaxDepth { return fmt.Sprint(value) }`.
//
// Returns ast.Stmt which is the depth-guard if statement.
func depthGuardSprint() ast.Stmt {
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{
			X:  goastutil.CachedIdent(identDepth),
			Op: token.GTR,
			Y:  goastutil.CachedIdent(pikoClickHouseMaxDepthConst),
		},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CallExpr(
			goastutil.SelectorExpr(identFmt, "Sprint"),
			goastutil.CachedIdent(identValue),
		))),
	)
}

// depthGuardLiteral emits the recursion cap on the literal helper, namely `if depth >
// pikoClickHouseMaxDepth { return quoted(escape(fmt.Sprint(value))) }`. The fallback is
// the single-quoted form (unlike the format helper's bare fmt.Sprint) so a cyclic or
// deeply nested pointer/interface value still terminates as a single valid literal.
//
// Returns ast.Stmt which is the depth-guard if statement.
func depthGuardLiteral() ast.Stmt {
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{
			X:  goastutil.CachedIdent(identDepth),
			Op: token.GTR,
			Y:  goastutil.CachedIdent(pikoClickHouseMaxDepthConst),
		},
		goastutil.BlockStmt(goastutil.ReturnStmt(quotedFormattedValue())),
	)
}

// nextDepth builds the `depth + 1` expression used when recursing one structural level
// deeper.
//
// Returns ast.Expr which is the incremented depth expression.
func nextDepth() ast.Expr {
	return &ast.BinaryExpr{
		X:  goastutil.CachedIdent(identDepth),
		Op: token.ADD,
		Y:  goastutil.IntLit(1),
	}
}

// buildClickHouseLiteralFunc constructs the AST for the exported pikoClickHouseLiteral.
//
// It delegates to the depth-carrying variant with a starting depth of zero, so
// pikoClickHouseLiteral(value any) string returns pikoClickHouseLiteralDepth(value, 0).
//
// Returns *ast.FuncDecl holding the function declaration.
func buildClickHouseLiteralFunc() *ast.FuncDecl {
	body := goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CallExpr(
		goastutil.CachedIdent(pikoClickHouseLiteralDepthFunc),
		goastutil.CachedIdent(identValue),
		goastutil.IntLit(0),
	)))
	decl := goastutil.FuncDecl(
		pikoClickHouseLiteralFunc,
		goastutil.FieldList(goastutil.Field(identValue, goastutil.CachedIdent(emitter_shared.IdentAny))),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identString))),
		body,
	)
	decl.Doc = literalFuncDoc()
	return decl
}

// buildClickHouseLiteralDepthFunc constructs the AST for pikoClickHouseLiteralDepth,
// which holds the real literal-wrapping logic and carries the recursion depth.
//
// A composite element binds verbatim into the surrounding [...] / {...} that ClickHouse
// re-parses, so only values it reads bare may be emitted bare: nil renders as NULL,
// numerics and booleans render as their literal text, and nested slices/arrays/maps
// render as their own bracketed composites. Every other shape (strings, byte slices, time
// values, fmt.Stringer values, and the fmt.Sprint fallback) is single-quoted with its
// backslashes and single quotes escaped, so a value whose text contains a quote, comma,
// bracket, or backslash cannot break out of its element or smuggle extra members. The
// shape is:
//
//	func pikoClickHouseLiteralDepth(value any, depth int) string {
//	    if value == nil { return "NULL" }
//	    if s, ok := value.(string); ok { return quoted(escape(s)) }
//	    if b, ok := value.([]byte); ok { return quoted(escape(string(b))) }
//	    if t, ok := value.(time.Time); ok { return quoted(format(t)) }
//	    switch value.(type) {
//	    case bool, int, ..., complex128: return format(value) // bare
//	    }
//	    rv := reflect.ValueOf(value)
//	    switch rv.Kind() {
//	    case reflect.Ptr, reflect.Interface: // NULL when nil; else literal of Elem
//	    case reflect.Slice, reflect.Array, reflect.Map: return format(value) // bare
//	    case reflect.Bool, reflect.Int, ..., reflect.Complex128: return format(value)
//	    }
//	    return quoted(escape(format(value))) // Stringer / Sprint fallback
//	}
//
// Returns *ast.FuncDecl holding the function declaration.
func buildClickHouseLiteralDepthFunc() *ast.FuncDecl {
	body := goastutil.BlockStmt(
		nilLiteralGuard(identValue),
		quotedAssertBranch(identValue, goastutil.CachedIdent(identString), identS, identS),
		quotedAssertBranch(identValue, &ast.ArrayType{Elt: goastutil.CachedIdent("byte")}, "b",
			goastutil.CallExpr(goastutil.CachedIdent(identString), goastutil.CachedIdent("b"))),
		timeValueLiteralBranch(identValue, "t"),
		bareNumericTypeSwitch(),
		depthGuardLiteral(),
		assignReflectValueOf(identReflectValue, identValue),
		buildLiteralReflectKindSwitch(identReflectValue),
		goastutil.ReturnStmt(quotedFormattedValue()),
	)
	return goastutil.FuncDecl(
		pikoClickHouseLiteralDepthFunc,
		depthParams(),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(identString))),
		body,
	)
}

var (
	// numericKindNames lists the reflect.Kind selector names for the numeric and boolean
	// kinds a composite element may emit bare. Centralised so the concrete-type switch and
	// the reflect-kind switch stay in agreement on which shapes are unquoted.
	numericKindNames = []string{
		"Bool",
		"Int", "Int8", "Int16", "Int32", "Int64",
		"Uint", "Uint8", "Uint16", "Uint32", "Uint64", "Uintptr",
		"Float32", "Float64",
		"Complex64", "Complex128",
	}

	// numericConcreteTypes lists the concrete built-in numeric and boolean type names
	// matched by the literal helper's type switch before it falls back to reflection.
	numericConcreteTypes = []string{
		"bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64",
		"complex64", "complex128",
	}
)

// bareNumericTypeSwitch emits a `switch value.(type) { case bool, int, ...: return
// pikoClickHouseFormatDepth(value, depth) }` so a numeric or boolean composite element is
// formatted bare rather than single-quoted.
//
// Returns ast.Stmt which is the type-switch statement.
func bareNumericTypeSwitch() ast.Stmt {
	caseTypes := make([]ast.Expr, 0, len(numericConcreteTypes))
	for _, typeName := range numericConcreteTypes {
		caseTypes = append(caseTypes, goastutil.CachedIdent(typeName))
	}
	return &ast.TypeSwitchStmt{
		Assign: &ast.ExprStmt{X: &ast.TypeAssertExpr{X: goastutil.CachedIdent(identValue)}},
		Body: goastutil.BlockStmt(&ast.CaseClause{
			List: caseTypes,
			Body: []ast.Stmt{goastutil.ReturnStmt(bareFormattedValue())},
		}),
	}
}

// buildLiteralReflectKindSwitch builds the reflect.Kind switch for the literal helper.
//
// A pointer or interface unwraps to its element (NULL when nil) and re-dispatches through
// the literal helper one level deeper; a slice, array, or map renders bare as its own
// bracketed composite; a named numeric or boolean kind renders bare.
//
// Takes rvName (string) which is the reflect.Value variable name in the generated switch.
//
// Returns ast.Stmt which is the assembled reflect.Kind switch.
func buildLiteralReflectKindSwitch(rvName string) ast.Stmt {
	return &ast.SwitchStmt{
		Tag: goastutil.CallExpr(goastutil.SelectorExpr(rvName, "Kind")),
		Body: goastutil.BlockStmt(
			literalPointerInterfaceCase(rvName),
			literalContainerCase(rvName),
			literalNumericKindCase(),
		),
	}
}

// pointerInterfaceCaseFor builds the shared `case reflect.Ptr, reflect.Interface:` block,
// where a nil pointer renders as nilLiteral and any other value re-wraps the element
// through recurseFunc at depth+1. The literal and format helpers differ only in those two
// values.
//
// Takes rvName (string) which is the reflect.Value variable name in the generated switch.
// Takes nilLiteral (string) which is the rendering for a nil pointer ("NULL" or "").
// Takes recurseFunc (string) which names the depth helper to recurse through.
//
// Returns *ast.CaseClause which is the assembled case block.
func pointerInterfaceCaseFor(rvName, nilLiteral, recurseFunc string) *ast.CaseClause {
	isNil := goastutil.CallExpr(goastutil.SelectorExpr(rvName, "IsNil"))
	guard := goastutil.IfStmt(nil, isNil, goastutil.BlockStmt(
		goastutil.ReturnStmt(goastutil.StrLit(nilLiteral)),
	))
	recurse := goastutil.ReturnStmt(goastutil.CallExpr(
		goastutil.CachedIdent(recurseFunc),
		goastutil.CallExpr(&ast.SelectorExpr{
			X:   goastutil.CallExpr(goastutil.SelectorExpr(rvName, "Elem")),
			Sel: ast.NewIdent("Interface"),
		}),
		nextDepth(),
	))
	return &ast.CaseClause{
		List: []ast.Expr{
			goastutil.SelectorExpr(identReflect, "Ptr"),
			goastutil.SelectorExpr(identReflect, "Interface"),
		},
		Body: []ast.Stmt{guard, recurse},
	}
}

// literalPointerInterfaceCase builds the pointer/interface case for the literal helper. A
// nil pointer renders as NULL; otherwise the element re-wraps through it.
//
// Takes rvName (string) which is the reflect.Value variable name in the generated switch.
//
// Returns *ast.CaseClause which is the assembled case block.
func literalPointerInterfaceCase(rvName string) *ast.CaseClause {
	return pointerInterfaceCaseFor(rvName, "NULL", pikoClickHouseLiteralDepthFunc)
}

// literalContainerCase builds the `case reflect.Slice, reflect.Array, reflect.Map:` block
// for the literal helper, which delegates to the format helper so the nested value
// renders as its own bare [...] / {...} composite rather than being single-quoted.
//
// Returns *ast.CaseClause which is the assembled container case block.
func literalContainerCase(_ string) *ast.CaseClause {
	return &ast.CaseClause{
		List: []ast.Expr{
			goastutil.SelectorExpr(identReflect, "Slice"),
			goastutil.SelectorExpr(identReflect, "Array"),
			goastutil.SelectorExpr(identReflect, "Map"),
		},
		Body: []ast.Stmt{goastutil.ReturnStmt(bareFormattedValue())},
	}
}

// literalNumericKindCase builds the numeric and boolean reflect.Kind case for the literal
// helper, catching named numeric types whose concrete type the type switch did not match,
// so they too render bare.
//
// Returns *ast.CaseClause which is the assembled numeric-kind case block.
func literalNumericKindCase() *ast.CaseClause {
	caseKinds := make([]ast.Expr, 0, len(numericKindNames))
	for _, kind := range numericKindNames {
		caseKinds = append(caseKinds, goastutil.SelectorExpr(identReflect, kind))
	}
	return &ast.CaseClause{
		List: caseKinds,
		Body: []ast.Stmt{goastutil.ReturnStmt(bareFormattedValue())},
	}
}

// bareFormattedValue builds `pikoClickHouseFormatDepth(value, depth)`, the unquoted
// rendering used for numerics, booleans, and nested composites.
//
// Returns ast.Expr which is the bare format call expression.
func bareFormattedValue() ast.Expr {
	return goastutil.CallExpr(
		goastutil.CachedIdent(pikoClickHouseFormatDepthFunc),
		goastutil.CachedIdent(identValue),
		goastutil.CachedIdent(identDepth),
	)
}

// quotedFormattedValue builds `"'" + escape(pikoClickHouseFormatDepth(value, depth)) +
// "'"`, the single-quoted-and-escaped rendering used for fmt.Stringer values and the
// fmt.Sprint fallback so their text cannot break out of the composite element.
//
// Returns ast.Expr which is the quoted and escaped format expression.
func quotedFormattedValue() ast.Expr {
	return concatStrings(
		goastutil.StrLit(singleQuote),
		clickHouseEscapeLiteralExpr(bareFormattedValue()),
		goastutil.StrLit(singleQuote),
	)
}

// nilLiteralGuard emits `if <name> == nil { return "NULL" }` for use inside the composite
// literal helper, so a nil array or map element renders as the ClickHouse NULL keyword.
//
// Takes name (string) which is the identifier the nil check runs against.
//
// Returns ast.Stmt which is the nil-guard if statement.
func nilLiteralGuard(name string) ast.Stmt {
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{
			X:  goastutil.CachedIdent(name),
			Op: token.EQL,
			Y:  goastutil.CachedIdent(emitter_shared.IdentNil),
		},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.StrLit("NULL"))),
	)
}

// nilGuardReturnEmpty emits `if <name> == nil { return "" }`.
//
// Takes name (string) which is the identifier the nil check runs against.
//
// Returns ast.Stmt which is the nil-guard if statement.
func nilGuardReturnEmpty(name string) ast.Stmt {
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{
			X:  goastutil.CachedIdent(name),
			Op: token.EQL,
			Y:  goastutil.CachedIdent(emitter_shared.IdentNil),
		},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.StrLit(""))),
	)
}

// assignReflectValueOf emits `<lhs> := reflect.ValueOf(<arg>)`.
//
// Takes lhs (string) which is the identifier the reflect.Value binds to.
// Takes arg (string) which is the identifier passed to reflect.ValueOf.
//
// Returns ast.Stmt which is the assignment statement.
func assignReflectValueOf(lhs, arg string) ast.Stmt {
	return goastutil.DefineStmt(
		lhs,
		goastutil.CallExpr(
			goastutil.SelectorExpr(identReflect, "ValueOf"),
			goastutil.CachedIdent(arg),
		),
	)
}

// buildValueTypeSwitch builds the type-switch statement that dispatches on the dynamic
// type of the `value` parameter.
//
// Returns ast.Stmt which is the assembled type-switch statement.
func buildValueTypeSwitch() ast.Stmt {
	timeImport := goastutil.SelectorExpr("time", "Time")
	bytesType := &ast.ArrayType{Elt: goastutil.CachedIdent("byte")}
	fmtStringer := goastutil.SelectorExpr(identFmt, "Stringer")
	stringErrorIface := &ast.InterfaceType{
		Methods: goastutil.FieldList(goastutil.Field("String", goastutil.FuncType(
			nil,
			goastutil.FieldList(
				goastutil.Field("", goastutil.CachedIdent(identString)),
				goastutil.Field("", goastutil.CachedIdent("error")),
			),
		))),
	}

	return &ast.TypeSwitchStmt{
		Assign: &ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent(identV)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{&ast.TypeAssertExpr{X: goastutil.CachedIdent(identValue)}},
		},
		Body: goastutil.BlockStmt(
			typeSwitchCase(timeImport, timeValueCaseBody()),
			typeSwitchCase(goastutil.CachedIdent(identString), goastutil.ReturnStmt(goastutil.CachedIdent(identV))),
			typeSwitchCase(bytesType, goastutil.ReturnStmt(
				goastutil.CallExpr(goastutil.CachedIdent(identString), goastutil.CachedIdent(identV)),
			)),
			typeSwitchCase(fmtStringer, goastutil.ReturnStmt(
				goastutil.CallExpr(goastutil.SelectorExpr(identV, "String")),
			)),
			typeSwitchCase(stringErrorIface, stringErrorCaseBody()),
		),
	}
}

// typeSwitchCase wraps a single case clause of a type switch.
//
// The statements argument may be either an *ast.BlockStmt, whose statements are
// flattened, or a single statement that becomes the only entry.
//
// Takes typ (ast.Expr) which is the case type matched by the clause.
// Takes statements (ast.Stmt) which is the case body, flattened when a block.
//
// Returns *ast.CaseClause which is the assembled case clause.
func typeSwitchCase(typ ast.Expr, statements ast.Stmt) *ast.CaseClause {
	var body []ast.Stmt
	if block, ok := statements.(*ast.BlockStmt); ok {
		body = block.List
	} else {
		body = []ast.Stmt{statements}
	}
	return &ast.CaseClause{List: []ast.Expr{typ}, Body: body}
}

// timeValueCaseBody builds the `case time.Time:` body that picks either date-only or
// date-with-time formatting based on whether the time component is zero.
//
// The value is normalised to UTC once (`u := v.UTC()`) and both the time-component zero
// check and the Format calls run against u. Computing the date-only or date-time decision
// on the original (possibly non-UTC) v while formatting v.UTC() corrupts non-UTC midnight
// values: 2024-01-01 00:00:00+05:00 has zero local time components but its UTC form is
// 2023-12-31 19:00:00, so deciding on v would emit the date-only "2023-12-31" and drop
// the time-of-day.
//
// Returns *ast.BlockStmt which is the assembled time.Time case body.
func timeValueCaseBody() *ast.BlockStmt {
	assignUTC := &ast.AssignStmt{
		Lhs: []ast.Expr{goastutil.CachedIdent(identUTC)},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{goastutil.CallExpr(goastutil.SelectorExpr(identV, "UTC"))},
	}

	zeroGuard := goastutil.IfStmt(
		nil,
		goastutil.CallExpr(goastutil.SelectorExpr(identUTC, "IsZero")),
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.StrLit(""))),
	)

	allTimePartsZero := chain(token.LAND,
		eqZero(goastutil.CallExpr(goastutil.SelectorExpr(identUTC, "Hour"))),
		eqZero(goastutil.CallExpr(goastutil.SelectorExpr(identUTC, "Minute"))),
		eqZero(goastutil.CallExpr(goastutil.SelectorExpr(identUTC, "Second"))),
		eqZero(goastutil.CallExpr(goastutil.SelectorExpr(identUTC, "Nanosecond"))),
	)
	dateOnly := goastutil.IfStmt(nil, allTimePartsZero, goastutil.BlockStmt(
		goastutil.ReturnStmt(formatCall(identUTC, "2006-01-02")),
	))
	dateTime := goastutil.ReturnStmt(formatCall(identUTC, "2006-01-02 15:04:05.999999999"))
	return goastutil.BlockStmt(assignUTC, zeroGuard, dateOnly, dateTime)
}

// formatCall builds the expression `<receiver>.Format(layout)` used by the time.Time case
// branches, where receiver is the UTC-normalised value.
//
// Takes receiver (string) which is the identifier to call Format on.
// Takes layout (string) which is the Go time layout string.
//
// Returns ast.Expr which is the Format call expression.
func formatCall(receiver, layout string) ast.Expr {
	return goastutil.CallExpr(
		goastutil.SelectorExpr(receiver, "Format"),
		goastutil.StrLit(layout),
	)
}

// stringErrorCaseBody builds the `case interface{String() (string, error)}:` body that
// calls String() and returns the result when err is nil, otherwise "".
//
// Returns *ast.BlockStmt which is the assembled case body.
func stringErrorCaseBody() *ast.BlockStmt {
	callString := goastutil.CallExpr(goastutil.SelectorExpr(identV, "String"))
	innerIf := &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent(identS), goastutil.CachedIdent("err")},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{callString},
		},
		Cond: &ast.BinaryExpr{
			X:  goastutil.CachedIdent("err"),
			Op: token.EQL,
			Y:  goastutil.CachedIdent(emitter_shared.IdentNil),
		},
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CachedIdent(identS))),
	}
	return goastutil.BlockStmt(innerIf, goastutil.ReturnStmt(goastutil.StrLit("")))
}

// buildReflectKindSwitch builds the switch on reflect.Kind that recurses through pointers
// and interfaces, joins slice and array elements as a ClickHouse array literal, and joins
// map entries as a ClickHouse map literal.
//
// Takes rvName (string) which is the reflect.Value variable name in the generated switch.
//
// Returns ast.Stmt which is the assembled reflect.Kind switch.
func buildReflectKindSwitch(rvName string) ast.Stmt {
	kindExpr := goastutil.CallExpr(goastutil.SelectorExpr(rvName, "Kind"))
	return &ast.SwitchStmt{
		Tag: kindExpr,
		Body: goastutil.BlockStmt(
			pointerInterfaceCase(rvName),
			sliceArrayCase(rvName),
			mapCase(rvName),
		),
	}
}

// pointerInterfaceCase builds the `case reflect.Ptr, reflect.Interface:` block that
// returns "" for nil pointers and recurses on the element otherwise.
//
// Takes rvName (string) which is the reflect.Value variable name in the generated switch.
//
// Returns *ast.CaseClause which is the assembled case block.
func pointerInterfaceCase(rvName string) *ast.CaseClause {
	return pointerInterfaceCaseFor(rvName, "", pikoClickHouseFormatDepthFunc)
}

// sliceArrayCase builds the slice and array reflect.Kind case block.
//
// It materialises each element as a ClickHouse literal and joins the parts with commas
// inside square brackets. A named byte-slice type, one whose dynamic type does not match
// the concrete []byte case above but whose element kind is uint8, is routed through the
// same string conversion as the concrete []byte case so it serialises as text rather than
// a bracketed list of byte values.
//
// Takes rvName (string) which is the reflect.Value variable name in the generated switch.
//
// Returns *ast.CaseClause which is the assembled slice and array case block.
func sliceArrayCase(rvName string) *ast.CaseClause {
	length := goastutil.CallExpr(goastutil.SelectorExpr(rvName, "Len"))
	makeParts := goastutil.DefineStmt(identParts, goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		&ast.ArrayType{Elt: goastutil.CachedIdent(identString)},
		length,
	))
	loop := &ast.RangeStmt{
		Key: goastutil.CachedIdent("i"),
		Tok: token.DEFINE,
		X:   length,
		Body: goastutil.BlockStmt(&ast.AssignStmt{
			Lhs: []ast.Expr{
				goastutil.IndexExpr(goastutil.CachedIdent(identParts), goastutil.CachedIdent("i")),
			},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{goastutil.CallExpr(
				goastutil.CachedIdent(pikoClickHouseLiteralDepthFunc),
				goastutil.CallExpr(&ast.SelectorExpr{
					X:   goastutil.CallExpr(goastutil.SelectorExpr(rvName, "Index"), goastutil.CachedIdent("i")),
					Sel: ast.NewIdent("Interface"),
				}),
				nextDepth(),
			)},
		}),
	}
	join := goastutil.CallExpr(
		goastutil.SelectorExpr(identStrings, "Join"),
		goastutil.CachedIdent(identParts),
		goastutil.StrLit(","),
	)
	result := concatStrings(goastutil.StrLit("["), join, goastutil.StrLit("]"))
	return &ast.CaseClause{
		List: []ast.Expr{
			goastutil.SelectorExpr(identReflect, "Slice"),
			goastutil.SelectorExpr(identReflect, "Array"),
		},
		Body: []ast.Stmt{namedByteSliceGuard(rvName), makeParts, loop, goastutil.ReturnStmt(result)},
	}
}

// namedByteSliceGuard emits the guard that converts a named byte-slice value to its
// string form before the generic element-join path runs.
//
// The emitted guard returns string(rv.Bytes()) when rv.Kind() is reflect.Slice and
// rv.Type().Elem().Kind() is reflect.Uint8. reflect.Value.Bytes is valid for a slice
// whose element kind is uint8, so this mirrors the concrete []byte case which returns
// string(v).
//
// Takes rvName (string) which is the reflect.Value variable name in the generated guard.
//
// Returns ast.Stmt which is the byte-slice guard if statement.
func namedByteSliceGuard(rvName string) ast.Stmt {
	isSlice := &ast.BinaryExpr{
		X:  goastutil.CallExpr(goastutil.SelectorExpr(rvName, "Kind")),
		Op: token.EQL,
		Y:  goastutil.SelectorExpr(identReflect, "Slice"),
	}
	return goastutil.IfStmt(
		nil,
		&ast.BinaryExpr{X: isSlice, Op: token.LAND, Y: elemKindUint8(rvName)},
		goastutil.BlockStmt(goastutil.ReturnStmt(goastutil.CallExpr(
			goastutil.CachedIdent(identString),
			goastutil.CallExpr(goastutil.SelectorExpr(rvName, "Bytes")),
		))),
	)
}

// elemKindUint8 builds the `rv.Type().Elem().Kind() == reflect.Uint8` test used to detect
// a byte slice via reflection.
//
// Takes rvName (string) which is the reflect.Value variable name in the generated test.
//
// Returns ast.Expr which is the element-kind equality expression.
func elemKindUint8(rvName string) ast.Expr {
	elemKind := goastutil.CallExpr(&ast.SelectorExpr{
		X: goastutil.CallExpr(&ast.SelectorExpr{
			X:   goastutil.CallExpr(goastutil.SelectorExpr(rvName, "Type")),
			Sel: ast.NewIdent("Elem"),
		}),
		Sel: ast.NewIdent("Kind"),
	})
	return &ast.BinaryExpr{
		X:  elemKind,
		Op: token.EQL,
		Y:  goastutil.SelectorExpr(identReflect, "Uint8"),
	}
}

// mapCase builds the `case reflect.Map:` block that joins each key and value pair as a
// ClickHouse map literal inside curly braces.
//
// Takes rvName (string) which is the reflect.Value variable name in the generated switch.
//
// Returns *ast.CaseClause which is the assembled map case block.
func mapCase(rvName string) *ast.CaseClause {
	keys := goastutil.DefineStmt("keys", goastutil.CallExpr(goastutil.SelectorExpr(rvName, "MapKeys")))
	partsInit := goastutil.DefineStmt(identParts, goastutil.CallExpr(
		goastutil.CachedIdent("make"),
		&ast.ArrayType{Elt: goastutil.CachedIdent(identString)},
		goastutil.IntLit(0),
		goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent("keys")),
	))

	keyLit := goastutil.CallExpr(
		goastutil.CachedIdent(pikoClickHouseLiteralDepthFunc),
		goastutil.CallExpr(goastutil.SelectorExpr("key", "Interface")),
		nextDepth(),
	)
	valueLit := goastutil.CallExpr(
		goastutil.CachedIdent(pikoClickHouseLiteralDepthFunc),
		goastutil.CallExpr(&ast.SelectorExpr{
			X:   goastutil.CallExpr(goastutil.SelectorExpr(rvName, "MapIndex"), goastutil.CachedIdent("key")),
			Sel: ast.NewIdent("Interface"),
		}),
		nextDepth(),
	)
	colonExpr := concatStrings(keyLit, goastutil.StrLit(":"), valueLit)
	appendStmt := &ast.AssignStmt{
		Lhs: []ast.Expr{goastutil.CachedIdent(identParts)},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{goastutil.CallExpr(
			goastutil.CachedIdent("append"),
			goastutil.CachedIdent(identParts),
			colonExpr,
		)},
	}

	loop := &ast.RangeStmt{
		Key:   goastutil.CachedIdent("_"),
		Value: goastutil.CachedIdent("key"),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent("keys"),
		Body:  goastutil.BlockStmt(appendStmt),
	}

	sortStmt := &ast.ExprStmt{X: goastutil.CallExpr(
		goastutil.SelectorExpr("slices", "Sort"),
		goastutil.CachedIdent(identParts),
	)}
	join := goastutil.CallExpr(
		goastutil.SelectorExpr(identStrings, "Join"),
		goastutil.CachedIdent(identParts),
		goastutil.StrLit(","),
	)
	result := concatStrings(goastutil.StrLit("{"), join, goastutil.StrLit("}"))
	return &ast.CaseClause{
		List: []ast.Expr{goastutil.SelectorExpr(identReflect, "Map")},
		Body: []ast.Stmt{keys, partsInit, loop, sortStmt, goastutil.ReturnStmt(result)},
	}
}

// quotedAssertBranch emits a type-asserting if branch that single-quotes the asserted
// value once the assertion holds.
//
// The branch reads `if <bind>, ok := <param>.(<typ>); ok { return "'" + <quotedValueExpr>
// + "'" }`, where the value expression escapes an embedded backslash to a doubled
// backslash and then escapes embedded single quotes to doubled single quotes via nested
// strings.ReplaceAll calls, as built by clickHouseEscapeLiteralExpr.
//
// Takes param (string) which is the parameter the assertion runs on.
// Takes typ (ast.Expr) which is the asserted type.
// Takes bind (string) which is the name the asserted value binds to.
// Takes valueExpr (any) which is the value to single-quote when bind is a string,
// otherwise the conversion expression.
//
// Returns *ast.IfStmt which is the assembled type-assert branch.
func quotedAssertBranch(param string, typ ast.Expr, bind string, valueExpr any) *ast.IfStmt {
	var source ast.Expr
	switch expr := valueExpr.(type) {
	case string:
		source = goastutil.CachedIdent(expr)
	case ast.Expr:
		source = expr
	}
	inner := clickHouseEscapeLiteralExpr(source)
	return &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent(bind), goastutil.CachedIdent(identOk)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{&ast.TypeAssertExpr{X: goastutil.CachedIdent(param), Type: typ}},
		},
		Cond: goastutil.CachedIdent(identOk),
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(
			concatStrings(goastutil.StrLit(singleQuote), inner, goastutil.StrLit(singleQuote)),
		)),
	}
}

// clickHouseEscapeLiteralExpr wraps source in the nested strings.ReplaceAll calls that
// escape a value for embedding inside a single-quoted ClickHouse literal.
//
// The backslash is escaped first, since it is the ClickHouse escape introducer, so
// doubling the single quotes afterwards cannot have its own backslashes re-doubled. A
// missing backslash escape would let a value such as "a\" or "a\'" break out of the
// quoted element when ClickHouse re-parses the bound composite literal, corrupting the
// value or injecting members. This mirrors escapeClickHouseStringBody on the parser side.
//
// Takes source (ast.Expr) which is the expression whose text is escaped.
//
// Returns ast.Expr which is the nested strings.ReplaceAll escape expression.
func clickHouseEscapeLiteralExpr(source ast.Expr) ast.Expr {
	backslashEscaped := goastutil.CallExpr(
		goastutil.SelectorExpr(identStrings, "ReplaceAll"),
		source,
		goastutil.StrLit("\\"),
		goastutil.StrLit("\\\\"),
	)
	return goastutil.CallExpr(
		goastutil.SelectorExpr(identStrings, "ReplaceAll"),
		backslashEscaped,
		goastutil.StrLit(singleQuote),
		goastutil.StrLit("''"),
	)
}

// timeValueLiteralBranch emits the time.Time branch of the literal helper.
//
// It asserts the value to time.Time and, when the assertion holds, returns "'" +
// pikoClickHouseFormat(t) + "'".
//
// Takes param (string) which is the parameter the assertion runs on.
// Takes bind (string) which is the name the asserted time value binds to.
//
// Returns *ast.IfStmt which is the assembled time.Time branch.
func timeValueLiteralBranch(param, bind string) *ast.IfStmt {
	inner := goastutil.CallExpr(
		goastutil.CachedIdent(pikoClickHouseFormatDepthFunc),
		goastutil.CachedIdent(bind),
		goastutil.CachedIdent(identDepth),
	)
	return &ast.IfStmt{
		Init: &ast.AssignStmt{
			Lhs: []ast.Expr{goastutil.CachedIdent(bind), goastutil.CachedIdent(identOk)},
			Tok: token.DEFINE,
			Rhs: []ast.Expr{&ast.TypeAssertExpr{
				X:    goastutil.CachedIdent(param),
				Type: goastutil.SelectorExpr("time", "Time"),
			}},
		},
		Cond: goastutil.CachedIdent(identOk),
		Body: goastutil.BlockStmt(goastutil.ReturnStmt(
			concatStrings(goastutil.StrLit(singleQuote), inner, goastutil.StrLit(singleQuote)),
		)),
	}
}

// concatStrings joins the supplied expressions with the `+` binary operator, producing a
// left-associative chain.
//
// It composes quoted-literal strings without resorting to fmt.Sprintf.
//
// Takes parts (...ast.Expr) which are the expressions joined in order.
//
// Returns ast.Expr which is the concatenation chain, or an empty string literal when no
// parts are supplied.
func concatStrings(parts ...ast.Expr) ast.Expr {
	if len(parts) == 0 {
		return goastutil.StrLit("")
	}
	expr := parts[0]
	for _, next := range parts[1:] {
		expr = &ast.BinaryExpr{X: expr, Op: token.ADD, Y: next}
	}
	return expr
}

// chain combines the supplied expressions with the given binary operator into a
// left-associative chain.
//
// Takes op (token.Token) which is the binary operator joining the expressions.
// Takes parts (...ast.Expr) which are the expressions joined in order.
//
// Returns ast.Expr which is the operator chain, or nil when no parts are supplied.
func chain(op token.Token, parts ...ast.Expr) ast.Expr {
	if len(parts) == 0 {
		return nil
	}
	expr := parts[0]
	for _, next := range parts[1:] {
		expr = &ast.BinaryExpr{X: expr, Op: op, Y: next}
	}
	return expr
}

// eqZero builds the equality test `<expr> == 0`.
//
// Takes expr (ast.Expr) which is the left-hand operand of the equality test.
//
// Returns ast.Expr which is the equality expression.
func eqZero(expr ast.Expr) ast.Expr {
	return &ast.BinaryExpr{X: expr, Op: token.EQL, Y: goastutil.IntLit(0)}
}

// formatFuncDoc returns the godoc comment group attached to the generated
// pikoClickHouseFormat function.
//
// Returns *ast.CommentGroup which is the assembled doc comment.
func formatFuncDoc() *ast.CommentGroup {
	lines := []string{
		"// pikoClickHouseFormat converts a Go value into the string form",
		"// the ClickHouse driver expects for a {name:Type} query parameter.",
		"//",
		"// Formatting rules:",
		"//   nil values serialise as the empty string.",
		"//   time.Time serialises as YYYY-MM-DD when only the date part is",
		"//   set, otherwise YYYY-MM-DD HH:MM:SS.fraction.",
		"//   fmt.Stringer values use their String() representation.",
		"//   Values implementing String() (string, error) use the returned",
		"//   string when err is nil.",
		"//   Slices and arrays of strings serialise as ['a','b','c'].",
		"//   Slices and arrays of other types serialise as [v1,v2,v3].",
		"//   Maps serialise as {'k':v} using ClickHouse Map literal syntax.",
		"//   Any other type falls back to fmt.Sprint.",
	}
	return commentGroup(lines)
}

// literalFuncDoc returns the godoc for pikoClickHouseLiteral.
//
// Returns *ast.CommentGroup which is the assembled doc comment.
func literalFuncDoc() *ast.CommentGroup {
	lines := []string{
		"// pikoClickHouseLiteral wraps a value the way ClickHouse expects",
		"// inside a composite literal. nil renders as NULL, numerics and",
		"// booleans render bare, and nested slices/arrays/maps render as",
		"// their own bracketed composites. Every other shape - strings,",
		"// byte slices, time values, fmt.Stringer values, and the",
		"// fmt.Sprint fallback - is single-quoted with its backslashes",
		"// doubled first (the escape introducer) and then its single",
		"// quotes doubled, so its text cannot break out of the element or",
		"// smuggle extra members.",
	}
	return commentGroup(lines)
}

// commentGroup wraps a slice of comment lines into an *ast.CommentGroup.
//
// The group lets the godoc render correctly on the parent declaration. Allocating one
// *ast.Comment per line per call is acceptable because the helper participates only in
// the offline codegen path; the hot-path code emitted downstream never traverses these
// AST nodes at runtime.
//
// Takes lines ([]string) which are the comment lines to wrap, in order.
//
// Returns *ast.CommentGroup which is the assembled comment group.
func commentGroup(lines []string) *ast.CommentGroup {
	comments := make([]*ast.Comment, 0, len(lines))
	for _, line := range lines {
		comments = append(comments, &ast.Comment{Text: line})
	}
	return &ast.CommentGroup{List: comments}
}
