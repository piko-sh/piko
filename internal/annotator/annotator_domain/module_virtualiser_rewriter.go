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

// Rewrites Go AST nodes for virtual components by transforming package names,
// import paths, and type references. Ensures virtual Go files have correct
// package structures and import statements for successful type checking and
// code generation.

import (
	"context"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// astRewriter applies documentation changes to an AST file.
type astRewriter struct {
	// vCtx holds the context for virtualisation operations.
	vCtx *virtualisationContext

	// vc is the virtual component being rewritten.
	vc *annotator_dto.VirtualComponent

	// ast holds the parsed Go source file that will be rewritten.
	ast *goast.File

	// pikoAliasToHash maps user import aliases (e.g., "card") to hashed package
	// names (e.g., "partials_card_abc123") for Piko imports. Used to rewrite
	// references like card.FormatPrice() to partials_card_abc123.FormatPrice().
	pikoAliasToHash map[string]string
}

// rewrite applies all changes to produce the final syntax tree.
//
// Returns *goast.File which is the rewritten syntax tree.
// Returns []string which contains any Piko import aliases that are shadowed by
// local variable declarations. The caller should emit diagnostic warnings for
// these.
// Returns error when import rewriting fails.
func (ar *astRewriter) rewrite(ctx context.Context) (*goast.File, []string, error) {
	ar.rewritePackageName()
	finalImports, useDecls, err := ar.rewriteImports(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("rewriting imports: %w", err)
	}
	shadowedAliases := ar.rewritePikoImportReferences()
	ar.rebuildDecls(finalImports, useDecls)
	return ar.ast, shadowedAliases, nil
}

// rewritePackageName updates the package name in the AST to use the hashed
// name.
func (ar *astRewriter) rewritePackageName() {
	if ar.ast.Name == nil {
		ar.ast.Name = goast.NewIdent(ar.vc.HashedName)
	}
	ar.ast.Name.Name = ar.vc.HashedName
}

// rewritePikoImportReferences walks the AST and rewrites all references to
// Piko import aliases (e.g., card.FormatPrice) to use the hashed package name
// instead (e.g., partials_card_abc123.FormatPrice). This prevents naming
// clashes when different components in a partial chain use the same alias for
// different imports.
//
// Returns []string which contains alias names that are shadowed by local
// variable declarations. The caller should emit diagnostic warnings for these.
func (ar *astRewriter) rewritePikoImportReferences() []string {
	if len(ar.pikoAliasToHash) == 0 {
		return nil
	}

	shadowedAliases := ar.detectShadowedAliases()

	goast.Inspect(ar.ast, func(n goast.Node) bool {
		selectorExpression, ok := n.(*goast.SelectorExpr)
		if !ok {
			return true
		}
		identifier, isIdent := selectorExpression.X.(*goast.Ident)
		if !isIdent {
			return true
		}
		if hashedName, found := ar.pikoAliasToHash[identifier.Name]; found {
			identifier.Name = hashedName
		}
		return true
	})

	return shadowedAliases
}

// detectShadowedAliases walks the AST to find any local variable declarations
// that shadow a Piko import alias.
//
// Returns []string which contains the names of shadowed aliases.
func (ar *astRewriter) detectShadowedAliases() []string {
	shadowedSet := make(map[string]bool)

	goast.Inspect(ar.ast, func(n goast.Node) bool {
		ar.collectShadowedFromNode(n, shadowedSet)
		return true
	})

	result := make([]string, 0, len(shadowedSet))
	for alias := range shadowedSet {
		result = append(result, alias)
	}
	return result
}

// collectShadowedFromNode checks a single AST node for Piko alias shadowing.
//
// Takes n (goast.Node) which is the AST node to inspect.
// Takes shadowedSet (map[string]bool) which collects identifiers that shadow
// the Piko alias.
func (ar *astRewriter) collectShadowedFromNode(n goast.Node, shadowedSet map[string]bool) {
	switch node := n.(type) {
	case *goast.AssignStmt:
		ar.collectShadowedFromAssign(node, shadowedSet)
	case *goast.ValueSpec:
		ar.collectShadowedFromValueSpec(node, shadowedSet)
	case *goast.FuncDecl:
		ar.collectShadowedFromFuncDecl(node, shadowedSet)
	case *goast.RangeStmt:
		ar.collectShadowedFromRangeStmt(node, shadowedSet)
	case *goast.ForStmt:
		ar.collectShadowedFromForStmt(node, shadowedSet)
	}
}

// collectShadowedFromAssign checks short variable declarations
// (card := something).
//
// Takes node (*goast.AssignStmt) which is the assignment to check.
// Takes shadowedSet (map[string]bool) which tracks shadowed identifiers.
func (ar *astRewriter) collectShadowedFromAssign(node *goast.AssignStmt, shadowedSet map[string]bool) {
	if node.Tok != token.DEFINE {
		return
	}
	for _, leftExpression := range node.Lhs {
		ar.markIfPikoAlias(leftExpression, shadowedSet)
	}
}

// collectShadowedFromValueSpec checks var declarations (var card = something).
//
// Takes node (*goast.ValueSpec) which contains the variable declaration to
// check.
// Takes shadowedSet (map[string]bool) which tracks shadowed identifiers.
func (ar *astRewriter) collectShadowedFromValueSpec(node *goast.ValueSpec, shadowedSet map[string]bool) {
	for _, name := range node.Names {
		ar.markIdentIfPikoAlias(name, shadowedSet)
	}
}

// collectShadowedFromFuncDecl checks function parameters and named return
// values for shadowed identifiers.
//
// Takes node (*goast.FuncDecl) which is the function declaration to inspect.
// Takes shadowedSet (map[string]bool) which accumulates shadowed identifiers.
func (ar *astRewriter) collectShadowedFromFuncDecl(node *goast.FuncDecl, shadowedSet map[string]bool) {
	if node.Type == nil {
		return
	}
	ar.collectShadowedFromFieldList(node.Type.Params, shadowedSet)
	ar.collectShadowedFromFieldList(node.Type.Results, shadowedSet)
}

// collectShadowedFromFieldList checks a field list (params or results) for
// shadowing.
//
// Takes fields (*goast.FieldList) which contains the parameters or results to
// check.
// Takes shadowedSet (map[string]bool) which tracks identifiers that shadow
// piko aliases.
func (ar *astRewriter) collectShadowedFromFieldList(fields *goast.FieldList, shadowedSet map[string]bool) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		for _, name := range field.Names {
			ar.markIdentIfPikoAlias(name, shadowedSet)
		}
	}
}

// collectShadowedFromRangeStmt checks range loop variables
// (for card := range items).
//
// Takes node (*goast.RangeStmt) which is the range statement to check.
// Takes shadowedSet (map[string]bool) which tracks shadowed identifiers.
func (ar *astRewriter) collectShadowedFromRangeStmt(node *goast.RangeStmt, shadowedSet map[string]bool) {
	ar.markIfPikoAlias(node.Key, shadowedSet)
	ar.markIfPikoAlias(node.Value, shadowedSet)
}

// collectShadowedFromForStmt checks for loop init statements (for card := 0;
// ...).
//
// Takes node (*goast.ForStmt) which is the for statement to inspect.
// Takes shadowedSet (map[string]bool) which collects shadowed variable names.
func (ar *astRewriter) collectShadowedFromForStmt(node *goast.ForStmt, shadowedSet map[string]bool) {
	assign, ok := node.Init.(*goast.AssignStmt)
	if !ok || assign.Tok != token.DEFINE {
		return
	}
	for _, leftExpression := range assign.Lhs {
		ar.markIfPikoAlias(leftExpression, shadowedSet)
	}
}

// markIfPikoAlias marks the expression as shadowed if it is an identifier
// that is a Piko alias.
//
// Takes expression (goast.Expr) which is the expression to check.
// Takes shadowedSet (map[string]bool) which tracks shadowed identifiers.
func (ar *astRewriter) markIfPikoAlias(expression goast.Expr, shadowedSet map[string]bool) {
	if identifier, ok := expression.(*goast.Ident); ok {
		ar.markIdentIfPikoAlias(identifier, shadowedSet)
	}
}

// markIdentIfPikoAlias marks the identifier as shadowed if it is a Piko alias.
//
// Takes identifier (*goast.Ident) which is the identifier to check.
// Takes shadowedSet (map[string]bool) which collects shadowed alias names.
func (ar *astRewriter) markIdentIfPikoAlias(identifier *goast.Ident, shadowedSet map[string]bool) {
	if _, isPikoAlias := ar.pikoAliasToHash[identifier.Name]; isPikoAlias {
		shadowedSet[identifier.Name] = true
	}
}

// rebuildDecls rebuilds the AST declarations with the given imports.
//
// Takes finalImports (map[string]*goast.ImportSpec) which holds the imports
// to include in the rebuilt AST.
// Takes useDecls ([]goast.Decl) which provides extra declarations to add
// before the existing ones.
func (ar *astRewriter) rebuildDecls(finalImports map[string]*goast.ImportSpec, useDecls []goast.Decl) {
	ar.ast.Imports = nil
	finalDecls := make([]goast.Decl, 0, len(ar.ast.Decls))
	for _, declaration := range ar.ast.Decls {
		if gen, ok := declaration.(*goast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		finalDecls = append(finalDecls, declaration)
	}
	finalDecls = append(useDecls, finalDecls...)
	if len(finalImports) > 0 {
		importDecl := buildImportDecl(finalImports)
		ar.ast.Decls = append([]goast.Decl{importDecl}, finalDecls...)
	} else {
		ar.ast.Decls = finalDecls
	}
}

// resolvePikoImport resolves a .pk import path to its canonical Go package
// path and hashed package name by looking up the component graph.
//
// Takes importPath (string) which is the .pk import path to resolve.
//
// Returns canonicalPath (string) which is the resolved Go package path.
// Returns hashedName (string) which is the hashed component package name.
// Returns err (error) when the import path cannot be resolved or the
// component is not found in the graph.
func (ar *astRewriter) resolvePikoImport(ctx context.Context, importPath string) (canonicalPath, hashedName string, err error) {
	resolvedPath, err := ar.vCtx.resolver.ResolvePKPath(ctx, importPath, ar.vc.Source.SourcePath)
	if err != nil {
		return "", "", fmt.Errorf("could not resolve import '%s'", importPath)
	}
	targetHash, ok := ar.vCtx.graph.PathToHashedName[resolvedPath]
	if !ok {
		return "", "", fmt.Errorf("internal error: could not find hash for resolved path '%s'", resolvedPath)
	}
	targetVirtualComp, ok := ar.vCtx.virtualModule.ComponentsByHash[targetHash]
	if !ok {
		return "", "", fmt.Errorf("internal error: could not find virtual component for hash '%s'", targetHash)
	}
	return targetVirtualComp.CanonicalGoPackagePath, targetVirtualComp.HashedName, nil
}

// rewriteImports is the core of the virtualisation logic.
//
// It rewrites all Piko component imports (.pk) to their canonical Go package
// paths. For standard Go packages that are only used in the template (not in
// the script), it keeps the named import for the TypeInspector while also
// injecting a "use" declaration (e.g., `var _ = alias.Member`) to prevent
// "imported and not used" errors from the Go toolchain.
//
// Returns map[string]*goast.ImportSpec which contains the final import map.
// Returns []goast.Decl which contains the generated use declarations.
// Returns error when processing of imports fails.
func (ar *astRewriter) rewriteImports(ctx context.Context) (map[string]*goast.ImportSpec, []goast.Decl, error) {
	templatePackageUses := ar.discoverTemplateUses()
	scriptUses := ar.discoverScriptUses()

	finalImports := make(map[string]*goast.ImportSpec)
	var useDecls []goast.Decl

	if err := ar.processOriginalImports(ctx, templatePackageUses, scriptUses, finalImports, &useDecls); err != nil {
		return nil, nil, fmt.Errorf("processing original imports: %w", err)
	}

	if err := ar.processCommentOnlyPikoImports(ctx, finalImports, &useDecls); err != nil {
		return nil, nil, fmt.Errorf("processing comment-only Piko imports: %w", err)
	}

	return finalImports, useDecls, nil
}

// templateMemberInfo tracks how a member is used in a template.
type templateMemberInfo struct {
	// isCall indicates whether the member is used as a function call. When true,
	// no use declaration is created because referencing a generic function without
	// instantiation is not valid Go.
	isCall bool
}

// discoverTemplateUses finds all package aliases and the members accessed
// through each alias in the template.
//
// Returns map[string]map[string]templateMemberInfo which maps each alias name
// to the set of member names used with that alias, along with how they are
// used.
func (ar *astRewriter) discoverTemplateUses() map[string]map[string]templateMemberInfo {
	templatePackageUses := make(map[string]map[string]templateMemberInfo)
	if ar.vc.Source.Template == nil {
		return templatePackageUses
	}

	ar.vc.Source.Template.Walk(func(node *ast_domain.TemplateNode) bool {
		walkModuleNodeExpressions(node, func(expression ast_domain.Expression) {
			ar.recordExpressionUsage(expression, templatePackageUses)
		})
		return true
	})

	return templatePackageUses
}

// recordExpressionUsage records package usage for a single expression.
//
// Takes expression (ast_domain.Expression) which is the expression to analyse.
// Takes uses (map[string]map[string]templateMemberInfo) which collects
// package member usage.
func (*astRewriter) recordExpressionUsage(expression ast_domain.Expression, uses map[string]map[string]templateMemberInfo) {
	rootIdent, memberName, isCall := getModuleRootAndMemberWithCallInfo(expression)
	if rootIdent == nil {
		return
	}

	if _, ok := uses[rootIdent.Name]; !ok {
		uses[rootIdent.Name] = make(map[string]templateMemberInfo)
	}
	if memberName == "" {
		return
	}

	existing := uses[rootIdent.Name][memberName]
	if !existing.isCall || isCall {
		uses[rootIdent.Name][memberName] = templateMemberInfo{isCall: isCall}
	}
}

// discoverScriptUses finds all package names used in the Go script.
//
// Returns map[string]bool which holds the set of package names found.
func (ar *astRewriter) discoverScriptUses() map[string]bool {
	scriptUses := make(map[string]bool)
	goast.Inspect(ar.ast, func(n goast.Node) bool {
		selectorExpression, ok := n.(*goast.SelectorExpr)
		if !ok {
			return true
		}
		if identifier, isIdent := selectorExpression.X.(*goast.Ident); isIdent {
			scriptUses[identifier.Name] = true
		}
		return true
	})
	return scriptUses
}

// processOriginalImports checks each original import and decides whether to
// keep it.
//
// Takes templatePackageUses (map[string]map[string]templateMemberInfo) which
// tracks package usage in templates.
// Takes scriptUses (map[string]bool) which shows which packages are used in
// scripts.
// Takes finalImports (map[string]*goast.ImportSpec) which collects the imports
// to keep.
// Takes useDecls (*[]goast.Decl) which gathers use declarations for imports
// that are only used in templates.
//
// Returns error when an import path cannot be resolved.
func (ar *astRewriter) processOriginalImports(
	ctx context.Context,
	templatePackageUses map[string]map[string]templateMemberInfo,
	scriptUses map[string]bool,
	finalImports map[string]*goast.ImportSpec,
	useDecls *[]goast.Decl,
) error {
	for _, impSpec := range ar.ast.Imports {
		path := strings.Trim(impSpec.Path.Value, `"`)
		isPikoImport := strings.HasSuffix(strings.ToLower(path), ".pk")
		alias := getAliasFromSpec(impSpec)

		isUsedInScript := scriptUses[alias]
		_, isUsedInTemplate := templatePackageUses[alias]

		alias, isUsedInTemplate = ar.handleSideEffectImport(
			alias, path, isPikoImport, templatePackageUses, isUsedInTemplate,
		)

		if !isUsedInScript && !isUsedInTemplate {
			continue
		}

		canonicalPath, targetHashedName, err := ar.resolveImportPath(ctx, path, alias, isPikoImport)
		if err != nil {
			return fmt.Errorf("resolving import path %q (alias %q): %w", path, alias, err)
		}

		finalImports[canonicalPath] = createImportSpec(canonicalPath, targetHashedName)

		if isPikoImport && alias != targetHashedName {
			ar.pikoAliasToHash[alias] = targetHashedName
		}

		ar.maybeAddTemplateOnlyUseDecl(
			alias, isPikoImport, isUsedInScript, isUsedInTemplate, templatePackageUses, useDecls,
		)
	}
	return nil
}

// handleSideEffectImport processes side-effect imports like `import _ "..."`
// in templates.
//
// Takes alias (string) which is the import alias, usually "_".
// Takes path (string) which is the import path.
// Takes isPikoImport (bool) which indicates if this is a Piko import.
// Takes templatePackageUses (map[string]map[string]templateMemberInfo) which
// tracks package usage in templates.
// Takes isUsedInTemplate (bool) which indicates if the import is used in a
// template.
//
// Returns string which is the resolved import alias.
// Returns bool which indicates if the import is used in a template.
func (*astRewriter) handleSideEffectImport(
	alias, path string,
	isPikoImport bool,
	templatePackageUses map[string]map[string]templateMemberInfo,
	isUsedInTemplate bool,
) (string, bool) {
	if alias != sideEffectImportName || isPikoImport {
		return alias, isUsedInTemplate
	}

	defaultPackageName := path
	if lastSlash := strings.LastIndex(path, "/"); lastSlash != -1 {
		defaultPackageName = path[lastSlash+1:]
	}

	if _, ok := templatePackageUses[defaultPackageName]; ok {
		return defaultPackageName, true
	}
	return alias, isUsedInTemplate
}

// resolveImportPath resolves the canonical path and target name for an import.
//
// Takes path (string) which is the import path to resolve.
// Takes alias (string) which is the import alias used in the source.
// Takes isPikoImport (bool) which indicates whether this is a Piko
// component import (.pk suffix).
//
// Returns canonicalPath (string) which is the resolved canonical Go
// package path.
// Returns targetName (string) which is the hashed package name for Piko
// imports, or the alias for standard Go imports.
// Returns err (error) when a Piko import cannot be resolved.
func (ar *astRewriter) resolveImportPath(ctx context.Context, path, alias string, isPikoImport bool) (canonicalPath, targetName string, err error) {
	if isPikoImport {
		return ar.resolvePikoImport(ctx, path)
	}
	return path, alias, nil
}

// maybeAddTemplateOnlyUseDecl adds a use declaration when a package is only
// used in the template and not in the script.
//
// Takes alias (string) which specifies the package alias to check.
// Takes isPikoImport (bool) which indicates if this is a Piko framework import.
// Takes isUsedInScript (bool) which indicates if the package is used in the
// script section.
// Takes isUsedInTemplate (bool) which indicates if the package is used in the
// template section.
// Takes templatePackageUses (map[string]map[string]templateMemberInfo) which
// maps package aliases to the members they use.
// Takes useDecls (*[]goast.Decl) which collects the generated use declarations.
func (*astRewriter) maybeAddTemplateOnlyUseDecl(
	alias string,
	isPikoImport, isUsedInScript, isUsedInTemplate bool,
	templatePackageUses map[string]map[string]templateMemberInfo,
	useDecls *[]goast.Decl,
) {
	if !isUsedInTemplate || isUsedInScript || isPikoImport {
		return
	}

	members, ok := templatePackageUses[alias]
	if !ok || len(members) == 0 {
		return
	}

	var memberToUse string
	for member, info := range members {
		if !info.isCall {
			memberToUse = member
			break
		}
	}

	if memberToUse == "" {
		return
	}

	if useDecl := createUseDecl(alias, memberToUse); useDecl != nil {
		*useDecls = append(*useDecls, useDecl)
	}
}

// processCommentOnlyPikoImports handles Piko imports that appear only in
// comments.
//
// Takes finalImports (map[string]*goast.ImportSpec) which collects the import
// specs to include in the output.
// Takes useDecls (*[]goast.Decl) which collects use declarations for the
// imports.
//
// Returns error when a Piko import path cannot be resolved.
func (ar *astRewriter) processCommentOnlyPikoImports(
	ctx context.Context,
	finalImports map[string]*goast.ImportSpec,
	useDecls *[]goast.Decl,
) error {
	for _, pikoImport := range ar.vc.Source.PikoImports {
		canonicalPath, targetHashedName, err := ar.resolvePikoImport(ctx, pikoImport.Path)
		if err != nil {
			return fmt.Errorf("resolving Piko import %q: %w", pikoImport.Path, err)
		}
		if _, exists := finalImports[canonicalPath]; exists {
			continue
		}

		finalImports[canonicalPath] = createImportSpec(canonicalPath, targetHashedName)

		if pikoImport.Alias != targetHashedName {
			ar.pikoAliasToHash[pikoImport.Alias] = targetHashedName
		}

		if useDecl := createUseDecl(targetHashedName, renderFuncName); useDecl != nil {
			*useDecls = append(*useDecls, useDecl)
		}
	}
	return nil
}

// newASTRewriter creates a new AST rewriter for the given virtual component.
//
// Takes vCtx (*virtualisationContext) which provides the virtualisation state.
// Takes vc (*annotator_dto.VirtualComponent) which specifies the component to
// rewrite.
//
// Returns *astRewriter which holds a deep copy of the component's AST.
func newASTRewriter(vCtx *virtualisationContext, vc *annotator_dto.VirtualComponent) *astRewriter {
	return &astRewriter{
		vCtx:            vCtx,
		vc:              vc,
		ast:             deepCopyASTFile(vc.Source.Script.AST),
		pikoAliasToHash: make(map[string]string),
	}
}

// deepCopyASTFile creates a full copy of an AST file by printing it to text
// and parsing it again.
//
// Takes original (*goast.File) which is the AST file to copy.
//
// Returns *goast.File which is a separate copy that does not share data with
// the original.
//
// Panics if the AST cannot be printed or parsed. This indicates an internal
// error.
func deepCopyASTFile(original *goast.File) *goast.File {
	if original == nil {
		return nil
	}
	var buffer strings.Builder
	fset := token.NewFileSet()
	if err := printer.Fprint(&buffer, fset, original); err != nil {
		panic(fmt.Sprintf("internal error: failed to print AST during deep copy: %v", err))
	}
	newFset := token.NewFileSet()
	newFile, err := parser.ParseFile(newFset, "", buffer.String(), parser.ParseComments)
	if err != nil {
		panic(fmt.Sprintf("internal error: failed to re-parse AST during deep copy: %v", err))
	}
	return newFile
}

// createUseDecl creates a variable declaration AST node that references a
// package member.
//
// Takes alias (string) which is the package alias or import name.
// Takes member (string) which is the exported symbol to reference.
//
// Returns goast.Decl which is the declaration node, or nil if the alias is
// empty, blank, dot, or if the member is empty.
func createUseDecl(alias, member string) goast.Decl {
	if alias == "" || alias == "_" || alias == "." || member == "" {
		return nil
	}
	return &goast.GenDecl{
		Tok: token.VAR,
		Specs: []goast.Spec{
			&goast.ValueSpec{
				Names:  []*goast.Ident{goast.NewIdent("_")},
				Values: []goast.Expr{&goast.SelectorExpr{X: goast.NewIdent(alias), Sel: goast.NewIdent(member)}},
			},
		},
	}
}

// buildImportDecl creates an import declaration from the given import specs.
//
// Takes imports (map[string]*goast.ImportSpec) which maps import paths to
// their specs.
//
// Returns *goast.GenDecl which contains the import specs sorted by path.
func buildImportDecl(imports map[string]*goast.ImportSpec) *goast.GenDecl {
	declaration := &goast.GenDecl{Tok: token.IMPORT, Specs: make([]goast.Spec, 0, len(imports))}
	if len(imports) > 1 {
		declaration.Lparen = 1
	}
	sortedPaths := make([]string, 0, len(imports))
	for path := range imports {
		sortedPaths = append(sortedPaths, path)
	}
	slices.Sort(sortedPaths)
	for _, path := range sortedPaths {
		declaration.Specs = append(declaration.Specs, imports[path])
	}
	return declaration
}

// getAliasFromSpec extracts the alias name from an import specification.
//
// Takes spec (*goast.ImportSpec) which is the import to extract the alias from.
//
// Returns string which is the explicit alias if set, or the last part of
// the import path if no alias is given.
func getAliasFromSpec(spec *goast.ImportSpec) string {
	if spec.Name != nil {
		return spec.Name.Name
	}
	path := strings.Trim(spec.Path.Value, `"`)
	if lastSlash := strings.LastIndex(path, "/"); lastSlash != -1 {
		return path[lastSlash+1:]
	}
	return path
}

// walkModuleNodeExpressions walks a template node and calls the visit function
// for each expression found in directives and attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to walk.
// Takes visit (func(...)) which is called for each expression found.
func walkModuleNodeExpressions(node *ast_domain.TemplateNode, visit func(ast_domain.Expression)) {
	if node == nil {
		return
	}

	visitDirectiveExpr(node.DirIf, visit)
	visitDirectiveExpr(node.DirElseIf, visit)
	visitDirectiveExpr(node.DirFor, visit)
	visitDirectiveExpr(node.DirShow, visit)
	visitDirectiveExpr(node.DirModel, visit)
	visitDirectiveExpr(node.DirClass, visit)
	visitDirectiveExpr(node.DirStyle, visit)
	visitDirectiveExpr(node.DirText, visit)
	visitDirectiveExpr(node.DirHTML, visit)
	visitDirectiveExpr(node.DirKey, visit)

	visitDynamicAttrExprs(node.DynamicAttributes, visit)
	visitRichTextExprs(node.RichText, visit)
	visitEventExprs(node.OnEvents, visit)
	visitEventExprs(node.CustomEvents, visit)
	visitBindExprs(node.Binds, visit)
}

// visitDirectiveExpr calls the visit function with the expression from a
// directive if both the directive and its expression are not nil.
//
// Takes directive (*ast_domain.Directive) which is the directive to check.
// Takes visit (func(...)) which is called with the expression if present.
func visitDirectiveExpr(directive *ast_domain.Directive, visit func(ast_domain.Expression)) {
	if directive != nil && directive.Expression != nil {
		visit(directive.Expression)
	}
}

// visitDynamicAttrExprs calls a visitor function on each expression in the
// given dynamic attributes.
//
// Takes attrs ([]ast_domain.DynamicAttribute) which contains the dynamic
// attributes to process.
// Takes visit (func(...)) which is called for each expression that is not nil.
func visitDynamicAttrExprs(attrs []ast_domain.DynamicAttribute, visit func(ast_domain.Expression)) {
	for i := range attrs {
		if attrs[i].Expression != nil {
			visit(attrs[i].Expression)
		}
	}
}

// visitRichTextExprs calls the visitor function for each expression found in
// the given rich text parts.
//
// Takes parts ([]ast_domain.TextPart) which contains the text parts to scan.
// Takes visit (func(...)) which is called for each expression that is not a
// literal.
func visitRichTextExprs(parts []ast_domain.TextPart, visit func(ast_domain.Expression)) {
	for i := range parts {
		if !parts[i].IsLiteral && parts[i].Expression != nil {
			visit(parts[i].Expression)
		}
	}
}

// visitEventExprs calls the visit function for each expression found in the
// event directives.
//
// Takes eventMaps (map[string][]ast_domain.Directive) which contains event
// directives grouped by event type.
// Takes visit (func(...)) which is called for each expression that is not nil.
func visitEventExprs(eventMaps map[string][]ast_domain.Directive, visit func(ast_domain.Expression)) {
	for _, events := range eventMaps {
		for i := range events {
			if events[i].Expression != nil {
				visit(events[i].Expression)
			}
		}
	}
}

// visitBindExprs calls the visit function on each expression found in the
// given bind directives, skipping any nil directives or expressions.
//
// Takes binds (map[string]*ast_domain.Directive) which contains the bind
// directives to process.
// Takes visit (func(...)) which is called for each non-nil expression.
func visitBindExprs(binds map[string]*ast_domain.Directive, visit func(ast_domain.Expression)) {
	for _, bind := range binds {
		if bind != nil && bind.Expression != nil {
			visit(bind.Expression)
		}
	}
}
