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
	"context"
	"go/ast"
	"go/token"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/goastutil"
)

const (
	// wrapperIdentArgsMap is the identifier for the arguments map parameter.
	wrapperIdentArgsMap = "argsMap"

	// wrapperIdentPiko is the package identifier for piko types in generated code.
	wrapperIdentPiko = "piko"

	// wrapperIdentOK is the identifier name for the boolean result of type
	// assertions.
	wrapperIdentOK = "ok"

	// goTypeString is the Go type name for string.
	goTypeString = "string"

	// goTypeInt is the Go type name for int.
	goTypeInt = "int"

	// goTypeInt64 is the Go type name for int64.
	goTypeInt64 = "int64"

	// goTypeFloat64 is the Go type name for float64.
	goTypeFloat64 = "float64"

	// goTypeBool is the Go type name for bool.
	goTypeBool = "bool"

	// blankParamName is the blank identifier used for discarded parameters.
	blankParamName = "_"
)

// ActionWrapperEmitter generates Go wrapper functions for type-safe action
// dispatch.
type ActionWrapperEmitter struct{}

// NewActionWrapperEmitter creates a new wrapper emitter.
//
// Returns *ActionWrapperEmitter which is ready for use.
func NewActionWrapperEmitter() *ActionWrapperEmitter {
	return &ActionWrapperEmitter{}
}

// EmitWrappers generates the action wrapper Go file using AST construction.
//
// Takes specs ([]annotator_dto.ActionSpec) which defines the actions to
// generate wrappers for.
//
// Returns []byte which contains the formatted Go source code.
// Returns error when AST formatting fails.
func (e *ActionWrapperEmitter) EmitWrappers(_ context.Context, specs []annotator_dto.ActionSpec) ([]byte, error) {
	fset := token.NewFileSet()
	file := e.buildWrappersAST(specs)

	needsPiko, needsMultipart := e.checkSpecialTypeImports(specs)
	needsBinder := e.checkBinderImport(specs)

	goastutil.AddImport(fset, file, actionLoggerPackagePath)
	goastutil.AddImport(fset, file, "context")
	if needsBinder {
		goastutil.AddNamedImport(fset, file, actionBinderPackageAlias, actionBinderPackagePath)
	}
	if needsPiko {
		goastutil.AddImport(fset, file, actionPikoPackagePath)
	}
	if needsMultipart {
		goastutil.AddImport(fset, file, actionMultipartPackagePath)
	}
	for i := range specs {
		spec := &specs[i]
		if actionNeedsAlias(spec.PackagePath, spec.PackageName) {
			goastutil.AddNamedImport(fset, file, spec.PackageName, spec.PackagePath)
		} else {
			goastutil.AddImport(fset, file, spec.PackagePath)
		}
	}

	return goastutil.FormatAST(fset, file)
}

// checkSpecialTypeImports checks if any action spec uses FileUpload or
// RawBody types.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action
// specifications to check.
//
// Returns needsPiko (bool) which indicates if piko imports are required.
// Returns needsMultipart (bool) which indicates if multipart imports are
// required.
func (*ActionWrapperEmitter) checkSpecialTypeImports(specs []annotator_dto.ActionSpec) (needsPiko, needsMultipart bool) {
	for i := range specs {
		spec := &specs[i]
		for _, param := range spec.CallParams {
			if param.IsFileUpload || param.IsFileUploadSlice || param.IsRawBody {
				needsPiko = true
			}
			if param.IsFileUpload || param.IsFileUploadSlice {
				needsMultipart = true
			}
		}
	}
	return needsPiko, needsMultipart
}

// checkBinderImport checks if any action spec requires the binder package for
// JSON-to-struct binding. This is needed when a parameter is a struct type or
// an unrecognised generic type.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action
// specifications to check.
//
// Returns bool which indicates if the binder import is required.
func (*ActionWrapperEmitter) checkBinderImport(specs []annotator_dto.ActionSpec) bool {
	for i := range specs {
		spec := &specs[i]
		for _, param := range spec.CallParams {
			if param.Name == blankParamName || param.IsFileUpload || param.IsFileUploadSlice || param.IsRawBody {
				continue
			}
			if param.Struct != nil {
				return true
			}
			switch param.GoType {
			case "string", "int", "int64", "float64", "bool":
				continue
			default:
				return true
			}
		}
	}
	return false
}

// specNeedsBinder checks whether a single action spec requires the binder
// package for JSON-to-struct binding.
//
// Takes spec (*annotator_dto.ActionSpec) which is the action specification to
// check.
//
// Returns bool which indicates if the spec has any parameters that need
// binder-based extraction.
func specNeedsBinder(spec *annotator_dto.ActionSpec) bool {
	for _, param := range spec.CallParams {
		if param.Name == blankParamName || param.IsFileUpload || param.IsFileUploadSlice || param.IsRawBody {
			continue
		}
		if param.Struct != nil {
			return true
		}
		switch param.GoType {
		case "string", "int", "int64", "float64", "bool":
			continue
		default:
			return true
		}
	}
	return false
}

// buildWrappersAST constructs the complete AST for the wrappers file.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action
// specifications to generate wrappers for.
//
// Returns *ast.File which is the complete AST ready for code generation.
func (e *ActionWrapperEmitter) buildWrappersAST(specs []annotator_dto.ActionSpec) *ast.File {
	sortedSpecs := actionSortSpecs(specs)

	decls := make([]ast.Decl, 0, len(sortedSpecs)+1)
	decls = append(decls, e.buildLogVarDecl())

	for i := range sortedSpecs {
		decls = append(decls, e.buildWrapperFunc(&sortedSpecs[i]))
	}

	return &ast.File{
		Name:  goastutil.CachedIdent(actionGeneratedPackageName),
		Decls: decls,
	}
}

// buildLogVarDecl builds a variable declaration AST node for the logger.
// It creates: var log = logger.GetLogger("piko/actions").
//
// Returns *ast.GenDecl which is the variable declaration node.
func (*ActionWrapperEmitter) buildLogVarDecl() *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{goastutil.CachedIdent("log")},
				Values: []ast.Expr{
					goastutil.CallExpr(
						goastutil.SelectorExpr("logger", "GetLogger"),
						goastutil.StrLit("piko/actions"),
					),
				},
			},
		},
	}
}

// buildWrapperFunc builds a single wrapper function declaration.
//
// Takes spec (*annotator_dto.ActionSpec) which defines the action to wrap.
//
// Returns *ast.FuncDecl which is the generated wrapper function AST node.
func (e *ActionWrapperEmitter) buildWrapperFunc(spec *annotator_dto.ActionSpec) *ast.FuncDecl {
	functionName := "invoke" + actionToPascalCase(spec.Name)
	pkgAlias := spec.PackageName

	statements := make([]ast.Stmt, 0)

	if specNeedsBinder(spec) {
		statements = append(statements, goastutil.DefineStmtMulti(
			[]string{"ctx", "l"},
			goastutil.CallExpr(
				goastutil.SelectorExpr("logger", "From"),
				goastutil.CachedIdent("ctx"),
				goastutil.CachedIdent("log"),
			),
		))
	}

	statements = append(statements, goastutil.DefineStmt(
		"a",
		goastutil.TypeAssertExpr(
			goastutil.CachedIdent("action"),
			goastutil.StarExpr(goastutil.SelectorExpr(pkgAlias, spec.StructName)),
		),
	))

	for _, param := range spec.CallParams {
		if param.Name == blankParamName {
			continue
		}
		paramStmts := e.buildParamExtraction(&param)
		statements = append(statements, paramStmts...)
	}

	callStmts := e.buildCallInvocation(spec)
	statements = append(statements, callStmts...)

	return goastutil.FuncDecl(
		functionName,
		goastutil.FieldList(
			goastutil.Field("ctx", goastutil.SelectorExpr("context", "Context")),
			goastutil.Field("action", goastutil.CachedIdent("any")),
			goastutil.Field(wrapperIdentArgsMap, goastutil.MapType(goastutil.CachedIdent("string"), goastutil.CachedIdent("any"))),
		),
		goastutil.FieldList(
			goastutil.Field("", goastutil.CachedIdent("any")),
			goastutil.Field("", goastutil.CachedIdent("error")),
		),
		goastutil.BlockStmt(statements...),
	)
}

// buildParamExtraction builds statements to extract a parameter from the arguments
// map.
//
// Takes param (*annotator_dto.ParamSpec) which specifies the parameter to
// extract, including its name, type, and any special handling requirements.
//
// Returns []ast.Stmt which contains the AST statements for extracting and
// converting the parameter value.
func (e *ActionWrapperEmitter) buildParamExtraction(param *annotator_dto.ParamSpec) []ast.Stmt {
	varName := param.Name
	jsonKey := param.JSONName

	if param.IsFileUpload {
		return e.buildFileUploadExtraction(varName, jsonKey)
	}
	if param.IsFileUploadSlice {
		return e.buildFileUploadSliceExtraction(varName, jsonKey)
	}
	if param.IsRawBody {
		return e.buildRawBodyExtraction(varName)
	}

	if param.Struct != nil {
		return e.buildStructParamExtraction(varName, jsonKey, param.Struct)
	}

	switch param.GoType {
	case goTypeString:
		return e.buildBasicTypeAssertion(varName, jsonKey, goTypeString)
	case goTypeInt:
		return e.buildIntConversion(varName, jsonKey, goTypeInt)
	case goTypeInt64:
		return e.buildIntConversion(varName, jsonKey, goTypeInt64)
	case goTypeFloat64:
		return e.buildBasicTypeAssertion(varName, jsonKey, goTypeFloat64)
	case goTypeBool:
		return e.buildBasicTypeAssertion(varName, jsonKey, goTypeBool)
	default:
		return e.buildGenericParamExtraction(varName, jsonKey, param.GoType)
	}
}

// buildStructParamExtraction builds statements for extracting a struct
// parameter.
//
// Takes varName (string) which specifies the variable name for the extracted
// value.
// Takes jsonKey (string) which specifies the JSON key to extract from.
// Takes typeSpec (*annotator_dto.TypeSpec) which describes the struct type
// to extract.
//
// Returns []ast.Stmt which contains the AST statements for JSON unmarshalling.
func (e *ActionWrapperEmitter) buildStructParamExtraction(varName, jsonKey string, typeSpec *annotator_dto.TypeSpec) []ast.Stmt {
	qualifiedType := wrapperQualifiedTypeName(typeSpec)
	return e.buildJSONUnmarshalExtraction(varName, jsonKey, qualifiedType)
}

// buildBasicTypeAssertion builds a type assertion statement of the form
// varName, _ := arguments["key"].(type).
//
// Takes varName (string) which is the variable name for the asserted value.
// Takes jsonKey (string) which is the key to look up in the arguments map.
// Takes typeName (string) which is the target type for the assertion.
//
// Returns []ast.Stmt which contains the type assertion assignment statement.
func (*ActionWrapperEmitter) buildBasicTypeAssertion(varName, jsonKey, typeName string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{varName, blankParamName},
			goastutil.TypeAssertExpr(
				goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit(jsonKey)),
				goastutil.CachedIdent(typeName),
			),
		),
	}
}

// buildIntConversion builds statements for int/int64 conversion from float64.
//
// Takes varName (string) which specifies the name for the converted variable.
// Takes jsonKey (string) which specifies the JSON key to extract the value
// from.
// Takes intType (string) which specifies the target integer type (int or
// int64).
//
// Returns []ast.Stmt which contains the AST statements for the conversion.
func (*ActionWrapperEmitter) buildIntConversion(varName, jsonKey, intType string) []ast.Stmt {
	rawVarName := varName + "Raw"
	return []ast.Stmt{
		goastutil.DefineStmtMulti(
			[]string{rawVarName, blankParamName},
			goastutil.TypeAssertExpr(
				goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit(jsonKey)),
				goastutil.CachedIdent(goTypeFloat64),
			),
		),
		goastutil.DefineStmt(
			varName,
			goastutil.CallExpr(goastutil.CachedIdent(intType), goastutil.CachedIdent(rawVarName)),
		),
	}
}

// buildGenericParamExtraction builds statements for generic type extraction
// using JSON.
//
// Takes varName (string) which specifies the variable name to assign.
// Takes jsonKey (string) which specifies the JSON key to extract.
// Takes goType (string) which specifies the target Go type for the value.
//
// Returns []ast.Stmt which contains the AST statements for the extraction.
func (e *ActionWrapperEmitter) buildGenericParamExtraction(varName, jsonKey, goType string) []ast.Stmt {
	return e.buildJSONUnmarshalExtraction(varName, jsonKey, goType)
}

// buildFileUploadExtraction builds statements for extracting a single
// piko.FileUpload parameter from the arguments map.
//
// Takes varName (string) which specifies the variable name for the extracted
// file upload.
// Takes jsonKey (string) which specifies the key to look up in the arguments map.
//
// Returns []ast.Stmt which contains a variable declaration and a type
// assertion if-statement that extracts the file header and creates a new
// FileUpload.
func (*ActionWrapperEmitter) buildFileUploadExtraction(varName, jsonKey string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.VarDecl(varName, goastutil.SelectorExpr(wrapperIdentPiko, "FileUpload")),
		goastutil.IfStmt(
			goastutil.DefineStmtMulti(
				[]string{"fh", wrapperIdentOK},
				goastutil.TypeAssertExpr(
					goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit(jsonKey)),
					goastutil.StarExpr(goastutil.SelectorExpr("multipart", "FileHeader")),
				),
			),
			goastutil.CachedIdent(wrapperIdentOK),
			goastutil.BlockStmt(
				goastutil.AssignStmt(
					goastutil.CachedIdent(varName),
					goastutil.CallExpr(
						goastutil.SelectorExpr(wrapperIdentPiko, "NewFileUpload"),
						goastutil.CachedIdent("fh"),
					),
				),
			),
		),
	}
}

// buildFileUploadSliceExtraction builds AST statements for extracting a
// []piko.FileUpload parameter from multipart file headers.
//
// Takes varName (string) which specifies the variable name to assign.
// Takes jsonKey (string) which specifies the key to look up in the arguments map.
//
// Returns []ast.Stmt which contains the variable declaration and conditional
// extraction logic.
func (*ActionWrapperEmitter) buildFileUploadSliceExtraction(varName, jsonKey string) []ast.Stmt {
	fileUploadSliceType := &ast.ArrayType{Elt: goastutil.SelectorExpr(wrapperIdentPiko, "FileUpload")}
	fileHeaderSliceType := &ast.ArrayType{Elt: goastutil.StarExpr(goastutil.SelectorExpr("multipart", "FileHeader"))}

	return []ast.Stmt{
		goastutil.VarDecl(varName, fileUploadSliceType),
		goastutil.IfStmt(
			goastutil.DefineStmtMulti(
				[]string{"fhs", wrapperIdentOK},
				goastutil.TypeAssertExpr(
					goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit(jsonKey)),
					fileHeaderSliceType,
				),
			),
			goastutil.CachedIdent(wrapperIdentOK),
			goastutil.BlockStmt(
				goastutil.AssignStmt(
					goastutil.CachedIdent(varName),
					goastutil.CallExpr(
						goastutil.CachedIdent("make"),
						&ast.ArrayType{Elt: goastutil.SelectorExpr(wrapperIdentPiko, "FileUpload")},
						goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent("fhs")),
					),
				),
				&ast.RangeStmt{
					Key:   goastutil.CachedIdent("i"),
					Value: goastutil.CachedIdent("fh"),
					Tok:   token.DEFINE,
					X:     goastutil.CachedIdent("fhs"),
					Body: goastutil.BlockStmt(
						goastutil.AssignStmt(
							goastutil.IndexExpr(goastutil.CachedIdent(varName), goastutil.CachedIdent("i")),
							goastutil.CallExpr(
								goastutil.SelectorExpr(wrapperIdentPiko, "NewFileUpload"),
								goastutil.CachedIdent("fh"),
							),
						),
					),
				},
			),
		),
	}
}

// buildRawBodyExtraction builds statements for extracting a piko.RawBody
// parameter from the arguments map using a type assertion.
//
// Takes varName (string) which is the name of the variable to assign the
// extracted RawBody value to.
//
// Returns []ast.Stmt which contains the variable declaration and conditional
// assignment statements.
//
// Note: RawBody is injected by the handler under the special key "_rawBody".
func (*ActionWrapperEmitter) buildRawBodyExtraction(varName string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.VarDecl(varName, goastutil.SelectorExpr(wrapperIdentPiko, "RawBody")),
		goastutil.IfStmt(
			goastutil.DefineStmtMulti(
				[]string{"rb", wrapperIdentOK},
				goastutil.TypeAssertExpr(
					goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit("_rawBody")),
					goastutil.SelectorExpr(wrapperIdentPiko, "RawBody"),
				),
			),
			goastutil.CachedIdent(wrapperIdentOK),
			goastutil.BlockStmt(
				goastutil.AssignStmt(
					goastutil.CachedIdent(varName),
					goastutil.CachedIdent("rb"),
				),
			),
		),
	}
}

// buildJSONUnmarshalExtraction builds binder-based extraction with error
// handling. Generates code that handles both nested JSON ({"input": {...}})
// and flat form data ({...}) using pikobinder.BindMap.
//
// Takes varName (string) which specifies the variable name to store the result.
// Takes jsonKey (string) which specifies the JSON key to extract.
// Takes typeName (string) which specifies the Go type for binding.
//
// Returns []ast.Stmt which contains the generated AST statements.
func (*ActionWrapperEmitter) buildJSONUnmarshalExtraction(varName, jsonKey, typeName string) []ast.Stmt {
	typeExpr := parseTypeExpr(typeName)

	return []ast.Stmt{
		goastutil.VarDecl(varName, typeExpr),
		&ast.IfStmt{
			Init: goastutil.DefineStmtMulti(
				[]string{"raw", wrapperIdentOK},
				goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit(jsonKey)),
			),
			Cond: goastutil.CachedIdent(wrapperIdentOK),
			Body: buildNestedBindBlock(varName, jsonKey),
			Else: &ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(wrapperIdentArgsMap)),
					Op: token.GTR,
					Y:  goastutil.IntLit(0),
				},
				Body: buildBindMapBlock(goastutil.CachedIdent(wrapperIdentArgsMap), varName, jsonKey, "Failed to bind action parameter from flat argsMap"),
			},
		},
	}
}

// buildCallInvocation builds the action Call method invocation statements.
//
// Takes spec (*annotator_dto.ActionSpec) which provides the action
// specification including call parameters and error handling details.
//
// Returns []ast.Stmt which contains the AST statements for invoking the Call
// method, with appropriate return handling based on whether errors are used.
func (*ActionWrapperEmitter) buildCallInvocation(spec *annotator_dto.ActionSpec) []ast.Stmt {
	arguments := make([]ast.Expr, 0, len(spec.CallParams))
	for _, param := range spec.CallParams {
		if param.Name == blankParamName {
			arguments = append(arguments, buildZeroValueExpr(&param))
			continue
		}
		argExpr := goastutil.CachedIdent(param.Name)
		if param.Optional {
			arguments = append(arguments, goastutil.AddressExpr(argExpr))
		} else {
			arguments = append(arguments, argExpr)
		}
	}

	callExpr := goastutil.CallExpr(
		goastutil.SelectorExprFrom(goastutil.CachedIdent("a"), "Call"),
		arguments...,
	)

	if spec.HasError {
		return []ast.Stmt{goastutil.ReturnStmt(callExpr)}
	}

	return []ast.Stmt{
		goastutil.DefineStmt("result", callExpr),
		goastutil.ReturnStmt(goastutil.CachedIdent("result"), goastutil.CachedIdent("nil")),
	}
}

// parseTypeExpr parses a type name string into an AST expression.
//
// Takes typeName (string) which is the type name to parse.
//
// Returns ast.Expr which is the parsed AST expression.
func parseTypeExpr(typeName string) ast.Expr {
	if strings.Contains(typeName, ".") {
		parts := strings.SplitN(typeName, ".", 2)
		return goastutil.SelectorExpr(parts[0], parts[1])
	}
	return goastutil.CachedIdent(typeName)
}

// buildNestedBindBlock builds the AST for the nested key extraction path.
// It generates a type assertion from any to map[string]any, then calls
// pikobinder.BindMap.
//
// Takes varName (string) which is the name of the variable to bind into.
// Takes jsonKey (string) which is the JSON key name for error logging.
//
// Returns *ast.BlockStmt which contains the type assertion and bind call.
func buildNestedBindBlock(varName, jsonKey string) *ast.BlockStmt {
	return goastutil.BlockStmt(
		&ast.IfStmt{
			Init: goastutil.DefineStmtMulti(
				[]string{"rawMap", wrapperIdentOK},
				goastutil.TypeAssertExpr(
					goastutil.CachedIdent("raw"),
					goastutil.MapType(goastutil.CachedIdent("string"), goastutil.CachedIdent("any")),
				),
			),
			Cond: goastutil.CachedIdent(wrapperIdentOK),
			Body: buildBindMapBlock(goastutil.CachedIdent("rawMap"), varName, jsonKey, "Failed to bind action parameter"),
		},
	)
}

// buildBindMapBlock builds the AST for binding a map[string]any to a struct
// using pikobinder.BindMap with IgnoreUnknownKeys(true).
//
// Takes sourceExpression (ast.Expr) which is the map expression to bind from.
// Takes varName (string) which is the name of the variable to bind into.
// Takes jsonKey (string) which is the JSON key name for error logging.
// Takes errorContext (string) which is the context message for error logging.
//
// Returns *ast.BlockStmt which contains the bind call and error handling.
func buildBindMapBlock(sourceExpression ast.Expr, varName, jsonKey, errorContext string) *ast.BlockStmt {
	return goastutil.BlockStmt(
		&ast.IfStmt{
			Init: goastutil.DefineStmt(
				"err",
				goastutil.CallExpr(
					goastutil.SelectorExpr(actionBinderPackageAlias, "BindMap"),
					goastutil.CachedIdent("ctx"),
					goastutil.AddressExpr(goastutil.CachedIdent(varName)),
					sourceExpression,
					goastutil.CallExpr(
						goastutil.SelectorExpr(actionBinderPackageAlias, "IgnoreUnknownKeys"),
						goastutil.CachedIdent("true"),
					),
				),
			),
			Cond: &ast.BinaryExpr{
				X:  goastutil.CachedIdent("err"),
				Op: token.NEQ,
				Y:  goastutil.CachedIdent("nil"),
			},
			Body: goastutil.BlockStmt(
				goastutil.ExprStmt(
					goastutil.CallExpr(
						goastutil.SelectorExpr("l", "Error"),
						goastutil.StrLit(errorContext),
						goastutil.CallExpr(goastutil.SelectorExpr("logger", "String"), goastutil.StrLit("param"), goastutil.StrLit(jsonKey)),
						goastutil.CallExpr(goastutil.SelectorExpr("logger", "Error"), goastutil.CachedIdent("err")),
					),
				),
				goastutil.ReturnStmt(goastutil.CachedIdent("nil"), goastutil.CachedIdent("err")),
			),
		},
	)
}

// buildZeroValueExpr builds the zero-value expression for a parameter type.
// This is used for blank identifier (_) parameters that are positionally
// required in the Call signature but carry no meaningful data.
//
// Takes param (*annotator_dto.ParamSpec) which describes the parameter type.
//
// Returns ast.Expr which is the zero-value expression for the type.
func buildZeroValueExpr(param *annotator_dto.ParamSpec) ast.Expr {
	if param.Optional {
		return goastutil.CachedIdent("nil")
	}

	switch param.GoType {
	case goTypeString:
		return goastutil.StrLit("")
	case goTypeInt, goTypeInt64, goTypeFloat64:
		return goastutil.IntLit(0)
	case goTypeBool:
		return goastutil.CachedIdent("false")
	default:
		typeExpr := parseTypeExpr(param.GoType)
		if param.Struct != nil {
			typeExpr = parseTypeExpr(wrapperQualifiedTypeName(param.Struct))
		}
		return goastutil.CompositeLit(typeExpr)
	}
}

// wrapperQualifiedTypeName returns the package-qualified type name for a
// struct type.
//
// Takes typeSpec (*annotator_dto.TypeSpec) which specifies the type to
// format.
//
// Returns string which is the qualified name in "package.TypeName" format, or
// empty if typeSpec is nil.
func wrapperQualifiedTypeName(typeSpec *annotator_dto.TypeSpec) string {
	if typeSpec == nil {
		return ""
	}
	packageName := typeSpec.PackageName
	if packageName == "" {
		parts := strings.Split(typeSpec.PackagePath, "/")
		packageName = parts[len(parts)-1]
	}
	return packageName + "." + typeSpec.Name
}
