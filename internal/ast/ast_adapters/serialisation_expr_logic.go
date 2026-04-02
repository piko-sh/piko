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
	"fmt"
	"maps"
	"reflect"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/ast/ast_schema/ast_schema_gen"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safeconv"
)

// expressionPayloadCount is the total number of expression payload types for
// array dispatch sizing.
const expressionPayloadCount = 24

// expressionEncoderEntry pairs a FlatBuffer payload type with a builder
// function for a specific concrete Expression type.
type expressionEncoderEntry struct {
	// build is the function that encodes the expression to FlatBuffers.
	build func(*encoder, ast_domain.Expression) (flatbuffers.UOffsetT, error)

	// payloadType is the FlatBuffer union discriminator for this entry.
	payloadType ast_schema_gen.ExpressionPayloadFB
}

var (
	// mapGoUnaryOpToFB maps the Go domain UnaryOp type to the FlatBuffers enum.
	mapGoUnaryOpToFB = map[ast_domain.UnaryOp]ast_schema_gen.UnaryOperatorFB{
		ast_domain.OpNot:    ast_schema_gen.UnaryOperatorFBNOT,
		ast_domain.OpNeg:    ast_schema_gen.UnaryOperatorFBNEG,
		ast_domain.OpAddrOf: ast_schema_gen.UnaryOperatorFBADDR_OF,
		ast_domain.OpTruthy: ast_schema_gen.UnaryOperatorFBTRUTHY,
	}

	// mapFBUnaryOpToGo maps the FlatBuffers enum for UnaryOperatorFB to the Go
	// domain type.
	mapFBUnaryOpToGo = map[ast_schema_gen.UnaryOperatorFB]ast_domain.UnaryOp{
		ast_schema_gen.UnaryOperatorFBNOT:     ast_domain.OpNot,
		ast_schema_gen.UnaryOperatorFBNEG:     ast_domain.OpNeg,
		ast_schema_gen.UnaryOperatorFBADDR_OF: ast_domain.OpAddrOf,
		ast_schema_gen.UnaryOperatorFBTRUTHY:  ast_domain.OpTruthy,
	}

	// mapGoBinaryOpToFB maps the Go domain BinaryOp type to the FlatBuffers enum.
	mapGoBinaryOpToFB = map[ast_domain.BinaryOp]ast_schema_gen.BinaryOperatorFB{
		ast_domain.OpEq:       ast_schema_gen.BinaryOperatorFBEQUAL,
		ast_domain.OpNe:       ast_schema_gen.BinaryOperatorFBNOTEQUAL,
		ast_domain.OpLooseEq:  ast_schema_gen.BinaryOperatorFBLOOSEEQUAL,
		ast_domain.OpLooseNe:  ast_schema_gen.BinaryOperatorFBLOOSENOTEQUAL,
		ast_domain.OpGt:       ast_schema_gen.BinaryOperatorFBGREATER,
		ast_domain.OpLt:       ast_schema_gen.BinaryOperatorFBLESS,
		ast_domain.OpGe:       ast_schema_gen.BinaryOperatorFBGTE,
		ast_domain.OpLe:       ast_schema_gen.BinaryOperatorFBLTE,
		ast_domain.OpAnd:      ast_schema_gen.BinaryOperatorFBAND,
		ast_domain.OpOr:       ast_schema_gen.BinaryOperatorFBOR,
		ast_domain.OpPlus:     ast_schema_gen.BinaryOperatorFBPLUS,
		ast_domain.OpMinus:    ast_schema_gen.BinaryOperatorFBMINUS,
		ast_domain.OpMul:      ast_schema_gen.BinaryOperatorFBMULTIPLY,
		ast_domain.OpDiv:      ast_schema_gen.BinaryOperatorFBDIVIDE,
		ast_domain.OpMod:      ast_schema_gen.BinaryOperatorFBMODULO,
		ast_domain.OpCoalesce: ast_schema_gen.BinaryOperatorFBCOALESCE,
	}

	// mapFBBinaryOpToGo maps the FlatBuffers enum for BinaryOperatorFB to the Go
	// domain type.
	mapFBBinaryOpToGo = make(map[ast_schema_gen.BinaryOperatorFB]ast_domain.BinaryOp)

	// expressionUnpackers is an array dispatch table mapping ExpressionPayloadFB
	// enum values to their corresponding unpacker functions.
	//
	// This is a pointer to break the initialisation cycle between the dispatch
	// table and the unpacker functions that recursively call unpackExpressionNode.
	expressionUnpackers *[expressionPayloadCount]expressionUnpacker

	// expressionEncoders maps concrete Expression types to their FlatBuffer
	// payload type and builder function. Used by buildExpressionNode for
	// table-driven dispatch instead of a type switch.
	expressionEncoders map[reflect.Type]expressionEncoderEntry
)

// buildExpressionNode is the main encoder for the Expression union. It
// determines the concrete type of the expression via the expressionEncoders
// dispatch table, calls the appropriate builder, and wraps the result in an
// ExpressionNodeFB wrapper table.
//
// Takes expression (ast_domain.Expression) which is the expression
// to encode into the FlatBuffer.
//
// Returns flatbuffers.UOffsetT which is the offset of the encoded
// wrapper table.
// Returns error when the expression type is unhandled or encoding
// fails.
func (s *encoder) buildExpressionNode(expression ast_domain.Expression) (flatbuffers.UOffsetT, error) {
	if expression == nil {
		return 0, nil
	}

	entry, ok := expressionEncoders[reflect.TypeOf(expression)]
	if !ok {
		return 0, fmt.Errorf("unhandled expression type for encoding: %T", expression)
	}

	payloadOffset, err := entry.build(s, expression)
	if err != nil {
		return 0, fmt.Errorf("building expression node: %w", err)
	}

	ast_schema_gen.ExpressionNodeFBStart(s.builder)
	ast_schema_gen.ExpressionNodeFBAddPayloadType(s.builder, entry.payloadType)
	ast_schema_gen.ExpressionNodeFBAddPayload(s.builder, payloadOffset)
	ast_schema_gen.ExpressionNodeFBAddSourceLength(s.builder, safeconv.IntToInt32(expression.GetSourceLength()))
	return ast_schema_gen.ExpressionNodeFBEnd(s.builder), nil
}

// buildMemberExpr serialises a member expression to FlatBuffers format.
//
// Takes expression (*ast_domain.MemberExpression) which is the
// member expression to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the
// serialised data.
// Returns error when serialising the base, property, or location
// fails.
func (s *encoder) buildMemberExpr(expression *ast_domain.MemberExpression) (flatbuffers.UOffsetT, error) {
	if expression == nil {
		return 0, nil
	}
	baseOff, err := s.buildExpressionNode(expression.Base)
	if err != nil {
		return 0, fmt.Errorf("serialise member expression base: %w", err)
	}
	propOff, err := s.buildExpressionNode(expression.Property)
	if err != nil {
		return 0, fmt.Errorf("serialise member expression property: %w", err)
	}
	locOff, err := s.buildLocation(&expression.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise member expression location: %w", err)
	}

	ast_schema_gen.MemberExprFBStart(s.builder)
	ast_schema_gen.MemberExprFBAddBase(s.builder, baseOff)
	ast_schema_gen.MemberExprFBAddProperty(s.builder, propOff)
	ast_schema_gen.MemberExprFBAddOptional(s.builder, expression.Optional)
	ast_schema_gen.MemberExprFBAddComputed(s.builder, expression.Computed)
	ast_schema_gen.MemberExprFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.MemberExprFBEnd(s.builder), nil
}

// buildIndexExpr serialises an index expression to the FlatBuffer format.
//
// Takes expression (*ast_domain.IndexExpression) which is the
// index expression to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the
// serialised data.
// Returns error when serialising any child node fails.
func (s *encoder) buildIndexExpr(expression *ast_domain.IndexExpression) (flatbuffers.UOffsetT, error) {
	if expression == nil {
		return 0, nil
	}
	baseOff, err := s.buildExpressionNode(expression.Base)
	if err != nil {
		return 0, fmt.Errorf("serialise index expression base: %w", err)
	}
	indexOff, err := s.buildExpressionNode(expression.Index)
	if err != nil {
		return 0, fmt.Errorf("serialise index expression index: %w", err)
	}
	locOff, err := s.buildLocation(&expression.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise index expression location: %w", err)
	}

	ast_schema_gen.IndexExprFBStart(s.builder)
	ast_schema_gen.IndexExprFBAddBase(s.builder, baseOff)
	ast_schema_gen.IndexExprFBAddIndex(s.builder, indexOff)
	ast_schema_gen.IndexExprFBAddOptional(s.builder, expression.Optional)
	ast_schema_gen.IndexExprFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.IndexExprFBEnd(s.builder), nil
}

// buildUnaryExpr serialises a unary expression to FlatBuffers format.
//
// Takes expression (*ast_domain.UnaryExpression) which is the
// unary expression to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the
// serialised data.
// Returns error when serialising the operand or location fails.
func (s *encoder) buildUnaryExpr(expression *ast_domain.UnaryExpression) (flatbuffers.UOffsetT, error) {
	if expression == nil {
		return 0, nil
	}
	rightOff, err := s.buildExpressionNode(expression.Right)
	if err != nil {
		return 0, fmt.Errorf("serialise unary expression operand: %w", err)
	}
	locOff, err := s.buildLocation(&expression.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise unary expression location: %w", err)
	}

	ast_schema_gen.UnaryExprFBStart(s.builder)
	ast_schema_gen.UnaryExprFBAddOperator(s.builder, mapGoUnaryOpToFB[expression.Operator])
	ast_schema_gen.UnaryExprFBAddRight(s.builder, rightOff)
	ast_schema_gen.UnaryExprFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.UnaryExprFBEnd(s.builder), nil
}

// buildBinaryExpr serialises a binary expression to the FlatBuffer format.
//
// Takes expression (*ast_domain.BinaryExpression) which is the
// binary expression to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the
// serialised expression.
// Returns error when serialising the left, right, or location
// fails.
func (s *encoder) buildBinaryExpr(expression *ast_domain.BinaryExpression) (flatbuffers.UOffsetT, error) {
	if expression == nil {
		return 0, nil
	}
	leftOff, err := s.buildExpressionNode(expression.Left)
	if err != nil {
		return 0, fmt.Errorf("serialise binary expression left operand: %w", err)
	}
	rightOff, err := s.buildExpressionNode(expression.Right)
	if err != nil {
		return 0, fmt.Errorf("serialise binary expression right operand: %w", err)
	}
	locOff, err := s.buildLocation(&expression.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise binary expression location: %w", err)
	}

	ast_schema_gen.BinaryExprFBStart(s.builder)
	ast_schema_gen.BinaryExprFBAddLeft(s.builder, leftOff)
	ast_schema_gen.BinaryExprFBAddOperator(s.builder, mapGoBinaryOpToFB[expression.Operator])
	ast_schema_gen.BinaryExprFBAddRight(s.builder, rightOff)
	ast_schema_gen.BinaryExprFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.BinaryExprFBEnd(s.builder), nil
}

// buildCallExpr serialises a call expression to FlatBuffers format.
//
// Takes expression (*ast_domain.CallExpression) which is the call
// expression to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the
// serialised call expression.
// Returns error when serialising the callee, arguments, or
// location fails.
func (s *encoder) buildCallExpr(expression *ast_domain.CallExpression) (flatbuffers.UOffsetT, error) {
	if expression == nil {
		return 0, nil
	}
	calleeOff, err := s.buildExpressionNode(expression.Callee)
	if err != nil {
		return 0, fmt.Errorf("serialise call expression callee: %w", err)
	}
	argsVec, err := s.buildExpressionVector(expression.Args)
	if err != nil {
		return 0, fmt.Errorf("serialise call expression arguments: %w", err)
	}
	locOff, err := s.buildLocation(&expression.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise call expression location: %w", err)
	}
	lparenOff, err := s.buildLocation(&expression.LparenLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise call expression lparen location: %w", err)
	}
	rparenOff, err := s.buildLocation(&expression.RparenLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise call expression rparen location: %w", err)
	}

	ast_schema_gen.CallExprFBStart(s.builder)
	ast_schema_gen.CallExprFBAddCallee(s.builder, calleeOff)
	ast_schema_gen.CallExprFBAddArguments(s.builder, argsVec)
	ast_schema_gen.CallExprFBAddLeftParenthesisLocation(s.builder, lparenOff)
	ast_schema_gen.CallExprFBAddRightParenthesisLocation(s.builder, rparenOff)
	ast_schema_gen.CallExprFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.CallExprFBEnd(s.builder), nil
}

// buildForInExpr serialises a for-in expression to its flatbuffer form.
//
// Takes expression (*ast_domain.ForInExpression) which is the
// expression to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the
// serialised data.
// Returns error when any child element fails to serialise.
//
//nolint:dupl // type-specific FlatBuffer encode/decode
func (s *encoder) buildForInExpr(expression *ast_domain.ForInExpression) (flatbuffers.UOffsetT, error) {
	if expression == nil {
		return 0, nil
	}
	idxVarOff, err := s.buildIdentifier(expression.IndexVariable)
	if err != nil {
		return 0, fmt.Errorf("serialise for-in expression index variable: %w", err)
	}
	itemVarOff, err := s.buildIdentifier(expression.ItemVariable)
	if err != nil {
		return 0, fmt.Errorf("serialise for-in expression item variable: %w", err)
	}
	collOff, err := s.buildExpressionNode(expression.Collection)
	if err != nil {
		return 0, fmt.Errorf("serialise for-in expression collection: %w", err)
	}
	locOff, err := s.buildLocation(&expression.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise for-in expression location: %w", err)
	}

	ast_schema_gen.ForInExprFBStart(s.builder)
	ast_schema_gen.ForInExprFBAddIndexVariable(s.builder, idxVarOff)
	ast_schema_gen.ForInExprFBAddItemVariable(s.builder, itemVarOff)
	ast_schema_gen.ForInExprFBAddCollection(s.builder, collOff)
	ast_schema_gen.ForInExprFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.ForInExprFBEnd(s.builder), nil
}

// buildArrayLiteral serialises an array literal to the FlatBuffer format.
//
// Takes lit (*ast_domain.ArrayLiteral) which is the array literal to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when serialising elements or location fails.
//
//nolint:dupl // type-specific FlatBuffer encode/decode
func (s *encoder) buildArrayLiteral(lit *ast_domain.ArrayLiteral) (flatbuffers.UOffsetT, error) {
	if lit == nil {
		return 0, nil
	}
	elementsVec, err := s.buildExpressionVector(lit.Elements)
	if err != nil {
		return 0, fmt.Errorf("serialise array literal elements: %w", err)
	}
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise array literal location: %w", err)
	}

	ast_schema_gen.ArrayLiteralFBStart(s.builder)
	ast_schema_gen.ArrayLiteralFBAddElements(s.builder, elementsVec)
	ast_schema_gen.ArrayLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.ArrayLiteralFBEnd(s.builder), nil
}

// buildObjectLiteral serialises an object literal AST node to FlatBuffers.
//
// Takes lit (*ast_domain.ObjectLiteral) which is the object literal to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised object.
// Returns error when serialising pairs or location fails.
//
//nolint:dupl // type-specific FlatBuffer encode/decode
func (s *encoder) buildObjectLiteral(lit *ast_domain.ObjectLiteral) (flatbuffers.UOffsetT, error) {
	if lit == nil {
		return 0, nil
	}

	pairsVectorOff, err := s.buildObjectLiteralPairs(lit.Pairs)
	if err != nil {
		return 0, fmt.Errorf("serialise object literal pairs: %w", err)
	}
	relativeLocOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise object literal location: %w", err)
	}

	ast_schema_gen.ObjectLiteralFBStart(s.builder)
	ast_schema_gen.ObjectLiteralFBAddPairs(s.builder, pairsVectorOff)
	ast_schema_gen.ObjectLiteralFBAddRelativeLocation(s.builder, relativeLocOff)
	return ast_schema_gen.ObjectLiteralFBEnd(s.builder), nil
}

// buildTernaryExpr serialises a ternary expression to the flatbuffer.
//
// Takes expression (*ast_domain.TernaryExpression) which is the
// ternary expression to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the
// serialised expression.
// Returns error when any child expression fails to serialise.
//
//nolint:dupl // type-specific FlatBuffer encode/decode
func (s *encoder) buildTernaryExpr(expression *ast_domain.TernaryExpression) (flatbuffers.UOffsetT, error) {
	if expression == nil {
		return 0, nil
	}

	conditionOff, err := s.buildExpressionNode(expression.Condition)
	if err != nil {
		return 0, fmt.Errorf("serialise ternary expression condition: %w", err)
	}
	consequentOff, err := s.buildExpressionNode(expression.Consequent)
	if err != nil {
		return 0, fmt.Errorf("serialise ternary expression consequent: %w", err)
	}
	alternateOff, err := s.buildExpressionNode(expression.Alternate)
	if err != nil {
		return 0, fmt.Errorf("serialise ternary expression alternate: %w", err)
	}

	relativeLocOff, err := s.buildLocation(&expression.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise ternary expression location: %w", err)
	}

	ast_schema_gen.TernaryExprFBStart(s.builder)
	ast_schema_gen.TernaryExprFBAddCondition(s.builder, conditionOff)
	ast_schema_gen.TernaryExprFBAddConsequent(s.builder, consequentOff)
	ast_schema_gen.TernaryExprFBAddAlternate(s.builder, alternateOff)
	ast_schema_gen.TernaryExprFBAddRelativeLocation(s.builder, relativeLocOff)
	return ast_schema_gen.TernaryExprFBEnd(s.builder), nil
}

// buildTemplateLiteral serialises a template literal to flatbuffer format.
//
// Takes lit (*ast_domain.TemplateLiteral) which is the template literal to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when serialising parts or location fails.
func (s *encoder) buildTemplateLiteral(lit *ast_domain.TemplateLiteral) (flatbuffers.UOffsetT, error) {
	if lit == nil {
		return 0, nil
	}
	partsVec, err := buildVectorOfValues(s, lit.Parts, (*encoder).buildTemplateLiteralPart)
	if err != nil {
		return 0, fmt.Errorf("serialise template literal parts: %w", err)
	}
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise template literal location: %w", err)
	}

	ast_schema_gen.TemplateLiteralFBStart(s.builder)
	ast_schema_gen.TemplateLiteralFBAddParts(s.builder, partsVec)
	ast_schema_gen.TemplateLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.TemplateLiteralFBEnd(s.builder), nil
}

// buildTemplateLiteralPart converts a template literal part to FlatBuffers.
//
// Takes part (*ast_domain.TemplateLiteralPart) which is the part to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised part.
// Returns error when building the expression or location fails.
func (s *encoder) buildTemplateLiteralPart(part *ast_domain.TemplateLiteralPart) (flatbuffers.UOffsetT, error) {
	if part == nil {
		return 0, nil
	}
	literalOff := s.builder.CreateString(part.Literal)
	expressionOffset, err := s.buildExpressionNode(part.Expression)
	if err != nil {
		return 0, fmt.Errorf("serialise template literal part expression: %w", err)
	}
	locOff, err := s.buildLocation(&part.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise template literal part location: %w", err)
	}

	ast_schema_gen.TemplateLiteralPartFBStart(s.builder)
	ast_schema_gen.TemplateLiteralPartFBAddIsLiteral(s.builder, part.IsLiteral)
	ast_schema_gen.TemplateLiteralPartFBAddLiteral(s.builder, literalOff)
	ast_schema_gen.TemplateLiteralPartFBAddExpression(s.builder, expressionOffset)
	ast_schema_gen.TemplateLiteralPartFBAddLocation(s.builder, locOff)
	return ast_schema_gen.TemplateLiteralPartFBEnd(s.builder), nil
}

// expressionUnpacker is a function type for unpacking expression payloads from
// FlatBuffers.
type expressionUnpacker func(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error)

// unpackExpressionNode is the main decoder for the Expression union. It uses
// an array dispatch table to route to the appropriate unpacker function based
// on payload type.
//
// Takes fb (*ast_schema_gen.ExpressionNodeFB) which is the flatbuffer node to
// decode.
//
// Returns ast_domain.Expression which is the decoded expression, or nil if the
// input is nil or has no payload.
// Returns error when the payload type is not recognised.
func (d *decoder) unpackExpressionNode(fb *ast_schema_gen.ExpressionNodeFB) (ast_domain.Expression, error) {
	if fb == nil {
		return nil, nil
	}

	payloadTable := &d.tableFB
	if !fb.Payload(payloadTable) {
		return nil, nil
	}

	payloadType := fb.PayloadType()
	if int(payloadType) >= len(expressionUnpackers) {
		return nil, fmt.Errorf("unhandled expression payload type: %s", payloadType)
	}

	unpacker := expressionUnpackers[payloadType]
	if unpacker == nil {
		return nil, nil
	}

	return unpacker(d, payloadTable, int(fb.SourceLength()))
}

// unpackMemberExpr converts a FlatBuffer member expression to a domain model.
//
// Takes fb (*ast_schema_gen.MemberExprFB) which is the FlatBuffer to convert.
// Takes sourceLength (int) which specifies the source length for the result.
//
// Returns *ast_domain.MemberExpression which is the converted domain model, or nil
// if fb is nil.
// Returns error when unpacking the base, property, or location fails.
func (d *decoder) unpackMemberExpr(fb *ast_schema_gen.MemberExprFB, sourceLength int) (*ast_domain.MemberExpression, error) {
	if fb == nil {
		return nil, nil
	}
	expression := &ast_domain.MemberExpression{
		Optional:     fb.Optional(),
		Computed:     fb.Computed(),
		SourceLength: sourceLength,
	}
	var err error
	expression.Base, err = d.unpackExpressionNode(fb.Base(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack member expression base: %w", err)
	}
	expression.Property, err = d.unpackExpressionNode(fb.Property(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack member expression property: %w", err)
	}
	expression.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack member expression location: %w", err)
	}
	return expression, nil
}

// unpackIndexExpr converts a FlatBuffer IndexExpr into a domain IndexExpr.
//
// Takes fb (*ast_schema_gen.IndexExprFB) which is the FlatBuffer to convert.
// Takes sourceLength (int) which specifies the source code length.
//
// Returns *ast_domain.IndexExpression which is the converted domain object.
// Returns error when unpacking the base, index, or location fails.
func (d *decoder) unpackIndexExpr(fb *ast_schema_gen.IndexExprFB, sourceLength int) (*ast_domain.IndexExpression, error) {
	if fb == nil {
		return nil, nil
	}
	expression := &ast_domain.IndexExpression{
		Optional:     fb.Optional(),
		SourceLength: sourceLength,
	}
	var err error
	expression.Base, err = d.unpackExpressionNode(fb.Base(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack index expression base: %w", err)
	}
	expression.Index, err = d.unpackExpressionNode(fb.Index(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack index expression index: %w", err)
	}
	expression.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack index expression location: %w", err)
	}
	return expression, nil
}

// unpackUnaryExpr converts a FlatBuffer unary expression to a domain model.
//
// Takes fb (*ast_schema_gen.UnaryExprFB) which is the FlatBuffer to convert.
// Takes sourceLength (int) which sets the source length for the node.
//
// Returns *ast_domain.UnaryExpression which is the converted unary expression, or
// nil if fb is nil.
// Returns error when converting the operand or location fails.
func (d *decoder) unpackUnaryExpr(fb *ast_schema_gen.UnaryExprFB, sourceLength int) (*ast_domain.UnaryExpression, error) {
	if fb == nil {
		return nil, nil
	}
	expression := &ast_domain.UnaryExpression{
		Operator:     mapFBUnaryOpToGo[fb.Operator()],
		SourceLength: sourceLength,
	}
	var err error
	expression.Right, err = d.unpackExpressionNode(fb.Right(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack unary expression operand: %w", err)
	}
	expression.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack unary expression location: %w", err)
	}
	return expression, nil
}

// unpackBinaryExpr converts a FlatBuffer binary expression into a domain
// object.
//
// Takes fb (*ast_schema_gen.BinaryExprFB) which is the FlatBuffer binary
// expression to convert.
// Takes sourceLength (int) which specifies the length of the source text.
//
// Returns *ast_domain.BinaryExpression which is the converted domain
// object, or nil
// if fb is nil.
// Returns error when unpacking the left, right, or location fields fails.
func (d *decoder) unpackBinaryExpr(fb *ast_schema_gen.BinaryExprFB, sourceLength int) (*ast_domain.BinaryExpression, error) {
	if fb == nil {
		return nil, nil
	}
	expression := &ast_domain.BinaryExpression{
		Operator:     mapFBBinaryOpToGo[fb.Operator()],
		SourceLength: sourceLength,
	}
	var err error
	expression.Left, err = d.unpackExpressionNode(fb.Left(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack binary expression left operand: %w", err)
	}
	expression.Right, err = d.unpackExpressionNode(fb.Right(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack binary expression right operand: %w", err)
	}
	expression.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack binary expression location: %w", err)
	}
	return expression, nil
}

// unpackCallExpr converts a FlatBuffer CallExprFB into a domain CallExpr.
//
// Takes fb (*ast_schema_gen.CallExprFB) which is the serialised call
// expression.
// Takes sourceLength (int) which specifies the length in the original source.
//
// Returns *ast_domain.CallExpression which is the deserialised call expression, or
// nil if fb is nil.
// Returns error when deserialising the callee, arguments, or location fails.
func (d *decoder) unpackCallExpr(fb *ast_schema_gen.CallExprFB, sourceLength int) (*ast_domain.CallExpression, error) {
	if fb == nil {
		return nil, nil
	}
	expression := &ast_domain.CallExpression{
		SourceLength: sourceLength,
	}
	var err error
	expression.Callee, err = d.unpackExpressionNode(fb.Callee(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack call expression callee: %w", err)
	}
	expression.Args, err = unpackVector(d, fb.ArgumentsLength(), fb.Arguments, (*decoder).unpackExpressionNode)
	if err != nil {
		return nil, fmt.Errorf("unpack call expression arguments: %w", err)
	}
	expression.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack call expression location: %w", err)
	}
	expression.LparenLocation, err = d.unpackLocation(fb.LeftParenthesisLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack call expression lparen location: %w", err)
	}
	expression.RparenLocation, err = d.unpackLocation(fb.RightParenthesisLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack call expression rparen location: %w", err)
	}
	return expression, nil
}

// unpackForInExpr converts a FlatBuffer for-in expression to a domain object.
//
// Takes fb (*ast_schema_gen.ForInExprFB) which is the serialised expression.
// Takes sourceLength (int) which gives the length in the source text.
//
// Returns *ast_domain.ForInExpression which is the converted domain expression.
// Returns error when any nested element fails to unpack.
func (d *decoder) unpackForInExpr(fb *ast_schema_gen.ForInExprFB, sourceLength int) (*ast_domain.ForInExpression, error) {
	if fb == nil {
		return nil, nil
	}
	expression := &ast_domain.ForInExpression{
		SourceLength: sourceLength,
	}
	var err error
	expression.IndexVariable, err = d.unpackIdentifier(fb.IndexVariable(&d.identFB), 0)
	if err != nil {
		return nil, fmt.Errorf("unpack for-in expression index variable: %w", err)
	}
	expression.ItemVariable, err = d.unpackIdentifier(fb.ItemVariable(&d.identFB), 0)
	if err != nil {
		return nil, fmt.Errorf("unpack for-in expression item variable: %w", err)
	}
	expression.Collection, err = d.unpackExpressionNode(fb.Collection(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack for-in expression collection: %w", err)
	}
	expression.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack for-in expression location: %w", err)
	}
	return expression, nil
}

// unpackArrayLiteral converts a FlatBuffer array literal into a domain model.
//
// Takes fb (*ast_schema_gen.ArrayLiteralFB) which is the FlatBuffer to unpack.
// Takes sourceLength (int) which specifies the source text length.
//
// Returns *ast_domain.ArrayLiteral which is the converted domain model.
// Returns error when unpacking elements or location fails.
//
//nolint:dupl // type-specific FlatBuffer encode/decode
func (d *decoder) unpackArrayLiteral(fb *ast_schema_gen.ArrayLiteralFB, sourceLength int) (*ast_domain.ArrayLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	lit := &ast_domain.ArrayLiteral{
		SourceLength: sourceLength,
	}
	var err error
	lit.Elements, err = unpackVector(d, fb.ElementsLength(), fb.Elements, (*decoder).unpackExpressionNode)
	if err != nil {
		return nil, fmt.Errorf("unpack array literal elements: %w", err)
	}
	lit.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack array literal location: %w", err)
	}
	return lit, nil
}

// unpackObjectLiteral converts a FlatBuffer object literal into a domain model.
//
// Takes fb (*ast_schema_gen.ObjectLiteralFB) which is the FlatBuffer to unpack.
// Takes sourceLength (int) which is the length in the source code.
//
// Returns *ast_domain.ObjectLiteral which is the unpacked domain object.
// Returns error when unpacking pairs or location fails.
func (d *decoder) unpackObjectLiteral(fb *ast_schema_gen.ObjectLiteralFB, sourceLength int) (*ast_domain.ObjectLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	lit := &ast_domain.ObjectLiteral{
		SourceLength: sourceLength,
	}
	var err error
	lit.Pairs, err = d.unpackObjectLiteralPairs(fb)
	if err != nil {
		return nil, fmt.Errorf("unpack object literal pairs: %w", err)
	}
	lit.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack object literal location: %w", err)
	}
	return lit, nil
}

// unpackTernaryExpr converts a FlatBuffer ternary expression into a domain
// ternary expression.
//
// Takes fb (*ast_schema_gen.TernaryExprFB) which is the FlatBuffer source.
// Takes sourceLength (int) which specifies the length in the source code.
//
// Returns *ast_domain.TernaryExpression which is the converted domain expression.
// Returns error when any sub-expression fails to unpack.
func (d *decoder) unpackTernaryExpr(fb *ast_schema_gen.TernaryExprFB, sourceLength int) (*ast_domain.TernaryExpression, error) {
	if fb == nil {
		return nil, nil
	}
	expression := &ast_domain.TernaryExpression{
		SourceLength: sourceLength,
	}
	var err error
	expression.Condition, err = d.unpackExpressionNode(fb.Condition(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack ternary expression condition: %w", err)
	}
	expression.Consequent, err = d.unpackExpressionNode(fb.Consequent(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack ternary expression consequent: %w", err)
	}
	expression.Alternate, err = d.unpackExpressionNode(fb.Alternate(&d.expressionNodeFB))
	if err != nil {
		return nil, fmt.Errorf("unpack ternary expression alternate: %w", err)
	}
	expression.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack ternary expression location: %w", err)
	}
	return expression, nil
}

// unpackTemplateLiteral converts a FlatBuffer template literal into a domain
// object.
//
// Takes fb (*ast_schema_gen.TemplateLiteralFB) which is the FlatBuffer data to
// convert.
// Takes sourceLength (int) which is the length in the source code.
//
// Returns *ast_domain.TemplateLiteral which is the converted domain object, or
// nil if fb is nil.
// Returns error when unpacking parts or location fails.
//
//nolint:dupl // type-specific FlatBuffer encode/decode
func (d *decoder) unpackTemplateLiteral(fb *ast_schema_gen.TemplateLiteralFB, sourceLength int) (*ast_domain.TemplateLiteral, error) {
	if fb == nil {
		return nil, nil
	}

	result := &ast_domain.TemplateLiteral{
		SourceLength: sourceLength,
	}

	var err error
	result.Parts, err = unpackVector(d, fb.PartsLength(), fb.Parts, (*decoder).unpackTemplateLiteralPart)
	if err != nil {
		return nil, fmt.Errorf("unpack template literal parts: %w", err)
	}

	result.RelativeLocation, err = d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack template literal location: %w", err)
	}

	return result, nil
}

// unpackTemplateLiteralPart converts a FlatBuffer template literal part to a
// domain object.
//
// Takes fb (*ast_schema_gen.TemplateLiteralPartFB) which is the FlatBuffer
// data to convert.
//
// Returns ast_domain.TemplateLiteralPart which is the converted domain object.
// Returns error when unpacking the expression or location fails.
func (d *decoder) unpackTemplateLiteralPart(fb *ast_schema_gen.TemplateLiteralPartFB) (ast_domain.TemplateLiteralPart, error) {
	if fb == nil {
		return ast_domain.TemplateLiteralPart{}, nil
	}
	part := ast_domain.TemplateLiteralPart{
		IsLiteral: fb.IsLiteral(),
		Literal:   mem.String(fb.Literal()),
	}
	var err error
	part.Expression, err = d.unpackExpressionNode(fb.Expression(&d.expressionNodeFB))
	if err != nil {
		return part, fmt.Errorf("unpack template literal part expression: %w", err)
	}
	part.RelativeLocation, err = d.unpackLocation(fb.Location(&d.locFB))
	if err != nil {
		return part, fmt.Errorf("unpack template literal part location: %w", err)
	}
	return part, nil
}

// buildExpressionVector builds a FlatBuffers vector from a slice of
// Expression interfaces.
//
// Takes items ([]ast_domain.Expression) which contains the expressions to
// serialise into the vector.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector.
// Returns error when any expression in the slice fails to build.
func (s *encoder) buildExpressionVector(items []ast_domain.Expression) (flatbuffers.UOffsetT, error) {
	if len(items) == 0 {
		return 0, nil
	}
	offsets := make([]flatbuffers.UOffsetT, len(items))
	for i := range items {
		off, err := s.buildExpressionNode(items[i])
		if err != nil {
			return 0, fmt.Errorf("failed to build expression item at index %d: %w", i, err)
		}
		offsets[i] = off
	}
	return createVector(s, offsets), nil
}

// unpackMemberExprPayload extracts a member expression from a flatbuffer table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the source code length for
// validation.
//
// Returns ast_domain.Expression which is the unpacked member expression.
// Returns error when the member expression cannot be decoded.
func unpackMemberExprPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.MemberExprFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackMemberExpr(&flatbuffer, sourceLength)
}

// unpackIndexExprPayload converts a FlatBuffer table into an index expression.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised index
// expression.
// Takes sourceLength (int) which specifies the source code length for
// validation.
//
// Returns ast_domain.Expression which is the unpacked index expression.
// Returns error when the index expression cannot be unpacked.
func unpackIndexExprPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.IndexExprFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackIndexExpr(&flatbuffer, sourceLength)
}

// unpackUnaryExprPayload extracts a unary expression from a flatbuffer table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised expression.
// Takes sourceLength (int) which specifies the source text length for
// validation.
//
// Returns ast_domain.Expression which is the decoded unary expression.
// Returns error when the expression cannot be unpacked.
func unpackUnaryExprPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.UnaryExprFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackUnaryExpr(&flatbuffer, sourceLength)
}

// unpackBinaryExprPayload unpacks a binary expression from a FlatBuffers table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the source text length for
// validation.
//
// Returns ast_domain.Expression which is the unpacked binary expression.
// Returns error when the binary expression cannot be unpacked.
func unpackBinaryExprPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.BinaryExprFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackBinaryExpr(&flatbuffer, sourceLength)
}

// unpackCallExprPayload extracts a call expression from a FlatBuffer table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised call
// expression.
// Takes sourceLength (int) which specifies the source text length for
// validation.
//
// Returns ast_domain.Expression which is the unpacked call expression.
// Returns error when the call expression cannot be unpacked.
func unpackCallExprPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.CallExprFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackCallExpr(&flatbuffer, sourceLength)
}

// unpackForInExprPayload unpacks a for-in expression from a flatbuffer table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised expression
// data.
// Takes sourceLength (int) which specifies the length of the source text.
//
// Returns ast_domain.Expression which is the unpacked for-in expression.
// Returns error when unpacking fails.
func unpackForInExprPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.ForInExprFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackForInExpr(&flatbuffer, sourceLength)
}

// unpackArrayLiteralPayload decodes an array literal from flatbuffer format.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised array literal.
// Takes sourceLength (int) which specifies the source text length.
//
// Returns ast_domain.Expression which is the decoded array literal.
// Returns error when decoding fails.
func unpackArrayLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.ArrayLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackArrayLiteral(&flatbuffer, sourceLength)
}

// unpackObjectLiteralPayload unpacks an object literal from a flatbuffer table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised object data.
// Takes sourceLength (int) which specifies the source length for validation.
//
// Returns ast_domain.Expression which is the unpacked object literal.
// Returns error when unpacking fails.
func unpackObjectLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.ObjectLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackObjectLiteral(&flatbuffer, sourceLength)
}

// unpackTernaryExprPayload extracts a ternary expression from a FlatBuffers
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised expression.
// Takes sourceLength (int) which specifies the source text length for
// validation.
//
// Returns ast_domain.Expression which is the unpacked ternary expression.
// Returns error when the expression cannot be decoded.
func unpackTernaryExprPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.TernaryExprFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackTernaryExpr(&flatbuffer, sourceLength)
}

// unpackTemplateLiteralPayload unpacks a template literal from a FlatBuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the original source length.
//
// Returns ast_domain.Expression which is the unpacked template literal.
// Returns error when unpacking fails.
func unpackTemplateLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.TemplateLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackTemplateLiteral(&flatbuffer, sourceLength)
}

// unpackIdentifierPayload extracts an identifier expression from a flatbuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised identifier.
// Takes sourceLength (int) which specifies the source code length for
// validation.
//
// Returns ast_domain.Expression which is the unpacked identifier expression.
// Returns error when the identifier cannot be unpacked.
func unpackIdentifierPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.IdentifierFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackIdentifier(&flatbuffer, sourceLength)
}

// unpackStringLiteralPayload extracts a string literal from a FlatBuffer table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the original source length.
//
// Returns ast_domain.Expression which is the unpacked string literal.
// Returns error when the string literal cannot be unpacked.
func unpackStringLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.StringLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackStringLiteral(&flatbuffer, sourceLength)
}

// unpackIntegerLiteralPayload extracts an integer literal from a flatbuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the source text length.
//
// Returns ast_domain.Expression which is the unpacked integer literal.
// Returns error when the integer literal cannot be unpacked.
func unpackIntegerLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.IntegerLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackIntegerLiteral(&flatbuffer, sourceLength)
}

// unpackFloatLiteralPayload extracts a float literal expression from a
// flatbuffer table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised float data.
// Takes sourceLength (int) which specifies the source text length.
//
// Returns ast_domain.Expression which is the decoded float literal.
// Returns error when decoding fails.
func unpackFloatLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.FloatLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackFloatLiteral(&flatbuffer, sourceLength)
}

// unpackBooleanLiteralPayload extracts a boolean literal from a FlatBuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised boolean d
// ata.
// Takes sourceLength (int) which specifies the source text length for
// validation.
//
// Returns ast_domain.Expression which is the decoded boolean literal.
// Returns error when the boolean literal cannot be unpacked.
func unpackBooleanLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.BooleanLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackBooleanLiteral(&flatbuffer, sourceLength)
}

// unpackNilLiteralPayload extracts a nil literal expression from the flatbuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised nil literal.
// Takes sourceLength (int) which specifies the source text length.
//
// Returns ast_domain.Expression which is the unpacked nil literal.
// Returns error when the nil literal cannot be unpacked.
func unpackNilLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.NilLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackNilLiteral(&flatbuffer, sourceLength)
}

// unpackDecimalLiteralPayload unpacks a decimal literal from a flatbuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the length of the source.
//
// Returns ast_domain.Expression which is the unpacked decimal literal.
// Returns error when unpacking fails.
func unpackDecimalLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.DecimalLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackDecimalLiteral(&flatbuffer, sourceLength)
}

// unpackBigIntLiteralPayload unpacks a big integer literal from the flatbuffer.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the source text length.
//
// Returns ast_domain.Expression which is the unpacked big integer literal.
// Returns error when unpacking fails.
func unpackBigIntLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.BigIntLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackBigIntLiteral(&flatbuffer, sourceLength)
}

// unpackRuneLiteralPayload extracts a rune literal expression from a
// FlatBuffers table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the original source length.
//
// Returns ast_domain.Expression which is the unpacked rune literal.
// Returns error when the rune literal cannot be unpacked.
func unpackRuneLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.RuneLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackRuneLiteral(&flatbuffer, sourceLength)
}

// unpackDateTimeLiteralPayload extracts a date-time literal from the flatbuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised literal data.
// Takes sourceLength (int) which specifies the original source length.
//
// Returns ast_domain.Expression which is the unpacked date-time literal.
// Returns error when unpacking fails.
func unpackDateTimeLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.DateTimeLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackDateTimeLiteral(&flatbuffer, sourceLength)
}

// unpackDateLiteralPayload extracts a date literal expression from a flatbuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised date literal.
// Takes sourceLength (int) which specifies the original source length.
//
// Returns ast_domain.Expression which is the unpacked date literal expression.
// Returns error when the date literal cannot be unpacked.
func unpackDateLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.DateLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackDateLiteral(&flatbuffer, sourceLength)
}

// unpackTimeLiteralPayload extracts a time literal expression from a
// flatbuffer table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised time literal.
// Takes sourceLength (int) which specifies the original source length.
//
// Returns ast_domain.Expression which is the decoded time literal.
// Returns error when the time literal cannot be unpacked.
func unpackTimeLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.TimeLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackTimeLiteral(&flatbuffer, sourceLength)
}

// unpackDurationLiteralPayload extracts a duration literal from a flatbuffer
// table.
//
// Takes d (*decoder) which provides the decoding context.
// Takes table (*flatbuffers.Table) which contains the serialised data.
// Takes sourceLength (int) which specifies the original source length.
//
// Returns ast_domain.Expression which is the unpacked duration literal.
// Returns error when the duration literal cannot be unpacked.
func unpackDurationLiteralPayload(d *decoder, table *flatbuffers.Table, sourceLength int) (ast_domain.Expression, error) {
	var flatbuffer ast_schema_gen.DurationLiteralFB
	flatbuffer.Init(table.Bytes, table.Pos)
	return d.unpackDurationLiteral(&flatbuffer, sourceLength)
}

func init() {
	for k, v := range mapGoBinaryOpToFB {
		mapFBBinaryOpToGo[v] = k
	}

	expressionEncoders = buildExpressionEncoders()
	expressionUnpackers = buildExpressionUnpackers()
}

// exprEnc is a shorthand constructor for expressionEncoderEntry.
//
// Takes pt (ast_schema_gen.ExpressionPayloadFB) which is the
// payload type discriminator.
// Takes fn (func) which is the encoder function for the type.
//
// Returns expressionEncoderEntry which is the constructed entry.
func exprEnc(
	pt ast_schema_gen.ExpressionPayloadFB,
	fn func(*encoder, ast_domain.Expression) (flatbuffers.UOffsetT, error),
) expressionEncoderEntry {
	return expressionEncoderEntry{payloadType: pt, build: fn}
}

// buildExpressionEncoders constructs the type-to-encoder dispatch
// map used by buildExpressionNode for table-driven encoding.
//
// Returns map[reflect.Type]expressionEncoderEntry which is the
// combined encoder dispatch map.
func buildExpressionEncoders() map[reflect.Type]expressionEncoderEntry {
	m := buildCompoundExpressionEncoders()
	maps.Copy(m, buildLiteralExpressionEncoders())
	return m
}

// buildCompoundExpressionEncoders returns encoders for compound
// expression types such as member access, index, unary, binary,
// call, for-in, array, object, ternary, template, and identifier.
//
// Returns map[reflect.Type]expressionEncoderEntry which is the
// compound expression encoder map.
func buildCompoundExpressionEncoders() map[reflect.Type]expressionEncoderEntry {
	return map[reflect.Type]expressionEncoderEntry{
		reflect.TypeFor[*ast_domain.MemberExpression](): exprEnc(ast_schema_gen.ExpressionPayloadFBMemberExprFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildMemberExpr(e.(*ast_domain.MemberExpression))
		}),
		reflect.TypeFor[*ast_domain.IndexExpression](): exprEnc(ast_schema_gen.ExpressionPayloadFBIndexExprFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildIndexExpr(e.(*ast_domain.IndexExpression))
		}),
		reflect.TypeFor[*ast_domain.UnaryExpression](): exprEnc(ast_schema_gen.ExpressionPayloadFBUnaryExprFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildUnaryExpr(e.(*ast_domain.UnaryExpression))
		}),
		reflect.TypeFor[*ast_domain.BinaryExpression](): exprEnc(ast_schema_gen.ExpressionPayloadFBBinaryExprFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildBinaryExpr(e.(*ast_domain.BinaryExpression))
		}),
		reflect.TypeFor[*ast_domain.CallExpression](): exprEnc(ast_schema_gen.ExpressionPayloadFBCallExprFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildCallExpr(e.(*ast_domain.CallExpression))
		}),
		reflect.TypeFor[*ast_domain.ForInExpression](): exprEnc(ast_schema_gen.ExpressionPayloadFBForInExprFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildForInExpr(e.(*ast_domain.ForInExpression))
		}),
		reflect.TypeFor[*ast_domain.ArrayLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBArrayLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildArrayLiteral(e.(*ast_domain.ArrayLiteral))
		}),
		reflect.TypeFor[*ast_domain.ObjectLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBObjectLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildObjectLiteral(e.(*ast_domain.ObjectLiteral))
		}),
		reflect.TypeFor[*ast_domain.TernaryExpression](): exprEnc(ast_schema_gen.ExpressionPayloadFBTernaryExprFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildTernaryExpr(e.(*ast_domain.TernaryExpression))
		}),
		reflect.TypeFor[*ast_domain.TemplateLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBTemplateLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildTemplateLiteral(e.(*ast_domain.TemplateLiteral))
		}),
		reflect.TypeFor[*ast_domain.Identifier](): exprEnc(ast_schema_gen.ExpressionPayloadFBIdentifierFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildIdentifier(e.(*ast_domain.Identifier))
		}),
	}
}

// buildLiteralExpressionEncoders returns encoders for scalar
// literal expression types such as string, integer, float,
// boolean, nil, decimal, bigint, rune, and temporal literals.
//
// Returns map[reflect.Type]expressionEncoderEntry which is the
// literal expression encoder map.
func buildLiteralExpressionEncoders() map[reflect.Type]expressionEncoderEntry {
	return map[reflect.Type]expressionEncoderEntry{
		reflect.TypeFor[*ast_domain.StringLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBStringLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildStringLiteral(e.(*ast_domain.StringLiteral))
		}),
		reflect.TypeFor[*ast_domain.IntegerLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBIntegerLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildIntegerLiteral(e.(*ast_domain.IntegerLiteral))
		}),
		reflect.TypeFor[*ast_domain.FloatLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBFloatLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildFloatLiteral(e.(*ast_domain.FloatLiteral))
		}),
		reflect.TypeFor[*ast_domain.BooleanLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBBooleanLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildBooleanLiteral(e.(*ast_domain.BooleanLiteral))
		}),
		reflect.TypeFor[*ast_domain.NilLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBNilLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildNilLiteral(e.(*ast_domain.NilLiteral))
		}),
		reflect.TypeFor[*ast_domain.DecimalLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBDecimalLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildDecimalLiteral(e.(*ast_domain.DecimalLiteral))
		}),
		reflect.TypeFor[*ast_domain.BigIntLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBBigIntLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildBigIntLiteral(e.(*ast_domain.BigIntLiteral))
		}),
		reflect.TypeFor[*ast_domain.RuneLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBRuneLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildRuneLiteral(e.(*ast_domain.RuneLiteral))
		}),
		reflect.TypeFor[*ast_domain.DateTimeLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBDateTimeLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildDateTimeLiteral(e.(*ast_domain.DateTimeLiteral))
		}),
		reflect.TypeFor[*ast_domain.DateLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBDateLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildDateLiteral(e.(*ast_domain.DateLiteral))
		}),
		reflect.TypeFor[*ast_domain.TimeLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBTimeLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildTimeLiteral(e.(*ast_domain.TimeLiteral))
		}),
		reflect.TypeFor[*ast_domain.DurationLiteral](): exprEnc(ast_schema_gen.ExpressionPayloadFBDurationLiteralFB, func(s *encoder, e ast_domain.Expression) (flatbuffers.UOffsetT, error) {
			return s.buildDurationLiteral(e.(*ast_domain.DurationLiteral))
		}),
	}
}

// buildExpressionUnpackers constructs the array dispatch table
// mapping ExpressionPayloadFB enum values to unpacker functions.
//
// Returns *[expressionPayloadCount]expressionUnpacker which is the
// populated dispatch table.
func buildExpressionUnpackers() *[expressionPayloadCount]expressionUnpacker {
	return &[expressionPayloadCount]expressionUnpacker{
		ast_schema_gen.ExpressionPayloadFBNONE:              nil,
		ast_schema_gen.ExpressionPayloadFBIdentifierFB:      unpackIdentifierPayload,
		ast_schema_gen.ExpressionPayloadFBStringLiteralFB:   unpackStringLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBIntegerLiteralFB:  unpackIntegerLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBFloatLiteralFB:    unpackFloatLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBBooleanLiteralFB:  unpackBooleanLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBNilLiteralFB:      unpackNilLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBMemberExprFB:      unpackMemberExprPayload,
		ast_schema_gen.ExpressionPayloadFBIndexExprFB:       unpackIndexExprPayload,
		ast_schema_gen.ExpressionPayloadFBUnaryExprFB:       unpackUnaryExprPayload,
		ast_schema_gen.ExpressionPayloadFBBinaryExprFB:      unpackBinaryExprPayload,
		ast_schema_gen.ExpressionPayloadFBCallExprFB:        unpackCallExprPayload,
		ast_schema_gen.ExpressionPayloadFBForInExprFB:       unpackForInExprPayload,
		ast_schema_gen.ExpressionPayloadFBArrayLiteralFB:    unpackArrayLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBObjectLiteralFB:   unpackObjectLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBTernaryExprFB:     unpackTernaryExprPayload,
		ast_schema_gen.ExpressionPayloadFBTemplateLiteralFB: unpackTemplateLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBDecimalLiteralFB:  unpackDecimalLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBDateTimeLiteralFB: unpackDateTimeLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBDateLiteralFB:     unpackDateLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBTimeLiteralFB:     unpackTimeLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBDurationLiteralFB: unpackDurationLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBRuneLiteralFB:     unpackRuneLiteralPayload,
		ast_schema_gen.ExpressionPayloadFBBigIntLiteralFB:   unpackBigIntLiteralPayload,
	}
}
