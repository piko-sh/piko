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
	"slices"

	goast "go/ast"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_domain"
)

// nodeEmissionParams groups parameters for creating a nodeEmissionContext.
// Field ordering is set for memory alignment.
type nodeEmissionParams struct {
	// Node is the template node to emit.
	Node *ast_domain.TemplateNode

	// ParentSliceExpression is the slice expression to which
	// emitted nodes are appended.
	ParentSliceExpression goast.Expr

	// PartialScopeID is the HashedName of the current partial for CSS scoping.
	PartialScopeID string

	// MainComponentScope is the hashed name of the main component being built.
	MainComponentScope string

	// Siblings holds the sibling template nodes used during iteration.
	Siblings []*ast_domain.TemplateNode

	// Index is the position of this node among its siblings.
	Index int

	// IsRootNode indicates whether this node is at the top level of the template.
	IsRootNode bool
}

// nodeEmissionContext holds the data needed to emit a single template node.
type nodeEmissionContext struct {
	// ctx is the context for cancellation and timeout handling.
	ctx context.Context

	// parentSliceExpr is the slice expression to which emitted nodes are added.
	parentSliceExpr goast.Expr

	// node is the template node being emitted.
	node *ast_domain.TemplateNode

	// partialScopeID holds the HashedName of the current partial for CSS scoping.
	// Element nodes in a partial use this as their `partial` attribute value;
	// empty string for pages (no scoping).
	partialScopeID string

	// mainComponentScope is the HashedName of the main component being generated.
	// Used to distinguish slotted content (same scope as main) from nested partial
	// content (different scope).
	mainComponentScope string

	// loopIterInfo holds pre-extracted loop iterable info for p-for nodes.
	// When set, the for_emitter uses this instead of re-emitting the collection
	// expression, enabling accurate child slice capacity and avoiding
	// double-evaluation.
	loopIterInfo *LoopIterableInfo

	// siblings holds the sibling template nodes used during iteration.
	siblings []*ast_domain.TemplateNode

	// index is the position of the current node within its siblings.
	index int

	// isRootNode indicates whether this node is at the top level of the template.
	isRootNode bool
}

// emitNode is the main recursive function that chooses which emitter to use.
//
// Takes emitCtx (*nodeEmissionContext) which provides the emission context.
//
// Returns statements ([]goast.Stmt) which contains the generated Go statements.
// Returns nodesConsumed (int) which shows how many nodes were used.
// Returns diagnostics ([]*ast_domain.Diagnostic) which holds any problems found.
func (b *astBuilder) emitNode(emitCtx *nodeEmissionContext) (statements []goast.Stmt, nodesConsumed int, diagnostics []*ast_domain.Diagnostic) {
	return b.emitNodeWithContext(emitCtx)
}

// emitNodeWithContext converts a node into Go AST statements using the given
// context.
//
// Takes emitCtx (*nodeEmissionContext) which provides the node and settings
// for the emission.
//
// Returns []goast.Stmt which contains the generated Go AST statements.
// Returns int which is the number of nodes that were processed.
// Returns []*ast_domain.Diagnostic which contains any errors found.
func (b *astBuilder) emitNodeWithContext(emitCtx *nodeEmissionContext) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	if emitCtx.node.TagName == "piko:content" {
		return b.emitContentTag(emitCtx.ctx, emitCtx.node, emitCtx.parentSliceExpr)
	}

	nodeForEmission := b.prepareNodeForEmission(emitCtx)
	if diagnostic := b.validateNodeForEmission(emitCtx.node, nodeForEmission); diagnostic != nil {
		return nil, 1, []*ast_domain.Diagnostic{diagnostic}
	}

	return b.dispatchNodeEmission(emitCtx, nodeForEmission)
}

// prepareNodeForEmission prepares a node for emission by adding partial
// metadata if needed. Uses the context to avoid a flag parameter.
//
// Takes emitCtx (*nodeEmissionContext) which provides the emission context.
//
// Returns *ast_domain.TemplateNode which is the prepared node, either the
// original or with partial metadata added.
func (b *astBuilder) prepareNodeForEmission(emitCtx *nodeEmissionContext) *ast_domain.TemplateNode {
	if !b.shouldAddPartialMetadata(emitCtx) {
		return emitCtx.node
	}

	return b.addPartialMetadataToNode(emitCtx.node)
}

// shouldAddPartialMetadata determines if partial metadata should be added to
// a node.
//
// Uses context instead of a separate isRootNode flag. Always returns true for
// root element nodes in non-page partials, even if the node has PartialInfo
// from a nested partial. In that case, both partial IDs are accumulated.
//
// Takes emitCtx (*nodeEmissionContext) which provides the emission context
// including root node status.
//
// Returns bool which indicates whether partial metadata should be added.
func (b *astBuilder) shouldAddPartialMetadata(emitCtx *nodeEmissionContext) bool {
	return !b.emitter.config.IsPage && emitCtx.isRootNode && emitCtx.node.NodeType == ast_domain.NodeElement
}

// addPartialMetadataToNode clones a node and adds partial metadata attributes.
//
// When the node has nested PartialInfo (i.e., it is a partial's root that is
// itself another partial), the attributes accumulate values in space-separated
// format: "outer inner". This enables CSS selectors using [partial~=xxx] to
// match any partial in the chain.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to clone
// and augment with partial metadata.
//
// Returns *ast_domain.TemplateNode which is a deep clone of the input node
// with partial, partial_name, and optionally partial_src attributes added.
func (b *astBuilder) addPartialMetadataToNode(node *ast_domain.TemplateNode) *ast_domain.TemplateNode {
	nodeForEmission := node.DeepClone()

	mainComponent, err := generator_domain.GetMainComponent(b.emitter.AnnotationResult)
	if err != nil {
		return nodeForEmission
	}

	partialValue := mainComponent.HashedName
	partialNameValue := mainComponent.PartialName
	partialSrcValue := mainComponent.PartialSrc

	if node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil {
		nestedPackage := node.GoAnnotations.PartialInfo.PartialPackageName
		if nestedVC, ok := b.emitter.AnnotationResult.VirtualModule.ComponentsByHash[nestedPackage]; ok {
			partialValue += " " + nestedVC.HashedName
			partialNameValue += " " + nestedVC.PartialName
			if nestedVC.IsPublic {
				partialSrcValue += " " + nestedVC.PartialSrc
			}
		}
	}

	nodeForEmission.Attributes = append(nodeForEmission.Attributes,
		ast_domain.HTMLAttribute{
			Name:           "partial",
			Value:          partialValue,
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
		ast_domain.HTMLAttribute{
			Name:           "partial_name",
			Value:          partialNameValue,
			Location:       ast_domain.Location{},
			NameLocation:   ast_domain.Location{},
			AttributeRange: ast_domain.Range{},
		},
	)

	if mainComponent.IsPublic {
		nodeForEmission.Attributes = append(nodeForEmission.Attributes,
			ast_domain.HTMLAttribute{
				Name:           "partial_src",
				Value:          partialSrcValue,
				Location:       ast_domain.Location{},
				NameLocation:   ast_domain.Location{},
				AttributeRange: ast_domain.Range{},
			},
		)
	}

	return nodeForEmission
}

// validateNodeForEmission checks whether a node can be emitted after
// preparation.
//
// Takes originalNode (*ast_domain.TemplateNode) which is the node before
// preparation.
// Takes preparedNode (*ast_domain.TemplateNode) which is the node after
// preparation.
//
// Returns *ast_domain.Diagnostic which describes any error found, or nil if
// the node is valid.
func (b *astBuilder) validateNodeForEmission(
	originalNode *ast_domain.TemplateNode,
	preparedNode *ast_domain.TemplateNode,
) *ast_domain.Diagnostic {
	if preparedNode == originalNode {
		return nil
	}

	_, err := generator_domain.GetMainComponent(b.emitter.AnnotationResult)
	if err != nil {
		return ast_domain.NewDiagnostic(
			ast_domain.Error,
			"Internal Emitter Error: "+err.Error(),
			originalNode.TagName,
			originalNode.Location,
			"",
		)
	}

	return nil
}

// dispatchNodeEmission sends a template node to the right emitter based on its
// type and directives.
//
// Takes emitCtx (*nodeEmissionContext) which holds the current emission state.
// Takes nodeForEmission (*ast_domain.TemplateNode) which is the node to emit.
//
// Returns []goast.Stmt which contains the generated Go statements.
// Returns int which shows how many sibling nodes were used.
// Returns []*ast_domain.Diagnostic which contains any issues found.
func (b *astBuilder) dispatchNodeEmission(
	emitCtx *nodeEmissionContext,
	nodeForEmission *ast_domain.TemplateNode,
) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	if nodeForEmission.DirFor != nil {
		if emitCtx.loopIterInfo != nil {
			statements, diagnostics := b.forEmitter.emitWithExtractedIterable(
				emitCtx.ctx, nodeForEmission, emitCtx.parentSliceExpr,
				emitCtx.loopIterInfo, emitCtx.partialScopeID, emitCtx.mainComponentScope,
			)
			return statements, 1, diagnostics
		}
		statements, diagnostics := b.forEmitter.emit(emitCtx.ctx, nodeForEmission, emitCtx.parentSliceExpr, emitCtx.partialScopeID, emitCtx.mainComponentScope)
		return statements, 1, diagnostics
	}

	if nodeForEmission.DirIf != nil {
		return b.ifEmitter.emitChain(emitCtx.ctx, nodeForEmission, emitCtx.siblings, emitCtx.index, emitCtx.parentSliceExpr, emitCtx.partialScopeID, emitCtx.mainComponentScope)
	}

	if b.isElseClauseNode(emitCtx.node, nodeForEmission) {
		return nil, 1, nil
	}

	if b.canEmitAsStatic(nodeForEmission) {
		return b.emitStaticNode(emitCtx.ctx, nodeForEmission, emitCtx.parentSliceExpr, emitCtx.partialScopeID)
	}

	if nodeForEmission.NodeType == ast_domain.NodeFragment {
		return b.emitFragment(emitCtx, nodeForEmission)
	}

	return b.emitDynamicNode(emitCtx.ctx, nodeForEmission, emitCtx.parentSliceExpr, emitCtx.partialScopeID)
}

// isElseClauseNode checks if a node is an else or else-if clause.
//
// Takes original (*ast_domain.TemplateNode) which is the original node to check.
// Takes prepared (*ast_domain.TemplateNode) which is the prepared node to check.
//
// Returns bool which is true if either node has an else or else-if directive.
func (*astBuilder) isElseClauseNode(original, prepared *ast_domain.TemplateNode) bool {
	return original.DirElseIf != nil || original.DirElse != nil ||
		prepared.DirElseIf != nil || prepared.DirElse != nil
}

// canEmitAsStatic determines if a node can be emitted as a static node.
// Nodes with srcset data must go through dynamic emission to generate srcset
// attributes.
//
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true if the node can be emitted statically.
func (b *astBuilder) canEmitAsStatic(node *ast_domain.TemplateNode) bool {
	return b.emitter.config.EnableStaticHoisting &&
		node.GoAnnotations != nil &&
		node.GoAnnotations.IsStatic &&
		!b.nodeContainsForLoops(node) &&
		!b.nodeContainsPikoContent(node) &&
		!b.nodeContainsRichText(node) &&
		len(node.GoAnnotations.Srcset) == 0
}

// emitStaticNode outputs a template node as a static node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to output.
// Takes parentSliceExpr (goast.Expr) which is the slice to append the node to.
// Takes partialScopeID (string) which is the current partial's HashedName for
// CSS scoping.
//
// Returns []goast.Stmt which contains the statements to append the static node.
// Returns int which is the count of nodes output.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from
// registering the static node.
func (b *astBuilder) emitStaticNode(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
	partialScopeID string,
) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	staticVarIdent, staticDiags := b.staticEmitter.registerStaticNode(ctx, node, partialScopeID)
	statements := []goast.Stmt{appendToSlice(parentSliceExpr, staticVarIdent)}
	return statements, 1, staticDiags
}

// emitFragment creates Go AST statements for a fragment node.
//
// Takes emitCtx (*nodeEmissionContext) which provides the context for code
// generation.
// Takes nodeForEmission (*ast_domain.TemplateNode) which is the fragment node
// to process.
//
// Returns []goast.Stmt which contains the generated Go AST statements.
// Returns int which is the number of statements created.
// Returns []*ast_domain.Diagnostic which contains any errors or warnings.
func (b *astBuilder) emitFragment(
	emitCtx *nodeEmissionContext,
	nodeForEmission *ast_domain.TemplateNode,
) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	if b.fragmentHasDynamicFeatures(nodeForEmission) {
		return b.emitDynamicNode(emitCtx.ctx, nodeForEmission, emitCtx.parentSliceExpr, emitCtx.partialScopeID)
	}

	return b.emitFragmentChildren(emitCtx, nodeForEmission)
}

// fragmentHasDynamicFeatures checks if a fragment has dynamic features
// requiring special handling.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to check.
//
// Returns bool which is true if the node has partial info annotations or
// dynamic attributes.
func (*astBuilder) fragmentHasDynamicFeatures(node *ast_domain.TemplateNode) bool {
	return (node.GoAnnotations != nil && node.GoAnnotations.PartialInfo != nil) ||
		len(node.DynamicAttributes) > 0
}

// emitFragmentChildren emits all children of a fragment node.
//
// Takes emitCtx (*nodeEmissionContext) which provides the emission context.
// Takes nodeForEmission (*ast_domain.TemplateNode) which is the fragment node
// whose children are emitted.
//
// Returns []goast.Stmt which contains the statements for all child nodes.
// Returns int which is the number of nodes consumed (always 1).
// Returns []*ast_domain.Diagnostic which contains any diagnostics from
// child emissions.
func (b *astBuilder) emitFragmentChildren(
	emitCtx *nodeEmissionContext,
	nodeForEmission *ast_domain.TemplateNode,
) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	var allStmts []goast.Stmt
	var allDiags []*ast_domain.Diagnostic

	i := 0
	for i < len(nodeForEmission.Children) {
		child := nodeForEmission.Children[i]
		childCtx := newNodeEmissionContext(emitCtx.ctx, nodeEmissionParams{
			Node:                  child,
			ParentSliceExpression: emitCtx.parentSliceExpr,
			Index:                 i,
			Siblings:              nodeForEmission.Children,
			IsRootNode:            emitCtx.isRootNode,
			PartialScopeID:        emitCtx.partialScopeID,
			MainComponentScope:    emitCtx.mainComponentScope,
		})
		childStmts, consumed, childDiags := b.emitNode(childCtx)
		allStmts = append(allStmts, childStmts...)
		allDiags = append(allDiags, childDiags...)
		i += consumed
	}

	return allStmts, 1, allDiags
}

// emitDynamicNode creates statements for a dynamic template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
// Takes parentSliceExpr (goast.Expr) which is the slice to append the new node
// to.
// Takes partialScopeID (string) which is the current partial's HashedName for
// CSS scoping.
//
// Returns []goast.Stmt which contains the statements to create and append the
// node.
// Returns int which is the number of nodes created (always 1).
// Returns []*ast_domain.Diagnostic which contains any warnings or errors from
// processing.
func (b *astBuilder) emitDynamicNode(
	ctx context.Context,
	node *ast_domain.TemplateNode,
	parentSliceExpr goast.Expr,
	partialScopeID string,
) ([]goast.Stmt, int, []*ast_domain.Diagnostic) {
	nodeVarName, nodeStmts, nodeDiags := b.nodeEmitter.emit(ctx, node, partialScopeID)
	nodeStmts = append(nodeStmts, appendToSlice(parentSliceExpr, cachedIdent(nodeVarName)))
	return nodeStmts, 1, nodeDiags
}

// nodeContainsForLoops checks if a node or any of its descendants contain a
// p-for directive. This prevents treating nodes with internal loops as static,
// which would cause issues with dynamic key expressions.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to check.
//
// Returns bool which is true if a p-for directive is found in the subtree.
func (b *astBuilder) nodeContainsForLoops(node *ast_domain.TemplateNode) bool {
	if node == nil {
		return false
	}

	if node.DirFor != nil {
		return true
	}

	return slices.ContainsFunc(node.Children, b.nodeContainsForLoops)
}

// nodeContainsPikoContent checks if a node or any of its descendants
// contain a <piko:content /> tag.
//
// This prevents treating nodes with piko:content children as static, since
// piko:content requires runtime fetching of contentAST from CollectionData.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to check.
//
// Returns bool which is true if a piko:content tag is found.
func (b *astBuilder) nodeContainsPikoContent(node *ast_domain.TemplateNode) bool {
	if node == nil {
		return false
	}

	if node.TagName == "piko:content" {
		return true
	}

	return slices.ContainsFunc(node.Children, b.nodeContainsPikoContent)
}

// nodeContainsRichText checks if a node or any of its descendants contain
// RichText (interpolations). This prevents treating nodes with dynamic
// interpolations as static, since RichText requires runtime evaluation of
// expressions like {{ state.Query }}.
//
// Takes node (*ast_domain.TemplateNode) which is the root node to check.
//
// Returns bool which is true if the node or any descendant contains RichText.
func (b *astBuilder) nodeContainsRichText(node *ast_domain.TemplateNode) bool {
	if node == nil {
		return false
	}

	if len(node.RichText) > 0 {
		return true
	}

	return slices.ContainsFunc(node.Children, b.nodeContainsRichText)
}

// newNodeEmissionContext creates a context for template node emission.
//
// Takes params (nodeEmissionParams) which holds the emission settings.
//
// Returns *nodeEmissionContext which is ready for use in template output.
func newNodeEmissionContext(ctx context.Context, params nodeEmissionParams) *nodeEmissionContext {
	return &nodeEmissionContext{
		siblings:           params.Siblings,
		ctx:                ctx,
		parentSliceExpr:    params.ParentSliceExpression,
		node:               params.Node,
		partialScopeID:     params.PartialScopeID,
		mainComponentScope: params.MainComponentScope,
		loopIterInfo:       nil,
		index:              params.Index,
		isRootNode:         params.IsRootNode,
	}
}
