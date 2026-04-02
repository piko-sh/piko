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

package ast_adapters

import (
	"context"
	"fmt"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/ast/ast_schema/ast_schema_gen"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safeconv"
)

var (
	// goToFBDirectiveType maps Go DirectiveType constants to their
	// FlatBuffer equivalents, indexed by Go DirectiveType.
	//
	// This mapping is necessary because the enum orderings differ
	// between the Go domain types and the FlatBuffer schema.
	//
	// paired with fbToGoDirectiveType.
	//
	//nolint:dupl // forward lookup
	goToFBDirectiveType = [...]ast_schema_gen.DirectiveType{
		ast_domain.DirectiveIf:       ast_schema_gen.DirectiveTypeIF,
		ast_domain.DirectiveElseIf:   ast_schema_gen.DirectiveTypeELSEIF,
		ast_domain.DirectiveElse:     ast_schema_gen.DirectiveTypeELSE,
		ast_domain.DirectiveFor:      ast_schema_gen.DirectiveTypeFOR,
		ast_domain.DirectiveShow:     ast_schema_gen.DirectiveTypeSHOW,
		ast_domain.DirectiveBind:     ast_schema_gen.DirectiveTypeBIND,
		ast_domain.DirectiveModel:    ast_schema_gen.DirectiveTypeMODEL,
		ast_domain.DirectiveOn:       ast_schema_gen.DirectiveTypeON,
		ast_domain.DirectiveEvent:    ast_schema_gen.DirectiveTypeEVENT,
		ast_domain.DirectiveClass:    ast_schema_gen.DirectiveTypeCLASS,
		ast_domain.DirectiveStyle:    ast_schema_gen.DirectiveTypeSTYLE,
		ast_domain.DirectiveText:     ast_schema_gen.DirectiveTypeTEXT,
		ast_domain.DirectiveHTML:     ast_schema_gen.DirectiveTypeHTML,
		ast_domain.DirectiveRef:      ast_schema_gen.DirectiveTypeREF,
		ast_domain.DirectiveSlot:     ast_schema_gen.DirectiveTypeSLOT,
		ast_domain.DirectiveKey:      ast_schema_gen.DirectiveTypeKEY,
		ast_domain.DirectiveContext:  ast_schema_gen.DirectiveTypeCONTEXT,
		ast_domain.DirectiveScaffold: ast_schema_gen.DirectiveTypeSCAFFOLD,
	}

	// fbToGoDirectiveType maps FlatBuffer DirectiveType constants back
	// to Go equivalents, indexed by FlatBuffer DirectiveType.
	//
	// paired with goToFBDirectiveType.
	//
	//nolint:dupl // reverse lookup
	fbToGoDirectiveType = [...]ast_domain.DirectiveType{
		ast_schema_gen.DirectiveTypeIF:       ast_domain.DirectiveIf,
		ast_schema_gen.DirectiveTypeELSEIF:   ast_domain.DirectiveElseIf,
		ast_schema_gen.DirectiveTypeELSE:     ast_domain.DirectiveElse,
		ast_schema_gen.DirectiveTypeFOR:      ast_domain.DirectiveFor,
		ast_schema_gen.DirectiveTypeSHOW:     ast_domain.DirectiveShow,
		ast_schema_gen.DirectiveTypeBIND:     ast_domain.DirectiveBind,
		ast_schema_gen.DirectiveTypeON:       ast_domain.DirectiveOn,
		ast_schema_gen.DirectiveTypeEVENT:    ast_domain.DirectiveEvent,
		ast_schema_gen.DirectiveTypeMODEL:    ast_domain.DirectiveModel,
		ast_schema_gen.DirectiveTypeREF:      ast_domain.DirectiveRef,
		ast_schema_gen.DirectiveTypeCLASS:    ast_domain.DirectiveClass,
		ast_schema_gen.DirectiveTypeSTYLE:    ast_domain.DirectiveStyle,
		ast_schema_gen.DirectiveTypeTEXT:     ast_domain.DirectiveText,
		ast_schema_gen.DirectiveTypeHTML:     ast_domain.DirectiveHTML,
		ast_schema_gen.DirectiveTypeKEY:      ast_domain.DirectiveKey,
		ast_schema_gen.DirectiveTypeCONTEXT:  ast_domain.DirectiveContext,
		ast_schema_gen.DirectiveTypeSCAFFOLD: ast_domain.DirectiveScaffold,
		ast_schema_gen.DirectiveTypeSLOT:     ast_domain.DirectiveSlot,
	}
)

// buildTemplateAST converts a TemplateAST domain object to FlatBuffers format.
//
// Takes ast (*ast_domain.TemplateAST) which is the template AST to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised data.
// Returns error when serialising root nodes or diagnostics fails.
func (s *encoder) buildTemplateAST(ast *ast_domain.TemplateAST) (flatbuffers.UOffsetT, error) {
	if ast == nil {
		return 0, nil
	}

	rootNodeOffsets, err := buildVectorOfPtrs(s, ast.RootNodes, (*encoder).buildTemplateNode)
	if err != nil {
		return 0, fmt.Errorf("building template AST root nodes: %w", err)
	}

	diagOffsets, err := buildVectorOfPtrs(s, ast.Diagnostics, (*encoder).buildDiagnostic)
	if err != nil {
		return 0, fmt.Errorf("building template AST diagnostics: %w", err)
	}

	var sourcePathOff flatbuffers.UOffsetT
	if ast.SourcePath != nil {
		sourcePathOff = s.builder.CreateString(*ast.SourcePath)
	}

	var expiresAt int64
	if ast.ExpiresAtUnixNano != nil {
		expiresAt = *ast.ExpiresAtUnixNano
	}

	var metadata flatbuffers.UOffsetT
	if ast.Metadata != nil {
		metadata = s.builder.CreateString(*ast.Metadata)
	}

	ast_schema_gen.TemplateASTFBStart(s.builder)
	ast_schema_gen.TemplateASTFBAddRootNodes(s.builder, rootNodeOffsets)
	ast_schema_gen.TemplateASTFBAddDiagnostics(s.builder, diagOffsets)
	if ast.SourcePath != nil {
		ast_schema_gen.TemplateASTFBAddSourcePath(s.builder, sourcePathOff)
	}
	ast_schema_gen.TemplateASTFBAddTidied(s.builder, ast.Tidied)
	ast_schema_gen.TemplateASTFBAddExpiresAtUnixNano(s.builder, expiresAt)
	ast_schema_gen.TemplateASTFBAddMetadata(s.builder, metadata)
	ast_schema_gen.TemplateASTFBAddSourceSize(s.builder, ast.SourceSize)
	return ast_schema_gen.TemplateASTFBEnd(s.builder), nil
}

// templateNodeDirectiveOffsets holds FlatBuffer offsets for all directive
// fields of a template node.
type templateNodeDirectiveOffsets struct {
	// dirIf is the FlatBuffer offset for the if directive.
	dirIf flatbuffers.UOffsetT

	// dirElseIf is the FlatBuffer offset for the else-if directive.
	dirElseIf flatbuffers.UOffsetT

	// dirElse is the FlatBuffer offset for the else directive.
	dirElse flatbuffers.UOffsetT

	// dirFor is the FlatBuffer offset for the for-loop directive.
	dirFor flatbuffers.UOffsetT

	// dirShow is the FlatBuffer offset for the show directive.
	dirShow flatbuffers.UOffsetT

	// dirModel is the FlatBuffer offset for the model directive.
	dirModel flatbuffers.UOffsetT

	// dirRef is the FlatBuffer offset for the ref directive.
	dirRef flatbuffers.UOffsetT

	// dirSlot holds the FlatBuffer offset for the slot directive.
	dirSlot flatbuffers.UOffsetT

	// dirClass is the FlatBuffer offset for the class directive.
	dirClass flatbuffers.UOffsetT

	// dirStyle is the FlatBuffer offset for the style directive.
	dirStyle flatbuffers.UOffsetT

	// dirText is the FlatBuffer offset for the text directive.
	dirText flatbuffers.UOffsetT

	// dirHTML stores the FlatBuffer offset for the HTML directive.
	dirHTML flatbuffers.UOffsetT

	// dirKey is the FlatBuffer offset for the key directive.
	dirKey flatbuffers.UOffsetT

	// dirContext is the FlatBuffer offset for the context directive.
	dirContext flatbuffers.UOffsetT

	// dirScaffold is the offset for the scaffold directive in the FlatBuffer.
	dirScaffold flatbuffers.UOffsetT
}

// buildTemplateNodeDirectives builds all directive offsets for a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
//
// Returns templateNodeDirectiveOffsets which holds the built directive offsets.
// Returns error when any directive fails to build.
func (s *encoder) buildTemplateNodeDirectives(node *ast_domain.TemplateNode) (templateNodeDirectiveOffsets, error) {
	var off templateNodeDirectiveOffsets
	var err error

	if off.dirIf, err = s.buildDirective(node.DirIf); err != nil {
		return off, err
	}
	if off.dirElseIf, err = s.buildDirective(node.DirElseIf); err != nil {
		return off, err
	}
	if off.dirElse, err = s.buildDirective(node.DirElse); err != nil {
		return off, err
	}
	if off.dirFor, err = s.buildDirective(node.DirFor); err != nil {
		return off, err
	}
	if off.dirShow, err = s.buildDirective(node.DirShow); err != nil {
		return off, err
	}
	if off.dirModel, err = s.buildDirective(node.DirModel); err != nil {
		return off, err
	}
	if off.dirRef, err = s.buildDirective(node.DirRef); err != nil {
		return off, err
	}
	if off.dirSlot, err = s.buildDirective(node.DirSlot); err != nil {
		return off, err
	}
	if off.dirClass, err = s.buildDirective(node.DirClass); err != nil {
		return off, err
	}
	if off.dirStyle, err = s.buildDirective(node.DirStyle); err != nil {
		return off, err
	}
	if off.dirText, err = s.buildDirective(node.DirText); err != nil {
		return off, err
	}
	if off.dirHTML, err = s.buildDirective(node.DirHTML); err != nil {
		return off, err
	}
	if off.dirKey, err = s.buildDirective(node.DirKey); err != nil {
		return off, err
	}
	if off.dirContext, err = s.buildDirective(node.DirContext); err != nil {
		return off, err
	}
	if off.dirScaffold, err = s.buildDirective(node.DirScaffold); err != nil {
		return off, err
	}
	return off, nil
}

// templateNodeVectorOffsets holds FlatBuffer offsets for vector fields.
type templateNodeVectorOffsets struct {
	// richText is the vector offset for serialised rich text parts.
	richText flatbuffers.UOffsetT

	// attrs is the FlatBuffers offset for the template node's HTML attributes
	// vector.
	attrs flatbuffers.UOffsetT

	// dynAttrs is the offset for the dynamic attributes vector.
	dynAttrs flatbuffers.UOffsetT

	// diagnostics is the offset to the vector of diagnostic messages for this node.
	diagnostics flatbuffers.UOffsetT

	// children is the offset to the vector of child template nodes.
	children flatbuffers.UOffsetT

	// directives is the FlatBuffer offset for the node's directive vector.
	directives flatbuffers.UOffsetT

	// onEvents is the FlatBuffer offset for the serialised on-event handlers
	// vector.
	onEvents flatbuffers.UOffsetT

	// customEvents is the offset for custom event bindings in the FlatBuffer.
	customEvents flatbuffers.UOffsetT

	// binds is the offset to the serialised binds map vector.
	binds flatbuffers.UOffsetT

	// attributeWriters is the offset to the vector of attribute writer directives.
	attributeWriters flatbuffers.UOffsetT
}

// buildTemplateNodeVectors builds all vector offsets for a template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
//
// Returns templateNodeVectorOffsets which contains the built vector offsets.
// Returns error when any vector building operation fails.
func (s *encoder) buildTemplateNodeVectors(node *ast_domain.TemplateNode) (templateNodeVectorOffsets, error) {
	var off templateNodeVectorOffsets
	var err error

	if off.richText, err = buildVectorOfValues(s, node.RichText, (*encoder).buildTextPart); err != nil {
		return off, err
	}
	if off.attrs, err = buildVectorOfValues(s, node.Attributes, (*encoder).buildHTMLAttribute); err != nil {
		return off, err
	}
	if off.dynAttrs, err = buildVectorOfValues(s, node.DynamicAttributes, (*encoder).buildDynamicAttribute); err != nil {
		return off, err
	}
	if off.diagnostics, err = buildVectorOfPtrs(s, node.Diagnostics, (*encoder).buildDiagnostic); err != nil {
		return off, err
	}
	if off.children, err = buildVectorOfPtrs(s, node.Children, (*encoder).buildTemplateNode); err != nil {
		return off, err
	}
	if off.directives, err = buildVectorOfValues(s, node.Directives, (*encoder).buildDirective); err != nil {
		return off, err
	}
	if off.onEvents, err = s.buildOnEventsMap(node.OnEvents); err != nil {
		return off, err
	}
	if off.customEvents, err = s.buildCustomEventsMap(node.CustomEvents); err != nil {
		return off, err
	}
	if off.binds, err = s.buildBindsMap(node.Binds); err != nil {
		return off, err
	}
	if off.attributeWriters, err = buildVectorOfPtrs(s, node.AttributeWriters, (*encoder).buildDirectWriter); err != nil {
		return off, err
	}

	return off, nil
}

// templateNodeAnnotationOffsets holds FlatBuffer offsets for annotation fields.
type templateNodeAnnotationOffsets struct {
	// key is the FlatBuffer offset for the template node's key expression.
	key flatbuffers.UOffsetT

	// goAnn is the offset for Go generator annotations.
	goAnn flatbuffers.UOffsetT

	// runAnn is the offset to the runtime annotations vector.
	runAnn flatbuffers.UOffsetT

	// textContentWriter is the offset to the writer for text content.
	textContentWriter flatbuffers.UOffsetT
}

// buildTemplateNodeAnnotations builds annotation and key offsets for a
// template node.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to process.
//
// Returns templateNodeAnnotationOffsets which contains the built offsets.
// Returns error when building any child node fails.
func (s *encoder) buildTemplateNodeAnnotations(node *ast_domain.TemplateNode) (templateNodeAnnotationOffsets, error) {
	var off templateNodeAnnotationOffsets
	var err error

	if off.key, err = s.buildExpressionNode(node.Key); err != nil {
		return off, err
	}
	if off.goAnn, err = s.buildGoGeneratorAnnotation(node.GoAnnotations); err != nil {
		return off, err
	}
	if off.runAnn, err = s.buildRuntimeAnnotation(node.RuntimeAnnotations); err != nil {
		return off, err
	}
	if off.textContentWriter, err = s.buildDirectWriter(node.TextContentWriter); err != nil {
		return off, err
	}

	return off, nil
}

// addDirectivesToFlatBuffer adds all directive offsets to the FlatBuffer
// builder.
//
// Takes directive (templateNodeDirectiveOffsets) which contains the pre-built
// offsets for each directive type.
func (s *encoder) addDirectivesToFlatBuffer(directive templateNodeDirectiveOffsets) {
	ast_schema_gen.TemplateNodeFBAddDirectiveIf(s.builder, directive.dirIf)
	ast_schema_gen.TemplateNodeFBAddDirectiveElseIf(s.builder, directive.dirElseIf)
	ast_schema_gen.TemplateNodeFBAddDirectiveElse(s.builder, directive.dirElse)
	ast_schema_gen.TemplateNodeFBAddDirectiveFor(s.builder, directive.dirFor)
	ast_schema_gen.TemplateNodeFBAddDirectiveShow(s.builder, directive.dirShow)
	ast_schema_gen.TemplateNodeFBAddDirectiveModel(s.builder, directive.dirModel)
	ast_schema_gen.TemplateNodeFBAddDirectiveRef(s.builder, directive.dirRef)
	ast_schema_gen.TemplateNodeFBAddDirectiveSlot(s.builder, directive.dirSlot)
	ast_schema_gen.TemplateNodeFBAddDirectiveClass(s.builder, directive.dirClass)
	ast_schema_gen.TemplateNodeFBAddDirectiveStyle(s.builder, directive.dirStyle)
	ast_schema_gen.TemplateNodeFBAddDirectiveText(s.builder, directive.dirText)
	ast_schema_gen.TemplateNodeFBAddDirectiveHtml(s.builder, directive.dirHTML)
	ast_schema_gen.TemplateNodeFBAddDirectiveKey(s.builder, directive.dirKey)
	ast_schema_gen.TemplateNodeFBAddDirectiveContext(s.builder, directive.dirContext)
	ast_schema_gen.TemplateNodeFBAddDirectiveScaffold(s.builder, directive.dirScaffold)
}

// buildTemplateNode converts a template node to its FlatBuffers representation.
//
// Takes node (*ast_domain.TemplateNode) which is the template node to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the built node.
// Returns error when building nested components fails.
func (s *encoder) buildTemplateNode(node *ast_domain.TemplateNode) (flatbuffers.UOffsetT, error) {
	if node == nil {
		return 0, nil
	}

	tagNameOff := s.builder.CreateString(node.TagName)
	textContentOff := s.builder.CreateString(node.TextContent)
	innerHTMLOff := s.builder.CreateString(node.InnerHTML)

	var prerenderedHTMLOff flatbuffers.UOffsetT
	if len(node.PrerenderedHTML) > 0 {
		ast_schema_gen.TemplateNodeFBStartPrerenderedHtmlVector(s.builder, len(node.PrerenderedHTML))
		for i := len(node.PrerenderedHTML) - 1; i >= 0; i-- {
			s.builder.PrependByte(node.PrerenderedHTML[i])
		}
		prerenderedHTMLOff = s.builder.EndVector(len(node.PrerenderedHTML))
	}

	rangeOffs, err := s.buildTemplateNodeRanges(node)
	if err != nil {
		return 0, fmt.Errorf("building template node ranges: %w", err)
	}

	vecOff, err := s.buildTemplateNodeVectors(node)
	if err != nil {
		return 0, fmt.Errorf("building template node vectors: %w", err)
	}
	dirOff, err := s.buildTemplateNodeDirectives(node)
	if err != nil {
		return 0, fmt.Errorf("building template node directives: %w", err)
	}
	annOff, err := s.buildTemplateNodeAnnotations(node)
	if err != nil {
		return 0, fmt.Errorf("building template node annotations: %w", err)
	}

	strOff := templateNodeStringOffsets{tagName: tagNameOff, textContent: textContentOff, innerHTML: innerHTMLOff, prerenderedHTML: prerenderedHTMLOff}
	return s.assembleTemplateNodeFB(node, strOff, rangeOffs, vecOff, dirOff, annOff), nil
}

// templateNodeStringOffsets holds FlatBuffers offsets for string fields.
type templateNodeStringOffsets struct {
	// tagName is the FlatBuffers offset for the node's tag name.
	tagName flatbuffers.UOffsetT

	// textContent is the FlatBuffers offset for the node's text content.
	textContent flatbuffers.UOffsetT

	// innerHTML is the FlatBuffers offset for the node's inner HTML content.
	innerHTML flatbuffers.UOffsetT

	// prerenderedHTML is the FlatBuffers offset for precomputed HTML bytes.
	prerenderedHTML flatbuffers.UOffsetT
}

// templateNodeRangeOffsets holds the location and range offsets for a
// TemplateNode.
type templateNodeRangeOffsets struct {
	// location is the FlatBuffers offset for the node's source location.
	location flatbuffers.UOffsetT

	// nodeRange is the FlatBuffer offset for the full source range of the node.
	nodeRange flatbuffers.UOffsetT

	// openingTagRange is the offset for the opening tag's position in the source.
	openingTagRange flatbuffers.UOffsetT

	// closingTagRange is the FlatBuffers offset for the closing tag's position.
	closingTagRange flatbuffers.UOffsetT
}

// buildTemplateNodeRanges builds the location and range offsets for a
// TemplateNode.
//
// Takes node (*ast_domain.TemplateNode) which provides the template node to
// extract ranges from.
//
// Returns templateNodeRangeOffsets which contains the built location and range
// offsets.
// Returns error when any location or range cannot be built.
func (s *encoder) buildTemplateNodeRanges(node *ast_domain.TemplateNode) (templateNodeRangeOffsets, error) {
	var rangeOffsets templateNodeRangeOffsets
	var err error
	rangeOffsets.location, err = s.buildLocation(&node.Location)
	if err != nil {
		return rangeOffsets, err
	}
	rangeOffsets.nodeRange, err = s.buildRange(&node.NodeRange)
	if err != nil {
		return rangeOffsets, err
	}
	rangeOffsets.openingTagRange, err = s.buildRange(&node.OpeningTagRange)
	if err != nil {
		return rangeOffsets, err
	}
	rangeOffsets.closingTagRange, err = s.buildRange(&node.ClosingTagRange)
	return rangeOffsets, err
}

// assembleTemplateNodeFB builds the final TemplateNodeFB table from pre-built
// offsets.
//
// Takes node (*ast_domain.TemplateNode) which provides the source node data.
// Takes strOff (templateNodeStringOffsets) which contains string field offsets.
// Takes rangeOffs (templateNodeRangeOffsets) which contains location offsets.
// Takes vecOff (templateNodeVectorOffsets) which contains vector offsets.
// Takes dirOff (templateNodeDirectiveOffsets) which contains directive offsets.
// Takes annOff (templateNodeAnnotationOffsets) which contains annotation
// offsets.
//
// Returns flatbuffers.UOffsetT which is the offset of the completed table.
func (s *encoder) assembleTemplateNodeFB(
	node *ast_domain.TemplateNode,
	strOff templateNodeStringOffsets,
	rangeOffs templateNodeRangeOffsets,
	vecOff templateNodeVectorOffsets,
	dirOff templateNodeDirectiveOffsets,
	annOff templateNodeAnnotationOffsets,
) flatbuffers.UOffsetT {
	ast_schema_gen.TemplateNodeFBStart(s.builder)
	ast_schema_gen.TemplateNodeFBAddNodeType(s.builder, ast_schema_gen.NodeType(safeconv.IntToUint8(int(node.NodeType))))
	ast_schema_gen.TemplateNodeFBAddTagName(s.builder, strOff.tagName)
	ast_schema_gen.TemplateNodeFBAddLocation(s.builder, rangeOffs.location)
	ast_schema_gen.TemplateNodeFBAddTextContent(s.builder, strOff.textContent)
	ast_schema_gen.TemplateNodeFBAddRichText(s.builder, vecOff.richText)
	ast_schema_gen.TemplateNodeFBAddInnerHtml(s.builder, strOff.innerHTML)
	ast_schema_gen.TemplateNodeFBAddIsContentEditable(s.builder, node.IsContentEditable)
	ast_schema_gen.TemplateNodeFBAddPreserveWhitespace(s.builder, node.PreserveWhitespace)
	ast_schema_gen.TemplateNodeFBAddNodeRange(s.builder, rangeOffs.nodeRange)
	ast_schema_gen.TemplateNodeFBAddOpeningTagRange(s.builder, rangeOffs.openingTagRange)
	ast_schema_gen.TemplateNodeFBAddClosingTagRange(s.builder, rangeOffs.closingTagRange)
	ast_schema_gen.TemplateNodeFBAddAttributes(s.builder, vecOff.attrs)
	ast_schema_gen.TemplateNodeFBAddDynamicAttributes(s.builder, vecOff.dynAttrs)
	ast_schema_gen.TemplateNodeFBAddDiagnostics(s.builder, vecOff.diagnostics)
	ast_schema_gen.TemplateNodeFBAddChildren(s.builder, vecOff.children)
	ast_schema_gen.TemplateNodeFBAddDirectives(s.builder, vecOff.directives)
	s.addDirectivesToFlatBuffer(dirOff)
	ast_schema_gen.TemplateNodeFBAddOnEvents(s.builder, vecOff.onEvents)
	ast_schema_gen.TemplateNodeFBAddCustomEvents(s.builder, vecOff.customEvents)
	ast_schema_gen.TemplateNodeFBAddBinds(s.builder, vecOff.binds)
	ast_schema_gen.TemplateNodeFBAddKey(s.builder, annOff.key)
	ast_schema_gen.TemplateNodeFBAddGoAnnotations(s.builder, annOff.goAnn)
	ast_schema_gen.TemplateNodeFBAddRuntimeAnnotations(s.builder, annOff.runAnn)
	ast_schema_gen.TemplateNodeFBAddAttributeWriters(s.builder, vecOff.attributeWriters)
	ast_schema_gen.TemplateNodeFBAddTextContentWriter(s.builder, annOff.textContentWriter)
	ast_schema_gen.TemplateNodeFBAddPreferredFormat(s.builder, ast_schema_gen.FormatHintFB(node.PreferredFormat))
	ast_schema_gen.TemplateNodeFBAddPrerenderedHtml(s.builder, strOff.prerenderedHTML)
	return ast_schema_gen.TemplateNodeFBEnd(s.builder)
}

// buildDirective converts a directive AST node into a FlatBuffers offset.
//
// Takes directive (*ast_domain.Directive) which is the directive to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised directive.
// Returns error when any nested expression or location fails to serialise.
func (s *encoder) buildDirective(directive *ast_domain.Directive) (flatbuffers.UOffsetT, error) {
	if directive == nil {
		return 0, nil
	}

	offsets, err := s.buildDirectiveFieldOffsets(directive)
	if err != nil {
		return 0, err
	}

	return s.assembleDirectiveFB(directive, offsets), nil
}

// directiveOffsets groups all pre-built FlatBuffer offsets needed to assemble
// a directive.
type directiveOffsets struct {
	// argument is the offset for the directive's argument string.
	argument flatbuffers.UOffsetT

	// modifier is the offset for the directive's modifier string.
	modifier flatbuffers.UOffsetT

	// rawExpr is the offset for the directive's raw expression string.
	rawExpr flatbuffers.UOffsetT

	// expression is the offset for the directive's serialised expression.
	expression flatbuffers.UOffsetT

	// chainKey is the offset for the directive's chain key string.
	chainKey flatbuffers.UOffsetT

	// location is the offset for the directive's source location.
	location flatbuffers.UOffsetT

	// nameLocation is the offset for the directive name's source location.
	nameLocation flatbuffers.UOffsetT

	// attributeRange is the offset for the directive's attribute range.
	attributeRange flatbuffers.UOffsetT

	// goAnn is the offset for the directive's Go annotation.
	goAnn flatbuffers.UOffsetT

	// eventModifiers is the offset for the directive's event modifier vector.
	eventModifiers flatbuffers.UOffsetT
}

// buildDirectiveFieldOffsets pre-builds all string, expression, location, and
// annotation offsets for a directive.
//
// Takes directive (*ast_domain.Directive) which is the directive to process.
//
// Returns directiveOffsets which contains all pre-built offsets.
// Returns error when any nested serialisation fails.
func (s *encoder) buildDirectiveFieldOffsets(directive *ast_domain.Directive) (directiveOffsets, error) {
	var o directiveOffsets
	var err error

	o.argument = s.builder.CreateString(directive.Arg)
	o.modifier = s.builder.CreateString(directive.Modifier)
	o.rawExpr = s.builder.CreateString(directive.RawExpression)

	if o.expression, err = s.buildExpressionNode(directive.Expression); err != nil {
		return o, err
	}
	if o.chainKey, err = s.buildExpressionNode(directive.ChainKey); err != nil {
		return o, err
	}
	if o.location, err = s.buildLocation(&directive.Location); err != nil {
		return o, err
	}
	if o.nameLocation, err = s.buildLocation(&directive.NameLocation); err != nil {
		return o, err
	}
	if o.attributeRange, err = s.buildRange(&directive.AttributeRange); err != nil {
		return o, err
	}
	if o.goAnn, err = s.buildGoGeneratorAnnotation(directive.GoAnnotations); err != nil {
		return o, err
	}

	o.eventModifiers = s.buildEventModifiersVector(directive.EventModifiers)

	return o, nil
}

// buildEventModifiersVector serialises a slice of event modifier strings into a
// FlatBuffer vector.
//
// Takes modifiers ([]string) which contains the event modifiers.
//
// Returns flatbuffers.UOffsetT which is the vector offset, or 0 when empty.
func (s *encoder) buildEventModifiersVector(modifiers []string) flatbuffers.UOffsetT {
	if len(modifiers) == 0 {
		return 0
	}
	modOffs := make([]flatbuffers.UOffsetT, len(modifiers))
	for i, m := range modifiers {
		modOffs[i] = s.builder.CreateString(m)
	}
	ast_schema_gen.DirectiveFBStartEventModifiersVector(s.builder, len(modOffs))
	for i := len(modOffs) - 1; i >= 0; i-- {
		s.builder.PrependUOffsetT(modOffs[i])
	}
	return s.builder.EndVector(len(modOffs))
}

// assembleDirectiveFB writes the directive table using pre-built offsets.
//
// Takes directive (*ast_domain.Directive) which provides type and flags.
// Takes o (directiveOffsets) which contains the pre-built field offsets.
//
// Returns flatbuffers.UOffsetT which is the completed table offset.
func (s *encoder) assembleDirectiveFB(directive *ast_domain.Directive, o directiveOffsets) flatbuffers.UOffsetT {
	var fbType ast_schema_gen.DirectiveType
	if int(directive.Type) < len(goToFBDirectiveType) {
		fbType = goToFBDirectiveType[directive.Type]
	} else {
		fbType = ast_schema_gen.DirectiveType(safeconv.IntToUint16(int(directive.Type)))
	}

	ast_schema_gen.DirectiveFBStart(s.builder)
	ast_schema_gen.DirectiveFBAddType(s.builder, fbType)
	ast_schema_gen.DirectiveFBAddArgument(s.builder, o.argument)
	ast_schema_gen.DirectiveFBAddModifier(s.builder, o.modifier)
	ast_schema_gen.DirectiveFBAddRawExpression(s.builder, o.rawExpr)
	ast_schema_gen.DirectiveFBAddExpression(s.builder, o.expression)
	ast_schema_gen.DirectiveFBAddChainKey(s.builder, o.chainKey)
	ast_schema_gen.DirectiveFBAddLocation(s.builder, o.location)
	ast_schema_gen.DirectiveFBAddNameLocation(s.builder, o.nameLocation)
	ast_schema_gen.DirectiveFBAddAttributeRange(s.builder, o.attributeRange)
	ast_schema_gen.DirectiveFBAddGoAnnotations(s.builder, o.goAnn)
	if o.eventModifiers != 0 {
		ast_schema_gen.DirectiveFBAddEventModifiers(s.builder, o.eventModifiers)
	}
	ast_schema_gen.DirectiveFBAddIsStaticEvent(s.builder, directive.IsStaticEvent)

	return ast_schema_gen.DirectiveFBEnd(s.builder)
}

// buildHTMLAttribute serialises an HTML attribute to flatbuffer format.
//
// Takes attr (*ast_domain.HTMLAttribute) which is the attribute to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised attribute.
// Returns error when building location or range data fails.
func (s *encoder) buildHTMLAttribute(attr *ast_domain.HTMLAttribute) (flatbuffers.UOffsetT, error) {
	if attr == nil {
		return 0, nil
	}
	nameOff := s.builder.CreateString(attr.Name)
	valueOff := s.builder.CreateString(attr.Value)
	locOff, err := s.buildLocation(&attr.Location)
	if err != nil {
		return 0, err
	}
	nameLocOff, err := s.buildLocation(&attr.NameLocation)
	if err != nil {
		return 0, err
	}
	attributeRangeOffset, err := s.buildRange(&attr.AttributeRange)
	if err != nil {
		return 0, err
	}

	ast_schema_gen.HTMLAttributeFBStart(s.builder)
	ast_schema_gen.HTMLAttributeFBAddName(s.builder, nameOff)
	ast_schema_gen.HTMLAttributeFBAddValue(s.builder, valueOff)
	ast_schema_gen.HTMLAttributeFBAddLocation(s.builder, locOff)
	ast_schema_gen.HTMLAttributeFBAddNameLocation(s.builder, nameLocOff)
	ast_schema_gen.HTMLAttributeFBAddAttributeRange(s.builder, attributeRangeOffset)
	return ast_schema_gen.HTMLAttributeFBEnd(s.builder), nil
}

// buildDynamicAttribute serialises a dynamic attribute node to the flatbuffer.
//
// Takes attr (*ast_domain.DynamicAttribute) which is the dynamic attribute to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised attribute.
// Returns error when serialising child nodes fails.
func (s *encoder) buildDynamicAttribute(attr *ast_domain.DynamicAttribute) (flatbuffers.UOffsetT, error) {
	if attr == nil {
		return 0, nil
	}
	nameOff := s.builder.CreateString(attr.Name)
	rawExprOff := s.builder.CreateString(attr.RawExpression)

	expressionOffset, err := s.buildExpressionNode(attr.Expression)
	if err != nil {
		return 0, err
	}

	locOff, err := s.buildLocation(&attr.Location)
	if err != nil {
		return 0, err
	}
	nameLocOff, err := s.buildLocation(&attr.NameLocation)
	if err != nil {
		return 0, err
	}
	attributeRangeOffset, err := s.buildRange(&attr.AttributeRange)
	if err != nil {
		return 0, err
	}
	goAnnOff, err := s.buildGoGeneratorAnnotation(attr.GoAnnotations)
	if err != nil {
		return 0, err
	}

	ast_schema_gen.DynamicAttributeFBStart(s.builder)
	ast_schema_gen.DynamicAttributeFBAddName(s.builder, nameOff)
	ast_schema_gen.DynamicAttributeFBAddRawExpression(s.builder, rawExprOff)
	ast_schema_gen.DynamicAttributeFBAddExpression(s.builder, expressionOffset)
	ast_schema_gen.DynamicAttributeFBAddLocation(s.builder, locOff)
	ast_schema_gen.DynamicAttributeFBAddNameLocation(s.builder, nameLocOff)
	ast_schema_gen.DynamicAttributeFBAddAttributeRange(s.builder, attributeRangeOffset)
	ast_schema_gen.DynamicAttributeFBAddGoAnnotations(s.builder, goAnnOff)

	return ast_schema_gen.DynamicAttributeFBEnd(s.builder), nil
}

// buildTextPart serialises a text part to its FlatBuffer representation.
//
// Takes part (*ast_domain.TextPart) which is the text part to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised text part.
// Returns error when serialising the expression or location fails.
func (s *encoder) buildTextPart(part *ast_domain.TextPart) (flatbuffers.UOffsetT, error) {
	if part == nil {
		return 0, nil
	}
	literalOff := s.builder.CreateString(part.Literal)
	rawExprOff := s.builder.CreateString(part.RawExpression)

	expressionOffset, err := s.buildExpressionNode(part.Expression)
	if err != nil {
		return 0, err
	}

	locOff, err := s.buildLocation(&part.Location)
	if err != nil {
		return 0, err
	}

	goAnnOff, err := s.buildGoGeneratorAnnotation(part.GoAnnotations)
	if err != nil {
		return 0, err
	}

	ast_schema_gen.TextPartFBStart(s.builder)
	ast_schema_gen.TextPartFBAddIsLiteral(s.builder, part.IsLiteral)
	ast_schema_gen.TextPartFBAddLiteral(s.builder, literalOff)
	ast_schema_gen.TextPartFBAddRawExpression(s.builder, rawExprOff)
	ast_schema_gen.TextPartFBAddExpression(s.builder, expressionOffset)
	ast_schema_gen.TextPartFBAddLocation(s.builder, locOff)
	ast_schema_gen.TextPartFBAddGoAnnotations(s.builder, goAnnOff)

	return ast_schema_gen.TextPartFBEnd(s.builder), nil
}

// buildDirectWriter converts a DirectWriter into a FlatBuffer table.
//
// Takes dw (*ast_domain.DirectWriter) which holds the writer parts to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the built table.
// Returns error when the conversion fails.
func (s *encoder) buildDirectWriter(dw *ast_domain.DirectWriter) (flatbuffers.UOffsetT, error) {
	if dw == nil || dw.Len() == 0 {
		return 0, nil
	}

	nameOff := s.builder.CreateString(dw.Name)

	partOffsets := make([]flatbuffers.UOffsetT, dw.Len())
	for i := range dw.Len() {
		part := dw.Part(i)
		if part == nil {
			continue
		}

		var strOff flatbuffers.UOffsetT
		if part.Type == ast_domain.WriterPartString || part.Type == ast_domain.WriterPartEscapeString {
			strOff = s.builder.CreateString(part.StringValue)
		}

		ast_schema_gen.WriterPartFBStart(s.builder)
		ast_schema_gen.WriterPartFBAddType(s.builder, ast_schema_gen.WriterPartTypeFB(part.Type))
		if part.Type == ast_domain.WriterPartString || part.Type == ast_domain.WriterPartEscapeString {
			ast_schema_gen.WriterPartFBAddS(s.builder, strOff)
		}
		ast_schema_gen.WriterPartFBAddI(s.builder, part.IntValue)
		ast_schema_gen.WriterPartFBAddU(s.builder, part.UintValue)
		ast_schema_gen.WriterPartFBAddF(s.builder, part.FloatValue)
		ast_schema_gen.WriterPartFBAddB(s.builder, part.BoolValue)
		partOffsets[i] = ast_schema_gen.WriterPartFBEnd(s.builder)
	}

	ast_schema_gen.DirectWriterFBStartPartsVector(s.builder, len(partOffsets))
	for i := len(partOffsets) - 1; i >= 0; i-- {
		s.builder.PrependUOffsetT(partOffsets[i])
	}
	partsVec := s.builder.EndVector(len(partOffsets))

	ast_schema_gen.DirectWriterFBStart(s.builder)
	ast_schema_gen.DirectWriterFBAddName(s.builder, nameOff)
	ast_schema_gen.DirectWriterFBAddParts(s.builder, partsVec)
	return ast_schema_gen.DirectWriterFBEnd(s.builder), nil
}

// unpackTemplateAST converts a FlatBuffer template AST into a domain model.
//
// Takes fb (*ast_schema_gen.TemplateASTFB) which is the serialised template
// AST to convert.
//
// Returns *ast_domain.TemplateAST which is the converted domain model.
// Returns error when unpacking root nodes or diagnostics fails.
func (d *decoder) unpackTemplateAST(ctx context.Context, fb *ast_schema_gen.TemplateASTFB) (*ast_domain.TemplateAST, error) {
	if fb == nil {
		return nil, nil
	}

	ast := &ast_domain.TemplateAST{
		SourcePath: new(mem.String(fb.SourcePath())),
		Tidied:     fb.Tidied(),
	}

	var err error
	ast.RootNodes, err = d.unpackTemplateNodeVector(ctx, fb.RootNodesLength(), fb.RootNodes)
	if err != nil {
		return nil, fmt.Errorf("unpacking template AST root nodes: %w", err)
	}

	ast.Diagnostics, err = unpackPtrVector(d, fb.DiagnosticsLength(), fb.Diagnostics, (*decoder).unpackDiagnostic)
	if err != nil {
		return nil, fmt.Errorf("unpacking template AST diagnostics: %w", err)
	}

	expiresAt := fb.ExpiresAtUnixNano()
	if expiresAt > 0 {
		ast.ExpiresAtUnixNano = &expiresAt
	}

	metadata := mem.String(fb.Metadata())
	if len(metadata) > 0 {
		ast.Metadata = &metadata
	}

	ast.SourceSize = fb.SourceSize()

	return ast, nil
}

// unpackTemplateNodeVector deserialises a vector of TemplateNodeFB entries,
// threading ctx through each node for logging within DirectWriter unpacking.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes length (int) which is the number of entries in the vector.
// Takes accessor (func(*ast_schema_gen.TemplateNodeFB, int) bool) which
// retrieves the entry at the given index into the provided flatbuffer.
//
// Returns []*ast_domain.TemplateNode which contains the deserialised nodes.
// Returns error when any node fails to unpack.
func (d *decoder) unpackTemplateNodeVector(ctx context.Context, length int, accessor func(*ast_schema_gen.TemplateNodeFB, int) bool) ([]*ast_domain.TemplateNode, error) {
	if length == 0 {
		return nil, nil
	}
	result := make([]*ast_domain.TemplateNode, length)
	var nodeFB ast_schema_gen.TemplateNodeFB
	for i := range length {
		if accessor(&nodeFB, i) {
			node, err := d.unpackTemplateNode(ctx, &nodeFB)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack pointer item at index %d: %w", i, err)
			}
			result[i] = node
		}
	}
	return result, nil
}

// unpackDirectWriters deserialises the attribute writer vector from a
// TemplateNodeFB, threading ctx for the logging site in unpackDirectWriter.
//
// Takes ctx (context.Context) which carries the request-scoped logger.
// Takes fb (*ast_schema_gen.TemplateNodeFB) which provides the serialised
// writer vector.
//
// Returns []*ast_domain.DirectWriter which contains the deserialised writers.
// Returns error when any writer fails to unpack.
func (d *decoder) unpackDirectWriters(ctx context.Context, fb *ast_schema_gen.TemplateNodeFB) ([]*ast_domain.DirectWriter, error) {
	length := fb.AttributeWritersLength()
	if length == 0 {
		return nil, nil
	}
	result := make([]*ast_domain.DirectWriter, length)
	for i := range length {
		if fb.AttributeWriters(&d.dwFB, i) {
			writer, err := d.unpackDirectWriter(ctx, &d.dwFB)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack pointer item at index %d: %w", i, err)
			}
			result[i] = writer
		}
	}
	return result, nil
}

// unpackTemplateNodeRanges unpacks location and range fields into the node.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which provides the source data.
// Takes node (*ast_domain.TemplateNode) which receives the unpacked fields.
//
// Returns error when unpacking any location or range field fails.
func (d *decoder) unpackTemplateNodeRanges(fb *ast_schema_gen.TemplateNodeFB, node *ast_domain.TemplateNode) error {
	var err error

	if d.skipRanges {
		return nil
	}

	if locFB := fb.Location(&d.locFB); locFB != nil {
		node.Location, err = d.unpackLocation(locFB)
		if err != nil {
			return fmt.Errorf("unpacking Location: %w", err)
		}
	}
	if nodeRangeFB := fb.NodeRange(&d.rangeFB); nodeRangeFB != nil {
		node.NodeRange, err = d.unpackRange(nodeRangeFB)
		if err != nil {
			return fmt.Errorf("unpacking NodeRange: %w", err)
		}
	}
	if openingTagRangeFB := fb.OpeningTagRange(&d.rangeFB); openingTagRangeFB != nil {
		node.OpeningTagRange, err = d.unpackRange(openingTagRangeFB)
		if err != nil {
			return fmt.Errorf("unpacking OpeningTagRange: %w", err)
		}
	}
	if closingTagRangeFB := fb.ClosingTagRange(&d.rangeFB); closingTagRangeFB != nil {
		node.ClosingTagRange, err = d.unpackRange(closingTagRangeFB)
		if err != nil {
			return fmt.Errorf("unpacking ClosingTagRange: %w", err)
		}
	}

	return nil
}

// unpackTemplateNodeVectors unpacks all vector fields into the node.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which is the FlatBuffer source.
// Takes node (*ast_domain.TemplateNode) which receives the unpacked vectors.
//
// Returns error when any vector unpacking fails.
func (d *decoder) unpackTemplateNodeVectors(ctx context.Context, fb *ast_schema_gen.TemplateNodeFB, node *ast_domain.TemplateNode) error {
	var err error

	if node.RichText, err = unpackVector(d, fb.RichTextLength(), fb.RichText, (*decoder).unpackTextPart); err != nil {
		return fmt.Errorf("unpacking RichText: %w", err)
	}
	if node.Attributes, err = unpackVector(d, fb.AttributesLength(), fb.Attributes, (*decoder).unpackHTMLAttribute); err != nil {
		return fmt.Errorf("unpacking Attributes: %w", err)
	}
	if node.DynamicAttributes, err = unpackVector(d, fb.DynamicAttributesLength(), fb.DynamicAttributes, (*decoder).unpackDynamicAttribute); err != nil {
		return fmt.Errorf("unpacking DynamicAttributes: %w", err)
	}
	if node.Diagnostics, err = unpackPtrVector(d, fb.DiagnosticsLength(), fb.Diagnostics, (*decoder).unpackDiagnostic); err != nil {
		return fmt.Errorf("unpacking Diagnostics: %w", err)
	}
	if node.Children, err = d.unpackTemplateNodeVector(ctx, fb.ChildrenLength(), fb.Children); err != nil {
		return fmt.Errorf("unpacking Children: %w", err)
	}
	if node.Directives, err = unpackVector(d, fb.DirectivesLength(), fb.Directives, (*decoder).unpackDirectiveValue); err != nil {
		return fmt.Errorf("unpacking Directives: %w", err)
	}
	if node.OnEvents, err = d.unpackOnEventsMap(fb); err != nil {
		return fmt.Errorf("unpacking OnEvents: %w", err)
	}
	if node.CustomEvents, err = d.unpackCustomEventsMap(fb); err != nil {
		return fmt.Errorf("unpacking CustomEvents: %w", err)
	}
	if node.Binds, err = d.unpackBindsMap(fb); err != nil {
		return fmt.Errorf("unpacking Binds: %w", err)
	}
	if node.AttributeWriters, err = d.unpackDirectWriters(ctx, fb); err != nil {
		return fmt.Errorf("unpacking AttributeWriters: %w", err)
	}

	return nil
}

// unpackTemplateNodeDirectives unpacks all distributed directive fields into
// the node.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which is the source flatbuffer.
// Takes node (*ast_domain.TemplateNode) which receives the unpacked directives.
//
// Returns error when any directive fails to unpack.
func (d *decoder) unpackTemplateNodeDirectives(fb *ast_schema_gen.TemplateNodeFB, node *ast_domain.TemplateNode) error {
	var err error
	if node.DirIf, err = d.unpackDirective(fb.DirectiveIf(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirIf: %w", err)
	}
	if node.DirElseIf, err = d.unpackDirective(fb.DirectiveElseIf(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirElseIf: %w", err)
	}
	if node.DirElse, err = d.unpackDirective(fb.DirectiveElse(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirElse: %w", err)
	}
	if node.DirFor, err = d.unpackDirective(fb.DirectiveFor(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirFor: %w", err)
	}
	if node.DirShow, err = d.unpackDirective(fb.DirectiveShow(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirShow: %w", err)
	}
	if node.DirModel, err = d.unpackDirective(fb.DirectiveModel(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirModel: %w", err)
	}
	if node.DirRef, err = d.unpackDirective(fb.DirectiveRef(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirRef: %w", err)
	}
	if node.DirSlot, err = d.unpackDirective(fb.DirectiveSlot(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirSlot: %w", err)
	}
	if node.DirClass, err = d.unpackDirective(fb.DirectiveClass(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirClass: %w", err)
	}
	if node.DirStyle, err = d.unpackDirective(fb.DirectiveStyle(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirStyle: %w", err)
	}
	if node.DirText, err = d.unpackDirective(fb.DirectiveText(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirText: %w", err)
	}
	if node.DirHTML, err = d.unpackDirective(fb.DirectiveHtml(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirHTML: %w", err)
	}
	if node.DirKey, err = d.unpackDirective(fb.DirectiveKey(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirKey: %w", err)
	}
	if node.DirContext, err = d.unpackDirective(fb.DirectiveContext(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirContext: %w", err)
	}
	if node.DirScaffold, err = d.unpackDirective(fb.DirectiveScaffold(&d.dirFB)); err != nil {
		return fmt.Errorf("unpacking DirScaffold: %w", err)
	}
	return nil
}

// unpackTemplateNodeAnnotations unpacks annotation and key fields into the
// node.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which provides the serialised
// template node data.
// Takes node (*ast_domain.TemplateNode) which receives the unpacked fields.
//
// Returns error when any annotation or key field fails to unpack.
func (d *decoder) unpackTemplateNodeAnnotations(ctx context.Context, fb *ast_schema_gen.TemplateNodeFB, node *ast_domain.TemplateNode) error {
	var err error

	if node.Key, err = d.unpackExpressionNode(fb.Key(&d.expressionNodeFB)); err != nil {
		return fmt.Errorf("unpacking Key: %w", err)
	}
	if node.GoAnnotations, err = d.unpackGoGeneratorAnnotation(fb.GoAnnotations(&d.goAnnotFB)); err != nil {
		return fmt.Errorf("unpacking GoAnnotations: %w", err)
	}
	if node.RuntimeAnnotations, err = d.unpackRuntimeAnnotation(fb.RuntimeAnnotations(&d.rtAnnotFB)); err != nil {
		return fmt.Errorf("unpacking RuntimeAnnotations: %w", err)
	}
	if node.TextContentWriter, err = d.unpackDirectWriter(ctx, fb.TextContentWriter(&d.dwFB)); err != nil {
		return fmt.Errorf("unpacking TextContentWriter: %w", err)
	}

	return nil
}

// unpackTemplateNode converts a FlatBuffer template node to a domain object.
//
// Takes fb (*ast_schema_gen.TemplateNodeFB) which is the FlatBuffer node to
// convert.
//
// Returns *ast_domain.TemplateNode which is the converted domain object, or
// nil if fb is nil.
// Returns error when unpacking ranges, vectors, directives, or annotations
// fails.
func (d *decoder) unpackTemplateNode(ctx context.Context, fb *ast_schema_gen.TemplateNodeFB) (*ast_domain.TemplateNode, error) {
	if fb == nil {
		return nil, nil
	}

	node := &ast_domain.TemplateNode{
		NodeType:           ast_domain.NodeType(fb.NodeType()),
		TagName:            mem.String(fb.TagName()),
		TextContent:        mem.String(fb.TextContent()),
		InnerHTML:          mem.String(fb.InnerHtml()),
		IsContentEditable:  fb.IsContentEditable(),
		PreserveWhitespace: fb.PreserveWhitespace(),
		PreferredFormat:    ast_domain.FormatHint(fb.PreferredFormat()),
		PrerenderedHTML:    fb.PrerenderedHtmlBytes(),
	}

	if err := d.unpackTemplateNodeRanges(fb, node); err != nil {
		return nil, fmt.Errorf("unpacking template node ranges: %w", err)
	}
	if err := d.unpackTemplateNodeVectors(ctx, fb, node); err != nil {
		return nil, fmt.Errorf("unpacking template node vectors: %w", err)
	}
	if err := d.unpackTemplateNodeDirectives(fb, node); err != nil {
		return nil, fmt.Errorf("unpacking template node directives: %w", err)
	}
	if err := d.unpackTemplateNodeAnnotations(ctx, fb, node); err != nil {
		return nil, fmt.Errorf("unpacking template node annotations: %w", err)
	}

	return node, nil
}

// unpackDirectWriter converts a FlatBuffer DirectWriterFB into a domain
// DirectWriter by extracting its name and reconstructing its parts.
//
// Takes fb (*ast_schema_gen.DirectWriterFB) which is the serialised writer.
//
// Returns *ast_domain.DirectWriter which is the reconstructed domain object.
// Returns error when deserialisation fails.
func (*decoder) unpackDirectWriter(ctx context.Context, fb *ast_schema_gen.DirectWriterFB) (*ast_domain.DirectWriter, error) {
	if fb == nil || fb.PartsLength() == 0 {
		return nil, nil
	}

	dw := ast_domain.GetDirectWriter()

	dw.SetName(mem.String(fb.Name()))

	for i := range fb.PartsLength() {
		var partFB ast_schema_gen.WriterPartFB
		if !fb.Parts(&partFB, i) {
			continue
		}

		partType := ast_domain.WriterPartType(partFB.Type())
		switch partType {
		case ast_domain.WriterPartString:
			dw.AppendString(mem.String(partFB.S()))
		case ast_domain.WriterPartEscapeString:
			dw.AppendEscapeString(mem.String(partFB.S()))
		case ast_domain.WriterPartInt:
			dw.AppendInt(partFB.I())
		case ast_domain.WriterPartUint:
			dw.AppendUint(partFB.U())
		case ast_domain.WriterPartFloat:
			dw.AppendFloat(partFB.F())
		case ast_domain.WriterPartBool:
			dw.AppendBool(partFB.B())
		default:
			_, l := logger_domain.From(ctx, log)
			l.Warn("Unknown writer part type during deserialisation",
				logger_domain.Int("part_type", int(partType)),
				logger_domain.String("writer_name", dw.Name))
		}
	}

	return dw, nil
}

// unpackDirective converts a FlatBuffer directive into a domain directive.
//
// Takes fb (*ast_schema_gen.DirectiveFB) which is the FlatBuffer directive to
// convert.
//
// Returns *ast_domain.Directive which is the converted domain directive, or
// nil if fb is nil.
// Returns error when any nested field fails to unpack.
func (d *decoder) unpackDirective(fb *ast_schema_gen.DirectiveFB) (*ast_domain.Directive, error) {
	if fb == nil {
		return nil, nil
	}

	var goType ast_domain.DirectiveType
	fbTypeVal := fb.Type()
	if int(fbTypeVal) < len(fbToGoDirectiveType) {
		goType = fbToGoDirectiveType[fbTypeVal]
	} else {
		goType = ast_domain.DirectiveType(fbTypeVal)
	}

	directive := &ast_domain.Directive{
		Type:          goType,
		Arg:           mem.String(fb.Argument()),
		Modifier:      mem.String(fb.Modifier()),
		RawExpression: mem.String(fb.RawExpression()),
		IsStaticEvent: fb.IsStaticEvent(),
	}

	if n := fb.EventModifiersLength(); n > 0 {
		directive.EventModifiers = make([]string, n)
		for i := range n {
			directive.EventModifiers[i] = mem.String(fb.EventModifiers(i))
		}
	}

	var err error
	directive.Expression, err = d.unpackExpressionNode(fb.Expression(&d.expressionNodeFB))
	if err != nil {
		return nil, err
	}
	directive.ChainKey, err = d.unpackExpressionNode(fb.ChainKey(&d.expressionNodeFB))
	if err != nil {
		return nil, err
	}
	if !d.skipRanges {
		directive.Location, err = d.unpackLocation(fb.Location(&d.locFB))
		if err != nil {
			return nil, err
		}
		directive.NameLocation, err = d.unpackLocation(fb.NameLocation(&d.locFB))
		if err != nil {
			return nil, err
		}
		directive.AttributeRange, err = d.unpackRange(fb.AttributeRange(&d.rangeFB))
		if err != nil {
			return nil, err
		}
	}
	directive.GoAnnotations, err = d.unpackGoGeneratorAnnotation(fb.GoAnnotations(&d.goAnnotFB))
	if err != nil {
		return nil, err
	}

	return directive, nil
}

// unpackHTMLAttribute converts a FlatBuffer HTML attribute to a domain model.
//
// Takes fb (*ast_schema_gen.HTMLAttributeFB) which is the serialised attribute.
//
// Returns ast_domain.HTMLAttribute which is the converted domain object.
// Returns error when location or range unpacking fails.
func (d *decoder) unpackHTMLAttribute(fb *ast_schema_gen.HTMLAttributeFB) (ast_domain.HTMLAttribute, error) {
	if fb == nil {
		return ast_domain.HTMLAttribute{}, nil
	}
	var location, nameLocation ast_domain.Location
	var attributeRange ast_domain.Range
	if !d.skipRanges {
		var err error
		location, err = d.unpackLocation(fb.Location(&d.locFB))
		if err != nil {
			return ast_domain.HTMLAttribute{}, err
		}
		nameLocation, err = d.unpackLocation(fb.NameLocation(&d.locFB))
		if err != nil {
			return ast_domain.HTMLAttribute{}, err
		}
		attributeRange, err = d.unpackRange(fb.AttributeRange(&d.rangeFB))
		if err != nil {
			return ast_domain.HTMLAttribute{}, err
		}
	}

	return ast_domain.HTMLAttribute{
		Name:           mem.String(fb.Name()),
		Value:          mem.String(fb.Value()),
		Location:       location,
		NameLocation:   nameLocation,
		AttributeRange: attributeRange,
	}, nil
}

// unpackDynamicAttribute converts a FlatBuffer dynamic attribute to a domain
// model representation.
//
// Takes fb (*ast_schema_gen.DynamicAttributeFB) which is the FlatBuffer to
// convert.
//
// Returns ast_domain.DynamicAttribute which contains the unpacked attribute
// data.
// Returns error when unpacking the expression, location, or range fails.
func (d *decoder) unpackDynamicAttribute(fb *ast_schema_gen.DynamicAttributeFB) (ast_domain.DynamicAttribute, error) {
	if fb == nil {
		return ast_domain.DynamicAttribute{}, nil
	}

	attr := ast_domain.DynamicAttribute{
		Name:          mem.String(fb.Name()),
		RawExpression: mem.String(fb.RawExpression()),
	}

	var err error

	attr.Expression, err = d.unpackExpressionNode(fb.Expression(&d.expressionNodeFB))
	if err != nil {
		return attr, err
	}

	attr.GoAnnotations, err = d.unpackGoGeneratorAnnotation(fb.GoAnnotations(&d.goAnnotFB))
	if err != nil {
		return attr, err
	}

	if !d.skipRanges {
		attr.Location, err = d.unpackLocation(fb.Location(&d.locFB))
		if err != nil {
			return attr, err
		}
		attr.NameLocation, err = d.unpackLocation(fb.NameLocation(&d.locFB))
		if err != nil {
			return attr, err
		}
		attr.AttributeRange, err = d.unpackRange(fb.AttributeRange(&d.rangeFB))
		if err != nil {
			return attr, err
		}
	}

	return attr, nil
}

// unpackTextPart converts a FlatBuffer TextPartFB into a domain TextPart.
//
// Takes fb (*ast_schema_gen.TextPartFB) which is the FlatBuffer to convert.
//
// Returns ast_domain.TextPart which is the converted domain object.
// Returns error when unpacking the expression or location fails.
func (d *decoder) unpackTextPart(fb *ast_schema_gen.TextPartFB) (ast_domain.TextPart, error) {
	if fb == nil {
		return ast_domain.TextPart{}, nil
	}

	part := ast_domain.TextPart{
		IsLiteral:     fb.IsLiteral(),
		Literal:       mem.String(fb.Literal()),
		RawExpression: mem.String(fb.RawExpression()),
	}

	var err error

	part.Expression, err = d.unpackExpressionNode(fb.Expression(&d.expressionNodeFB))
	if err != nil {
		return part, err
	}

	part.GoAnnotations, err = d.unpackGoGeneratorAnnotation(fb.GoAnnotations(&d.goAnnotFB))
	if err != nil {
		return part, err
	}

	if !d.skipRanges {
		part.Location, err = d.unpackLocation(fb.Location(&d.locFB))
		if err != nil {
			return part, err
		}
	}

	return part, nil
}

// unpackDirectiveValue returns a Directive by value for use with unpackVector
// when working with value slices.
//
// Takes fb (*ast_schema_gen.DirectiveFB) which is the flatbuffer directive to
// unpack.
//
// Returns ast_domain.Directive which is the unpacked directive value.
// Returns error when the underlying unpackDirective call fails.
func (d *decoder) unpackDirectiveValue(fb *ast_schema_gen.DirectiveFB) (ast_domain.Directive, error) {
	directive, err := d.unpackDirective(fb)
	if err != nil {
		return ast_domain.Directive{}, fmt.Errorf("unpacking directive value: %w", err)
	}
	if directive == nil {
		return ast_domain.Directive{}, nil
	}
	return *directive, nil
}
