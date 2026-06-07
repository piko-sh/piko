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
	"context"
	goast "go/ast"
	"go/token"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// maxPartialScopeDepth bounds the recursion used to classify and collect partial
	// invocations across a template tree. It mirrors the limit applied when the tree is
	// built and walked elsewhere, so a pathologically deep template cannot overflow the
	// stack here.
	maxPartialScopeDepth = 10000
)

// buildPartialRenderCalls creates the top-level render calls for partial invocations. It
// skips any invocation that depends on loop variables, as those must be rendered inside
// the loop body by the for-emitter.
//
// Takes result (*annotator_dto.AnnotationResult) which provides the annotation context
// for code generation.
// Takes sortedInvocations ([]*annotator_dto.PartialInvocation) which lists the partial
// invocations to process in order.
//
// Returns []goast.Stmt which contains the generated render call statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics produced.
func (b *astBuilder) buildPartialRenderCalls(
	result *annotator_dto.AnnotationResult,
	sortedInvocations []*annotator_dto.PartialInvocation,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var statements []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	invocationsByKey := make(map[string]*annotator_dto.PartialInvocation, len(sortedInvocations))
	for _, inv := range sortedInvocations {
		invocationsByKey[inv.InvocationKey] = inv
	}

	guardedKeys := b.emitter.conditionallyGuardedKeys()

	_, l := logger_domain.From(context.Background(), log)
	for _, invocation := range sortedInvocations {
		visited := make(map[string]bool)
		if isPartialInvocationLoopDependent(invocation, invocationsByKey, visited) {
			l.Trace("[Generator] Skipping hoisted render call for loop-dependent partial",
				logger_domain.String("invocationKey", invocation.InvocationKey),
			)
			continue
		}

		if guardedKeys[invocation.InvocationKey] {
			l.Trace("[Generator] Skipping hoisted render call for conditionally-guarded partial",
				logger_domain.String("invocationKey", invocation.InvocationKey),
			)
			continue
		}

		l.Trace("[Generator] Generating hoisted render call for static partial",
			logger_domain.String("invocationKey", invocation.InvocationKey),
		)

		pInfo := partialInfoFromInvocation(invocation)

		renderStmts, renderDiags := b.emitPartialRenderCall(pInfo, result)
		statements = append(statements, renderStmts...)
		allDiags = append(allDiags, renderDiags...)
	}

	return statements, allDiags
}

// emitPartialRenderCall builds the Go statements for a partial invocation.
//
// Takes pInfo (*ast_domain.PartialInvocationInfo) which describes the partial to call.
// Takes result (*annotator_dto.AnnotationResult) which collects annotations.
//
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any problems found while building
// props.
func (b *astBuilder) emitPartialRenderCall(
	pInfo *ast_domain.PartialInvocationInfo,
	result *annotator_dto.AnnotationResult,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	key := pInfo.InvocationKey
	packageName := pInfo.PartialPackageName

	reqVar := cachedIdent("r")

	propsVar, propsStmts, propsDiags := b.buildPartialProps(pInfo, result, key)

	renderStmts := buildPartialRenderCallBlock(reqVar, propsVar, key, packageName)

	statements := make([]goast.Stmt, 0, len(propsStmts)+len(renderStmts))
	statements = append(statements, propsStmts...)
	statements = append(statements, renderStmts...)

	return statements, propsDiags
}

// buildPartialProps builds the props variable for a partial invocation.
//
// Takes pInfo (*ast_domain.PartialInvocationInfo) which contains the partial invocation
// details.
// Takes result (*annotator_dto.AnnotationResult) which provides annotation context.
// Takes key (string) which identifies the props variable uniquely.
//
// Returns *goast.Ident which is the identifier for the props variable.
// Returns []goast.Stmt which contains the prerequisite statements including the props
// assignment.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from building the props
// literal.
func (b *astBuilder) buildPartialProps(
	pInfo *ast_domain.PartialInvocationInfo,
	result *annotator_dto.AnnotationResult,
	key string,
) (*goast.Ident, []goast.Stmt, []*ast_domain.Diagnostic) {
	propsVar := cachedIdent("props_" + key)
	propsLit, propsPrereqs, propsDiags := b.buildPropsLiteral(pInfo, result, pInfo.InvokerPackageAlias)

	propsPrereqs = append(propsPrereqs, defineAndAssign(propsVar.Name, propsLit))
	return propsVar, propsPrereqs, propsDiags
}

// buildPropsLiteral creates the props struct literal for a partial call.
//
// Takes pInfo (*ast_domain.PartialInvocationInfo) which provides the partial invocation
// details including passed props and location.
// Takes result (*annotator_dto.AnnotationResult) which contains the virtual module with
// component definitions.
// Takes currentPackageHash (string) which identifies the current package for type
// resolution.
//
// Returns *goast.CompositeLit which is the constructed props literal.
// Returns []goast.Stmt which contains prerequisite statements needed before the literal.
// Returns []*ast_domain.Diagnostic which contains any errors encountered during
// construction.
func (b *astBuilder) buildPropsLiteral(
	pInfo *ast_domain.PartialInvocationInfo,
	result *annotator_dto.AnnotationResult,
	currentPackageHash string,
) (*goast.CompositeLit, []goast.Stmt, []*ast_domain.Diagnostic) {
	var allPrereqs []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	partialComponent, ok := result.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	if !ok {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			"Internal Emitter Error: Could not find virtual component for partial "+pInfo.PartialPackageName,
			pInfo.PartialAlias,
			pInfo.Location,
			"",
		)
		propsLit := &goast.CompositeLit{
			Type: &goast.SelectorExpr{X: cachedIdent(pInfo.PartialPackageName), Sel: cachedIdent("Props")},
		}
		return propsLit, nil, []*ast_domain.Diagnostic{diagnostic}
	}

	rawPropsTypeExpr := partialComponent.Source.Script.PropsTypeExpression
	if rawPropsTypeExpr == nil {
		rawPropsTypeExpr = &goast.SelectorExpr{X: cachedIdent(facadePackageName), Sel: cachedIdent(NoPropsTypeName)}
	}

	finalPropsTypeExpr := determinePropsTypeExpr(rawPropsTypeExpr, pInfo.PartialPackageName, currentPackageHash)
	propsLit := &goast.CompositeLit{Type: finalPropsTypeExpr, Elts: []goast.Expr{}}

	propKeys := slices.Sorted(maps.Keys(pInfo.PassedProps))

	for _, propName := range propKeys {
		propVal := pInfo.PassedProps[propName]
		goExpr, prereqs, diagnostics := b.expressionEmitter.emit(propVal.Expression)
		allPrereqs = append(allPrereqs, prereqs...)
		allDiags = append(allDiags, diagnostics...)

		fieldName := propVal.GoFieldName
		if fieldName == "" {
			fieldName = propToField(propName)
		}

		kv := &goast.KeyValueExpr{Key: cachedIdent(fieldName), Value: goExpr}
		propsLit.Elts = append(propsLit.Elts, kv)
	}

	return propsLit, allPrereqs, allDiags
}

// isPartialInvocationLoopDependent checks if a partial invocation depends on loop
// variables, either directly via its props or indirectly via its dependencies (e.g., a
// nested partial whose parent is inside a p-for loop).
//
// Takes invocation (*annotator_dto.PartialInvocation) which is the partial invocation to
// check for loop dependencies.
// Takes invocationsByKey (map[string]*annotator_dto.PartialInvocation) which maps
// invocation keys to their invocations, used to look up dependencies.
// Takes visited (map[string]bool) which tracks already-visited invocation keys to prevent
// infinite recursion in circular dependency scenarios.
//
// Returns bool which is true if any passed prop or request override is loop dependent, or
// if any dependency is loop dependent.
func isPartialInvocationLoopDependent(
	invocation *annotator_dto.PartialInvocation,
	invocationsByKey map[string]*annotator_dto.PartialInvocation,
	visited map[string]bool,
) bool {
	if visited[invocation.InvocationKey] {
		return false
	}
	visited[invocation.InvocationKey] = true

	for _, propVal := range invocation.PassedProps {
		if propVal.IsLoopDependent {
			return true
		}
	}

	for _, propVal := range invocation.RequestOverrides {
		if propVal.IsLoopDependent {
			return true
		}
	}

	for _, depKey := range invocation.DependsOn {
		depInvocation, ok := invocationsByKey[depKey]
		if !ok {
			continue
		}
		if isPartialInvocationLoopDependent(depInvocation, invocationsByKey, visited) {
			return true
		}
	}

	return false
}

// buildPartialRenderCallBlock builds the statements that call a partial's Render method
// and handle any errors it returns.
//
// Takes reqVar (*goast.Ident) which is the request variable to pass. Takes propsVar
// (*goast.Ident) which is the props variable to pass. Takes key (string) which names the
// partial and is used to create variable names. Takes packageName (string) which is the
// package that contains the Render method.
//
// Returns []goast.Stmt which contains the render call, error handler, and placeholder
// assignments for data, meta, and error values.
func buildPartialRenderCallBlock(reqVar, propsVar *goast.Ident, key, packageName string) []goast.Stmt {
	dataVar := cachedIdent(packageName + "Data_" + key)
	metaVar := cachedIdent(packageName + "Meta_" + key)
	errVar := cachedIdent(packageName + "Err_" + key)

	renderCall := &goast.CallExpr{
		Fun:  &goast.SelectorExpr{X: cachedIdent(packageName), Sel: cachedIdent("Render")},
		Args: []goast.Expr{reqVar, propsVar},
	}

	return []goast.Stmt{
		&goast.AssignStmt{
			Lhs: []goast.Expr{dataVar, metaVar, errVar},
			Tok: token.DEFINE,
			Rhs: []goast.Expr{renderCall},
		},
		buildPartialErrorHandler(errVar, packageName),
		&goast.AssignStmt{Lhs: []goast.Expr{cachedIdent("_")}, Tok: token.ASSIGN, Rhs: []goast.Expr{dataVar}},
		&goast.AssignStmt{Lhs: []goast.Expr{cachedIdent("_")}, Tok: token.ASSIGN, Rhs: []goast.Expr{metaVar}},
		&goast.AssignStmt{Lhs: []goast.Expr{cachedIdent("_")}, Tok: token.ASSIGN, Rhs: []goast.Expr{errVar}},
	}
}

// buildPartialErrorHandler creates the error handling block for partial render calls.
//
// Takes errVar (*goast.Ident) which is the error variable to check.
// Takes packageName (string) which is the partial name for error messages.
//
// Returns goast.Stmt which is an if statement that adds a diagnostic and returns early
// when the error is not nil.
func buildPartialErrorHandler(errVar *goast.Ident, packageName string) goast.Stmt {
	return &goast.IfStmt{
		Cond: &goast.BinaryExpr{X: errVar, Op: token.NEQ, Y: cachedIdent("nil")},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.AssignStmt{
				Lhs: []goast.Expr{cachedIdent(DiagnosticsVarName)},
				Tok: token.ASSIGN,
				Rhs: []goast.Expr{&goast.CallExpr{
					Fun: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("AppendDiagnostic")},
					Args: []goast.Expr{
						cachedIdent(DiagnosticsVarName),
						&goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("Error")},
						&goast.CallExpr{Fun: &goast.SelectorExpr{X: errVar, Sel: cachedIdent("Error")}, Args: []goast.Expr{}},
						strLit("R002"),
						strLit(""),
						strLit("Partial Render() error for " + packageName),
						intLit(0),
						intLit(0),
					},
				}},
			},
			&goast.ReturnStmt{Results: []goast.Expr{
				cachedIdent("nil"),
				&goast.CompositeLit{Type: &goast.SelectorExpr{
					X:   cachedIdent(runtimePackageName),
					Sel: cachedIdent("InternalMetadata"),
				}},
				cachedIdent(DiagnosticsVarName),
			}},
		}},
	}
}

// determinePropsTypeExpr finds the correct type expression for props.
//
// Takes rawPropsTypeExpr (goast.Expr) which is the raw type expression to process.
// Takes partialPackageName (string) which is the package name for types without a package
// prefix.
// Takes currentPackageHash (string) which identifies the current package for comparison.
//
// Returns goast.Expr which is the adjusted type expression. If the type is in the current
// package, the package prefix is removed. Otherwise, the partial package name is added as
// a prefix.
func determinePropsTypeExpr(
	rawPropsTypeExpr goast.Expr,
	partialPackageName string,
	currentPackageHash string,
) goast.Expr {
	var propsOriginPackage string

	if selExpr, isSel := rawPropsTypeExpr.(*goast.SelectorExpr); isSel {
		if identifier, isIdent := selExpr.X.(*goast.Ident); isIdent {
			propsOriginPackage = identifier.Name
		}
	} else if _, isIdent := rawPropsTypeExpr.(*goast.Ident); isIdent {
		propsOriginPackage = partialPackageName
	}

	if propsOriginPackage == currentPackageHash {
		return goastutil.UnqualifyTypeExpr(rawPropsTypeExpr)
	}

	if identifier, isIdent := rawPropsTypeExpr.(*goast.Ident); isIdent {
		return &goast.SelectorExpr{X: cachedIdent(partialPackageName), Sel: identifier}
	}

	return rawPropsTypeExpr
}

// propToField converts a property name from kebab-case to PascalCase.
//
// Takes propName (string) which is the kebab-case name to convert.
//
// Returns string which is the PascalCase field name.
func propToField(propName string) string {
	parts := strings.Split(propName, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// isConditionalNode reports whether the node is the head of, or a branch of, a
// conditional chain (p-if, p-else-if, p-else). Such a node governs the conditional
// rendering of itself and its descendants.
//
// Takes node (*ast_domain.TemplateNode) which is the node to test.
//
// Returns bool which is true when the node carries a conditional directive.
func isConditionalNode(node *ast_domain.TemplateNode) bool {
	return node.DirIf != nil || node.DirElseIf != nil || node.DirElse != nil
}

// conditionalGuardInfo records, per invocation key, whether any occurrence sits under a
// conditional and whether any occurrence sits unguarded (outside every conditional).
type conditionalGuardInfo struct {
	// underCond is true when at least one occurrence of the invocation sits under a
	// conditional.
	underCond bool

	// hasUnguardedOccurrence is true when at least one occurrence sits outside every
	// conditional.
	hasUnguardedOccurrence bool
}

// collectConditionallyGuardedKeys returns the invocation keys that must be rendered
// inside their governing conditional rather than hoisted to the function entry. A key
// qualifies when every occurrence sits under a conditional, so hoisting it would evaluate
// its props unconditionally and defeat the guard.
//
// A key with any unguarded occurrence (at the function root, or directly in a loop that
// is not itself under a conditional) is excluded: it stays hoisted so its single
// deduplicated output variable remains in scope at that unguarded reference. Keys whose
// occurrences span several distinct conditionals, but none unguarded, still qualify and
// are rendered inside each governing branch.
//
// Takes ast (*ast_domain.TemplateAST) which is the component's annotated template.
//
// Returns map[string]bool which is the set of conditionally-guarded invocation keys.
func collectConditionallyGuardedKeys(ast *ast_domain.TemplateAST) map[string]bool {
	infos := make(map[string]*conditionalGuardInfo)
	if ast != nil {
		for _, node := range ast.RootNodes {
			collectGuardInfoRecursive(node, false, 0, infos)
		}
	}

	guarded := make(map[string]bool, len(infos))
	for key, info := range infos {
		if info.underCond && !info.hasUnguardedOccurrence {
			guarded[key] = true
		}
	}
	return guarded
}

// collectGuardInfoRecursive walks the tree tracking whether the current position is under
// a conditional, recording for every partial invocation whether that occurrence is
// guarded. Recursion is bounded by maxPartialScopeDepth so a pathologically deep template
// cannot overflow the stack.
//
// Takes node (*ast_domain.TemplateNode) which is the current node.
// Takes underConditional (bool) which is true when an ancestor is conditional.
// Takes depth (int) which is the current recursion depth.
// Takes infos (map[string]*conditionalGuardInfo) which accumulates per-key data.
func collectGuardInfoRecursive(node *ast_domain.TemplateNode, underConditional bool, depth int, infos map[string]*conditionalGuardInfo) {
	if node == nil || depth > maxPartialScopeDepth {
		return
	}

	nodeUnderConditional := underConditional || isConditionalNode(node)

	if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
		recordGuardInfo(infos, node.GoAnnotations.PartialInfo.InvocationKey, nodeUnderConditional)
	}

	for _, child := range node.Children {
		collectGuardInfoRecursive(child, nodeUnderConditional, depth+1, infos)
	}
}

// recordGuardInfo registers an occurrence of an invocation key, tracking whether the key
// is ever seen under a conditional and whether it is ever seen unguarded.
//
// Takes infos (map[string]*conditionalGuardInfo) which accumulates per-key data.
// Takes key (string) which is the invocation key.
// Takes underConditional (bool) which is true when this occurrence is under a
// conditional.
func recordGuardInfo(infos map[string]*conditionalGuardInfo, key string, underConditional bool) {
	info := infos[key]
	if info == nil {
		infos[key] = &conditionalGuardInfo{underCond: underConditional, hasUnguardedOccurrence: !underConditional}
		return
	}
	if underConditional {
		info.underCond = true
	} else {
		info.hasUnguardedOccurrence = true
	}
}

// collectGovernedPartialInvocations returns the partial invocations directly governed by
// a scope node (a loop or conditional branch): those in its subtree not separated from it
// by a deeper loop or conditional, which render their own partials. The scope node itself
// is included when it is a partial invocation.
//
// Takes scopeNode (*ast_domain.TemplateNode) which is the loop or conditional node.
// Takes uniqueInvocations ([]*annotator_dto.PartialInvocation) which is the canonical
// list.
//
// Returns []*annotator_dto.PartialInvocation which are the governed invocations, each
// appearing once even when several nodes in the scope share an invocation key (those
// nodes reuse a single deduplicated render).
func collectGovernedPartialInvocations(scopeNode *ast_domain.TemplateNode, uniqueInvocations []*annotator_dto.PartialInvocation) []*annotator_dto.PartialInvocation {
	var invocations []*annotator_dto.PartialInvocation
	seen := make(map[string]bool)
	collectGovernedPartialsRecursive(scopeNode, scopeNode, 0, uniqueInvocations, &invocations, seen)
	return invocations
}

// collectGovernedPartialsRecursive walks the scope subtree, stopping at a deeper loop or
// conditional, collecting each distinct partial invocation it reaches. Recursion is
// bounded by maxPartialScopeDepth so a pathologically deep template cannot overflow the
// stack.
//
// Takes node (*ast_domain.TemplateNode) which is the current node.
// Takes scopeNode (*ast_domain.TemplateNode) which is the governing scope root.
// Takes depth (int) which is the current recursion depth.
// Takes uniqueInvocations ([]*annotator_dto.PartialInvocation) which is the canonical
// list.
// Takes out (*[]*annotator_dto.PartialInvocation) which accumulates the results.
// Takes seen (map[string]bool) which deduplicates by invocation key within the scope.
func collectGovernedPartialsRecursive(
	node, scopeNode *ast_domain.TemplateNode,
	depth int,
	uniqueInvocations []*annotator_dto.PartialInvocation,
	out *[]*annotator_dto.PartialInvocation,
	seen map[string]bool,
) {
	if node == nil || depth > maxPartialScopeDepth {
		return
	}

	if node != scopeNode && (node.DirFor != nil || isConditionalNode(node)) {
		return
	}

	if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
		key := node.GoAnnotations.PartialInfo.InvocationKey
		if !seen[key] {
			if invocation := findCanonicalInvocation(uniqueInvocations, key); invocation != nil {
				seen[key] = true
				*out = append(*out, invocation)
			}
		}
	}

	for _, child := range node.Children {
		collectGovernedPartialsRecursive(child, scopeNode, depth+1, uniqueInvocations, out, seen)
	}
}

// findCanonicalInvocation returns the canonical invocation data for the given key, or nil
// when it is not present.
//
// Takes uniqueInvocations ([]*annotator_dto.PartialInvocation) which is the canonical
// list.
// Takes invocationKey (string) which identifies the invocation.
//
// Returns *annotator_dto.PartialInvocation which is the matching invocation, or nil.
func findCanonicalInvocation(uniqueInvocations []*annotator_dto.PartialInvocation, invocationKey string) *annotator_dto.PartialInvocation {
	for _, invocation := range uniqueInvocations {
		if invocation.InvocationKey == invocationKey {
			return invocation
		}
	}
	return nil
}

// partialInfoFromInvocation maps a canonical partial invocation onto the per-call info
// the render emitters consume, so the hoisted, conditional, and loop render paths share
// one field mapping.
//
// Takes invocation (*annotator_dto.PartialInvocation) which is the canonical invocation.
//
// Returns *ast_domain.PartialInvocationInfo which describes the partial to call.
func partialInfoFromInvocation(invocation *annotator_dto.PartialInvocation) *ast_domain.PartialInvocationInfo {
	return &ast_domain.PartialInvocationInfo{
		InvocationKey:        invocation.InvocationKey,
		PartialAlias:         invocation.PartialAlias,
		PartialPackageName:   invocation.PartialHashedName,
		RequestOverrides:     invocation.RequestOverrides,
		PassedProps:          invocation.PassedProps,
		InvokerPackageAlias:  invocation.InvokerHashedName,
		InvokerInvocationKey: invocation.InvokerInvocationKey,
		Location:             invocation.Location,
	}
}
