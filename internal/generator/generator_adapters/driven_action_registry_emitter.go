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
	"cmp"
	"context"
	"go/ast"
	"go/token"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/goastutil"
)

const (
	// registryTypeString is the Go type name used for string fields in generated code.
	registryTypeString = "string"

	// registryTypeAny is the type name for Go's any type in generated code.
	registryTypeAny = "any"

	// registryTypeError is the error type identifier used in generated code.
	registryTypeError = "error"
)

// ActionRegistryEmitter generates Go code that maps action names to handlers.
type ActionRegistryEmitter struct{}

// NewActionRegistryEmitter creates a new registry emitter.
//
// Returns *ActionRegistryEmitter which is ready for use.
func NewActionRegistryEmitter() *ActionRegistryEmitter {
	return &ActionRegistryEmitter{}
}

// EmitRegistry generates the action registry Go file using AST construction.
//
// Takes specs ([]annotator_dto.ActionSpec) which defines the actions to
// include in the registry.
//
// Returns []byte which contains the formatted Go source code.
// Returns error when AST formatting fails.
func (e *ActionRegistryEmitter) EmitRegistry(_ context.Context, specs []annotator_dto.ActionSpec) ([]byte, error) {
	fset := token.NewFileSet()
	file := e.buildRegistryAST(specs)

	goastutil.AddImport(fset, file, "context")
	goastutil.AddImport(fset, file, actionPikoPackagePath)
	goastutil.AddNamedImport(fset, file, actionJSONPackageAlias, actionJSONPackagePath)
	goastutil.AddImport(fset, file, actionReflectPackagePath)
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

// buildRegistryAST constructs the complete AST for the registry file.
//
// Takes specs ([]annotator_dto.ActionSpec) which provides the action
// specifications to include in the generated registry.
//
// Returns *ast.File which is the complete AST ready for code generation.
func (e *ActionRegistryEmitter) buildRegistryAST(specs []annotator_dto.ActionSpec) *ast.File {
	return &ast.File{
		Name: goastutil.CachedIdent(actionGeneratedPackageName),
		Decls: []ast.Decl{
			e.buildInitFunction(specs),
			e.buildActionHandlerTypeDecl(),
			e.buildRegistryFunction(specs),
		},
	}
}

// buildInitFunction builds the init() function that registers all actions
// and pretouches JSON types for performance.
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action
// specifications to register.
//
// Returns *ast.FuncDecl which is the generated init function AST node.
func (e *ActionRegistryEmitter) buildInitFunction(specs []annotator_dto.ActionSpec) *ast.FuncDecl {
	sortedSpecs := actionSortSpecs(specs)

	mapElts := make([]ast.Expr, 0, len(sortedSpecs))
	for i := range sortedSpecs {
		mapElts = append(mapElts, e.buildActionEntry(&sortedSpecs[i]))
	}

	registerCall := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr("piko", "RegisterActions"),
		goastutil.CompositeLit(
			goastutil.MapType(
				goastutil.CachedIdent(registryTypeString),
				goastutil.SelectorExpr("piko", "ActionHandlerEntry"),
			),
			mapElts...,
		),
	))

	pretouchTypes := e.collectPretouchTypes(specs)

	statements := []ast.Stmt{registerCall}
	if len(pretouchTypes) > 0 {
		statements = append(statements, e.buildPretouchStatements(pretouchTypes)...)
	}

	return goastutil.FuncDecl(
		"init",
		nil,
		nil,
		goastutil.BlockStmt(statements...),
	)
}

// collectPretouchTypes collects unique struct types from action specs for
// pretouch. Only types with valid package paths are included (skips builtins).
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action
// specifications to extract types from.
//
// Returns []annotator_dto.TypeSpec which contains the unique types sorted
// by their fully qualified names.
func (*ActionRegistryEmitter) collectPretouchTypes(specs []annotator_dto.ActionSpec) []annotator_dto.TypeSpec {
	seen := make(map[string]bool)
	var result []annotator_dto.TypeSpec

	for i := range specs {
		result = collectParamTypes(specs[i].CallParams, seen, result)
		result = collectReturnType(specs[i].ReturnType, seen, result)
	}

	slices.SortFunc(result, func(a, b annotator_dto.TypeSpec) int {
		return cmp.Compare(a.PackagePath+"."+a.Name, b.PackagePath+"."+b.Name)
	})

	return result
}

// buildPretouchStatements builds the pretouch initialisation statements.
//
// Takes types ([]annotator_dto.TypeSpec) which specifies the types to
// pretouch for JSON serialisation.
//
// Returns []ast.Stmt which contains the variable declarations and loop for
// pretouching all specified types.
func (*ActionRegistryEmitter) buildPretouchStatements(types []annotator_dto.TypeSpec) []ast.Stmt {
	typeElts := make([]ast.Expr, 0, len(types))
	for _, t := range types {
		packageName := t.PackageName
		if packageName == "" {
			pkgParts := strings.Split(t.PackagePath, "/")
			packageName = pkgParts[len(pkgParts)-1]
		}
		typeElts = append(typeElts, goastutil.CallExpr(
			&ast.IndexExpr{
				X:     goastutil.SelectorExpr("reflect", "TypeFor"),
				Index: goastutil.SelectorExpr(packageName, t.Name),
			},
		))
	}

	pretouchTypesDecl := goastutil.DefineStmt(
		"pretouchTypes",
		goastutil.CompositeLit(
			&ast.ArrayType{Elt: goastutil.SelectorExpr("reflect", "Type")},
			typeElts...,
		),
	)

	forStmt := &ast.RangeStmt{
		Key:   goastutil.CachedIdent("_"),
		Value: goastutil.CachedIdent("t"),
		Tok:   token.DEFINE,
		X:     goastutil.CachedIdent("pretouchTypes"),
		Body: goastutil.BlockStmt(
			&ast.AssignStmt{
				Lhs: []ast.Expr{goastutil.CachedIdent("_")},
				Tok: token.ASSIGN,
				Rhs: []ast.Expr{
					goastutil.CallExpr(
						goastutil.SelectorExpr(actionJSONPackageAlias, "Pretouch"),
						goastutil.CachedIdent("t"),
					),
				},
			},
		),
	}

	return []ast.Stmt{pretouchTypesDecl, forStmt}
}

// buildActionHandlerTypeDecl builds the ActionHandler type declaration.
//
// Returns *ast.GenDecl which contains the struct type definition for
// ActionHandler with fields for Name, Method, Create, Invoke, and HasSSE.
func (e *ActionRegistryEmitter) buildActionHandlerTypeDecl() *ast.GenDecl {
	return goastutil.GenDeclType(
		"ActionHandler",
		goastutil.StructType(
			goastutil.Field("Name", goastutil.CachedIdent(registryTypeString)),
			goastutil.Field("Method", goastutil.CachedIdent(registryTypeString)),
			goastutil.Field("Create", goastutil.FuncType(nil, goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(registryTypeAny))))),
			goastutil.Field("Invoke", e.buildInvokeFuncType()),
			goastutil.Field("HasSSE", goastutil.CachedIdent("bool")),
		),
	)
}

// buildInvokeFuncType builds the AST representation for an invoke function
// type with signature:
// func(ctx context.Context, action any, arguments map[string]any) (any, error).
//
// Returns *ast.FuncType which defines the function type for action invocation.
func (*ActionRegistryEmitter) buildInvokeFuncType() *ast.FuncType {
	return goastutil.FuncType(
		goastutil.FieldList(
			goastutil.Field("ctx", goastutil.SelectorExpr("context", "Context")),
			goastutil.Field("action", goastutil.CachedIdent(registryTypeAny)),
			goastutil.Field("args", goastutil.MapType(goastutil.CachedIdent(registryTypeString), goastutil.CachedIdent(registryTypeAny))),
		),
		goastutil.FieldList(
			goastutil.Field("", goastutil.CachedIdent(registryTypeAny)),
			goastutil.Field("", goastutil.CachedIdent(registryTypeError)),
		),
	)
}

// buildRegistryFunction builds the Registry() function.
//
// Takes specs ([]annotator_dto.ActionSpec) which provides the action
// specifications to include in the registry.
//
// Returns *ast.FuncDecl which is the generated Registry function declaration.
func (e *ActionRegistryEmitter) buildRegistryFunction(specs []annotator_dto.ActionSpec) *ast.FuncDecl {
	sortedSpecs := actionSortSpecs(specs)

	mapElts := make([]ast.Expr, 0, len(sortedSpecs))
	for i := range sortedSpecs {
		mapElts = append(mapElts, e.buildActionEntry(&sortedSpecs[i]))
	}

	returnStmt := goastutil.ReturnStmt(
		goastutil.CompositeLit(
			goastutil.MapType(goastutil.CachedIdent(registryTypeString), goastutil.CachedIdent("ActionHandler")),
			mapElts...,
		),
	)

	return goastutil.FuncDecl(
		"Registry",
		nil,
		goastutil.FieldList(goastutil.Field("", goastutil.MapType(goastutil.CachedIdent(registryTypeString), goastutil.CachedIdent("ActionHandler")))),
		goastutil.BlockStmt(returnStmt),
	)
}

// buildActionEntry builds a single map entry for an action.
//
// Takes spec (*annotator_dto.ActionSpec) which defines the action to build
// an entry for.
//
// Returns *ast.KeyValueExpr which is the AST node representing the map entry.
func (e *ActionRegistryEmitter) buildActionEntry(spec *annotator_dto.ActionSpec) *ast.KeyValueExpr {
	pkgAlias := spec.PackageName
	invokeFuncName := "invoke" + actionToPascalCase(spec.Name)

	return goastutil.KeyValueExpr(
		goastutil.StrLit(spec.Name),
		goastutil.CompositeLit(
			nil,
			goastutil.KeyValueIdent("Name", goastutil.StrLit(spec.Name)),
			goastutil.KeyValueIdent("Method", goastutil.StrLit(spec.HTTPMethod)),
			goastutil.KeyValueIdent("Create", e.buildCreateFunc(pkgAlias, spec.StructName)),
			goastutil.KeyValueIdent("Invoke", goastutil.CachedIdent(invokeFuncName)),
			goastutil.KeyValueIdent("HasSSE", goastutil.BoolIdent(spec.HasSSE)),
		),
	)
}

// buildCreateFunc builds a function literal that returns a new instance of a
// struct, producing code of the form: func() any { return &pkg.StructName{} }.
//
// Takes pkgAlias (string) which specifies the package alias for the struct.
// Takes structName (string) which specifies the name of the struct to create.
//
// Returns *ast.FuncLit which is the generated function literal AST node.
func (*ActionRegistryEmitter) buildCreateFunc(pkgAlias, structName string) *ast.FuncLit {
	return goastutil.FuncLit(
		goastutil.FuncType(nil, goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(registryTypeAny)))),
		goastutil.BlockStmt(
			goastutil.ReturnStmt(
				goastutil.AddressExpr(
					goastutil.CompositeLit(goastutil.SelectorExpr(pkgAlias, structName)),
				),
			),
		),
	)
}

// collectParamTypes adds struct types from call parameters to the result.
//
// Takes params ([]annotator_dto.ParamSpec) which contains the parameters
// to scan for struct types.
// Takes seen (map[string]bool) which tracks already processed types to avoid
// duplicates.
// Takes result ([]annotator_dto.TypeSpec) which is the slice to append
// new types to.
//
// Returns []annotator_dto.TypeSpec which contains the updated result with
// any new struct types appended.
func collectParamTypes(params []annotator_dto.ParamSpec, seen map[string]bool, result []annotator_dto.TypeSpec) []annotator_dto.TypeSpec {
	for _, param := range params {
		if param.Struct == nil || param.Struct.PackagePath == "" {
			continue
		}
		key := param.Struct.PackagePath + "." + param.Struct.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, *param.Struct)
	}
	return result
}

// collectReturnType adds a return type to the result slice if it is a struct
// that has not already been seen.
//
// Takes returnType (*annotator_dto.TypeSpec) which is the type to collect.
// Takes seen (map[string]bool) which tracks already processed types by key.
// Takes result ([]annotator_dto.TypeSpec) which is the accumulator slice.
//
// Returns []annotator_dto.TypeSpec which is the updated result slice.
func collectReturnType(returnType *annotator_dto.TypeSpec, seen map[string]bool, result []annotator_dto.TypeSpec) []annotator_dto.TypeSpec {
	if returnType == nil || returnType.PackagePath == "" {
		return result
	}
	key := returnType.PackagePath + "." + returnType.Name
	if seen[key] {
		return result
	}
	seen[key] = true
	return append(result, *returnType)
}

// actionSortSpecs returns a sorted copy of specs by name.
//
// Takes specs ([]annotator_dto.ActionSpec) which is the slice to sort.
//
// Returns []annotator_dto.ActionSpec which is a new slice sorted by name.
func actionSortSpecs(specs []annotator_dto.ActionSpec) []annotator_dto.ActionSpec {
	sorted := make([]annotator_dto.ActionSpec, len(specs))
	copy(sorted, specs)
	slices.SortFunc(sorted, func(a, b annotator_dto.ActionSpec) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return sorted
}

// actionNeedsAlias checks if a package import needs an alias.
//
// Takes packagePath (string) which is the full import path of the package.
// Takes packageName (string) which is the declared package name.
//
// Returns bool which is true when the last path segment differs from the
// package name.
func actionNeedsAlias(packagePath, packageName string) bool {
	parts := strings.Split(packagePath, "/")
	lastPart := parts[len(parts)-1]
	return lastPart != packageName
}

// actionToPascalCase converts a dot-notation name to PascalCase.
//
// Takes name (string) which is the dot-separated string to convert.
//
// Returns string which is the PascalCase version of the input.
func actionToPascalCase(name string) string {
	var b strings.Builder
	for part := range strings.SplitSeq(name, ".") {
		if len(part) > 0 {
			b.WriteString(strings.ToUpper(part[:1]))
			b.WriteString(part[1:])
		}
	}
	return b.String()
}
