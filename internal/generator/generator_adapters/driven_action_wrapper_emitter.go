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

	// wrapperIdentOK is the identifier name for the boolean result of type assertions.
	wrapperIdentOK = "ok"

	// wrapperIdentLogger is the package identifier for the logger in generated code.
	wrapperIdentLogger = "logger"

	// actionLogVarName is the package-level logger variable the wrapper file declares. It is
	// deliberately not "log": that is a plausible name for a user action package, and the
	// variable and the import of such a package would then redeclare one another.
	actionLogVarName = "pikoActionLog"

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

// bindReturnStyle selects the return statement a generated bind failure branch emits, so
// the same parameter extraction serves both the two-result invoke wrapper and the
// error-only bind function.
type bindReturnStyle int

const (
	// bindReturnValueAndError emits "return nil, err" for the invoke wrapper, whose
	// signature returns a result alongside its error.
	bindReturnValueAndError bindReturnStyle = iota

	// bindReturnErrorOnly emits "return err" for the bind function, whose only result is an
	// error.
	bindReturnErrorOnly
)

var (
	// wrapperBodyLocalNames lists every identifier a generated wrapper or bind body declares
	// for itself. A call parameter named after one of these would shadow it, so each
	// parameter claims a name against this set rather than using the user's spelling.
	wrapperBodyLocalNames = []string{
		"a", "action", "argsMap", "ctx", "err", "fh", "fhs", "fu", "i", "l", "ok", "raw",
		"rawMap", "rb", "result",
	}
)

// actionParamLocals derives the local variable name each call parameter is extracted
// into.
//
// Takes spec (*annotator_dto.ActionSpec) which holds the call parameters.
//
// Returns []string which holds one local name per parameter, empty for a blank parameter.
func actionParamLocals(spec *annotator_dto.ActionSpec) []string {
	used := make(map[string]struct{}, len(wrapperBodyLocalNames)+len(spec.CallParams))
	for _, name := range wrapperBodyLocalNames {
		used[name] = struct{}{}
	}

	locals := make([]string, len(spec.CallParams))
	for i := range spec.CallParams {
		if spec.CallParams[i].Name == blankParamName {
			continue
		}
		sanitised := goastutil.SanitiseGoIdentifier(spec.CallParams[i].Name)
		locals[i] = goastutil.ReserveIdentifier(sanitised, used)
	}

	return locals
}

// returnStmt builds the return statement a bind failure branch ends with.
//
// Returns ast.Stmt which returns the bind error in this style's result shape.
func (style bindReturnStyle) returnStmt() ast.Stmt {
	if style == bindReturnErrorOnly {
		return goastutil.ReturnStmt(goastutil.ErrIdent())
	}
	return goastutil.ReturnStmt(goastutil.NilIdent(), goastutil.ErrIdent())
}

// ActionWrapperEmitter generates Go wrapper functions for type-safe action dispatch.
type ActionWrapperEmitter struct{}

// NewActionWrapperEmitter creates a new wrapper emitter.
//
// Returns *ActionWrapperEmitter which is ready for use.
func NewActionWrapperEmitter() *ActionWrapperEmitter {
	return &ActionWrapperEmitter{}
}

// EmitWrappers generates the action wrapper Go file using AST construction.
//
// Takes specs ([]annotator_dto.ActionSpec) which defines the actions to generate wrappers
// for.
//
// Returns []byte which contains the formatted Go source code.
// Returns error when AST formatting fails.
func (e *ActionWrapperEmitter) EmitWrappers(_ context.Context, specs []annotator_dto.ActionSpec) ([]byte, error) {
	fset := token.NewFileSet()
	naming := newActionNaming(specs)
	file := e.buildWrappersAST(&naming)

	needsPiko, needsMultipart := e.checkSpecialTypeImports(specs)
	needsBinder := e.checkBinderImport(specs)

	goastutil.AddImport(fset, file, actionLoggerPackagePath)
	goastutil.AddImport(fset, file, actionContextPackagePath)
	if needsBinder {
		goastutil.AddNamedImport(fset, file, actionBinderPackageAlias, actionBinderPackagePath)
	}
	if needsPiko {
		goastutil.AddImport(fset, file, actionPikoPackagePath)
	}
	if needsMultipart {
		goastutil.AddImport(fset, file, actionMultipartPackagePath)
	}
	naming.addPackageImports(fset, file)

	return goastutil.FormatAST(fset, file)
}

// checkSpecialTypeImports checks if any action spec uses FileUpload or RawBody types.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action specifications to
// check.
//
// Returns needsPiko (bool) which indicates if piko imports are required.
// Returns needsMultipart (bool) which indicates if multipart imports are required.
func (*ActionWrapperEmitter) checkSpecialTypeImports(specs []annotator_dto.ActionSpec) (needsPiko, needsMultipart bool) {
	for i := range specs {
		specPiko, specMultipart := actionSpecialTypeImports(&specs[i])
		needsPiko = needsPiko || specPiko
		needsMultipart = needsMultipart || specMultipart
	}

	return needsPiko, needsMultipart
}

// actionSpecialTypeImports reports the imports one action's signature calls for.
//
// A streaming action needs piko, because its generated bind function records the bound
// arguments through piko.SetActionInput. A file upload needs piko for its helper types
// and multipart for the header type, whether the upload is a parameter of its own or a
// field of a struct parameter.
//
// Takes spec (*annotator_dto.ActionSpec) which is the action to inspect.
//
// Returns bool which is true when the generated file must import piko.
// Returns bool which is true when it must import mime/multipart.
func actionSpecialTypeImports(spec *annotator_dto.ActionSpec) (needsPiko, needsMultipart bool) {
	needsPiko = spec.HasSSE

	for i := range spec.CallParams {
		param := &spec.CallParams[i]

		switch {
		case param.IsFileUpload, param.IsFileUploadSlice:
			needsPiko, needsMultipart = true, true
		case param.IsRawBody:
			needsPiko = true
		}

		if structSpecHasFileUpload(param.Struct) {
			needsPiko, needsMultipart = true, true
		}
	}

	return needsPiko, needsMultipart
}

// checkBinderImport checks if any action spec requires the binder package for
// JSON-to-struct binding. This is needed when a parameter is a struct type or an
// unrecognised generic type.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action specifications to
// check.
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

// specNeedsBinder checks whether a single action spec requires the binder package for
// JSON-to-struct binding.
//
// Takes spec (*annotator_dto.ActionSpec) which is the action specification to check.
//
// Returns bool which indicates if the spec has any parameters that need binder-based
// extraction.
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
// Takes naming (*actionNaming) which contains the action specifications to generate
// wrappers for and the names they are emitted under.
//
// Returns *ast.File which is the complete AST ready for code generation.
func (e *ActionWrapperEmitter) buildWrappersAST(naming *actionNaming) *ast.File {
	decls := make([]ast.Decl, 0, len(naming.specs)*2+1)
	decls = append(decls, e.buildLogVarDecl())

	for i := range naming.specs {
		decls = append(decls, e.buildWrapperFunc(naming, i))
		if naming.specs[i].HasSSE {
			decls = append(decls, e.buildBindFunc(naming, i))
		}
	}

	return &ast.File{
		Name:  goastutil.CachedIdent(actionGeneratedPackageName),
		Decls: decls,
	}
}

// buildLogVarDecl builds a variable declaration AST node for the logger. It creates: var
// pikoActionLog = logger.GetLogger("piko/actions").
//
// Returns *ast.GenDecl which is the variable declaration node.
func (*ActionWrapperEmitter) buildLogVarDecl() *ast.GenDecl {
	return &ast.GenDecl{
		Tok: token.VAR,
		Specs: []ast.Spec{
			&ast.ValueSpec{
				Names: []*ast.Ident{goastutil.CachedIdent(actionLogVarName)},
				Values: []ast.Expr{
					goastutil.CallExpr(
						goastutil.SelectorExpr(wrapperIdentLogger, "GetLogger"),
						goastutil.StrLit("piko/actions"),
					),
				},
			},
		},
	}
}

// buildWrapperFunc builds a single wrapper function declaration.
//
// Takes naming (*actionNaming) which holds the action specifications and their emitted
// names.
// Takes index (int) which selects the action to wrap.
//
// Returns *ast.FuncDecl which is the generated wrapper function AST node.
func (e *ActionWrapperEmitter) buildWrapperFunc(naming *actionNaming, index int) *ast.FuncDecl {
	spec := &naming.specs[index]
	functionName := naming.wrapperNames[index]
	pkgAlias := naming.qualifier(spec.PackagePath, spec.PackageName)

	locals := naming.paramLocals[index]

	statements := e.buildBindStatements(naming, spec, locals, bindReturnValueAndError)

	statements = append(statements, goastutil.DefineStmt(
		"a",
		goastutil.TypeAssertExpr(
			goastutil.CachedIdent("action"),
			goastutil.StarExpr(goastutil.SelectorExpr(pkgAlias, spec.StructName)),
		),
	))

	statements = append(statements, e.buildCallInvocation(naming, spec, locals)...)

	return goastutil.FuncDecl(
		functionName,
		e.buildDispatchParams(),
		goastutil.FieldList(
			goastutil.Field("", goastutil.CachedIdent("any")),
			goastutil.Field("", goastutil.CachedIdent("error")),
		),
		goastutil.BlockStmt(statements...),
	)
}

// buildBindFunc builds the bind function declaration for an SSE-capable action.
//
// Takes naming (*actionNaming) which holds the action specifications and their emitted
// names.
// Takes index (int) which selects the action to build a bind function for.
//
// Returns *ast.FuncDecl which is the generated bind function AST node.
func (e *ActionWrapperEmitter) buildBindFunc(naming *actionNaming, index int) *ast.FuncDecl {
	spec := &naming.specs[index]

	locals := naming.paramLocals[index]

	statements := e.buildBindStatements(naming, spec, locals, bindReturnErrorOnly)
	statements = append(statements,
		e.buildSetInputStmt(spec, locals), goastutil.ReturnStmt(goastutil.NilIdent()))

	return goastutil.FuncDecl(
		naming.bindNames[index],
		e.buildDispatchParams(),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent("error"))),
		goastutil.BlockStmt(statements...),
	)
}

// buildDispatchParams builds the parameter list shared by the invoke wrapper and the bind
// function, so the two stay signature-compatible with the runtime's entry fields.
//
// Returns *ast.FieldList which declares (ctx, action, argsMap).
func (*ActionWrapperEmitter) buildDispatchParams() *ast.FieldList {
	return goastutil.FieldList(
		goastutil.Field("ctx", goastutil.SelectorExpr(actionContextPackageAlias, "Context")),
		goastutil.Field("action", goastutil.CachedIdent("any")),
		goastutil.Field(wrapperIdentArgsMap, goastutil.MapType(goastutil.CachedIdent("string"), goastutil.CachedIdent("any"))),
	)
}

// buildBindStatements builds the logger preamble and the per-parameter extraction shared
// by the invoke wrapper and the bind function.
//
// Takes naming (*actionNaming) which provides the aliases parameter packages are
// qualified by.
// Takes spec (*annotator_dto.ActionSpec) which describes the action's call parameters.
// Takes locals ([]string) which holds the variable name each parameter is extracted into.
// Takes style (bindReturnStyle) which selects the result shape of a bind failure return.
//
// Returns []ast.Stmt which declares and populates one variable per bound parameter.
func (e *ActionWrapperEmitter) buildBindStatements(
	naming *actionNaming,
	spec *annotator_dto.ActionSpec,
	locals []string,
	style bindReturnStyle,
) []ast.Stmt {
	statements := make([]ast.Stmt, 0, len(spec.CallParams)+1)

	if specNeedsBinder(spec) {
		statements = append(statements, goastutil.DefineStmtMulti(
			[]string{"ctx", "l"},
			goastutil.CallExpr(
				goastutil.SelectorExpr(wrapperIdentLogger, "From"),
				goastutil.CachedIdent("ctx"),
				goastutil.CachedIdent(actionLogVarName),
			),
		))
	}

	for index := range spec.CallParams {
		if spec.CallParams[index].Name == blankParamName {
			continue
		}
		statements = append(statements,
			e.buildParamExtraction(naming, &spec.CallParams[index], locals[index], style)...)
	}

	return statements
}

// buildSetInputStmt builds the piko.SetActionInput call that records the bound parameters
// on the action's metadata.
//
// Takes spec (*annotator_dto.ActionSpec) which describes the action's call parameters.
// Takes locals ([]string) which holds the variable name each parameter was extracted
// into, in call order.
//
// Returns ast.Stmt which is the SetActionInput call.
func (*ActionWrapperEmitter) buildSetInputStmt(spec *annotator_dto.ActionSpec, locals []string) ast.Stmt {
	arguments := make([]ast.Expr, 0, len(spec.CallParams)+1)
	arguments = append(arguments, goastutil.CachedIdent("action"))

	for index := range spec.CallParams {
		param := &spec.CallParams[index]
		if param.Name == blankParamName {
			continue
		}
		argExpr := goastutil.CachedIdent(locals[index])
		if param.Optional {
			arguments = append(arguments, goastutil.AddressExpr(argExpr))
			continue
		}
		arguments = append(arguments, argExpr)
	}

	return goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr(wrapperIdentPiko, "SetActionInput"),
		arguments...,
	))
}

// buildParamExtraction builds statements to extract a parameter from the arguments map.
//
// Takes naming (*actionNaming) which provides the alias a struct parameter's package is
// qualified by.
// Takes param (*annotator_dto.ParamSpec) which specifies the parameter to extract,
// including its type and any special handling requirements.
// Takes varName (string) which is the local variable the value is extracted into. It is
// claimed against the names the wrapper body already uses, so it is not always the name
// the user gave the parameter.
// Takes style (bindReturnStyle) which selects the result shape of any bind failure
// return.
//
// Returns []ast.Stmt which contains the AST statements for extracting and converting the
// parameter value.
func (e *ActionWrapperEmitter) buildParamExtraction(
	naming *actionNaming,
	param *annotator_dto.ParamSpec,
	varName string,
	style bindReturnStyle,
) []ast.Stmt {
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
		return e.buildStructParamExtraction(naming, varName, jsonKey, param.Struct, style)
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
		return e.buildGenericParamExtraction(varName, jsonKey, param.GoType, style)
	}
}

// buildStructParamExtraction builds statements for extracting a struct parameter.
//
// Takes naming (*actionNaming) which provides the alias the struct's package is qualified
// by.
// Takes varName (string) which specifies the variable name for the extracted value.
// Takes jsonKey (string) which specifies the JSON key to extract from.
// Takes typeSpec (*annotator_dto.TypeSpec) which describes the struct type to extract.
// Takes style (bindReturnStyle) which selects the result shape of any bind failure
// return.
//
// Returns []ast.Stmt which contains the AST statements for JSON unmarshalling.
func (e *ActionWrapperEmitter) buildStructParamExtraction(naming *actionNaming, varName, jsonKey string, typeSpec *annotator_dto.TypeSpec, style bindReturnStyle) []ast.Stmt {
	if structSpecHasFileUpload(typeSpec) {
		return e.buildStructParamWithFileUploads(naming, varName, jsonKey, typeSpec, style)
	}
	qualifiedType := wrapperQualifiedTypeName(naming, typeSpec)
	return e.buildStructBindExtraction(varName, jsonKey, qualifiedType, style)
}

// buildStructParamWithFileUploads extracts a struct parameter that contains
// piko.FileUpload fields. The generated client sends such a struct as multipart form with
// the file(s) as top-level form fields, so each FileUpload field is pulled from the args
// map as a *multipart.FileHeader and removed, then the remaining flat fields bind into
// the struct.
//
// Takes naming (*actionNaming) which provides the alias the struct's package is qualified
// by.
// Takes varName (string) which is the local variable name for the struct.
// Takes jsonKey (string) which is the JSON key, used for binder error context.
// Takes typeSpec (*annotator_dto.TypeSpec) which describes the struct and its fields.
// Takes style (bindReturnStyle) which selects the result shape of any bind failure
// return.
//
// Returns []ast.Stmt which declares the struct, extracts its files, and binds the rest.
func (*ActionWrapperEmitter) buildStructParamWithFileUploads(naming *actionNaming, varName, jsonKey string, typeSpec *annotator_dto.TypeSpec, style bindReturnStyle) []ast.Stmt {
	qualifiedType := wrapperQualifiedTypeName(naming, typeSpec)
	stmts := []ast.Stmt{goastutil.VarDecl(varName, parseTypeExpr(qualifiedType))}

	for i := range typeSpec.Fields {
		field := typeSpec.Fields[i]
		if !field.IsFileUpload {
			continue
		}
		stmts = append(stmts, buildStructFileUploadFieldExtraction(varName, field.Name, field.JSONName, field.IsPointer)...)
	}

	stmts = append(stmts, buildFlatBindBlock(varName, jsonKey, style))
	return stmts
}

// buildBasicTypeAssertion builds a type assertion statement of the form varName, _ :=
// arguments["key"].(type).
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
// Takes jsonKey (string) which specifies the JSON key to extract the value from.
// Takes intType (string) which specifies the target integer type (int or int64).
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

// buildGenericParamExtraction builds statements for generic type extraction using JSON.
//
// Takes varName (string) which specifies the variable name to assign.
// Takes jsonKey (string) which specifies the JSON key to extract.
// Takes goType (string) which specifies the target Go type for the value.
// Takes style (bindReturnStyle) which selects the result shape of any bind failure
// return.
//
// Returns []ast.Stmt which contains the AST statements for the extraction.
func (e *ActionWrapperEmitter) buildGenericParamExtraction(varName, jsonKey, goType string, style bindReturnStyle) []ast.Stmt {
	return e.buildGuardedBindExtraction(varName, jsonKey, goType, style)
}

// buildFileUploadExtraction builds statements for extracting a single piko.FileUpload
// parameter from the arguments map.
//
// Takes varName (string) which specifies the variable name for the extracted file upload.
// Takes jsonKey (string) which specifies the key to look up in the arguments map.
//
// Returns []ast.Stmt which contains a variable declaration and a type assertion
// if-statement that extracts the file header and creates a new FileUpload.
func (*ActionWrapperEmitter) buildFileUploadExtraction(varName, jsonKey string) []ast.Stmt {
	return []ast.Stmt{
		goastutil.VarDecl(varName, goastutil.SelectorExpr(wrapperIdentPiko, "FileUpload")),
		goastutil.IfStmt(
			goastutil.DefineStmtMulti(
				[]string{"fh", wrapperIdentOK},
				goastutil.TypeAssertExpr(
					goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit(jsonKey)),
					goastutil.StarExpr(goastutil.SelectorExpr(actionMultipartPackageAlias, "FileHeader")),
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

// buildFileUploadSliceExtraction builds AST statements for extracting a []piko.FileUpload
// parameter from multipart file headers.
//
// Takes varName (string) which specifies the variable name to assign.
// Takes jsonKey (string) which specifies the key to look up in the arguments map.
//
// Returns []ast.Stmt which contains the variable declaration and conditional extraction
// logic.
func (*ActionWrapperEmitter) buildFileUploadSliceExtraction(varName, jsonKey string) []ast.Stmt {
	fileUploadSliceType := &ast.ArrayType{Elt: goastutil.SelectorExpr(wrapperIdentPiko, "FileUpload")}
	fileHeaderSliceType := &ast.ArrayType{Elt: goastutil.StarExpr(goastutil.SelectorExpr(actionMultipartPackageAlias, "FileHeader"))}

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

// buildRawBodyExtraction builds statements for extracting a piko.RawBody parameter from
// the arguments map using a type assertion.
//
// Takes varName (string) which is the name of the variable to assign the extracted
// RawBody value to.
//
// Returns []ast.Stmt which contains the variable declaration and conditional assignment
// statements.
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

// buildStructBindExtraction builds binder-based extraction for a struct parameter. It
// generates a single pikobinder.BindMap call whose source comes from
// pikobinder.ActionInputSource, which resolves nested JSON ({"input": {...}}) and flat
// form data ({...}) alike.
//
// Takes varName (string) which specifies the variable name to store the result.
// Takes jsonKey (string) which specifies the JSON key to extract.
// Takes typeName (string) which specifies the Go type for binding.
// Takes style (bindReturnStyle) which selects the result shape of any bind failure
// return.
//
// Returns []ast.Stmt which contains the generated AST statements.
func (*ActionWrapperEmitter) buildStructBindExtraction(varName, jsonKey, typeName string, style bindReturnStyle) []ast.Stmt {
	typeExpr := parseTypeExpr(typeName)

	source := goastutil.CallExpr(
		goastutil.SelectorExpr(actionBinderPackageAlias, "ActionInputSource"),
		goastutil.CachedIdent(wrapperIdentArgsMap),
		goastutil.StrLit(jsonKey),
	)

	return []ast.Stmt{
		goastutil.VarDecl(varName, typeExpr),
		buildBindMapStmt(source, varName, jsonKey, "Failed to bind action parameter", style),
	}
}

// buildGuardedBindExtraction builds binder-based extraction for a parameter that is not a
// struct, such as a slice, a map or a sized integer.
//
// Takes varName (string) which specifies the variable name to store the result.
// Takes jsonKey (string) which specifies the JSON key to extract.
// Takes typeName (string) which specifies the Go type for binding.
// Takes style (bindReturnStyle) which selects the result shape of any bind failure
// return.
//
// Returns []ast.Stmt which contains the generated AST statements.
func (*ActionWrapperEmitter) buildGuardedBindExtraction(varName, jsonKey, typeName string, style bindReturnStyle) []ast.Stmt {
	typeExpr := parseTypeExpr(typeName)

	return []ast.Stmt{
		goastutil.VarDecl(varName, typeExpr),
		&ast.IfStmt{
			Init: goastutil.DefineStmtMulti(
				[]string{"raw", wrapperIdentOK},
				goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit(jsonKey)),
			),
			Cond: goastutil.CachedIdent(wrapperIdentOK),
			Body: goastutil.BlockStmt(
				&ast.IfStmt{
					Init: goastutil.DefineStmtMulti(
						[]string{"rawMap", wrapperIdentOK},
						goastutil.TypeAssertExpr(
							goastutil.CachedIdent("raw"),
							goastutil.MapType(goastutil.CachedIdent("string"), goastutil.CachedIdent("any")),
						),
					),
					Cond: goastutil.CachedIdent(wrapperIdentOK),
					Body: goastutil.BlockStmt(
						buildBindMapStmt(goastutil.CachedIdent("rawMap"), varName, jsonKey, "Failed to bind action parameter", style),
					),
				},
			),
			Else: &ast.IfStmt{
				Cond: &ast.BinaryExpr{
					X:  goastutil.CallExpr(goastutil.CachedIdent("len"), goastutil.CachedIdent(wrapperIdentArgsMap)),
					Op: token.GTR,
					Y:  goastutil.IntLit(0),
				},
				Body: goastutil.BlockStmt(
					buildBindMapStmt(goastutil.CachedIdent(wrapperIdentArgsMap), varName, jsonKey, "Failed to bind action parameter from flat argsMap", style),
				),
			},
		},
	}
}

// buildCallInvocation builds the action Call method invocation statements.
//
// Takes naming (*actionNaming) which provides the alias a discarded struct argument's
// package is qualified by.
// Takes spec (*annotator_dto.ActionSpec) which provides the action specification
// including call parameters and error handling details.
// Takes locals ([]string) which holds the variable name each parameter was extracted
// into, in call order.
//
// Returns []ast.Stmt which contains the AST statements for invoking the Call method, with
// appropriate return handling based on whether errors are used.
func (*ActionWrapperEmitter) buildCallInvocation(
	naming *actionNaming,
	spec *annotator_dto.ActionSpec,
	locals []string,
) []ast.Stmt {
	arguments := make([]ast.Expr, 0, len(spec.CallParams))
	for index := range spec.CallParams {
		param := &spec.CallParams[index]
		if param.Name == blankParamName {
			arguments = append(arguments, buildZeroValueExpr(naming, param))
			continue
		}
		argExpr := goastutil.CachedIdent(locals[index])
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
		goastutil.ReturnStmt(goastutil.CachedIdent("result"), goastutil.NilIdent()),
	}
}

// parseTypeExpr parses a type name string into an AST expression.
//
// Takes typeName (string) which is the type name to parse.
//
// Returns ast.Expr which is the parsed AST expression.
func parseTypeExpr(typeName string) ast.Expr {
	if pkg, name, ok := strings.Cut(typeName, "."); ok {
		return goastutil.SelectorExpr(pkg, name)
	}
	return goastutil.CachedIdent(typeName)
}

// buildBindMapStmt builds the AST for binding a map[string]any to a struct.
//
// The failure branch logs at warning level: every error it reports describes the
// request's payload, whether a value would not convert or a `validate:"..."` tag rejected
// it, and malformed input from a client is not a fault of the server.
//
// Takes sourceExpression (ast.Expr) which is the map expression to bind from.
// Takes varName (string) which is the name of the variable to bind into.
// Takes jsonKey (string) which is the JSON key name for error logging.
// Takes errorContext (string) which is the context message for error logging.
// Takes style (bindReturnStyle) which selects the result shape of the failure return.
//
// Returns ast.Stmt which contains the bind call and error handling.
func buildBindMapStmt(sourceExpression ast.Expr, varName, jsonKey, errorContext string, style bindReturnStyle) ast.Stmt {
	return &ast.IfStmt{
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
				goastutil.CallExpr(
					goastutil.SelectorExpr(actionBinderPackageAlias, "WithDocumentScaleLimits"),
				),
				goastutil.CallExpr(
					goastutil.SelectorExpr(actionBinderPackageAlias, "WithValidation"),
					goastutil.CachedIdent("true"),
				),
			),
		),
		Cond: &ast.BinaryExpr{
			X:  goastutil.ErrIdent(),
			Op: token.NEQ,
			Y:  goastutil.NilIdent(),
		},
		Body: goastutil.BlockStmt(
			goastutil.ExprStmt(
				goastutil.CallExpr(
					goastutil.SelectorExpr("l", "Warn"),
					goastutil.StrLit(errorContext),
					goastutil.CallExpr(goastutil.SelectorExpr(wrapperIdentLogger, "String"), goastutil.StrLit("param"), goastutil.StrLit(jsonKey)),
					goastutil.CallExpr(goastutil.SelectorExpr(wrapperIdentLogger, "Error"), goastutil.ErrIdent()),
				),
			),
			style.returnStmt(),
		),
	}
}

// buildZeroValueExpr builds the zero-value expression for a parameter type. This is used
// for blank identifier (_) parameters that are positionally required in the Call
// signature but carry no meaningful data.
//
// Takes naming (*actionNaming) which provides the alias the parameter's package is
// qualified by.
// Takes param (*annotator_dto.ParamSpec) which describes the parameter type.
//
// Returns ast.Expr which is the zero-value expression for the type.
func buildZeroValueExpr(naming *actionNaming, param *annotator_dto.ParamSpec) ast.Expr {
	if param.Optional {
		return goastutil.NilIdent()
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
			typeExpr = parseTypeExpr(wrapperQualifiedTypeName(naming, param.Struct))
		}
		return goastutil.CompositeLit(typeExpr)
	}
}

// wrapperQualifiedTypeName returns the package-qualified type name for a struct type.
//
// The qualifier comes from the naming, so a type declared in an action package is
// referenced through the same alias the file imports that package under.
//
// Takes naming (*actionNaming) which provides the alias the type's package is qualified
// by.
// Takes typeSpec (*annotator_dto.TypeSpec) which specifies the type to format.
//
// Returns string which is the qualified name in "package.TypeName" format, or empty if
// typeSpec is nil.
func wrapperQualifiedTypeName(naming *actionNaming, typeSpec *annotator_dto.TypeSpec) string {
	if typeSpec == nil {
		return ""
	}
	return naming.qualifier(typeSpec.PackagePath, typeSpec.PackageName) + actionNameSeparator + typeSpec.Name
}

// structSpecHasFileUpload reports whether a struct parameter has any piko.FileUpload
// fields, which must be extracted from the multipart form rather than bound from the
// body.
//
// Takes typeSpec (*annotator_dto.TypeSpec) which describes the struct parameter.
//
// Returns bool which is true when the struct has at least one FileUpload field.
func structSpecHasFileUpload(typeSpec *annotator_dto.TypeSpec) bool {
	if typeSpec == nil {
		return false
	}
	for i := range typeSpec.Fields {
		if typeSpec.Fields[i].IsFileUpload {
			return true
		}
	}
	return false
}

// buildStructFileUploadFieldExtraction assigns a struct FileUpload field from the
// multipart header in the args map and then removes that key so the binder does not try
// to bind it. A pointer field is assigned by address, since piko.NewFileUpload returns a
// value.
//
// Takes varName (string) which is the struct variable name.
// Takes fieldName (string) which is the Go field name to assign.
// Takes jsonKey (string) which is the args-map key holding the *multipart.FileHeader.
// Takes isPointer (bool) which is true when the field type is a pointer to FileUpload.
//
// Returns []ast.Stmt which contains the type-asserted assignment and the delete call.
func buildStructFileUploadFieldExtraction(varName, fieldName, jsonKey string, isPointer bool) []ast.Stmt {
	target := goastutil.SelectorExprFrom(goastutil.CachedIdent(varName), fieldName)
	newUpload := goastutil.CallExpr(
		goastutil.SelectorExpr(wrapperIdentPiko, "NewFileUpload"),
		goastutil.CachedIdent("fh"),
	)

	var body *ast.BlockStmt
	if isPointer {
		body = goastutil.BlockStmt(
			goastutil.DefineStmt("fu", newUpload),
			goastutil.AssignStmt(target, &ast.UnaryExpr{Op: token.AND, X: goastutil.CachedIdent("fu")}),
		)
	} else {
		body = goastutil.BlockStmt(goastutil.AssignStmt(target, newUpload))
	}

	return []ast.Stmt{
		goastutil.IfStmt(
			goastutil.DefineStmtMulti(
				[]string{"fh", wrapperIdentOK},
				goastutil.TypeAssertExpr(
					goastutil.IndexExpr(goastutil.CachedIdent(wrapperIdentArgsMap), goastutil.StrLit(jsonKey)),
					goastutil.StarExpr(goastutil.SelectorExpr(actionMultipartPackageAlias, "FileHeader")),
				),
			),
			goastutil.CachedIdent(wrapperIdentOK),
			body,
		),
		goastutil.ExprStmt(
			goastutil.CallExpr(
				goastutil.CachedIdent("delete"),
				goastutil.CachedIdent(wrapperIdentArgsMap),
				goastutil.StrLit(jsonKey),
			),
		),
	}
}

// buildFlatBindBlock binds the remaining flat args-map entries into the struct via
// pikobinder.BindMap, once file fields have been removed.
//
// The bind is unconditional: an upload-only request leaves the args map empty, and the
// struct's non-file fields must still face their `validate:"..."` tags.
//
// Takes varName (string) which is the struct variable name.
// Takes jsonKey (string) which is the JSON key, used for binder error context.
// Takes style (bindReturnStyle) which selects the result shape of any bind failure
// return.
//
// Returns ast.Stmt which is the BindMap call and its error handling.
func buildFlatBindBlock(varName, jsonKey string, style bindReturnStyle) ast.Stmt {
	return buildBindMapStmt(
		goastutil.CachedIdent(wrapperIdentArgsMap),
		varName,
		jsonKey,
		"Failed to bind action parameter from flat argsMap",
		style,
	)
}
