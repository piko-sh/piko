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

package pml_domain

import (
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/pml/pml_dto"
)

var _ Transformer = (*engine)(nil)

const (
	// defaultContainerWidth is the default width in pixels for email containers.
	defaultContainerWidth = 600.0

	// maxLogPreviewLength is the maximum length of text content shown in debug
	// logs.
	maxLogPreviewLength = 50
)

// engine implements the Transformer interface.
// It orchestrates a two-pass transformation: autowrap to ensure structural
// validity, then transform to convert PML components into email-safe HTML.
type engine struct {
	// registry stores PML components by tag name for lookup during transformation.
	registry ComponentRegistry

	// mediaQueryCollector gathers CSS media queries during transformation.
	mediaQueryCollector MediaQueryCollector

	// msoConditionalCollector gathers MSO conditional comments for Outlook
	// email compatibility.
	msoConditionalCollector MSOConditionalCollector
}

// Transform applies PML changes to a template using a two-pass method.
// It implements the Transformer interface.
//
// Takes ast (*ast_domain.TemplateAST) which is the template to transform.
// Takes config (*pml_dto.Config) which sets how the transformation behaves.
//
// Returns *ast_domain.TemplateAST which is the changed template.
// Returns string which holds the CSS made for the breakpoint.
// Returns []*Error which lists any problems found during the process.
func (e *engine) Transform(
	ctx context.Context,
	ast *ast_domain.TemplateAST,
	config *pml_dto.Config,
) (*ast_domain.TemplateAST, string, []*Error) {
	ctx, l := logger_domain.From(ctx, log)
	if ast == nil {
		return nil, "", []*Error{{Message: "Cannot transform nil AST", Severity: SeverityError}}
	}
	if config == nil {
		config = pml_dto.DefaultConfig()
	}

	l.Internal("Starting PML transformation", logger_domain.String("breakpoint", config.Breakpoint))

	l.Internal("PML Pass 1: Running autowrapping.")
	e.autowrapTree(ctx, ast)
	l.Internal("PML autowrapping pass complete.")

	rootCtx := e.setupTransformationContext(config)
	if ast.SourcePath != nil {
		rootCtx.SourceFilePath = *ast.SourcePath
	}
	transformedRootNodes, allDiagnostics := e.transformRootNodes(ctx, ast.RootNodes, rootCtx)

	transformedAST := e.buildTransformedAST(ast, transformedRootNodes)
	finalCSS := e.generateFinalCSS(config.Breakpoint)

	l.Internal("PML transformation completed", logger_domain.Int("diagnosticCount", len(allDiagnostics)))

	return transformedAST, finalCSS, allDiagnostics
}

// TransformForEmail transforms a template AST for email rendering contexts.
//
// It initialises an email-specific transformation context with asset registry
// support and returns the collected asset requests alongside the transformed
// AST. Use when rendering email templates to enable automatic CID (Content-ID)
// embedding of local assets referenced in <pml-img> tags.
//
// Takes ast (*ast_domain.TemplateAST) which is the template to transform.
// Takes config (*pml_dto.Config) which specifies rendering options.
//
// Returns *ast_domain.TemplateAST which is the transformed template AST.
// Returns string which contains the generated CSS including media queries
// and MSO conditionals.
// Returns []*email_dto.EmailAssetRequest which lists assets that need to be
// fetched and embedded.
// Returns []*Error which contains warnings and errors encountered during
// transformation.
func (e *engine) TransformForEmail(
	ctx context.Context,
	ast *ast_domain.TemplateAST,
	config *pml_dto.Config,
) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*Error) {
	ctx, l := logger_domain.From(ctx, log)
	if ast == nil {
		return nil, "", nil, []*Error{{Message: "Cannot transform nil AST", Severity: SeverityError}}
	}
	if config == nil {
		config = pml_dto.DefaultConfig()
	}

	l.Internal("Starting PML email transformation", logger_domain.String("breakpoint", config.Breakpoint))

	l.Internal("PML Pass 1: Running autowrapping.")
	e.autowrapTree(ctx, ast)
	l.Internal("PML autowrapping pass complete.")

	rootCtx := e.setupEmailTransformationContext(config)
	if ast.SourcePath != nil {
		rootCtx.SourceFilePath = *ast.SourcePath
	}
	transformedRootNodes, allDiagnostics := e.transformRootNodesForEmail(ctx, ast.RootNodes, rootCtx)

	transformedAST := e.buildTransformedAST(ast, transformedRootNodes)
	finalCSS := e.generateFinalCSS(config.Breakpoint)
	assetRequests := e.extractAssetRequests(rootCtx)

	l.Internal("PML email transformation completed",
		logger_domain.Int("diagnosticCount", len(allDiagnostics)),
		logger_domain.Int("assetRequestCount", len(assetRequests)),
	)

	return transformedAST, finalCSS, assetRequests, allDiagnostics
}

// setupTransformationContext creates and sets up the root transformation
// context.
//
// Takes config (*pml_dto.Config) which specifies the transformation settings.
//
// Returns *TransformationContext which is the root context ready for use.
func (e *engine) setupTransformationContext(config *pml_dto.Config) *TransformationContext {
	rootCtx := newRootTransformationContext(config, defaultContainerWidth, e.registry)
	rootCtx.MediaQueryCollector = e.mediaQueryCollector
	rootCtx.MSOConditionalCollector = e.msoConditionalCollector
	return rootCtx
}

// setupEmailTransformationContext creates and configures the email-specific
// transformation context.
//
// Takes config (*pml_dto.Config) which provides the email configuration.
//
// Returns *TransformationContext which is the configured context ready for
// email transformation.
func (e *engine) setupEmailTransformationContext(config *pml_dto.Config) *TransformationContext {
	rootCtx := newRootTransformationContextForEmail(config, defaultContainerWidth, e.registry)
	rootCtx.MediaQueryCollector = e.mediaQueryCollector
	rootCtx.MSOConditionalCollector = e.msoConditionalCollector
	return rootCtx
}

// transformRootNodes transforms all root nodes and collects diagnostics.
//
// Takes rootNodes ([]*ast_domain.TemplateNode) which are the root template
// nodes to transform.
// Takes rootCtx (*TransformationContext) which provides the transformation
// context.
//
// Returns []*ast_domain.TemplateNode which contains the transformed nodes.
// Returns []*Error which contains any diagnostics collected during
// transformation.
func (e *engine) transformRootNodes(
	ctx context.Context,
	rootNodes []*ast_domain.TemplateNode,
	rootCtx *TransformationContext,
) ([]*ast_domain.TemplateNode, []*Error) {
	return e.doTransformRootNodes(ctx, rootNodes, rootCtx, "PML Pass 2: Running component transformation.")
}

// transformRootNodesForEmail transforms all root nodes for email rendering
// and collects diagnostics.
//
// Takes rootNodes ([]*ast_domain.TemplateNode) which contains the template
// nodes to transform.
// Takes rootCtx (*TransformationContext) which provides the transformation
// context.
//
// Returns []*ast_domain.TemplateNode which contains the transformed nodes.
// Returns []*Error which contains any diagnostics collected during
// transformation.
func (e *engine) transformRootNodesForEmail(
	ctx context.Context,
	rootNodes []*ast_domain.TemplateNode,
	rootCtx *TransformationContext,
) ([]*ast_domain.TemplateNode, []*Error) {
	return e.doTransformRootNodes(ctx, rootNodes, rootCtx, "PML Pass 2: Running email component transformation.")
}

// doTransformRootNodes iterates over root nodes, transforms each one, and
// collects diagnostics. It is the shared implementation for both standard
// and email transformation passes.
//
// Takes rootNodes ([]*ast_domain.TemplateNode) which are the root template
// nodes to transform.
// Takes rootCtx (*TransformationContext) which provides the transformation
// context.
// Takes logMessage (string) which is the message to log before processing.
//
// Returns []*ast_domain.TemplateNode which contains the transformed nodes.
// Returns []*Error which contains any diagnostics collected during
// transformation.
func (e *engine) doTransformRootNodes(
	ctx context.Context,
	rootNodes []*ast_domain.TemplateNode,
	rootCtx *TransformationContext,
	logMessage string,
) ([]*ast_domain.TemplateNode, []*Error) {
	_, l := logger_domain.From(ctx, log)
	var allDiagnostics []*Error
	transformedRootNodes := make([]*ast_domain.TemplateNode, 0, len(rootNodes))

	l.Internal(logMessage)
	for _, rootNode := range rootNodes {
		transformedNode, diagnostics := e.transformNode(rootNode, rootCtx, nil, nil)
		if transformedNode != nil {
			transformedRootNodes = append(transformedRootNodes, transformedNode)
		}
		allDiagnostics = append(allDiagnostics, diagnostics...)
	}

	return transformedRootNodes, allDiagnostics
}

// buildTransformedAST constructs a transformed AST by combining the original
// AST metadata with new root nodes.
//
// Takes originalAST (*ast_domain.TemplateAST) which provides the source
// metadata and diagnostics to preserve.
// Takes transformedRootNodes ([]*ast_domain.TemplateNode) which contains the
// new root nodes for the transformed tree.
//
// Returns *ast_domain.TemplateAST which is the new AST with transformed nodes
// and preserved metadata from the original.
func (*engine) buildTransformedAST(
	originalAST *ast_domain.TemplateAST,
	transformedRootNodes []*ast_domain.TemplateNode,
) *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes:         transformedRootNodes,
		Diagnostics:       originalAST.Diagnostics,
		SourcePath:        originalAST.SourcePath,
		SourceSize:        originalAST.SourceSize,
		ExpiresAtUnixNano: nil,
		Metadata:          nil,
		Tidied:            false,
	}
}

// generateFinalCSS generates the final CSS output from collected media queries
// and MSO conditionals.
//
// Takes breakpoint (string) which specifies the target breakpoint for CSS.
//
// Returns string which contains the combined CSS output.
func (e *engine) generateFinalCSS(breakpoint string) string {
	mediaQueryCSS := e.mediaQueryCollector.GenerateCSS(breakpoint)
	msoConditionalBlock := e.msoConditionalCollector.GenerateConditionalBlock()

	var finalCSSBuilder strings.Builder
	if mediaQueryCSS != "" {
		finalCSSBuilder.WriteString(mediaQueryCSS)
	}
	if msoConditionalBlock != "" {
		if finalCSSBuilder.Len() > 0 {
			finalCSSBuilder.WriteString("\n")
		}
		finalCSSBuilder.WriteString(msoConditionalBlock)
	}

	return finalCSSBuilder.String()
}

// extractAssetRequests collects all asset requests from the transformation
// context.
//
// Takes rootCtx (*TransformationContext) which provides the context containing
// the email asset registry.
//
// Returns []*email_dto.EmailAssetRequest which contains the collected asset
// requests, or nil if no registry exists.
func (*engine) extractAssetRequests(rootCtx *TransformationContext) []*email_dto.EmailAssetRequest {
	var assetRequests []*email_dto.EmailAssetRequest
	if rootCtx.EmailAssetRegistry != nil {
		assetRequests = rootCtx.EmailAssetRegistry.Requests
	}
	return assetRequests
}

// autowrapTree starts the post-order autowrapping traversal for the entire
// AST.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes ast (*ast_domain.TemplateAST) which provides the tree to traverse.
func (e *engine) autowrapTree(ctx context.Context, ast *ast_domain.TemplateAST) {
	e.autowrapPostOrderRecursive(ctx, ast.RootNodes, nil)
}

// autowrapPostOrderRecursive performs a post-order walk that processes
// all of a node's children first and then applies autowrapping to that
// node's direct children, using a bottom-up approach to prevent
// infinite loops.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes nodes ([]*ast_domain.TemplateNode) which are the nodes to process.
func (e *engine) autowrapPostOrderRecursive(ctx context.Context, nodes []*ast_domain.TemplateNode, _ /* parent */ *ast_domain.TemplateNode) {
	if nodes == nil {
		return
	}
	for _, node := range nodes {
		isEndingTag := e.checkAndLogEndingTag(ctx, node)

		if !isEndingTag {
			e.autowrapPostOrderRecursive(ctx, node.Children, node)
		}

		if !isEndingTag {
			node.Children = autowrapChildren(node.Children, node)
		}
	}
}

// checkAndLogEndingTag checks if a node is an ending-tag PML component and
// logs debug information.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes node (*ast_domain.TemplateNode) which is the node to check.
//
// Returns bool which is true when the node is an ending-tag component.
func (e *engine) checkAndLogEndingTag(ctx context.Context, node *ast_domain.TemplateNode) bool {
	if node.NodeType != ast_domain.NodeElement || !strings.HasPrefix(node.TagName, "pml-") {
		return false
	}

	if e.registry == nil {
		return false
	}

	comp, found := e.registry.Get(node.TagName)
	if !found || !comp.IsEndingTag() {
		return false
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("Skipping autowrap for ending-tag component",
		logger_domain.String("tagName", node.TagName),
		logger_domain.Int("childCount", len(node.Children)))

	e.logFirstChildDetails(ctx, node)

	return true
}

// logFirstChildDetails logs details about the first child of a node for
// debugging purposes.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes node (*ast_domain.TemplateNode) which is the parent node to inspect.
func (*engine) logFirstChildDetails(ctx context.Context, node *ast_domain.TemplateNode) {
	if len(node.Children) == 0 {
		return
	}

	_, l := logger_domain.From(ctx, log)
	firstChild := node.Children[0]
	textPreview := firstChild.TextContent
	if len(textPreview) > maxLogPreviewLength {
		textPreview = textPreview[:maxLogPreviewLength]
	}

	l.Trace("First child details",
		logger_domain.String("nodeType", firstChild.NodeType.String()),
		logger_domain.String("tagName", firstChild.TagName),
		logger_domain.String("textContent", textPreview))
}

// transformNode recursively transforms a single node and its children
// as the core of the second pass, assuming a structurally valid AST
// and performing no autowrapping.
//
// Takes node (*ast_domain.TemplateNode) which is the node to transform.
// Takes ctx (*TransformationContext) which provides the transformation
// state.
// Takes parentNode (*ast_domain.TemplateNode) which is the parent
// template node.
// Takes parentComponent (Component) which is the parent component
// instance.
//
// Returns *ast_domain.TemplateNode which is the transformed node.
// Returns []*Error which contains any diagnostics from the
// transformation.
func (e *engine) transformNode(
	node *ast_domain.TemplateNode,
	ctx *TransformationContext,
	parentNode *ast_domain.TemplateNode,
	parentComponent Component,
) (*ast_domain.TemplateNode, []*Error) {
	if node == nil {
		return nil, nil
	}

	if node.NodeType == ast_domain.NodeElement && strings.HasPrefix(node.TagName, "pml-") {
		return e.transformPMLComponent(node, ctx, parentNode, parentComponent)
	}

	return e.transformNonPMLNode(node, ctx)
}

// transformPMLComponent handles the transformation of a PML component node.
//
// Takes node (*ast_domain.TemplateNode) which is the component node to
// transform.
// Takes ctx (*TransformationContext) which provides the transformation state.
// Takes parentNode (*ast_domain.TemplateNode) which is the parent template
// node.
// Takes parentComponent (Component) which is the parent component instance.
//
// Returns *ast_domain.TemplateNode which is the transformed node.
// Returns []*Error which contains any diagnostics from the transformation.
func (e *engine) transformPMLComponent(
	node *ast_domain.TemplateNode,
	ctx *TransformationContext,
	parentNode *ast_domain.TemplateNode,
	parentComponent Component,
) (*ast_domain.TemplateNode, []*Error) {
	comp, found := e.registry.Get(node.TagName)
	if !found {
		diagnostic := &Error{
			Message:  fmt.Sprintf("Unknown PML component: %s", node.TagName),
			TagName:  node.TagName,
			Severity: SeverityError,
			Location: node.Location,
		}
		return node, []*Error{diagnostic}
	}

	childCtx := e.createChildContext(ctx, node, comp, parentNode, parentComponent)
	nodeToTransform := node.DeepClone()

	var allDiagnostics []*Error

	if !comp.IsEndingTag() {
		transformedChildren, childDiagnostics := e.transformChildren(nodeToTransform.Children, childCtx, nodeToTransform, comp)
		nodeToTransform.Children = transformedChildren
		allDiagnostics = append(allDiagnostics, childDiagnostics...)
	}

	transformedNode, transformDiagnostics := comp.Transform(nodeToTransform, childCtx)
	allDiagnostics = append(allDiagnostics, transformDiagnostics...)
	allDiagnostics = append(allDiagnostics, childCtx.Diagnostics()...)

	return transformedNode, allDiagnostics
}

// transformNonPMLNode handles the transformation of non-PML nodes by recursing
// on their children.
//
// Takes node (*ast_domain.TemplateNode) which is the node to transform.
// Takes ctx (*TransformationContext) which provides the transformation state.
//
// Returns *ast_domain.TemplateNode which is a deep clone with transformed
// children.
// Returns []*Error which contains any diagnostics from child transformations.
func (e *engine) transformNonPMLNode(
	node *ast_domain.TemplateNode,
	ctx *TransformationContext,
) (*ast_domain.TemplateNode, []*Error) {
	transformedChildren, allDiagnostics := e.transformChildren(node.Children, ctx, node, nil)

	newNode := node.DeepClone()
	newNode.Children = transformedChildren
	return newNode, allDiagnostics
}

// transformChildren transforms a slice of child nodes.
//
// Takes children ([]*ast_domain.TemplateNode) which are the nodes to transform.
// Takes ctx (*TransformationContext) which provides the transformation context.
// Takes parentNode (*ast_domain.TemplateNode) which is the parent of the
// children.
// Takes parentComponent (Component) which is the component containing the
// nodes.
//
// Returns []*ast_domain.TemplateNode which are the transformed child nodes.
// Returns []*Error which contains any errors from the transformation.
func (e *engine) transformChildren(
	children []*ast_domain.TemplateNode,
	ctx *TransformationContext,
	parentNode *ast_domain.TemplateNode,
	parentComponent Component,
) ([]*ast_domain.TemplateNode, []*Error) {
	var allDiagnostics []*Error
	transformedChildren := make([]*ast_domain.TemplateNode, 0, len(children))

	for _, child := range children {
		transformedChild, childDiagnostics := e.transformNode(child, ctx, parentNode, parentComponent)
		if transformedChild != nil {
			transformedChildren = append(transformedChildren, transformedChild)
		}
		allDiagnostics = append(allDiagnostics, childDiagnostics...)
	}

	return transformedChildren, allDiagnostics
}

// createChildContext creates a transformation context for a child component.
//
// Takes ctx (*TransformationContext) which provides the current context to
// clone.
// Takes node (*ast_domain.TemplateNode) which is the child node being
// processed.
// Takes comp (Component) which is the child component.
// Takes parentNode (*ast_domain.TemplateNode) which is the parent template
// node.
// Takes parentComponent (Component) which is the parent component.
//
// Returns *TransformationContext which is the new context for the child with
// sibling count and group state set based on the parent.
func (e *engine) createChildContext(
	ctx *TransformationContext,
	node *ast_domain.TemplateNode,
	comp Component,
	parentNode *ast_domain.TemplateNode,
	parentComponent Component,
) *TransformationContext {
	childCtx := ctx.CloneForChild(node, comp, parentNode, parentComponent)

	if parentNode != nil {
		childCtx.SiblingCount = e.countPMLSiblings(parentNode)
	}

	if parentComponent != nil && parentComponent.TagName() == "pml-no-stack" {
		childCtx.IsInsideGroup = true
	}

	return childCtx
}

// countPMLSiblings counts the number of PML component siblings in a parent
// node.
//
// Takes parentNode (*ast_domain.TemplateNode) which contains the children to
// count.
//
// Returns int which is the count of child elements with a "pml-" tag prefix.
func (*engine) countPMLSiblings(parentNode *ast_domain.TemplateNode) int {
	siblingCount := 0
	for _, child := range parentNode.Children {
		if child.NodeType == ast_domain.NodeElement && strings.HasPrefix(child.TagName, "pml-") {
			siblingCount++
		}
	}
	return siblingCount
}

// NewTransformer creates and returns a new PML transformation engine.
//
// The collectors are injected as dependencies to avoid circular dependencies
// with adapters.
//
// Takes registry (ComponentRegistry) which provides access to PML components.
// Takes mediaQueryCollector (MediaQueryCollector) which gathers media queries.
// Takes msoConditionalCollector (MSOConditionalCollector) which gathers MSO
// conditional comments.
//
// Returns Transformer which is the configured transformation engine.
func NewTransformer(
	registry ComponentRegistry,
	mediaQueryCollector MediaQueryCollector,
	msoConditionalCollector MSOConditionalCollector,
) Transformer {
	return &engine{
		registry:                registry,
		mediaQueryCollector:     mediaQueryCollector,
		msoConditionalCollector: msoConditionalCollector,
	}
}
