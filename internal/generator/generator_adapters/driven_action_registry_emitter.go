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
	"path"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

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

	// actionNameSeparator separates the package part of an action name from the struct part,
	// as in "email.Contact".
	actionNameSeparator = "."

	// actionWrapperPrefix begins every generated wrapper function name.
	actionWrapperPrefix = "invoke"

	// actionBindPrefix begins every generated bind function name.
	actionBindPrefix = "bind"

	// actionHandlerTypeName is the name of the handler struct the registry file declares.
	actionHandlerTypeName = "ActionHandler"

	// actionRegistryFuncName is the name of the accessor function the registry file
	// declares.
	actionRegistryFuncName = "Registry"

	// actionInitFuncName is the name of the registration function the registry file
	// declares.
	actionInitFuncName = "init"

	// actionContextPackagePath is the import path for the context package.
	actionContextPackagePath = "context"

	// actionContextPackageAlias is the identifier the context package is referenced by in
	// generated code.
	actionContextPackageAlias = "context"

	// actionReflectPackageAlias is the identifier the reflect package is referenced by in
	// generated code.
	actionReflectPackageAlias = "reflect"

	// actionMultipartPackageAlias is the identifier the mime/multipart package is referenced
	// by in generated code.
	actionMultipartPackageAlias = "multipart"
)

var (
	// actionReservedIdentifiers lists every package-scope identifier the generated action
	// files already bind: the fixed import aliases and the declarations the emitters write
	// themselves. No package alias and no wrapper function name may take one of these, so
	// they seed the set of names already in use.
	actionReservedIdentifiers = []string{
		actionContextPackageAlias,
		actionReflectPackageAlias,
		actionMultipartPackageAlias,
		actionJSONPackageAlias,
		actionBinderPackageAlias,
		actionGeneratedPackageName,
		actionHandlerTypeName,
		actionRegistryFuncName,
		actionInitFuncName,
		actionLogVarName,
		wrapperIdentPiko,
		wrapperIdentLogger,
	}
)

// actionNaming holds every generated name the two action emitters have to agree on: the
// order the actions are emitted in, the file-local import alias each referenced package
// is referenced by, and the wrapper function name each action dispatches through.
type actionNaming struct {
	// packageAliases maps an import path to its file-local alias.
	packageAliases map[string]string

	// referenced records the import paths qualifier has been asked for.
	referenced map[string]struct{}

	// used holds every package-scope identifier the generated file has claimed. It is kept
	// past construction so a package first referenced later can still claim an alias that
	// collides with nothing.
	used map[string]struct{}

	// packagePaths lists the aliased import paths in sorted order, so imports are added in a
	// stable order.
	packagePaths []string

	// specs holds the action specifications in emission order.
	specs []annotator_dto.ActionSpec

	// wrapperNames holds the wrapper function name of each entry in specs, by index.
	wrapperNames []string

	// bindNames holds the bind function name of each entry in specs, by index. The entry is
	// empty for an action with no SSE transport, which needs no bind function.
	bindNames []string

	// paramLocals holds the local variable name each call parameter is extracted into,
	// indexed by spec then by parameter. The entry is empty for a blank parameter, which is
	// never extracted.
	paramLocals [][]string
}

// newActionNaming derives the aliases and wrapper names for a set of action specs.
//
// Takes specs ([]annotator_dto.ActionSpec) which are the actions being emitted.
//
// Returns actionNaming which holds the sorted specs and the names derived for them.
func newActionNaming(specs []annotator_dto.ActionSpec) actionNaming {
	sorted := actionSortSpecs(specs)
	packagePaths, stems := actionPackageStems(sorted)

	used := make(map[string]struct{}, len(actionReservedIdentifiers)+len(packagePaths)+len(sorted))
	for _, reserved := range actionReservedIdentifiers {
		used[reserved] = struct{}{}
	}

	packageAliases := make(map[string]string, len(packagePaths))
	for _, packagePath := range packagePaths {
		alias := goastutil.GoPackageAliasWithStem(stems[packagePath], packagePath)
		packageAliases[packagePath] = goastutil.ReserveIdentifier(alias, used)
	}

	wrapperNames := make([]string, len(sorted))
	bindNames := make([]string, len(sorted))
	paramLocals := make([][]string, len(sorted))
	for i := range sorted {
		pascalName := actionToPascalCase(sorted[i].Name)
		wrapperName := goastutil.SanitiseGoIdentifier(actionWrapperPrefix + pascalName)
		wrapperNames[i] = goastutil.ReserveIdentifier(wrapperName, used)

		if !sorted[i].HasSSE {
			continue
		}
		bindName := goastutil.SanitiseGoIdentifier(actionBindPrefix + pascalName)
		bindNames[i] = goastutil.ReserveIdentifier(bindName, used)
	}

	for i := range sorted {
		paramLocals[i] = actionParamLocals(&sorted[i])
	}

	return actionNaming{
		packageAliases: packageAliases,
		referenced:     make(map[string]struct{}, len(packagePaths)),
		packagePaths:   packagePaths,
		specs:          sorted,
		wrapperNames:   wrapperNames,
		bindNames:      bindNames,
		paramLocals:    paramLocals,
		used:           used,
	}
}

// qualifier returns the identifier a reference into a package must be prefixed with, and
// records that the file being built now names that package.
//
// Takes packagePath (string) which is the import path being referenced.
// Takes packageName (string) which is the declared package name, used as the readable
// stem for a package the specs never named.
//
// Returns string which is the selector qualifier for that package.
func (n *actionNaming) qualifier(packagePath, packageName string) string {
	if alias, exists := n.packageAliases[packagePath]; exists {
		n.referenced[packagePath] = struct{}{}
		return alias
	}

	stem := actionPackageNameOrDefault(packageName, packagePath)
	alias := goastutil.ReserveIdentifier(goastutil.GoPackageAliasWithStem(stem, packagePath), n.used)
	n.packageAliases[packagePath] = alias
	n.referenced[packagePath] = struct{}{}

	position, _ := slices.BinarySearch(n.packagePaths, packagePath)
	n.packagePaths = slices.Insert(n.packagePaths, position, packagePath)

	return alias
}

// addPackageImports adds an aliased import for every package the generated file actually
// qualifies a reference by.
//
// Takes fset (*token.FileSet) which positions the added import specs.
// Takes file (*ast.File) which receives the imports.
func (n *actionNaming) addPackageImports(fset *token.FileSet, file *ast.File) {
	for _, packagePath := range n.packagePaths {
		if _, referenced := n.referenced[packagePath]; !referenced {
			continue
		}
		goastutil.AddNamedImport(fset, file, n.packageAliases[packagePath], packagePath)
	}
}

// actionPackageStems collects the packages the generated files may need to import and the
// readable stem each of their aliases is built from.
//
// Takes specs ([]annotator_dto.ActionSpec) which are the actions being emitted.
//
// Returns []string which are the unique import paths in sorted order.
// Returns map[string]string which maps each import path to its declared package name.
func actionPackageStems(specs []annotator_dto.ActionSpec) ([]string, map[string]string) {
	stems := make(map[string]string, len(specs))
	paths := make([]string, 0, len(specs))

	for i := range specs {
		spec := &specs[i]
		paths = actionAddPackageStem(spec.PackagePath, spec.PackageName, stems, paths)

		for j := range spec.CallParams {
			paths = actionAddTypePackageStem(spec.CallParams[j].Struct, stems, paths)
		}
		paths = actionAddTypePackageStem(spec.ReturnType, stems, paths)
	}

	slices.Sort(paths)
	return paths, stems
}

// actionAddTypePackageStem records the package a parameter or return type is declared in.
//
// Takes typeSpec (*annotator_dto.TypeSpec) which is the type, possibly nil for a
// parameter that is not a struct or an action that returns only an error.
// Takes stems (map[string]string) which gains the package's readable alias stem.
// Takes paths ([]string) which is the accumulator of import paths.
//
// Returns []string which is the updated accumulator.
func actionAddTypePackageStem(typeSpec *annotator_dto.TypeSpec, stems map[string]string, paths []string) []string {
	if typeSpec == nil {
		return paths
	}
	return actionAddPackageStem(typeSpec.PackagePath, typeSpec.PackageName, stems, paths)
}

// actionAddPackageStem records one package's alias stem, ignoring a path already recorded
// and a builtin type's empty path, which names no package to import.
//
// Takes packagePath (string) which is the import path.
// Takes packageName (string) which is the declared package name, possibly empty.
// Takes stems (map[string]string) which gains the package's readable alias stem.
// Takes paths ([]string) which is the accumulator of import paths.
//
// Returns []string which is the updated accumulator.
func actionAddPackageStem(packagePath, packageName string, stems map[string]string, paths []string) []string {
	if packagePath == "" {
		return paths
	}
	if _, exists := stems[packagePath]; exists {
		return paths
	}
	stems[packagePath] = actionPackageNameOrDefault(packageName, packagePath)
	return append(paths, packagePath)
}

// actionPackageNameOrDefault returns the declared package name, falling back to the last
// segment of the import path when the annotator recorded no name.
//
// Takes packageName (string) which is the declared package name, possibly empty.
// Takes packagePath (string) which is the import path to fall back to.
//
// Returns string which is the package name to use.
func actionPackageNameOrDefault(packageName, packagePath string) string {
	return cmp.Or(packageName, path.Base(packagePath))
}

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
// Takes specs ([]annotator_dto.ActionSpec) which defines the actions to include in the
// registry.
//
// Returns []byte which contains the formatted Go source code.
// Returns error when AST formatting fails.
func (e *ActionRegistryEmitter) EmitRegistry(_ context.Context, specs []annotator_dto.ActionSpec) ([]byte, error) {
	fset := token.NewFileSet()
	naming := newActionNaming(specs)
	file := e.buildRegistryAST(&naming)

	goastutil.AddImport(fset, file, actionContextPackagePath)
	goastutil.AddImport(fset, file, actionPikoPackagePath)
	goastutil.AddNamedImport(fset, file, actionJSONPackageAlias, actionJSONPackagePath)
	goastutil.AddImport(fset, file, actionReflectPackagePath)
	naming.addPackageImports(fset, file)

	return goastutil.FormatAST(fset, file)
}

// buildRegistryAST constructs the complete AST for the registry file.
//
// Takes naming (*actionNaming) which provides the action specifications to include in the
// generated registry and the names they are emitted under.
//
// Returns *ast.File which is the complete AST ready for code generation.
func (e *ActionRegistryEmitter) buildRegistryAST(naming *actionNaming) *ast.File {
	return &ast.File{
		Name: goastutil.CachedIdent(actionGeneratedPackageName),
		Decls: []ast.Decl{
			e.buildInitFunction(naming),
			e.buildActionHandlerTypeDecl(),
			e.buildRegistryFunction(naming),
		},
	}
}

// buildInitFunction builds the init() function that registers all actions and pretouches
// JSON types for performance.
//
// Takes naming (*actionNaming) which contains the action specifications to register and
// the names they are emitted under.
//
// Returns *ast.FuncDecl which is the generated init function AST node.
func (e *ActionRegistryEmitter) buildInitFunction(naming *actionNaming) *ast.FuncDecl {
	mapElts := make([]ast.Expr, 0, len(naming.specs))
	for i := range naming.specs {
		mapElts = append(mapElts, e.buildActionEntry(naming, i))
	}

	registerCall := goastutil.ExprStmt(goastutil.CallExpr(
		goastutil.SelectorExpr(wrapperIdentPiko, "RegisterActions"),
		goastutil.CompositeLit(
			goastutil.MapType(
				goastutil.CachedIdent(registryTypeString),
				goastutil.SelectorExpr(wrapperIdentPiko, "ActionHandlerEntry"),
			),
			mapElts...,
		),
	))

	pretouchTypes := e.collectPretouchTypes(naming.specs)

	statements := []ast.Stmt{registerCall}
	if len(pretouchTypes) > 0 {
		statements = append(statements, e.buildPretouchStatements(naming, pretouchTypes)...)
	}

	return goastutil.FuncDecl(
		actionInitFuncName,
		nil,
		nil,
		goastutil.BlockStmt(statements...),
	)
}

// collectPretouchTypes collects unique struct types from action specs for pretouch. Only
// types with valid package paths are included (skips builtins).
//
// Takes specs ([]annotator_dto.ActionSpec) which contains the action specifications to
// extract types from.
//
// Returns []annotator_dto.TypeSpec which contains the unique types sorted by their fully
// qualified names.
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
// Takes naming (*actionNaming) which provides the alias each type's package is qualified
// by.
// Takes types ([]annotator_dto.TypeSpec) which specifies the types to pretouch for JSON
// serialisation.
//
// Returns []ast.Stmt which contains the variable declarations and loop for pretouching
// all specified types.
func (*ActionRegistryEmitter) buildPretouchStatements(naming *actionNaming, types []annotator_dto.TypeSpec) []ast.Stmt {
	typeElts := make([]ast.Expr, 0, len(types))
	for _, t := range types {
		typeElts = append(typeElts, goastutil.CallExpr(
			&ast.IndexExpr{
				X:     goastutil.SelectorExpr(actionReflectPackageAlias, "TypeFor"),
				Index: goastutil.SelectorExpr(naming.qualifier(t.PackagePath, t.PackageName), t.Name),
			},
		))
	}

	pretouchTypesDecl := goastutil.DefineStmt(
		"pretouchTypes",
		goastutil.CompositeLit(
			&ast.ArrayType{Elt: goastutil.SelectorExpr(actionReflectPackageAlias, "Type")},
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
// The Bind field must mirror piko.ActionHandlerEntry: buildActionEntry emits one keyed
// composite literal that is used both for the piko-typed registration in init and for the
// locally-typed map the declaration describes, so a field missing here fails to compile.
//
// Returns *ast.GenDecl which contains the struct type definition for ActionHandler with
// fields for Name, Method, Create, Invoke, Bind, and HasSSE.
func (e *ActionRegistryEmitter) buildActionHandlerTypeDecl() *ast.GenDecl {
	return goastutil.GenDeclType(
		actionHandlerTypeName,
		goastutil.StructType(
			goastutil.Field("Name", goastutil.CachedIdent(registryTypeString)),
			goastutil.Field("Method", goastutil.CachedIdent(registryTypeString)),
			goastutil.Field("Create", goastutil.FuncType(nil, goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(registryTypeAny))))),
			goastutil.Field("Invoke", e.buildInvokeFuncType()),
			goastutil.Field("Bind", e.buildBindFuncType()),
			goastutil.Field("HasSSE", goastutil.CachedIdent("bool")),
		),
	)
}

// buildBindFuncType builds the AST representation for a bind function type with
// signature: func(ctx context.Context, action any, args map[string]any) error.
//
// Returns *ast.FuncType which defines the function type for SSE input binding.
func (*ActionRegistryEmitter) buildBindFuncType() *ast.FuncType {
	return goastutil.FuncType(
		goastutil.FieldList(
			goastutil.Field("ctx", goastutil.SelectorExpr(actionContextPackageAlias, "Context")),
			goastutil.Field("action", goastutil.CachedIdent(registryTypeAny)),
			goastutil.Field("args", goastutil.MapType(goastutil.CachedIdent(registryTypeString), goastutil.CachedIdent(registryTypeAny))),
		),
		goastutil.FieldList(goastutil.Field("", goastutil.CachedIdent(registryTypeError))),
	)
}

// buildInvokeFuncType builds the AST representation for an invoke function type with
// signature: func(ctx context.Context, action any, arguments map[string]any) (any,
// error).
//
// Returns *ast.FuncType which defines the function type for action invocation.
func (*ActionRegistryEmitter) buildInvokeFuncType() *ast.FuncType {
	return goastutil.FuncType(
		goastutil.FieldList(
			goastutil.Field("ctx", goastutil.SelectorExpr(actionContextPackageAlias, "Context")),
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
// Takes naming (*actionNaming) which provides the action specifications to include in the
// registry and the names they are emitted under.
//
// Returns *ast.FuncDecl which is the generated Registry function declaration.
func (e *ActionRegistryEmitter) buildRegistryFunction(naming *actionNaming) *ast.FuncDecl {
	mapElts := make([]ast.Expr, 0, len(naming.specs))
	for i := range naming.specs {
		mapElts = append(mapElts, e.buildActionEntry(naming, i))
	}

	returnStmt := goastutil.ReturnStmt(
		goastutil.CompositeLit(
			goastutil.MapType(goastutil.CachedIdent(registryTypeString), goastutil.CachedIdent(actionHandlerTypeName)),
			mapElts...,
		),
	)

	return goastutil.FuncDecl(
		actionRegistryFuncName,
		nil,
		goastutil.FieldList(goastutil.Field("", goastutil.MapType(goastutil.CachedIdent(registryTypeString), goastutil.CachedIdent(actionHandlerTypeName)))),
		goastutil.BlockStmt(returnStmt),
	)
}

// buildActionEntry builds a single map entry for an action.
//
// Takes naming (*actionNaming) which holds the action specifications and their emitted
// names.
// Takes index (int) which selects the action to build an entry for.
//
// Returns *ast.KeyValueExpr which is the AST node representing the map entry.
func (e *ActionRegistryEmitter) buildActionEntry(naming *actionNaming, index int) *ast.KeyValueExpr {
	spec := &naming.specs[index]
	pkgAlias := naming.qualifier(spec.PackagePath, spec.PackageName)

	fields := []ast.Expr{
		goastutil.KeyValueIdent("Name", goastutil.StrLit(spec.Name)),
		goastutil.KeyValueIdent("Method", goastutil.StrLit(spec.HTTPMethod)),
		goastutil.KeyValueIdent("Create", e.buildCreateFunc(pkgAlias, spec.StructName)),
		goastutil.KeyValueIdent("Invoke", goastutil.CachedIdent(naming.wrapperNames[index])),
	}

	if bindName := naming.bindNames[index]; bindName != "" {
		fields = append(fields, goastutil.KeyValueIdent("Bind", goastutil.CachedIdent(bindName)))
	}

	fields = append(fields, goastutil.KeyValueIdent("HasSSE", goastutil.BoolIdent(spec.HasSSE)))

	return goastutil.KeyValueExpr(
		goastutil.StrLit(spec.Name),
		goastutil.CompositeLit(nil, fields...),
	)
}

// buildCreateFunc builds a function literal that returns a new instance of a struct,
// producing code of the form: func() any { return &pkg.StructName{} }.
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
// Takes params ([]annotator_dto.ParamSpec) which contains the parameters to scan for
// struct types.
// Takes seen (map[string]bool) which tracks already processed types to avoid duplicates.
// Takes result ([]annotator_dto.TypeSpec) which is the slice to append new types to.
//
// Returns []annotator_dto.TypeSpec which contains the updated result with any new struct
// types appended.
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

// collectReturnType adds a return type to the result slice if it is a struct that has not
// already been seen.
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
		return cmp.Or(
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.PackagePath, b.PackagePath),
			cmp.Compare(a.StructName, b.StructName),
		)
	})
	return sorted
}

// actionToPascalCase converts a dot-notation name to PascalCase.
//
// Takes name (string) which is the dot-separated string to convert.
//
// Returns string which is the PascalCase version of the input.
func actionToPascalCase(name string) string {
	var builder strings.Builder
	builder.Grow(len(name))

	for part := range strings.SplitSeq(name, actionNameSeparator) {
		if part == "" {
			continue
		}
		first, width := utf8.DecodeRuneInString(part)
		_, _ = builder.WriteRune(unicode.ToUpper(first))
		_, _ = builder.WriteString(part[width:])
	}

	return builder.String()
}
