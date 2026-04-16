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

package ast_domain

// Provides AST transformation utilities including tidying, validation, and
// parallel processing for template trees. Implements ParseAndTransform for
// parsing HTML with automatic validation and cleanup, supporting both
// sequential and parallel execution.

import (
	"context"
	"fmt"
	gohtml "html"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/logger/logger_domain"
)

// parallelWalkThresholdBytes is the file size in bytes above which parallel
// processing is used instead of sequential processing.
const parallelWalkThresholdBytes = 32 * 1024

var (
	// expressionParserPool reuses ExpressionParser instances to reduce allocation
	// pressure during template parsing.
	expressionParserPool = sync.Pool{
		New: func() any {
			return NewExpressionParser(context.Background(), "", "")
		},
	}

	// forceSequentialProcessing overrides parallel processing when set, forcing
	// all AST transformations to run sequentially.
	forceSequentialProcessing atomic.Bool

	// rawStringDirectives defines directives whose values are raw strings, not
	// expressions. These directives skip expression parsing and keep RawExpression
	// as-is.
	rawStringDirectives = map[DirectiveType]bool{
		DirectiveRef:      true,
		DirectiveSlot:     true,
		DirectiveTimeline: true,
	}
)

// keyAssigner assigns unique keys to template parts during AST processing.
type keyAssigner struct {
	// tree is the parsed syntax tree of the documentation template.
	tree *TemplateAST

	// sourcePath is the path to the file where the issue was found.
	sourcePath string

	// contextKeys maps each context key name to its next available index.
	contextKeys map[string]int

	// defaultParts holds the template parts used when no context overrides apply.
	defaultParts []TemplateLiteralPart
}

// assignKeysAndProcessDirectives performs a tree walk that assigns keys
// to all nodes in a single pass.
//
// Takes sourcePath (string) which is the file path for error reporting.
// Takes tree (*TemplateAST) which is the template tree to process.
func (ka *keyAssigner) assignKeysAndProcessDirectives(sourcePath string, tree *TemplateAST) {
	currentContextParts := ka.defaultParts

	for _, root := range ka.tree.RootNodes {
		if root.DirContext != nil && root.DirContext.Expression != nil {
			currentContextParts = ka.getKeyParts(root.DirContext.Expression)
		}

		key := ka.getContextKey(currentContextParts)
		index := ka.contextKeys[key]
		ka.contextKeys[key]++

		basePath := make([]TemplateLiteralPart, len(currentContextParts), len(currentContextParts)+1)
		copy(basePath, currentContextParts)
		basePath = append(basePath, TemplateLiteralPart{Expression: nil, Literal: fmt.Sprintf(".%d", index), IsLiteral: true, RelativeLocation: Location{}})
		userKeyExpr := ka.getUserKeyExpression(root)

		if userKeyExpr != nil && root.DirFor == nil {
			root.Key = userKeyExpr
		} else if userKeyExpr != nil {
			pathWithKey := make([]TemplateLiteralPart, len(basePath), len(basePath)+1)
			copy(pathWithKey, basePath)
			pathWithKey = append(pathWithKey, TemplateLiteralPart{Expression: nil, Literal: ".", IsLiteral: true, RelativeLocation: Location{}})
			pathWithKey = append(pathWithKey, ka.getKeyParts(userKeyExpr)...)
			root.Key = buildExpressionFromParts(pathWithKey, root.Location)
		} else {
			root.Key = buildExpressionFromParts(basePath, root.Location)
		}

		ka.assignChildKeysAndProcessDirectives(root, sourcePath, tree)
	}
}

// assignChildKeysAndProcessDirectives assigns keys to children in a
// recursive walk.
//
// Takes parent (*TemplateNode) which is the node whose children need
// processing.
// Takes sourcePath (string) which is the file path for error reporting.
// Takes tree (*TemplateAST) which is the template tree for diagnostics.
func (ka *keyAssigner) assignChildKeysAndProcessDirectives(parent *TemplateNode, sourcePath string, tree *TemplateAST) {
	parentPathParts := ka.getPathPartsFromKey(parent.Key)

	for i, child := range parent.Children {
		var childBasePath []TemplateLiteralPart
		userKeyExpr := ka.getUserKeyExpression(child)

		if child.DirContext != nil && child.DirContext.Expression != nil {
			childBasePath = ka.getKeyParts(child.DirContext.Expression)
		} else {
			childBasePath = make([]TemplateLiteralPart, len(parentPathParts), len(parentPathParts)+1)
			copy(childBasePath, parentPathParts)
			childBasePath = append(childBasePath, TemplateLiteralPart{Expression: nil, Literal: fmt.Sprintf(":%d", i), IsLiteral: true, RelativeLocation: Location{}})
		}

		if userKeyExpr != nil {
			pathWithKey := make([]TemplateLiteralPart, len(childBasePath), len(childBasePath)+1)
			copy(pathWithKey, childBasePath)
			pathWithKey = append(pathWithKey, TemplateLiteralPart{Expression: nil, Literal: ".", IsLiteral: true, RelativeLocation: Location{}})
			pathWithKey = append(pathWithKey, ka.getKeyParts(userKeyExpr)...)
			child.Key = buildExpressionFromParts(pathWithKey, child.Location)
		} else {
			child.Key = buildExpressionFromParts(childBasePath, child.Location)
		}

		ka.assignChildKeysAndProcessDirectives(child, sourcePath, tree)
	}
}

// getPathPartsFromKey converts a key Expression into a slice of path parts
// for use in recursion.
//
// Takes keyExpr (Expression) which is the key to convert into path parts.
//
// Returns []TemplateLiteralPart which holds the path parts for recursion.
func (*keyAssigner) getPathPartsFromKey(keyExpr Expression) []TemplateLiteralPart {
	if keyExpr == nil {
		return nil
	}
	switch v := keyExpr.(type) {
	case *StringLiteral:
		return []TemplateLiteralPart{{Expression: nil, Literal: v.Value, IsLiteral: true, RelativeLocation: Location{}}}
	case *TemplateLiteral:
		return v.Parts
	default:
		return []TemplateLiteralPart{{Expression: v, Literal: "", IsLiteral: false, RelativeLocation: Location{}}}
	}
}

// getUserKeyExpression extracts the key expression from a template node.
//
// Takes node (*TemplateNode) which is the node to get the key from.
//
// Returns Expression which is the key expression, or nil if none is found.
func (*keyAssigner) getUserKeyExpression(node *TemplateNode) Expression {
	if node.DirKey != nil && node.DirKey.Expression != nil {
		return node.DirKey.Expression
	}
	if node.DirFor != nil {
		if forInExpr, ok := node.DirFor.Expression.(*ForInExpression); ok {
			if forInExpr.IndexVariable != nil {
				return forInExpr.IndexVariable
			}
			if forInExpr.ItemVariable != nil {
				return forInExpr.ItemVariable
			}
		}
		return &Identifier{GoAnnotations: nil, Name: "index", RelativeLocation: Location{}, SourceLength: 0}
	}
	return nil
}

// getKeyParts extracts the template literal parts from the given expression.
//
// Takes expression (Expression) which is the expression to extract
// parts from.
//
// Returns []TemplateLiteralPart which contains the path parts from
// the key.
func (ka *keyAssigner) getKeyParts(expression Expression) []TemplateLiteralPart {
	return ka.getPathPartsFromKey(expression)
}

// getContextKey returns the context key for the given template parts.
//
// Takes parts ([]TemplateLiteralPart) which contains the parsed template
// segments.
//
// Returns string which is the literal value if parts contains a single
// literal, otherwise returns "dctx" as a default key.
func (*keyAssigner) getContextKey(parts []TemplateLiteralPart) string {
	if len(parts) == 1 && parts[0].IsLiteral {
		return parts[0].Literal
	}
	return "dctx"
}

// ResetExpressionParserPool clears the expression parser pool for test
// isolation. Call via t.Cleanup(ResetExpressionParserPool) in tests.
func ResetExpressionParserPool() {
	expressionParserPool = sync.Pool{
		New: func() any {
			return NewExpressionParser(context.Background(), "", "")
		},
	}
}

// TidyAST runs a second phase of changes on the AST that makes the tree
// simpler for later stages like code generation. It assigns keys to nodes
// and links if-else chains.
//
// When tree is nil, returns at once.
//
// When tree.Tidied is already true, logs a warning and returns without making
// changes.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes tree (*TemplateAST) which is the parsed template tree to transform.
func TidyAST(ctx context.Context, tree *TemplateAST) {
	_, l := logger_domain.From(ctx, log)
	if tree == nil {
		return
	}
	if tree.Tidied {
		l.Warn("TidyAST should only be called once on a given tree.")
		return
	}

	sourcePath := ""
	if tree.SourcePath != nil {
		sourcePath = *tree.SourcePath
	}

	assigner := newKeyAssigner(tree, sourcePath)

	assigner.assignKeysAndProcessDirectives(sourcePath, tree)

	linkIfElseChainsRecursive(tree.RootNodes, tree)

	tree.Tidied = true

	HoistDiagnostics(tree)
}

// setSequentialProcessing enables or disables forced sequential processing.
//
// When enabled, expression parsing always uses the sequential path regardless
// of file size. Use it to ensure deterministic test runs.
//
// Takes enabled (bool) which controls whether sequential processing is forced.
func setSequentialProcessing(enabled bool) {
	forceSequentialProcessing.Store(enabled)
}

// applyTemplateTransformations runs the first stage of changes on a parsed
// template. It turns raw expression strings into structured Expression objects
// and moves directives from the raw Directives slice into their typed fields
// (such as DirIf and OnEvents).
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes tree (*TemplateAST) which is the parsed template to transform.
func applyTemplateTransformations(ctx context.Context, tree *TemplateAST) {
	if tree == nil {
		return
	}

	parseAllExpressions(ctx, tree)

	tree.Walk(func(node *TemplateNode) bool {
		node.distributeDirectives()
		return true
	})
}

// parseAllExpressions walks the AST to parse all raw expression strings.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes tree (*TemplateAST) which is the AST to process.
//
// Concurrent processing uses multiple goroutines when the tree size exceeds
// the parallel walk threshold. A mutex protects diagnostic collection.
func parseAllExpressions(ctx context.Context, tree *TemplateAST) {
	sourcePath := ""
	if tree.SourcePath != nil {
		sourcePath = *tree.SourcePath
	}

	if forceSequentialProcessing.Load() || tree.SourceSize < parallelWalkThresholdBytes {
		var allDiags []*Diagnostic
		tree.Walk(func(node *TemplateNode) bool {
			diagnostics := parseExpressionsForNode(ctx, node, sourcePath)
			if len(diagnostics) > 0 {
				allDiags = append(allDiags, diagnostics...)
			}
			return true
		})
		if len(allDiags) > 0 {
			tree.Diagnostics = append(tree.Diagnostics, allDiags...)
		}
		return
	}

	var mu sync.Mutex
	parseFunction := func(walkCtx context.Context, node *TemplateNode) error {
		diagnostics := parseExpressionsForNode(walkCtx, node, sourcePath)
		if len(diagnostics) > 0 {
			mu.Lock()
			tree.Diagnostics = append(tree.Diagnostics, diagnostics...)
			mu.Unlock()
		}
		return nil
	}
	_ = tree.ParallelWalk(ctx, runtime.NumCPU(), parseFunction)
}

// parseExpressionsForNode parses all expressions within a single template node.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes node (*TemplateNode) which contains directives, dynamic attributes,
// and rich text to parse.
// Takes sourcePath (string) which identifies the source file for diagnostics.
//
// Returns []*Diagnostic which contains any parsing errors found in the node.
func parseExpressionsForNode(ctx context.Context, node *TemplateNode, sourcePath string) []*Diagnostic {
	var allNodeDiags []*Diagnostic

	for i := range node.Directives {
		if diagnostics := parseExpressionForDirective(ctx, &node.Directives[i], sourcePath); len(diagnostics) > 0 {
			allNodeDiags = append(allNodeDiags, diagnostics...)
		}
	}

	for i := range node.DynamicAttributes {
		if diagnostics := parseExpressionForDynamicAttribute(ctx, &node.DynamicAttributes[i], sourcePath); len(diagnostics) > 0 {
			allNodeDiags = append(allNodeDiags, diagnostics...)
		}
	}

	for i := range node.RichText {
		part := &node.RichText[i]
		if !part.IsLiteral && part.RawExpression != "" {
			if diagnostics := parseExpressionForTextPart(ctx, part, sourcePath); len(diagnostics) > 0 {
				allNodeDiags = append(allNodeDiags, diagnostics...)
			}
		}
	}

	return allNodeDiags
}

// parseAndSetExpression parses a raw expression string and sets it on a target
// using the given callback.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes rawExpr (string) which is the expression text to parse.
// Takes location (Location) which specifies where the expression appears.
// Takes sourcePath (string) which identifies the source file.
// Takes setExpr (func(...)) which receives the parsed expression on success.
//
// Returns []*Diagnostic which contains any parse warnings or errors.
func parseAndSetExpression(ctx context.Context, rawExpr string, location Location, sourcePath string, setExpr func(Expression)) []*Diagnostic {
	if rawExpr == "" {
		return nil
	}

	parser, ok := expressionParserPool.Get().(*ExpressionParser)
	if !ok {
		_, l := logger_domain.From(ctx, log)
		l.Error("expressionParserPool returned unexpected type, allocating new instance")
		parser = &ExpressionParser{}
	}
	defer expressionParserPool.Put(parser)

	expressionToParse := gohtml.UnescapeString(rawExpr)
	parser.Reset(ctx, expressionToParse, sourcePath)
	parsed, diagnostics := parser.ParseExpression(ctx)

	if len(diagnostics) > 0 {
		adjustDiagnosticLocations(diagnostics, location, rawExpr)
	}

	if HasErrors(diagnostics) || parsed == nil {
		return diagnostics
	}

	setExpr(parsed)
	return diagnostics
}

// parseExpressionForDirective parses and sets the expression for a directive.
//
// When the directive is an else type with no raw expression, returns nil.
// When the directive is a raw string directive (such as p-ref), checks the raw
// value instead of parsing it as an expression.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes d (*Directive) which is the directive to parse.
// Takes sourcePath (string) which is the path to the source file.
//
// Returns []*Diagnostic which contains any parsing errors found.
func parseExpressionForDirective(ctx context.Context, d *Directive, sourcePath string) []*Diagnostic {
	if d.RawExpression == "" && d.Type == DirectiveElse {
		return nil
	}
	if rawStringDirectives[d.Type] {
		return validateRawStringDirective(d, sourcePath)
	}
	return parseAndSetExpression(ctx, d.RawExpression, d.Location, sourcePath, func(expression Expression) {
		d.Expression = expression
	})
}

// validateRawStringDirective checks directives that use raw string values
// instead of expressions (e.g. p-ref).
//
// Takes d (*Directive) which is the directive to check.
// Takes sourcePath (string) which is the path to the source file for error
// reporting.
//
// Returns []*Diagnostic which contains any errors found during checking.
func validateRawStringDirective(d *Directive, sourcePath string) []*Diagnostic {
	switch d.Type {
	case DirectiveRef:
		return validateRefDirective(d, sourcePath)
	case DirectiveSlot:
		return validateSlotDirective(d, sourcePath)
	default:
		return nil
	}
}

// validateRefDirective validates and normalises a p-ref directive value.
//
// The value must be a valid JavaScript identifier. Whitespace is trimmed
// and the normalised value is stored back in RawExpression so consumers
// do not need to trim repeatedly.
//
// Takes d (*Directive) which is the directive to validate and normalise.
// Takes sourcePath (string) which is the path to the source file for
// diagnostics.
//
// Returns []*Diagnostic which contains any validation errors found.
func validateRefDirective(d *Directive, sourcePath string) []*Diagnostic {
	d.RawExpression = strings.TrimSpace(d.RawExpression)

	if d.RawExpression == "" {
		return []*Diagnostic{
			NewDiagnosticWithCode(Error, "p-ref value cannot be empty", d.RawExpression, CodeInvalidDirectiveValue, d.Location, sourcePath),
		}
	}

	if !IsValidJSIdentifier(d.RawExpression) {
		return []*Diagnostic{
			NewDiagnosticWithCode(
				Error,
				fmt.Sprintf("p-ref value %q must be a valid JavaScript identifier (letters, numbers, _, $; cannot start with number)", d.RawExpression),
				d.RawExpression, CodeInvalidDirectiveValue, d.Location, sourcePath,
			),
		}
	}

	return nil
}

// validateSlotDirective validates and normalises a p-slot directive value.
// Whitespace is trimmed and the normalised value is stored back in
// RawExpression.
//
// Takes d (*Directive) which is the directive to validate and normalise.
// Takes sourcePath (string) which is the path to the source file for
// diagnostics.
//
// Returns []*Diagnostic which contains any validation errors found.
func validateSlotDirective(d *Directive, sourcePath string) []*Diagnostic {
	d.RawExpression = strings.TrimSpace(d.RawExpression)

	if d.RawExpression == "" {
		return []*Diagnostic{
			NewDiagnosticWithCode(Error, "p-slot value cannot be empty", d.RawExpression, CodeInvalidDirectiveValue, d.Location, sourcePath),
		}
	}

	return nil
}

// parseExpressionForDynamicAttribute parses the raw expression of a dynamic
// attribute and stores the parsed result.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes da (*DynamicAttribute) which holds the raw expression to parse.
// Takes sourcePath (string) which is the path to the source file for error
// messages.
//
// Returns []*Diagnostic which contains any parsing errors found.
func parseExpressionForDynamicAttribute(ctx context.Context, da *DynamicAttribute, sourcePath string) []*Diagnostic {
	return parseAndSetExpression(ctx, da.RawExpression, da.Location, sourcePath, func(expression Expression) {
		da.Expression = expression
	})
}

// parseExpressionForTextPart parses the raw expression in a text part and
// stores the result.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes part (*TextPart) which contains the raw expression to parse.
// Takes sourcePath (string) which gives the source file path for error
// messages.
//
// Returns []*Diagnostic which contains any parsing errors found.
func parseExpressionForTextPart(ctx context.Context, part *TextPart, sourcePath string) []*Diagnostic {
	return parseAndSetExpression(ctx, part.RawExpression, part.Location, sourcePath, func(expression Expression) {
		part.Expression = expression
	})
}

// adjustDiagnosticLocations updates diagnostic positions from value-relative
// to document-relative coordinates.
//
// Takes diagnostics ([]*Diagnostic) which contains the diagnostics to update.
// Takes baseLocation (Location) which specifies the starting position in the
// document.
// Takes rawValue (string) which provides the raw text for counting lines.
func adjustDiagnosticLocations(diagnostics []*Diagnostic, baseLocation Location, rawValue string) {
	for _, diagnostic := range diagnostics {
		errLineInValue := diagnostic.Location.Line
		errColInValue := diagnostic.Location.Column

		if errLineInValue == 1 {
			diagnostic.Location.Line = baseLocation.Line
			diagnostic.Location.Column = baseLocation.Column + errColInValue - 1
		} else {
			lines := strings.Split(rawValue, "\n")
			if errLineInValue > 1 && errLineInValue <= len(lines) {
				diagnostic.Location.Line = baseLocation.Line + errLineInValue - 1
				diagnostic.Location.Column = errColInValue
			}
		}
	}
}

// newKeyAssigner creates a key assigner for the given template tree.
//
// Takes tree (*TemplateAST) which is the parsed template to assign keys to.
// Takes sourcePath (string) which is the path to the source file.
//
// Returns *keyAssigner which is ready to use with default settings.
func newKeyAssigner(tree *TemplateAST, sourcePath string) *keyAssigner {
	return &keyAssigner{
		tree:         tree,
		sourcePath:   sourcePath,
		contextKeys:  make(map[string]int),
		defaultParts: []TemplateLiteralPart{{Expression: nil, Literal: "r", IsLiteral: true, RelativeLocation: Location{}}},
	}
}

// buildExpressionFromParts builds an Expression from template literal parts.
//
// Takes parts ([]TemplateLiteralPart) which contains the template segments to
// combine.
// Takes baseLocation (Location) which specifies the source position for the
// result.
//
// Returns Expression which is a StringLiteral when all parts are plain text,
// the single expression when only one dynamic part exists, or a TemplateLiteral
// when there is a mix of both types.
func buildExpressionFromParts(parts []TemplateLiteralPart, baseLocation Location) Expression {
	var cleanedParts []TemplateLiteralPart
	var hasDynamicPart bool
	var builder strings.Builder

	for _, part := range parts {
		if part.IsLiteral {
			builder.WriteString(part.Literal)
		} else {
			hasDynamicPart = true
			if builder.Len() > 0 {
				cleanedParts = append(cleanedParts, TemplateLiteralPart{
					Expression:       nil,
					Literal:          builder.String(),
					IsLiteral:        true,
					RelativeLocation: baseLocation,
				})
				builder.Reset()
			}
			cleanedParts = append(cleanedParts, part)
		}
	}
	if builder.Len() > 0 {
		cleanedParts = append(cleanedParts, TemplateLiteralPart{
			Expression:       nil,
			Literal:          builder.String(),
			IsLiteral:        true,
			RelativeLocation: baseLocation,
		})
	}

	if !hasDynamicPart {
		if len(cleanedParts) > 0 {
			return &StringLiteral{GoAnnotations: nil, Value: cleanedParts[0].Literal, RelativeLocation: baseLocation, SourceLength: 0}
		}
		return &StringLiteral{GoAnnotations: nil, Value: "", RelativeLocation: baseLocation, SourceLength: 0}
	}

	if len(cleanedParts) == 1 && !cleanedParts[0].IsLiteral {
		return cleanedParts[0].Expression
	}
	return &TemplateLiteral{GoAnnotations: nil, Parts: cleanedParts, RelativeLocation: baseLocation, SourceLength: 0}
}

// linkIfElseChains walks a list of sibling nodes and links `p-else-if` and
// `p-else` directives to their preceding `p-if` node using the `ChainKey`.
//
// Takes siblings ([]*TemplateNode) which is the list of sibling
// nodes to link.
func linkIfElseChains(siblings []*TemplateNode, _ *TemplateAST) {
	var currentIfNode *TemplateNode

	for _, node := range siblings {
		if breaksIfElseChain(node) {
			currentIfNode = nil
			continue
		}

		currentIfNode = processIfElseNode(node, currentIfNode)
	}
}

// linkIfElseChainsRecursive links if-else-elseif directive chains throughout
// the tree by walking siblings and their children.
//
// Takes siblings ([]*TemplateNode) which is the list of sibling nodes to
// process.
// Takes tree (*TemplateAST) which is the template tree for context.
func linkIfElseChainsRecursive(siblings []*TemplateNode, tree *TemplateAST) {
	linkIfElseChains(siblings, tree)

	for _, node := range siblings {
		if len(node.Children) > 0 {
			linkIfElseChainsRecursive(node.Children, tree)
		}
	}
}

// breaksIfElseChain checks whether a node breaks an if-else chain.
//
// A node breaks the chain if it is an element without any conditional
// directives, or if it is neither a comment nor whitespace-only text.
//
// Takes node (*TemplateNode) which is the node to check.
//
// Returns bool which is true if the node breaks the chain.
func breaksIfElseChain(node *TemplateNode) bool {
	if node.NodeType == NodeElement {
		return node.DirIf == nil && node.DirElseIf == nil && node.DirElse == nil
	}
	return node.NodeType != NodeComment && !isWhitespaceOnlyText(node)
}

// processIfElseNode handles a single node in an if-else chain and returns the
// updated chain head.
//
// Takes node (*TemplateNode) which is the node to check and link.
// Takes currentIfNode (*TemplateNode) which is the current head of the chain.
//
// Returns *TemplateNode which is the new chain head, or nil when an else node
// ends the chain.
func processIfElseNode(node *TemplateNode, currentIfNode *TemplateNode) *TemplateNode {
	if node.DirIf != nil {
		return node
	}

	if node.DirElseIf != nil {
		return linkElseIfToChain(node, currentIfNode)
	}

	if node.DirElse != nil {
		linkElseToChain(node, currentIfNode)
		return nil
	}

	return currentIfNode
}

// linkElseIfToChain connects an else-if node to an existing if-else chain.
//
// Takes node (*TemplateNode) which is the else-if node to add to the chain.
// Takes currentIfNode (*TemplateNode) which is the if node that starts the
// chain.
//
// Returns *TemplateNode which is the linked node, or nil if currentIfNode is
// nil.
func linkElseIfToChain(node *TemplateNode, currentIfNode *TemplateNode) *TemplateNode {
	if currentIfNode == nil {
		return nil
	}

	chainKey := getChainKey(currentIfNode)
	if chainKey != nil {
		node.DirElseIf.ChainKey = chainKey
	}

	return node
}

// linkElseToChain connects a p-else node to an existing if-else chain.
//
// Takes node (*TemplateNode) which is the p-else node to link.
// Takes currentIfNode (*TemplateNode) which is the first node in the chain.
func linkElseToChain(node *TemplateNode, currentIfNode *TemplateNode) {
	if currentIfNode == nil {
		return
	}

	chainKey := getChainKey(currentIfNode)
	if chainKey != nil {
		node.DirElse.ChainKey = chainKey
	}
}

// getChainKey returns the chain key from an if or else-if node.
//
// Takes node (*TemplateNode) which is the node to get the chain key from.
//
// Returns Expression which is the chain key, or nil if the node has no DirIf
// or DirElseIf directive.
func getChainKey(node *TemplateNode) Expression {
	if node.DirIf != nil {
		return node.Key
	}
	if node.DirElseIf != nil {
		return node.DirElseIf.ChainKey
	}
	return nil
}
