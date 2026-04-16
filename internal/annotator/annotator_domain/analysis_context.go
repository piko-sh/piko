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

package annotator_domain

// Provides the core context and symbol table structures for analysing
// component templates. Tracks variables, scopes, and type information
// whilst collecting diagnostics during template processing.

import (
	"fmt"
	goast "go/ast"
	"go/token"
	"maps"
	"slices"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// typeBuiltInFunction is the type name used for Go built-in functions.
	typeBuiltInFunction = "builtin_function"

	// initialSymbolCapacity is the starting size for symbol maps in a
	// symbol table.
	initialSymbolCapacity = 8
)

// Symbol represents a variable or identifier within a specific scope. It
// stores the resolved type, source location, and the final variable name for
// use in generated Go code.
type Symbol struct {
	// TypeInfo holds the resolved type information for this symbol.
	TypeInfo *ast_domain.ResolvedTypeInfo

	// Name is the identifier for the symbol.
	Name string

	// CodeGenVarName is the Go variable name for this symbol in generated code.
	// For partials, it is 'props_<invocationKey>' for props or
	// '<partialHashedName>Data_<invocationKey>' for state.
	CodeGenVarName string

	// SourceInvocationKey is the invocation key of the partial this
	// symbol originates from, used for explicit dependency tracking
	// between partial invocations.
	//
	// Empty for symbols not originating from a partial invocation
	// (e.g., request, built-ins).
	SourceInvocationKey string
}

// reservedSystemSymbols maps identifier names reserved by the
// template runtime that trigger a warning when shadowed.
var reservedSystemSymbols = map[string]struct{}{
	"request": {},
	"req":     {},
	"r":       {},
	"state":   {},
	"s":       {},
	"props":   {},
	"p":       {},
	"T":       {},
	"LT":      {},
	"F":       {},
	"LF":      {},
}

// SymbolTable tracks variable definitions and their visibility within scopes.
// It forms a linked list through the parent pointer to represent nested
// scopes.
type SymbolTable struct {
	// parent points to the outer scope; nil for the top-level scope.
	parent *SymbolTable

	// symbols maps symbol names to their definitions in this scope.
	symbols map[string]Symbol

	// cachedNames stores the result of AllSymbolNames for reuse.
	cachedNames []string

	// namesCached indicates whether cachedNames is valid.
	namesCached bool
}

// NewSymbolTable creates a new symbol table with an optional parent scope.
//
// Takes parent (*SymbolTable) which specifies the enclosing scope for symbol
// lookup, or nil for a root scope.
//
// Returns *SymbolTable which is the newly created symbol table.
func NewSymbolTable(parent *SymbolTable) *SymbolTable {
	return &SymbolTable{
		parent:  parent,
		symbols: make(map[string]Symbol, initialSymbolCapacity),
	}
}

// Define adds a symbol to the current scope.
//
// Takes symbol (Symbol) which specifies the symbol to add.
func (st *SymbolTable) Define(symbol Symbol) {
	st.symbols[symbol.Name] = symbol
	st.namesCached = false
}

// Find searches for a symbol by name, starting in the current scope and
// traversing up to parent scopes until the symbol is found.
//
// Takes name (string) which specifies the symbol name to search for.
//
// Returns Symbol which is the found symbol if it exists in any accessible
// scope.
// Returns bool which indicates whether the symbol was found.
func (st *SymbolTable) Find(name string) (Symbol, bool) {
	current := st
	for current != nil {
		if symbol, found := current.symbols[name]; found {
			return symbol, true
		}
		current = current.parent
	}
	return Symbol{}, false
}

// AllSymbolNames returns a sorted list of all unique symbol names visible
// from the current scope.
//
// Returns []string which contains the symbol names from this scope and all
// parent scopes, sorted alphabetically.
func (st *SymbolTable) AllSymbolNames() []string {
	if st.namesCached {
		return st.cachedNames
	}

	names := make(map[string]struct{})
	current := st
	for current != nil {
		for name := range current.symbols {
			if _, exists := names[name]; !exists {
				names[name] = struct{}{}
			}
		}
		current = current.parent
	}

	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	slices.Sort(result)

	st.cachedNames = result
	st.namesCached = true
	return result
}

// TranslationKeySet holds translation keys used for compile-time validation.
// It provides O(1) lookup to check if a key exists.
type TranslationKeySet struct {
	// LocalKeys holds translation keys from the component's <i18n> block.
	LocalKeys map[string]struct{}

	// GlobalKeys holds keys that can be used with T() from
	// project-level translations.
	GlobalKeys map[string]struct{}
}

// HasLocalKey returns true if the key exists in local translations.
//
// Takes key (string) which specifies the translation key to look up.
//
// Returns bool which is true if the key exists, false otherwise.
func (t *TranslationKeySet) HasLocalKey(key string) bool {
	if t == nil || t.LocalKeys == nil {
		return false
	}
	_, ok := t.LocalKeys[key]
	return ok
}

// HasGlobalKey returns true if the key exists in global translations.
//
// Takes key (string) which specifies the translation key to look up.
//
// Returns bool which is true if the key exists, false otherwise.
func (t *TranslationKeySet) HasGlobalKey(key string) bool {
	if t == nil || t.GlobalKeys == nil {
		return false
	}
	_, ok := t.GlobalKeys[key]
	return ok
}

// AnalysisContext is a lightweight, passive data structure that provides
// all the necessary context for the TypeResolver to analyse an
// expression. It represents the state at a specific point in the AST
// walk.
type AnalysisContext struct {
	// Logger provides structured logging for analysis operations.
	Logger logger_domain.Logger

	// Symbols holds the symbol table for variable lookup in the current scope.
	Symbols *SymbolTable

	// Diagnostics holds the list of messages found during analysis.
	Diagnostics *[]*ast_domain.Diagnostic

	// TranslationKeys holds the set of valid translation keys for checking.
	TranslationKeys *TranslationKeySet

	// KnownNonNilExpressions tracks expressions that are guaranteed non-nil
	// in the current scope, keyed by canonical string form (e.g.,
	// "props.FloorPlan").
	//
	// Populated when entering p-if blocks that guard against nil.
	KnownNonNilExpressions map[string]bool

	// CurrentGoFullPackagePath is the full import path of the current package.
	CurrentGoFullPackagePath string

	// CurrentGoPackageName is the short name of the Go package being analysed.
	CurrentGoPackageName string

	// CurrentGoSourcePath is the file path of the Go source file being analysed.
	CurrentGoSourcePath string

	// SFCSourcePath is the path to the source file being analysed.
	SFCSourcePath string
}

// NewRootAnalysisContext creates the top-level context for a component's
// analysis.
//
// Takes diagnostics (*[]*ast_domain.Diagnostic) which collects any issues
// found during analysis.
// Takes goPackagePath (string) which specifies the full import path of the
// package.
// Takes goPackageName (string) which specifies the package name.
// Takes goSourcePath (string) which specifies the path to the Go source file.
// Takes sfcSourcePath (string) which specifies the path to the SFC source
// file.
//
// Returns *AnalysisContext which is the initialised root context ready for
// analysis.
func NewRootAnalysisContext(diagnostics *[]*ast_domain.Diagnostic, goPackagePath, goPackageName, goSourcePath, sfcSourcePath string) *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(nil),
		Diagnostics:              diagnostics,
		TranslationKeys:          nil,
		CurrentGoFullPackagePath: goPackagePath,
		CurrentGoPackageName:     goPackageName,
		CurrentGoSourcePath:      goSourcePath,
		SFCSourcePath:            sfcSourcePath,
		Logger:                   log,
	}
}

// ForChildScope creates a new context for a nested scope, such as inside a
// p-for loop. The new context inherits from the parent but has its own symbol
// table.
//
// Returns *AnalysisContext which is the child context with inherited settings
// and a fresh symbol table.
func (ac *AnalysisContext) ForChildScope() *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(ac.Symbols),
		Diagnostics:              ac.Diagnostics,
		CurrentGoFullPackagePath: ac.CurrentGoFullPackagePath,
		CurrentGoPackageName:     ac.CurrentGoPackageName,
		CurrentGoSourcePath:      ac.CurrentGoSourcePath,
		SFCSourcePath:            ac.SFCSourcePath,
		Logger:                   ac.Logger,
		TranslationKeys:          ac.TranslationKeys,
		KnownNonNilExpressions:   ac.KnownNonNilExpressions,
	}
}

// ForChildScopeWithNilGuards creates a child scope with extra nil guards.
// The new context keeps the parent's guards and adds the ones provided.
//
// Takes guards ([]string) which contains expressions known to be non-nil.
//
// Returns *AnalysisContext which is the child context with combined guards.
func (ac *AnalysisContext) ForChildScopeWithNilGuards(guards []string) *AnalysisContext {
	child := ac.ForChildScope()
	if len(guards) > 0 {
		child.KnownNonNilExpressions = make(map[string]bool)
		maps.Copy(child.KnownNonNilExpressions, ac.KnownNonNilExpressions)
		for _, g := range guards {
			child.KnownNonNilExpressions[g] = true
		}
	} else if ac.KnownNonNilExpressions != nil {
		child.KnownNonNilExpressions = ac.KnownNonNilExpressions
	}
	return child
}

// IsKnownNonNil checks if an expression is guaranteed non-nil in the current
// scope.
//
// Only direct matches are considered. If "props.FloorPlan" is guarded,
// accessing props.FloorPlan.URL is safe because the base (props.FloorPlan) is
// directly guarded. However, props.User being guarded does NOT make
// props.User.Profile safe if props.User.Profile is itself a pointer field that
// could be nil.
//
// Takes expressionString (string) which is the expression string to check.
//
// Returns bool which is true if the expression is directly guarded.
func (ac *AnalysisContext) IsKnownNonNil(expressionString string) bool {
	if ac.KnownNonNilExpressions == nil {
		return false
	}
	return ac.KnownNonNilExpressions[expressionString]
}

// ForNewPackageContext creates a new context for analysing content from a
// different component (e.g., slotted content). It resets the symbol table and
// updates the package context.
//
// Takes goPackagePath (string) which specifies the full Go package path.
// Takes goPackageName (string) which specifies the Go package name.
// Takes goSourcePath (string) which specifies the path to the Go source file.
// Takes sfcSourcePath (string) which specifies the path to the SFC
// source file.
//
// Returns *AnalysisContext which is the new context with inherited settings.
func (ac *AnalysisContext) ForNewPackageContext(goPackagePath, goPackageName, goSourcePath, sfcSourcePath string) *AnalysisContext {
	return &AnalysisContext{
		Symbols:                  NewSymbolTable(ac.Symbols),
		Diagnostics:              ac.Diagnostics,
		CurrentGoFullPackagePath: goPackagePath,
		CurrentGoPackageName:     goPackageName,
		CurrentGoSourcePath:      goSourcePath,
		SFCSourcePath:            sfcSourcePath,
		Logger:                   ac.Logger,
		TranslationKeys:          ac.TranslationKeys,
		KnownNonNilExpressions:   ac.KnownNonNilExpressions,
	}
}

// SetTranslationKeys sets the translation keys for this context.
//
// Takes keys (*TranslationKeySet) which provides the keys used for
// translation.
func (ac *AnalysisContext) SetTranslationKeys(keys *TranslationKeySet) {
	ac.TranslationKeys = keys
}

// SetLocalTranslationKeys updates the local translation keys for partial
// contexts.
//
// Takes localKeys (map[string]struct{}) which contains the translation keys
// to set.
func (ac *AnalysisContext) SetLocalTranslationKeys(localKeys map[string]struct{}) {
	if ac.TranslationKeys == nil {
		ac.TranslationKeys = &TranslationKeySet{}
	}
	ac.TranslationKeys.LocalKeys = localKeys
}

// WithSymbol adds a symbol with the given name and type expression,
// then returns the context. This builder-style method simplifies test
// setup by enabling method chaining.
//
// Takes name (string) which specifies the symbol name.
// Takes typeExpression (goast.Expr) which specifies the Go type expression for
// the symbol.
//
// Returns *AnalysisContext which is the same context, enabling method
// chaining.
func (ac *AnalysisContext) WithSymbol(name string, typeExpression goast.Expr) *AnalysisContext {
	ac.Symbols.Define(Symbol{
		Name:           name,
		CodeGenVarName: name,
		TypeInfo:       newSimpleTypeInfo(typeExpression),
	})
	return ac
}

// WithTypedSymbol adds a fully-specified symbol to the context and returns
// the context. This builder-style method simplifies test setup by enabling
// method chaining for complex symbol definitions.
//
// Takes symbol (Symbol) which specifies the complete symbol to define.
//
// Returns *AnalysisContext which is the same context, enabling method
// chaining.
func (ac *AnalysisContext) WithTypedSymbol(symbol Symbol) *AnalysisContext {
	ac.Symbols.Define(symbol)
	return ac
}

// addDiagnosticWithPath adds a diagnostic to the context's list with a custom
// source path.
//
// Takes sev (ast_domain.Severity) which sets the severity level.
// Takes message (string) which provides the message text.
// Takes expression (string) which contains the expression that caused the issue.
// Takes location (ast_domain.Location) which specifies where the issue occurred.
// Takes path (string) which sets a custom source path. If empty, uses the
// default path from the context.
func (ac *AnalysisContext) addDiagnosticWithPath(sev ast_domain.Severity, message, expression string, location ast_domain.Location, path, code string) {
	sourcePath := ac.SFCSourcePath
	if path != "" {
		sourcePath = path
	}
	*ac.Diagnostics = append(*ac.Diagnostics, ast_domain.NewDiagnosticWithCode(sev, message, expression, code, location, sourcePath))
}

// addDiagnostic is a convenience method for adding a diagnostic to the
// context's collection.
//
// Takes severity (ast_domain.Severity) which specifies the diagnostic
// severity.
// Takes message (string) which provides the diagnostic message.
// Takes expression (string) which identifies the expression being
// diagnosed.
// Takes location (ast_domain.Location) which specifies the source
// location.
// Takes annotations (*ast_domain.GoGeneratorAnnotation) which provides
// optional source path override.
func (ac *AnalysisContext) addDiagnostic(severity ast_domain.Severity, message, expression string, location ast_domain.Location, annotations *ast_domain.GoGeneratorAnnotation, code string) {
	sourcePath := ac.SFCSourcePath
	if annotations != nil && annotations.OriginalSourcePath != nil {
		sourcePath = *annotations.OriginalSourcePath
	}
	*ac.Diagnostics = append(*ac.Diagnostics, ast_domain.NewDiagnosticWithCode(severity, message, expression, code, location, sourcePath))
}

// addDiagnosticForExpression adds a diagnostic with accurate SourceLength.
//
// Takes severity (ast_domain.Severity) which specifies the diagnostic
// severity.
// Takes message (string) which provides the diagnostic message.
// Takes expression (ast_domain.Expression) which identifies the code
// element.
// Takes location (ast_domain.Location) which specifies where the
// diagnostic occurs.
// Takes annotations (*ast_domain.GoGeneratorAnnotation) which provides
// the original source path if available.
func (ac *AnalysisContext) addDiagnosticForExpression(
	severity ast_domain.Severity, message string, expression ast_domain.Expression,
	location ast_domain.Location, annotations *ast_domain.GoGeneratorAnnotation, code string,
) {
	sourcePath := ac.SFCSourcePath
	if annotations != nil && annotations.OriginalSourcePath != nil {
		sourcePath = *annotations.OriginalSourcePath
	}
	diagnostic := ast_domain.NewDiagnosticForExpression(severity, message, expression, location, sourcePath)
	diagnostic.Code = code
	*ac.Diagnostics = append(*ac.Diagnostics, diagnostic)
}

// PopulateRootContext creates the top-level context for a main page component.
// It sets up standard variable names for the page's state and props.
//
// When the component has a collection, this also adds a 'data' symbol to hold
// the collection items.
//
// Takes ctx (*AnalysisContext) which receives the symbol definitions.
// Takes typeResolver (*TypeResolver) which resolves types for the
// component's data.
// Takes virtualComponent (*annotator_dto.VirtualComponent) which
// provides the component source and collection metadata.
func PopulateRootContext(ctx *AnalysisContext, typeResolver *TypeResolver, virtualComponent *annotator_dto.VirtualComponent) {
	populateContext(ctx, typeResolver, virtualComponent, "pageData", "props", "")

	if virtualComponent.Source.HasCollection {
		dataType := inferDataType(typeResolver, virtualComponent)
		ctx.Symbols.Define(Symbol{
			Name:                "data",
			CodeGenVarName:      "data",
			TypeInfo:            dataType,
			SourceInvocationKey: "",
		})
		ctx.Logger.Trace("[PopulateRootContext] Injected 'data' symbol for collection page")
	}
}

// defineGlobalSymbols adds symbols that are available in all component scopes.
//
// These include request aliases (request, req, r), translation functions
// (T, LT), formatting functions (F, LF), Go built-in functions (len, cap,
// append, min, max), and type conversion functions (string, int, float, bool,
// and others).
//
// Takes ctx (*AnalysisContext) which provides the symbol table to populate.
// Takes typeResolver (*TypeResolver) which resolves types for the defined symbols.
func defineGlobalSymbols(ctx *AnalysisContext, typeResolver *TypeResolver) {
	requestTypeExpr := &goast.StarExpr{X: &goast.SelectorExpr{X: goast.NewIdent(pikoFacadeAlias), Sel: goast.NewIdent(requestDataStruct)}}
	requestTypeInfo := typeResolver.newResolvedTypeInfo(ctx, requestTypeExpr)
	ctx.Symbols.Define(Symbol{Name: "request", CodeGenVarName: "r", TypeInfo: requestTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "req", CodeGenVarName: "r", TypeInfo: requestTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "r", CodeGenVarName: "r", TypeInfo: requestTypeInfo, SourceInvocationKey: ""})

	imports := typeResolver.inspector.GetImportsForFile(ctx.CurrentGoFullPackagePath, ctx.CurrentGoSourcePath)
	templaterDtoCanonicalPath := imports[pikoFacadeAlias]

	translationFuncTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:          goast.NewIdent(typeBuiltInFunction),
		PackageAlias:            pikoFacadeAlias,
		CanonicalPackagePath:    templaterDtoCanonicalPath,
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}

	ctx.Symbols.Define(Symbol{Name: "T", CodeGenVarName: "r.T", TypeInfo: translationFuncTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "LT", CodeGenVarName: "r.LT", TypeInfo: translationFuncTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "LF", CodeGenVarName: "r.LF", TypeInfo: translationFuncTypeInfo, SourceInvocationKey: ""})

	builtInFuncTypeInfo := newSimpleTypeInfo(goast.NewIdent(typeBuiltInFunction))
	ctx.Symbols.Define(Symbol{Name: "F", CodeGenVarName: "F", TypeInfo: builtInFuncTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "len", CodeGenVarName: "len", TypeInfo: builtInFuncTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "cap", CodeGenVarName: "cap", TypeInfo: builtInFuncTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "append", CodeGenVarName: "append", TypeInfo: builtInFuncTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "min", CodeGenVarName: "min", TypeInfo: builtInFuncTypeInfo, SourceInvocationKey: ""})
	ctx.Symbols.Define(Symbol{Name: "max", CodeGenVarName: "max", TypeInfo: builtInFuncTypeInfo, SourceInvocationKey: ""})

	coercionFunctions := []string{
		"string", "int", "int64", "int32", "int16",
		"float", "float64", "float32",
		"bool", "decimal", "bigint",
	}
	for _, name := range coercionFunctions {
		ctx.Symbols.Define(Symbol{Name: name, CodeGenVarName: name, TypeInfo: builtInFuncTypeInfo, SourceInvocationKey: ""})
	}
}

// defineComponentSymbols adds the state and props symbols for a component.
//
// It defines both full names (state, props) and short aliases (s, p) in the
// symbol table. When the component has no script, it returns without action.
//
// Takes ctx (*AnalysisContext) which holds the symbol table to update.
// Takes typeResolver (*TypeResolver) which resolves type expressions
// to type info.
// Takes virtualComponent (*annotator_dto.VirtualComponent) which is
// the component to define symbols for.
// Takes stateVar (string) which is the variable name for state in
// code output.
// Takes propsVar (string) which is the variable name for props in
// code output.
// Takes sourceInvocationKey (string) which is the invocation key for
// partial contexts, or empty for root pages.
func defineComponentSymbols(ctx *AnalysisContext, typeResolver *TypeResolver, virtualComponent *annotator_dto.VirtualComponent, stateVar, propsVar, sourceInvocationKey string) {
	if virtualComponent.Source.Script == nil {
		return
	}
	if virtualComponent.Source.Script.RenderReturnTypeExpression != nil {
		stateTypeInfo := typeResolver.newResolvedTypeInfo(ctx, virtualComponent.Source.Script.RenderReturnTypeExpression)
		ctx.Symbols.Define(Symbol{Name: "state", CodeGenVarName: stateVar, TypeInfo: stateTypeInfo, SourceInvocationKey: sourceInvocationKey})
		ctx.Symbols.Define(Symbol{Name: "s", CodeGenVarName: stateVar, TypeInfo: stateTypeInfo, SourceInvocationKey: sourceInvocationKey})
	}
	if virtualComponent.Source.Script.PropsTypeExpression != nil {
		propsTypeInfo := typeResolver.newResolvedTypeInfo(ctx, virtualComponent.Source.Script.PropsTypeExpression)
		ctx.Symbols.Define(Symbol{Name: "props", CodeGenVarName: propsVar, TypeInfo: propsTypeInfo, SourceInvocationKey: sourceInvocationKey})
		ctx.Symbols.Define(Symbol{Name: "p", CodeGenVarName: propsVar, TypeInfo: propsTypeInfo, SourceInvocationKey: sourceInvocationKey})
	}
}

// defineAndValidateLocalFunctions adds local function symbols to the symbol
// table and warns when user-defined functions hide reserved names.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and
// symbol table.
// Takes virtualComponent (*annotator_dto.VirtualComponent) which
// contains the script AST to process.
func defineAndValidateLocalFunctions(ctx *AnalysisContext, virtualComponent *annotator_dto.VirtualComponent) {
	if virtualComponent.Source.Script == nil || virtualComponent.Source.Script.AST == nil {
		return
	}

	fset := virtualComponent.Source.Script.Fset
	scriptStartLocation := virtualComponent.Source.Script.ScriptStartLocation

	for _, declaration := range virtualComponent.Source.Script.AST.Decls {
		if functionDeclaration, ok := declaration.(*goast.FuncDecl); ok && functionDeclaration.Recv == nil {
			functionName := functionDeclaration.Name.Name

			if !token.IsExported(functionName) {
				continue
			}

			if _, isReserved := reservedSystemSymbols[functionName]; isReserved {
				message := fmt.Sprintf("User-defined function '%s' shadows a built-in Piko system symbol. This may lead to unexpected behaviour.", functionName)
				tokenPos := fset.Position(functionDeclaration.Name.Pos())
				relativeLocation := ast_domain.Location{Line: tokenPos.Line, Column: tokenPos.Column, Offset: 0}
				finalLocation := scriptStartLocation.Add(relativeLocation)
				ctx.addDiagnosticWithPath(ast_domain.Warning, message, functionName, finalLocation, virtualComponent.Source.SourcePath, annotator_dto.CodeVariableShadowing)
			} else if _, isBuiltIn := builtInFunctions[functionName]; isBuiltIn {
				message := fmt.Sprintf("User-defined function '%s' shadows a global built-in function. This may lead to unexpected behaviour.", functionName)
				tokenPos := fset.Position(functionDeclaration.Name.Pos())
				relativeLocation := ast_domain.Location{Line: tokenPos.Line, Column: tokenPos.Column, Offset: 0}
				finalLocation := scriptStartLocation.Add(relativeLocation)
				ctx.addDiagnosticWithPath(ast_domain.Warning, message, functionName, finalLocation, virtualComponent.Source.SourcePath, annotator_dto.CodeVariableShadowing)
			}

			ctx.Symbols.Define(Symbol{
				Name:           functionName,
				CodeGenVarName: functionName,
				TypeInfo: &ast_domain.ResolvedTypeInfo{
					TypeExpression:          goast.NewIdent(typeFunction),
					PackageAlias:            ctx.CurrentGoPackageName,
					CanonicalPackagePath:    ctx.CurrentGoFullPackagePath,
					IsSynthetic:             false,
					IsExportedPackageSymbol: true,
					InitialPackagePath:      "",
					InitialFilePath:         "",
				},
				SourceInvocationKey: "",
			})
		}
	}

	defineExportedConstantsAndVariables(ctx, virtualComponent)
}

// defineExportedConstantsAndVariables adds exported package-level constants
// and variables from the script block to the symbol table. These are needed
// in templates and require cross-package names when the component is embedded
// in another.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and
// symbol table.
// Takes virtualComponent (*annotator_dto.VirtualComponent) which
// contains the script AST to process.
func defineExportedConstantsAndVariables(ctx *AnalysisContext, virtualComponent *annotator_dto.VirtualComponent) {
	if virtualComponent.Source.Script == nil || virtualComponent.Source.Script.AST == nil {
		return
	}

	for _, declaration := range virtualComponent.Source.Script.AST.Decls {
		processConstVarDecl(ctx, declaration)
	}
}

// processConstVarDecl handles a single declaration if it is a const or var.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes declaration (goast.Decl) which is the declaration to check and process.
func processConstVarDecl(ctx *AnalysisContext, declaration goast.Decl) {
	genDecl, ok := declaration.(*goast.GenDecl)
	if !ok {
		return
	}
	if genDecl.Tok != token.CONST && genDecl.Tok != token.VAR {
		return
	}

	for _, spec := range genDecl.Specs {
		processValueSpec(ctx, spec)
	}
}

// processValueSpec handles a single value spec from a const or var
// declaration.
//
// Takes ctx (*AnalysisContext) which provides the analysis state.
// Takes spec (goast.Spec) which is the value spec to process.
func processValueSpec(ctx *AnalysisContext, spec goast.Spec) {
	valueSpec, ok := spec.(*goast.ValueSpec)
	if !ok {
		return
	}

	for _, name := range valueSpec.Names {
		defineExportedSymbol(ctx, name.Name, valueSpec.Type)
	}
}

// defineExportedSymbol defines a symbol if it is exported.
//
// When the symbol name is not exported, returns without doing anything.
//
// When typeExpr is nil, uses "any" as the type.
//
// Takes ctx (*AnalysisContext) which provides the analysis state and symbol
// table.
// Takes symbolName (string) which is the name of the symbol to define.
// Takes typeExpr (goast.Expr) which is the type expression for the symbol.
func defineExportedSymbol(ctx *AnalysisContext, symbolName string, typeExpr goast.Expr) {
	if !token.IsExported(symbolName) {
		return
	}

	if typeExpr == nil {
		typeExpr = goast.NewIdent("any")
	}

	ctx.Symbols.Define(Symbol{
		Name:           symbolName,
		CodeGenVarName: symbolName,
		TypeInfo: &ast_domain.ResolvedTypeInfo{
			TypeExpression:          typeExpr,
			PackageAlias:            ctx.CurrentGoPackageName,
			CanonicalPackagePath:    ctx.CurrentGoFullPackagePath,
			IsSynthetic:             false,
			IsExportedPackageSymbol: true,
			InitialPackagePath:      "",
			InitialFilePath:         "",
		},
		SourceInvocationKey: "",
	})
}

// inferDataType returns the Go type for the data variable on collection pages.
//
// This function was planned to scan Render() for piko.GetData[T](r) calls to
// find typed data. However, the design changed: components now use
// GetData[T](r) directly in their Render function and pass processed data to
// templates via state. The data symbol is kept for possible future use but
// templates do not currently use it.
//
// Returns *ast_domain.ResolvedTypeInfo which is map[string]interface{} as a
// fallback type.
func inferDataType(_ *TypeResolver, _ *annotator_dto.VirtualComponent) *ast_domain.ResolvedTypeInfo {
	return &ast_domain.ResolvedTypeInfo{
		TypeExpression: &goast.MapType{
			Key:   goast.NewIdent("string"),
			Value: goast.NewIdent("interface{}"),
		},
		PackageAlias:            "",
		CanonicalPackagePath:    "",
		IsSynthetic:             false,
		IsExportedPackageSymbol: false,
		InitialPackagePath:      "",
		InitialFilePath:         "",
	}
}

// populateContext sets up the symbol table for a component's scope.
//
// It defines all standard symbols available within the component's template.
// These include state, props, request, their aliases, and any top-level
// functions.
//
// Takes ctx (*AnalysisContext) which provides the analysis context to
// set up.
// Takes typeResolver (*TypeResolver) which resolves types for symbol
// definitions.
// Takes virtualComponent (*annotator_dto.VirtualComponent) which
// specifies the component.
// Takes stateVar (string) which names the state variable alias.
// Takes propsVar (string) which names the props variable alias.
// Takes sourceInvocationKey (string) which is the invocation key if
// this is a partial context, or empty for root pages.
func populateContext(ctx *AnalysisContext, typeResolver *TypeResolver, virtualComponent *annotator_dto.VirtualComponent, stateVar, propsVar, sourceInvocationKey string) {
	ctx.Logger.Trace("[populateContext] Populating full context.", logger_domain.String("component", virtualComponent.Source.SourcePath))
	defineGlobalSymbols(ctx, typeResolver)
	defineComponentSymbols(ctx, typeResolver, virtualComponent, stateVar, propsVar, sourceInvocationKey)
	defineAndValidateLocalFunctions(ctx, virtualComponent)
}

// populatePartialContext creates the context for a partial component that has
// been called. It builds unique variable names for this specific call to avoid
// name clashes in the output code.
//
// Takes ctx (*AnalysisContext) which holds the current analysis state.
// Takes typeResolver (*TypeResolver) which resolves types during
// analysis.
// Takes virtualComponent (*annotator_dto.VirtualComponent) which is
// the partial component.
// Takes partialInfo (*ast_domain.PartialInvocationInfo) which provides
// call details including the package name and unique key.
func populatePartialContext(ctx *AnalysisContext, typeResolver *TypeResolver, virtualComponent *annotator_dto.VirtualComponent, partialInfo *ast_domain.PartialInvocationInfo) {
	stateVar := fmt.Sprintf("%sData_%s", partialInfo.PartialPackageName, partialInfo.InvocationKey)
	propsVar := fmt.Sprintf("props_%s", partialInfo.InvocationKey)
	populateContext(ctx, typeResolver, virtualComponent, stateVar, propsVar, partialInfo.InvocationKey)
}
