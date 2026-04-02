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
	"fmt"
	goast "go/ast"
	"go/token"
	"html"
	"slices"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_helpers"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// scopeSeparator is the separator used to join multiple scope IDs in
	// a scope chain for CSS scoping.
	scopeSeparator = " "

	// argTypeStatic marks an argument as a static literal value.
	argTypeStatic = "s"

	// templateNodeTypeName is the type name for template nodes in generated code.
	templateNodeTypeName = "TemplateNode"

	// initialStatementCapacity is the pre-allocation size for statement slices
	// (typically: init, NodeType, and optional content).
	initialStatementCapacity = 3
)

// StaticEmitter provides methods for hoisting and managing static AST nodes.
// It enables mocking and testing of static node handling.
type StaticEmitter interface {
	// registerStaticNode registers a static template node in the cross-reference
	// system.
	//
	// Takes node (*ast_domain.TemplateNode) which is the static node to register.
	// Takes partialScopeID (string) which is the current partial's HashedName for
	// CSS scoping.
	//
	// Returns *goast.Ident which is the identifier for the registered node.
	// Returns []*ast_domain.Diagnostic which contains any issues found during
	// registration.
	registerStaticNode(ctx context.Context, node *ast_domain.TemplateNode, partialScopeID string) (*goast.Ident, []*ast_domain.Diagnostic)

	// registerStaticAttributes registers a static attribute slice and returns the
	// variable name. If an identical attribute set was already registered, returns
	// the existing variable name (deduplication).
	//
	// Takes node (*ast_domain.TemplateNode) which provides the attributes.
	// Takes partialScopeID (string) which is the current partial's HashedName.
	//
	// Returns string which is the variable name for the static attribute slice,
	// or empty string if there are no attributes.
	registerStaticAttributes(node *ast_domain.TemplateNode, partialScopeID string) string

	// buildDeclarations builds and returns the AST declarations.
	//
	// Returns goast.Decl which contains the built declarations.
	buildDeclarations() goast.Decl

	// buildInitFunction builds an init function declaration.
	//
	// Returns goast.Decl which is the generated init function.
	buildInitFunction() goast.Decl
}

// staticEmitter creates static code declarations and init functions. It
// implements StaticEmitter and stores a mapping of node hashes to variable
// names.
type staticEmitter struct {
	// emitter provides variable naming and source mapping helpers.
	emitter *emitter

	// staticNodeCache maps node hashes to their root variable names.
	staticNodeCache map[string]string

	// staticAttrCache maps attribute content hashes to variable names.
	// This avoids duplicating attribute sets across nodes.
	staticAttrCache map[string]string

	// allStaticVarDecls maps variable names to their ValueSpec for the var block.
	allStaticVarDecls map[string]*goast.ValueSpec

	// staticAttrVarDecls maps variable names to their ValueSpec for attribute
	// slices.
	staticAttrVarDecls map[string]*goast.ValueSpec

	// mainComponentScope is the hashed name of the main component being built.
	// It is used to tell slotted content apart from nested partial content.
	mainComponentScope string

	// initFunctionStatements holds the statements to add to the init() function.
	initFunctionStatements []goast.Stmt
}

var _ StaticEmitter = (*staticEmitter)(nil)

// registerStaticNode registers a template node for static use. It checks a
// cache to avoid repeated work and then starts the recursive process to build
// the node.
//
// If the node is fully prerenderable and a prerenderer is available, the HTML
// is rendered at generation time and stored as bytes for zero-copy output at
// runtime.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to register.
// Takes partialScopeID (string) which is the current partial's HashedName for
// CSS scoping.
//
// Returns *goast.Ident which is the identifier for the registered static node.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from building
// the node.
func (se *staticEmitter) registerStaticNode(ctx context.Context, node *ast_domain.TemplateNode, partialScopeID string) (*goast.Ident, []*ast_domain.Diagnostic) {
	nodeHash := ast_domain.SerialiseNodeString(node) + ":" + partialScopeID

	if existingVarName, ok := se.staticNodeCache[nodeHash]; ok {
		return cachedIdent(existingVarName), nil
	}

	canPrerender := se.emitter.config.EnablePrerendering &&
		se.emitter.prerenderer != nil &&
		node.GoAnnotations != nil &&
		node.GoAnnotations.IsFullyPrerenderable

	if canPrerender {
		return se.registerPrerenderedNode(ctx, node, nodeHash, partialScopeID)
	}

	StaticNodeHoistedCount.Add(ctx, 1)

	rootVarName := se.emitter.nextStaticVarName()
	se.staticNodeCache[nodeHash] = rootVarName

	statements, diagnostics := se.buildStaticNodeRecursive(ctx, node, rootVarName, partialScopeID)

	se.initFunctionStatements = append(se.initFunctionStatements, statements...)

	return cachedIdent(rootVarName), diagnostics
}

// registerPrerenderedNode registers a node that will use prerendered HTML
// bytes. This generates a minimal TemplateNode with only PrerenderedHTML set.
//
// Before prerendering, this deep-clones the entire subtree and injects
// `partial` and `p-key` attributes into every element node. This provides CSS
// scoping and hydration work correctly for prerendered content.
//
// Takes node (*ast_domain.TemplateNode) which is the node to prerender.
// Takes nodeHash (string) which is the cache key for this node.
// Takes partialScopeID (string) which is the current partial's HashedName.
//
// Returns *goast.Ident which is the identifier for the registered node.
// Returns []*ast_domain.Diagnostic which contains any issues from prerendering.
func (se *staticEmitter) registerPrerenderedNode(ctx context.Context, node *ast_domain.TemplateNode, nodeHash, partialScopeID string) (*goast.Ident, []*ast_domain.Diagnostic) {
	ctx, l := logger_domain.From(ctx, log)
	nodeToRender := node.DeepCloneWithScopeAttributes(partialScopeID)

	preprocessEventsForPrerendering(nodeToRender)

	htmlBytes, err := se.emitter.prerenderer.RenderStaticNode(nodeToRender)
	if err != nil {
		l.Warn("Prerendering failed, falling back to AST emission",
			logger_domain.Error(err),
			logger_domain.String("tag", node.TagName))
		return se.registerStaticNodeFallback(ctx, node, nodeHash, partialScopeID)
	}

	PrerenderedNodeCount.Add(ctx, 1)

	rootVarName := se.emitter.nextStaticVarName()
	se.staticNodeCache[nodeHash] = rootVarName

	statements := se.buildPrerenderedNodeStatements(node, rootVarName, htmlBytes)
	se.initFunctionStatements = append(se.initFunctionStatements, statements...)

	return cachedIdent(rootVarName), nil
}

// registerStaticNodeFallback registers a static node without prerendering.
// Used when prerendering fails for any reason.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to register.
// Takes nodeHash (string) which is the unique hash identifying the node.
// Takes partialScopeID (string) which is the scope identifier for the partial.
//
// Returns *goast.Ident which is the identifier for the cached static node.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from building
// the node.
func (se *staticEmitter) registerStaticNodeFallback(ctx context.Context, node *ast_domain.TemplateNode, nodeHash, partialScopeID string) (*goast.Ident, []*ast_domain.Diagnostic) {
	StaticNodeHoistedCount.Add(ctx, 1)

	rootVarName := se.emitter.nextStaticVarName()
	se.staticNodeCache[nodeHash] = rootVarName

	statements, diagnostics := se.buildStaticNodeRecursive(ctx, node, rootVarName, partialScopeID)
	se.initFunctionStatements = append(se.initFunctionStatements, statements...)

	return cachedIdent(rootVarName), diagnostics
}

// buildPrerenderedNodeStatements generates statements for a prerendered node.
// Creates a minimal TemplateNode with only PrerenderedHTML set.
//
// Takes node (*ast_domain.TemplateNode) which provides source mapping info.
// Takes varName (string) which is the variable name for this node.
// Takes htmlBytes ([]byte) which is the prerendered HTML content.
//
// Returns []goast.Stmt which contains the initialisation statements.
func (se *staticEmitter) buildPrerenderedNodeStatements(node *ast_domain.TemplateNode, varName string, htmlBytes []byte) []goast.Stmt {
	nodeVarIdent := cachedIdent(varName)

	se.allStaticVarDecls[varName] = &goast.ValueSpec{
		Comment: se.emitter.sourceMappingCommentGroup(node),
		Names:   []*goast.Ident{nodeVarIdent},
		Type:    &goast.StarExpr{X: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(templateNodeTypeName)}},
	}

	statements := make([]goast.Stmt, 0, initialStatementCapacity)

	initExpr := &goast.UnaryExpr{
		Op: token.AND,
		X:  &goast.CompositeLit{Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(templateNodeTypeName)}},
	}
	statements = append(statements,
		assignExpression(varName, initExpr),
		&goast.AssignStmt{
			Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVarIdent, Sel: cachedIdent("NodeType")}},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{&goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(node.NodeType.String())}},
		},
		&goast.AssignStmt{
			Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVarIdent, Sel: cachedIdent("PrerenderedHTML")}},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{byteLit(htmlBytes)},
		})

	return statements
}

// buildStaticNodeRecursive generates Go statements for a static template node
// by delegating to specialised helpers for declaration, initialisation, and
// population of the node and its children.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes varName (string) which is the variable name to use for this node.
// Takes partialScopeID (string) which is the current partial's HashedName for
// CSS scoping.
//
// Returns []goast.Stmt which contains the generated statements.
// Returns []*ast_domain.Diagnostic which contains any diagnostics encountered.
func (se *staticEmitter) buildStaticNodeRecursive(ctx context.Context, node *ast_domain.TemplateNode, varName string, partialScopeID string) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	var statements []goast.Stmt
	var allDiags []*ast_domain.Diagnostic
	nodeVarIdent := cachedIdent(varName)

	se.registerStaticVarDecl(node, varName, nodeVarIdent)

	statements = append(statements, se.createNodeInitialisation(varName))

	se.setBasicNodeProperties(node, nodeVarIdent, &statements)

	if se.isContainerNode(node) {
		attributeStatements, attributeDiagnostics := se.buildStaticAttributes(node, nodeVarIdent, varName, partialScopeID)
		statements = append(statements, attributeStatements...)
		allDiags = append(allDiags, attributeDiagnostics...)

		childStmts, childDiags := se.buildStaticChildren(ctx, node, nodeVarIdent, varName, partialScopeID)
		statements = append(statements, childStmts...)
		allDiags = append(allDiags, childDiags...)
	}

	return statements, allDiags
}

// registerStaticVarDecl creates and registers a variable declaration for a
// static node.
//
// Takes node (*ast_domain.TemplateNode) which provides the source mapping.
// Takes varName (string) which names the variable in the declaration map.
// Takes nodeVarIdent (*goast.Ident) which is the identifier for the variable.
func (se *staticEmitter) registerStaticVarDecl(node *ast_domain.TemplateNode, varName string, nodeVarIdent *goast.Ident) {
	se.allStaticVarDecls[varName] = &goast.ValueSpec{
		Comment: se.emitter.sourceMappingCommentGroup(node),
		Names:   []*goast.Ident{nodeVarIdent},
		Type:    &goast.StarExpr{X: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(templateNodeTypeName)}},
	}
}

// createNodeInitialisation creates the initialisation statement for a static
// node.
//
// Takes varName (string) which specifies the variable name for the node.
//
// Returns goast.Stmt which is the assignment statement that sets up the node.
func (*staticEmitter) createNodeInitialisation(varName string) goast.Stmt {
	initExpr := &goast.UnaryExpr{
		Op: token.AND,
		X:  &goast.CompositeLit{Type: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(templateNodeTypeName)}},
	}
	return assignExpression(varName, initExpr)
}

// setBasicNodeProperties sets the basic runtime properties on a node variable.
// These properties are NodeType, TagName, and TextContent.
//
// Takes node (*ast_domain.TemplateNode) which provides the source node data.
// Takes nodeVarIdent (*goast.Ident) which identifies the target variable.
// Takes statements (*[]goast.Stmt) which collects the generated assignment
// statements.
func (*staticEmitter) setBasicNodeProperties(node *ast_domain.TemplateNode, nodeVarIdent *goast.Ident, statements *[]goast.Stmt) {
	addStmt := func(leftHandSide string, rightHandSide goast.Expr) {
		*statements = append(*statements, &goast.AssignStmt{
			Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVarIdent, Sel: cachedIdent(leftHandSide)}},
			Tok: token.ASSIGN,
			Rhs: []goast.Expr{rightHandSide},
		})
	}

	addStmt("NodeType", &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(node.NodeType.String())})
	if node.TagName != "" {
		addStmt("TagName", strLit(node.TagName))
	}
	if node.TextContent != "" {
		addStmt("TextContent", strLit(html.EscapeString(node.TextContent)))
	}
}

// isContainerNode checks if a node can contain attributes and children.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node is an element or fragment.
func (*staticEmitter) isContainerNode(node *ast_domain.TemplateNode) bool {
	return node.NodeType == ast_domain.NodeElement || node.NodeType == ast_domain.NodeFragment
}

// buildStaticAttributes generates code for static attributes including partial
// and key attributes. Uses collectAttributeEntries as the single source of
// truth for which attributes to emit.
//
// Takes node (*ast_domain.TemplateNode) which provides the attributes
// to emit.
// Takes nodeVarIdent (*goast.Ident) which identifies the Go variable
// for the node.
// Takes partialScopeID (string) which is the scope identifier for
// partial rendering.
//
// Returns []goast.Stmt which contains the attribute assignment
// statements.
// Returns []*ast_domain.Diagnostic which contains any issues found
// during emission.
func (se *staticEmitter) buildStaticAttributes(node *ast_domain.TemplateNode, nodeVarIdent *goast.Ident, _ string, partialScopeID string) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	effectiveKey := getEffectiveKeyExpression(node)
	if effectiveKey != nil {
		if _, ok := effectiveKey.(*ast_domain.StringLiteral); !ok {
			diagnostic := ast_domain.NewDiagnostic(ast_domain.Error, "Internal Emitter Error: a static node's key was not a StringLiteral", effectiveKey.String(), node.Location, "")
			return nil, []*ast_domain.Diagnostic{diagnostic}
		}
	}

	attrs := se.collectAttributeEntries(node, partialScopeID)
	if len(attrs) == 0 {
		return nil, nil
	}

	statements := make([]goast.Stmt, 0, len(attrs)+1)

	statements = append(statements, &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVarIdent, Sel: cachedIdent("Attributes")}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{&goast.CallExpr{
			Fun:  cachedIdent(MakeFuncName),
			Args: []goast.Expr{&goast.ArrayType{Elt: cachedIdent(fmt.Sprintf(HTMLAttributeTypeFmt, runtimePackageName))}, intLit(0), intLit(len(attrs))},
		}},
	})

	attributeSlice := &goast.SelectorExpr{X: nodeVarIdent, Sel: cachedIdent("Attributes")}

	for _, attr := range attrs {
		statements = append(statements, appendToSlice(attributeSlice, createHTMLAttributeLiteral(attr.Name, attr.Value)))
	}

	return statements, nil
}

// calculateAttributeCapacity works out the required capacity for the
// attributes slice.
//
// Takes node (*ast_domain.TemplateNode) which provides the template to check.
// Takes effectiveKey (ast_domain.Expression) which is the optional key
// directive expression.
// Takes partialScopeID (string) which is the current partial's HashedName for
// CSS scoping.
//
// Returns int which is the total capacity needed for all attributes.
func (se *staticEmitter) calculateAttributeCapacity(node *ast_domain.TemplateNode, effectiveKey ast_domain.Expression, partialScopeID string) int {
	capacity := len(node.Attributes)
	if effectiveKey != nil {
		capacity += SingleDirectiveAttrCount
	}
	if node.DirRef != nil && node.DirRef.RawExpression != "" {
		capacity += SingleDirectiveAttrCount
	}
	if se.hasPartialInfo(node) {
		capacity += PartialAttributeCapacity
		if se.hasPartialQueryProps(node) {
			capacity += PartialPropsAttributeCapacity
		}
	} else if partialScopeID != "" && node.NodeType == ast_domain.NodeElement && !nodeHasPartialAttribute(node) {
		capacity += SingleDirectiveAttrCount
	}
	capacity += se.countStaticEventAttributes(node)
	return capacity
}

// hasPartialInfo checks if the node has public partial information.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true when the node has partial info.
func (se *staticEmitter) hasPartialInfo(node *ast_domain.TemplateNode) bool {
	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return false
	}
	pInfo := node.GoAnnotations.PartialInfo
	_, ok := se.emitter.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	return ok
}

// hasPartialQueryProps checks whether a public partial node has query-bound
// primitive props that require a partial_props attribute.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when the partial has query-bound primitive props.
func (se *staticEmitter) hasPartialQueryProps(node *ast_domain.TemplateNode) bool {
	if node.GoAnnotations == nil || node.GoAnnotations.PartialInfo == nil {
		return false
	}
	pInfo := node.GoAnnotations.PartialInfo
	partialVC, ok := se.emitter.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
	if !ok || !partialVC.IsPublic {
		return false
	}
	return len(extractPrimitiveQueryPropsFromComponent(partialVC)) > 0
}

// buildStaticChildren creates code to build child nodes for a parent node.
//
// Takes node (*ast_domain.TemplateNode) which is the parent node whose
// children will be built.
// Takes nodeVarIdent (*goast.Ident) which is the identifier for the parent
// node variable in the generated code.
// Takes varName (string) which is the base name for child variable names.
// Takes partialScopeID (string) which is the current partial's hashed name
// for CSS scoping.
//
// Returns []goast.Stmt which contains the generated statements for building
// all child nodes.
// Returns []*ast_domain.Diagnostic which contains any problems found while
// building child nodes.
//
// Only creates the Children slice when there are children. This saves 24 bytes
// per node by not creating empty slice headers.
func (se *staticEmitter) buildStaticChildren(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	nodeVarIdent *goast.Ident,
	varName string,
	partialScopeID string,
) ([]goast.Stmt, []*ast_domain.Diagnostic) {
	if len(node.Children) == 0 {
		return nil, nil
	}

	estimatedCapacity := 1 + (len(node.Children) * 2)
	statements := make([]goast.Stmt, 0, estimatedCapacity)
	var diagnostics []*ast_domain.Diagnostic

	statements = append(statements, &goast.AssignStmt{
		Lhs: []goast.Expr{&goast.SelectorExpr{X: nodeVarIdent, Sel: cachedIdent("Children")}},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{&goast.CallExpr{
			Fun:  cachedIdent(MakeFuncName),
			Args: []goast.Expr{&goast.ArrayType{Elt: &goast.StarExpr{X: cachedIdent(fmt.Sprintf("%s.TemplateNode", runtimePackageName))}}, intLit(0), intLit(len(node.Children))},
		}},
	})

	childrenSliceExpr := &goast.SelectorExpr{X: nodeVarIdent, Sel: cachedIdent("Children")}
	for i, child := range node.Children {
		childVarName := fmt.Sprintf("%s_child_%d", varName, i)
		childScopeID := getEffectivePartialScopeID(child, partialScopeID, se.mainComponentScope)
		childStmts, childDiags := se.buildStaticNodeRecursive(ctx, child, childVarName, childScopeID)
		statements = append(statements, childStmts...)
		diagnostics = append(diagnostics, childDiags...)
		statements = append(statements, appendToSlice(childrenSliceExpr, cachedIdent(childVarName)))
	}

	return statements, diagnostics
}

// registerStaticAttributes registers a static attribute slice and returns the
// variable name. If an identical attribute set was already registered, returns
// the existing variable name (deduplication).
//
// This reuses the same attribute building logic as buildStaticAttributes but
// creates a package-level composite literal instead of init-time statements.
//
// Takes node (*ast_domain.TemplateNode) which provides the attributes.
// Takes partialScopeID (string) which is the current partial's HashedName.
//
// Returns string which is the variable name for the static attribute slice.
func (se *staticEmitter) registerStaticAttributes(node *ast_domain.TemplateNode, partialScopeID string) string {
	attrs := se.collectAttributeEntries(node, partialScopeID)
	if len(attrs) == 0 {
		return ""
	}

	hashKey := computeAttrHash(attrs)

	if existingVarName, ok := se.staticAttrCache[hashKey]; ok {
		return existingVarName
	}

	varName := se.emitter.nextStaticAttrVarName()
	se.staticAttrCache[hashKey] = varName

	se.staticAttrVarDecls[varName] = &goast.ValueSpec{
		Names:   []*goast.Ident{cachedIdent(varName)},
		Type:    &goast.ArrayType{Elt: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("HTMLAttribute")}},
		Comment: se.emitter.sourceMappingCommentGroup(node),
	}

	varIdent := cachedIdent(varName)

	elts := make([]goast.Expr, len(attrs))
	for i, attr := range attrs {
		elts[i] = createHTMLAttributeLiteral(attr.Name, attr.Value)
	}

	initStmt := &goast.AssignStmt{
		Lhs: []goast.Expr{varIdent},
		Tok: token.ASSIGN,
		Rhs: []goast.Expr{
			&goast.CompositeLit{
				Type: &goast.ArrayType{Elt: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent("HTMLAttribute")}},
				Elts: elts,
			},
		},
	}
	se.initFunctionStatements = append(se.initFunctionStatements, initStmt)

	return varName
}

// attributeEntry holds a single attribute name-value pair for code generation.
type attributeEntry struct {
	// Name is the attribute name.
	Name string

	// Value is the text content of the attribute.
	Value string
}

// collectAttributeEntries builds the list of attributes for a node.
// This is the single source of truth for which attributes a static node has.
//
// Order: partial info attrs, key attr, ref attr, regular attrs, event attrs.
//
// Takes node (*ast_domain.TemplateNode) which provides the template node.
// Takes partialScopeID (string) which is the current partial's HashedName.
//
// Returns []attributeEntry which contains all attributes in the correct order.
func (se *staticEmitter) collectAttributeEntries(node *ast_domain.TemplateNode, partialScopeID string) []attributeEntry {
	effectiveKey := getEffectiveKeyExpression(node)
	capacity := se.calculateAttributeCapacity(node, effectiveKey, partialScopeID)
	if capacity == 0 {
		return nil
	}

	attrs := make([]attributeEntry, 0, capacity)
	attrs = se.appendPartialInfoAttrs(attrs, node, partialScopeID)
	attrs = appendKeyAttr(attrs, effectiveKey)
	attrs = se.appendRefAttrs(attrs, node)
	attrs = appendRegularAttrs(attrs, node)
	attrs = se.appendStaticEventAttrs(attrs, node)
	return attrs
}

// appendPartialInfoAttrs adds partial scope or info attributes based on the
// node context.
//
// Takes attrs ([]attributeEntry) which is the existing attribute list to extend.
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
// Takes partialScopeID (string) which is the scope identifier for partial
// elements.
//
// Returns []attributeEntry which contains the original attributes plus any added
// partial attributes.
func (se *staticEmitter) appendPartialInfoAttrs(attrs []attributeEntry, node *ast_domain.TemplateNode, partialScopeID string) []attributeEntry {
	if se.hasPartialInfo(node) {
		pInfo := node.GoAnnotations.PartialInfo
		partialVC, ok := se.emitter.AnnotationResult.VirtualModule.ComponentsByHash[pInfo.PartialPackageName]
		if ok {
			attrs = append(attrs,
				attributeEntry{Name: "partial", Value: partialVC.HashedName},
				attributeEntry{Name: "partial_name", Value: partialVC.PartialName},
			)
			if partialVC.IsPublic {
				attrs = append(attrs, attributeEntry{Name: "partial_src", Value: partialVC.PartialSrc})
			}
			return attrs
		}
	}
	if partialScopeID != "" && node.NodeType == ast_domain.NodeElement && !nodeHasPartialAttribute(node) {
		return append(attrs, attributeEntry{Name: "partial", Value: partialScopeID})
	}
	return attrs
}

// appendRefAttrs adds p-ref and data-pk-partial attributes when present.
//
// Takes attrs ([]attributeEntry) which is the existing attribute list to extend.
// Takes node (*ast_domain.TemplateNode) which provides the directive reference.
//
// Returns []attributeEntry which is the attribute list with reference attributes
// added.
func (se *staticEmitter) appendRefAttrs(attrs []attributeEntry, node *ast_domain.TemplateNode) []attributeEntry {
	if node.DirRef == nil || node.DirRef.RawExpression == "" {
		return attrs
	}
	attrs = append(attrs, attributeEntry{Name: prefAttributeName, Value: html.EscapeString(node.DirRef.RawExpression)})
	if componentHash := se.emitter.config.HashedName; componentHash != "" {
		attrs = append(attrs, attributeEntry{Name: "data-pk-partial", Value: componentHash})
	}
	return attrs
}

// buildDeclarations creates the top-level var block for all static nodes and
// static attribute slices. It sorts variable names to ensure consistent output.
//
// Returns goast.Decl which is the generated declaration block, or nil if there
// are no static variable declarations.
func (se *staticEmitter) buildDeclarations() goast.Decl {
	totalDecls := len(se.allStaticVarDecls) + len(se.staticAttrVarDecls)
	if totalDecls == 0 {
		return nil
	}

	sortedNames := make([]string, 0, totalDecls)
	for name := range se.allStaticVarDecls {
		sortedNames = append(sortedNames, name)
	}
	for name := range se.staticAttrVarDecls {
		sortedNames = append(sortedNames, name)
	}
	slices.Sort(sortedNames)

	specs := make([]goast.Spec, len(sortedNames))
	for i, name := range sortedNames {
		if spec, ok := se.allStaticVarDecls[name]; ok {
			specs[i] = spec
		} else {
			specs[i] = se.staticAttrVarDecls[name]
		}
	}

	return &goast.GenDecl{
		Tok:    token.VAR,
		Lparen: 1,
		Specs:  specs,
	}
}

// buildInitFunction creates the init function that holds all setup statements
// for the static nodes.
//
// Returns goast.Decl which is the init function declaration, or nil if there
// are no setup statements.
func (se *staticEmitter) buildInitFunction() goast.Decl {
	if len(se.initFunctionStatements) == 0 {
		return nil
	}

	return &goast.FuncDecl{
		Name: cachedIdent("init"),
		Type: &goast.FuncType{Params: &goast.FieldList{}},
		Body: &goast.BlockStmt{List: se.initFunctionStatements},
	}
}

// appendStaticEventAttrs adds static event attributes (p-on:*, p-event:*) to
// the attribute list. Only processes events where IsStaticEvent == true and the
// emission rules dictate that an HTML attribute should be generated.
//
// Takes attrs ([]attributeEntry) which is the existing attribute list to extend.
// Takes node (*ast_domain.TemplateNode) which provides the event directives.
//
// Returns []attributeEntry which contains the original attributes plus any event
// attributes.
func (se *staticEmitter) appendStaticEventAttrs(attrs []attributeEntry, node *ast_domain.TemplateNode) []attributeEntry {
	attrs = se.appendStaticEventsFromMap(attrs, node.OnEvents, "p-on:", node)
	attrs = se.appendStaticEventsFromMap(attrs, node.CustomEvents, "p-event:", node)
	return attrs
}

// appendStaticEventsFromMap processes a single event map (OnEvents or
// CustomEvents) and adds static event attributes with pre-encoded payloads.
//
// Takes attrs ([]attributeEntry) which is the existing attribute list to extend.
// Takes events (map[string][]ast_domain.Directive) which is the event map to
// process.
// Takes prefix (string) which is the attribute prefix ("p-on:" or "p-event:").
// Takes node (*ast_domain.TemplateNode) which is the parent node for client
// script lookup.
//
// Returns []attributeEntry which contains the original attributes plus any event
// attributes.
func (se *staticEmitter) appendStaticEventsFromMap(
	attrs []attributeEntry,
	events map[string][]ast_domain.Directive,
	prefix string,
	node *ast_domain.TemplateNode,
) []attributeEntry {
	if len(events) == 0 {
		return attrs
	}

	eventNames := make([]string, 0, len(events))
	for name := range events {
		eventNames = append(eventNames, name)
	}
	slices.Sort(eventNames)

	for _, eventName := range eventNames {
		directives := events[eventName]
		for i := range directives {
			d := &directives[i]

			if !d.IsStaticEvent {
				continue
			}

			attributeName, shouldEmit := se.resolveStaticEventEmission(d, eventName, prefix, node)
			if !shouldEmit {
				continue
			}

			payload := se.buildStaticEventPayload(d)
			if payload == "" {
				continue
			}

			attrs = append(attrs, attributeEntry{Name: attributeName, Value: payload})
		}
	}
	return attrs
}

// resolveStaticEventEmission determines if a static event should produce an
// HTML attribute and returns the attribute name.
//
// Uses the same emission rules as
// attribute_emitter_actions.resolveEventHandlerEmission to ensure consistency
// between static and dynamic emission paths.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes eventName (string) which is the event name for the attribute key.
// Takes prefix (string) which is the attribute prefix ("p-on:" or "p-event:").
// Takes node (*ast_domain.TemplateNode) which is the parent node for client
// script lookup.
//
// Returns string which is the attribute name (e.g., "p-on:click").
// Returns bool which is true if the directive should produce an HTML attribute.
func (se *staticEmitter) resolveStaticEventEmission(
	d *ast_domain.Directive,
	eventName string,
	prefix string,
	node *ast_domain.TemplateNode,
) (string, bool) {
	attributeName := buildEventAttributeName(prefix, eventName, d.EventModifiers)

	switch d.Modifier {
	case actionModifierName, helperModifierName:
		return attributeName, true
	case "":
		if se.staticDirectiveHasClientScript(d, node) {
			return attributeName, true
		}
		return "", false
	default:
		return "", false
	}
}

// staticDirectiveHasClientScript checks whether a directive comes from a
// component with a client-side script. This is the static emitter's version of
// the same check in attributeEmitter.
//
// Takes d (*ast_domain.Directive) which is the directive to check.
// Takes node (*ast_domain.TemplateNode) which is the parent node for fallback
// lookup.
//
// Returns bool which is true if the directive's source has a client script.
func (se *staticEmitter) staticDirectiveHasClientScript(d *ast_domain.Directive, node *ast_domain.TemplateNode) bool {
	config := se.emitter.config

	if d.GoAnnotations != nil && d.GoAnnotations.OriginalSourcePath != nil {
		sourcePath := *d.GoAnnotations.OriginalSourcePath
		if config.SourcePathHasClientScript != nil {
			if hasScript, ok := config.SourcePathHasClientScript[sourcePath]; ok {
				return hasScript
			}
		}
	}

	if node != nil && node.GoAnnotations != nil && node.GoAnnotations.OriginalSourcePath != nil {
		sourcePath := *node.GoAnnotations.OriginalSourcePath
		if config.SourcePathHasClientScript != nil {
			if hasScript, ok := config.SourcePathHasClientScript[sourcePath]; ok {
				return hasScript
			}
		}
	}

	return config.HasClientScript
}

// buildStaticEventPayload extracts static argument values from a directive
// expression and encodes them as a base64 JSON payload string.
//
// Takes d (*ast_domain.Directive) which contains the event handler expression.
//
// Returns string which is the base64-encoded payload, or empty if encoding
// fails.
func (*staticEmitter) buildStaticEventPayload(d *ast_domain.Directive) string {
	callExpr := staticNormaliseToCallExpr(d.Expression)
	if callExpr == nil {
		return ""
	}

	functionName, ok := callExpr.Callee.(*ast_domain.Identifier)
	if !ok {
		return ""
	}

	arguments, valid := extractStaticArgs(callExpr.Args)
	if !valid {
		return ""
	}

	payload := templater_dto.ActionPayload{
		Function: functionName.Name,
		Args:     arguments,
	}

	return generator_helpers.EncodeStaticActionPayload(payload)
}

// countStaticEventAttributes counts the number of static event attributes that
// will be emitted for a node.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns int which is the count of static event attributes.
func (se *staticEmitter) countStaticEventAttributes(node *ast_domain.TemplateNode) int {
	count := 0
	count += se.countStaticEventsInMap(node.OnEvents, node)
	count += se.countStaticEventsInMap(node.CustomEvents, node)
	return count
}

// countStaticEventsInMap counts the number of static events in an event map
// that will produce HTML attributes.
//
// Takes events (map[string][]ast_domain.Directive) which is the event map.
// Takes node (*ast_domain.TemplateNode) which is the parent node for client
// script lookup.
//
// Returns int which is the count of emittable static events.
func (se *staticEmitter) countStaticEventsInMap(events map[string][]ast_domain.Directive, node *ast_domain.TemplateNode) int {
	count := 0
	for eventName, directives := range events {
		for i := range directives {
			d := &directives[i]
			if !d.IsStaticEvent {
				continue
			}
			_, shouldEmit := se.resolveStaticEventEmission(d, eventName, "", node)
			if shouldEmit {
				count++
			}
		}
	}
	return count
}

// newStaticEmitter creates a new static emitter with empty caches.
//
// Takes emitter (*emitter) which provides the base emitter functions.
// Takes mainComponentScope (string) which sets the scope for the main
// component.
//
// Returns *staticEmitter which is ready for use.
func newStaticEmitter(emitter *emitter, mainComponentScope string) *staticEmitter {
	return &staticEmitter{
		emitter:                emitter,
		staticNodeCache:        make(map[string]string),
		staticAttrCache:        make(map[string]string),
		allStaticVarDecls:      make(map[string]*goast.ValueSpec),
		staticAttrVarDecls:     make(map[string]*goast.ValueSpec),
		mainComponentScope:     mainComponentScope,
		initFunctionStatements: make([]goast.Stmt, 0),
	}
}

// byteLit creates a Go AST expression for a byte slice literal from bytes.
// Uses string literal conversion for readability: []byte("content").
//
// Takes b ([]byte) which provides the bytes to convert to a literal.
//
// Returns goast.Expr which is the byte slice literal expression.
func byteLit(b []byte) goast.Expr {
	return &goast.CallExpr{
		Fun:  &goast.ArrayType{Elt: cachedIdent("byte")},
		Args: []goast.Expr{strLit(string(b))},
	}
}

// getEffectivePartialScopeID finds the correct scope ID for a node based on
// its link to the main component and parent scope.
//
// When a node has its own OriginalPackageAlias from a nested partial, this returns
// a combined scope with both the parent scope and the node scope, letting
// parent CSS style elements in nested partials using the [partial~=xxx]
// selector.
//
// Slotted content (content passed from the main component to a child partial's
// slot) does not inherit the receiver's scope. Slotted content has its
// OriginalPackageAlias equal to the main component scope.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check for its own
// scope.
// Takes parentScopeID (string) which is the scope from the parent context.
// Takes mainComponentScope (string) which is the HashedName of the main
// component being built.
//
// Returns string which is the scope ID to use for this node.
func getEffectivePartialScopeID(node *ast_domain.TemplateNode, parentScopeID, mainComponentScope string) string {
	if node.GoAnnotations == nil || node.GoAnnotations.OriginalPackageAlias == nil {
		return parentScopeID
	}
	nodeScope := *node.GoAnnotations.OriginalPackageAlias

	if nodeScope == mainComponentScope {
		if containsScope(parentScopeID, mainComponentScope) {
			return nodeScope
		}
		if parentScopeID != "" && parentScopeID != nodeScope {
			return parentScopeID + scopeSeparator + nodeScope
		}
		return nodeScope
	}

	if getFirstScope(parentScopeID) == nodeScope {
		return nodeScope
	}

	if containsScope(parentScopeID, nodeScope) {
		return parentScopeID
	}

	if parentScopeID != "" && parentScopeID != nodeScope {
		return nodeScope + scopeSeparator + parentScopeID
	}
	return nodeScope
}

// getFirstScope returns the first scope from a space-separated scope chain.
//
// Takes scopeChain (string) which is the list of scope IDs separated by spaces.
//
// Returns string which is the first scope, or an empty string if the chain is
// empty.
func getFirstScope(scopeChain string) string {
	if scopeChain == "" {
		return ""
	}
	if before, _, found := strings.Cut(scopeChain, scopeSeparator); found {
		return before
	}
	return scopeChain
}

// containsScope checks if a scope chain contains a given scope.
//
// Takes scopeChain (string) which is the scope IDs separated by spaces.
// Takes scope (string) which is the scope to find.
//
// Returns bool which is true if the scope is in the chain.
func containsScope(scopeChain, scope string) bool {
	for s := range strings.SplitSeq(scopeChain, scopeSeparator) {
		if s == scope {
			return true
		}
	}
	return false
}

// appendKeyAttr adds the p-key attribute if the key is a string literal.
//
// Takes attrs ([]attributeEntry) which holds the current list of attributes.
// Takes effectiveKey (ast_domain.Expression) which is the key expression to
// check.
//
// Returns []attributeEntry which is the attribute list, with p-key added if the
// key was a string literal.
func appendKeyAttr(attrs []attributeEntry, effectiveKey ast_domain.Expression) []attributeEntry {
	if effectiveKey == nil {
		return attrs
	}
	if sl, ok := effectiveKey.(*ast_domain.StringLiteral); ok {
		return append(attrs, attributeEntry{Name: pkeyAttributeName, Value: html.EscapeString(sl.Value)})
	}
	return attrs
}

// appendRegularAttrs adds standard HTML attributes to the given slice, skipping
// dynamic class and style attributes.
//
// Takes attrs ([]attributeEntry) which is the slice to add attributes to.
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// read attributes from.
//
// Returns []attributeEntry which contains the original entries plus any standard
// attributes from the node, with values escaped for HTML.
func appendRegularAttrs(attrs []attributeEntry, node *ast_domain.TemplateNode) []attributeEntry {
	skipClass := hasDynamicClassContent(node)
	skipStyle := hasDynamicStyleContent(node)
	for i := range node.Attributes {
		attr := &node.Attributes[i]
		if shouldSkipDynamicAttr(attr.Name, skipClass, skipStyle) {
			continue
		}
		attrs = append(attrs, attributeEntry{Name: attr.Name, Value: html.EscapeString(attr.Value)})
	}
	return attrs
}

// shouldSkipDynamicAttr reports whether an attribute should be skipped because
// it is handled dynamically.
//
// Takes name (string) which is the attribute name to check.
// Takes skipClass (bool) which is true when class attributes are skipped.
// Takes skipStyle (bool) which is true when style attributes are skipped.
//
// Returns bool which is true when the attribute matches a dynamic type to skip.
func shouldSkipDynamicAttr(name string, skipClass, skipStyle bool) bool {
	if skipClass && strings.EqualFold(name, attributeNameClass) {
		return true
	}
	return skipStyle && strings.EqualFold(name, attributeNameStyle)
}

// computeAttrHash builds a unique key from a list of attributes.
//
// Takes attrs ([]attributeEntry) which contains the attributes to turn into a key.
//
// Returns string which is a key for finding duplicate attribute sets.
func computeAttrHash(attrs []attributeEntry) string {
	var builder strings.Builder
	for _, a := range attrs {
		builder.WriteString(a.Name)
		_ = builder.WriteByte(0)
		builder.WriteString(a.Value)
		_ = builder.WriteByte(0)
	}
	return builder.String()
}

// staticNormaliseToCallExpr converts an expression to a CallExpr.
// Uses the same logic as attribute_emitter_actions.normaliseToCallExpr.
//
// When the expression is a bare Identifier, wraps it with an implicit $event
// argument, making bare handler equivalent to handler($event).
//
// Takes expression (ast_domain.Expression) which is the expression to convert.
//
// Returns *ast_domain.CallExpression which is the converted call
// expression, or nil if the expression is not a CallExpr or
// Identifier.
func staticNormaliseToCallExpr(expression ast_domain.Expression) *ast_domain.CallExpression {
	if ce, isCall := expression.(*ast_domain.CallExpression); isCall {
		return ce
	}
	if identifier, isIdent := expression.(*ast_domain.Identifier); isIdent {
		return &ast_domain.CallExpression{
			Callee: identifier,
			Args: []ast_domain.Expression{
				&ast_domain.Identifier{Name: "$event"},
			},
		}
	}
	return nil
}

// extractStaticArgs converts a slice of expressions into action arguments.
//
// Takes exprs ([]ast_domain.Expression) which contains the expressions to
// convert.
//
// Returns []templater_dto.ActionArgument which holds the converted arguments.
// Returns bool which is true if all arguments are static.
func extractStaticArgs(exprs []ast_domain.Expression) ([]templater_dto.ActionArgument, bool) {
	arguments := make([]templater_dto.ActionArgument, 0, len(exprs))
	for _, expression := range exprs {
		argument, ok := extractStaticArg(expression)
		if !ok {
			return nil, false
		}
		arguments = append(arguments, argument)
	}
	return arguments, true
}

// extractStaticArg converts a static expression to an ActionArgument.
//
// Takes expression (ast_domain.Expression) which is the expression to convert.
//
// Returns templater_dto.ActionArgument which holds the converted argument value.
// Returns bool which is true if the expression is static, false otherwise.
func extractStaticArg(expression ast_domain.Expression) (templater_dto.ActionArgument, bool) {
	switch e := expression.(type) {
	case *ast_domain.Identifier:
		if e.Name == "$event" {
			return templater_dto.ActionArgument{Type: "e"}, true
		}
		if e.Name == "$form" {
			return templater_dto.ActionArgument{Type: "f"}, true
		}
		return templater_dto.ActionArgument{}, false
	case *ast_domain.StringLiteral:
		return templater_dto.ActionArgument{Type: argTypeStatic, Value: e.Value}, true
	case *ast_domain.IntegerLiteral:
		return templater_dto.ActionArgument{Type: argTypeStatic, Value: e.Value}, true
	case *ast_domain.FloatLiteral:
		return templater_dto.ActionArgument{Type: argTypeStatic, Value: e.Value}, true
	case *ast_domain.BooleanLiteral:
		return templater_dto.ActionArgument{Type: argTypeStatic, Value: e.Value}, true
	case *ast_domain.ObjectLiteral:
		staticObject, ok := extractStaticObject(e)
		if !ok {
			return templater_dto.ActionArgument{}, false
		}
		return templater_dto.ActionArgument{Type: argTypeStatic, Value: staticObject}, true
	case *ast_domain.ArrayLiteral:
		arr, ok := extractStaticArray(e)
		if !ok {
			return templater_dto.ActionArgument{}, false
		}
		return templater_dto.ActionArgument{Type: argTypeStatic, Value: arr}, true
	default:
		return templater_dto.ActionArgument{}, false
	}
}

// extractStaticValue extracts a static value from an expression, returning the
// raw Go value rather than an ActionArgument. Used for recursively
// extracting nested object and array literal contents.
//
// Takes expression (ast_domain.Expression) which is the
// expression to extract from.
//
// Returns any which is the extracted value.
// Returns bool which is true if the expression is a static literal.
func extractStaticValue(expression ast_domain.Expression) (any, bool) {
	switch e := expression.(type) {
	case *ast_domain.StringLiteral:
		return e.Value, true
	case *ast_domain.IntegerLiteral:
		return e.Value, true
	case *ast_domain.FloatLiteral:
		return e.Value, true
	case *ast_domain.BooleanLiteral:
		return e.Value, true
	case *ast_domain.ObjectLiteral:
		return extractStaticObject(e)
	case *ast_domain.ArrayLiteral:
		return extractStaticArray(e)
	default:
		return nil, false
	}
}

// extractStaticObject recursively extracts all key-value pairs from an object
// literal into a map[string]any. All values must be static literals.
//
// Takes objectLiteral (*ast_domain.ObjectLiteral) which is the object
// literal to extract.
//
// Returns map[string]any which holds the extracted key-value pairs.
// Returns bool which is true if all values are static literals.
func extractStaticObject(objectLiteral *ast_domain.ObjectLiteral) (map[string]any, bool) {
	result := make(map[string]any, len(objectLiteral.Pairs))
	for key, valExpr := range objectLiteral.Pairs {
		value, ok := extractStaticValue(valExpr)
		if !ok {
			return nil, false
		}
		result[key] = value
	}
	return result, true
}

// extractStaticArray recursively extracts all elements from an array literal
// into a []any. All elements must be static literals.
//
// Takes arr (*ast_domain.ArrayLiteral) which is the array literal to extract.
//
// Returns []any which holds the extracted element values.
// Returns bool which is true if all elements are static literals.
func extractStaticArray(arr *ast_domain.ArrayLiteral) ([]any, bool) {
	result := make([]any, 0, len(arr.Elements))
	for _, elemExpr := range arr.Elements {
		value, ok := extractStaticValue(elemExpr)
		if !ok {
			return nil, false
		}
		result = append(result, value)
	}
	return result, true
}

// preprocessEventsForPrerendering walks the node tree and encodes all static
// event directives as base64 JSON. This makes the prerendered HTML ready for
// the client-side JavaScript to parse.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to process.
// The node and its children are changed in place.
func preprocessEventsForPrerendering(node *ast_domain.TemplateNode) {
	if node == nil {
		return
	}
	encodeEventMap(node.OnEvents)
	encodeEventMap(node.CustomEvents)
	for _, child := range node.Children {
		preprocessEventsForPrerendering(child)
	}
}

// encodeEventMap encodes the RawExpression of each static event directive in
// the map to a base64 JSON payload.
//
// Takes events (map[string][]ast_domain.Directive) which contains the event
// directives to encode.
func encodeEventMap(events map[string][]ast_domain.Directive) {
	for eventName := range events {
		directives := events[eventName]
		for i := range directives {
			if !directives[i].IsStaticEvent {
				continue
			}
			payload := encodeDirectivePayload(&directives[i])
			if payload != "" {
				directives[i].RawExpression = payload
			}
		}
		events[eventName] = directives
	}
}

// encodeDirectivePayload extracts static argument values from a directive
// expression and encodes them as a base64 JSON payload string. This is a
// standalone version of buildStaticEventPayload for use during prerendering
// preprocessing.
//
// Takes d (*ast_domain.Directive) which contains the event handler expression.
//
// Returns string which is the base64-encoded payload, or empty if encoding
// fails.
func encodeDirectivePayload(d *ast_domain.Directive) string {
	callExpr := staticNormaliseToCallExpr(d.Expression)
	if callExpr == nil {
		return ""
	}
	functionName, ok := callExpr.Callee.(*ast_domain.Identifier)
	if !ok {
		return ""
	}
	arguments, valid := extractStaticArgs(callExpr.Args)
	if !valid {
		return ""
	}
	payload := templater_dto.ActionPayload{
		Function: functionName.Name,
		Args:     arguments,
	}
	return generator_helpers.EncodeStaticActionPayload(payload)
}
