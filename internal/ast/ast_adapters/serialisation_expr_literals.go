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

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/ast/ast_schema/ast_schema_gen"
	"piko.sh/piko/internal/mem"
)

// buildIdentifier converts an identifier to its FlatBuffer form.
//
// Takes identifier (*ast_domain.Identifier) which is the identifier to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the stored data.
// Returns error when the location cannot be built.
func (s *encoder) buildIdentifier(identifier *ast_domain.Identifier) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if identifier == nil {
		return 0, nil
	}
	nameOff := s.builder.CreateString(identifier.Name)
	locOff, err := s.buildLocation(&identifier.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise identifier location: %w", err)
	}

	ast_schema_gen.IdentifierFBStart(s.builder)
	ast_schema_gen.IdentifierFBAddName(s.builder, nameOff)
	ast_schema_gen.IdentifierFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.IdentifierFBEnd(s.builder), nil
}

// buildStringLiteral serialises a string literal into the FlatBuffer.
//
// Takes lit (*ast_domain.StringLiteral) which is the string literal to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when the location cannot be built.
func (s *encoder) buildStringLiteral(lit *ast_domain.StringLiteral) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if lit == nil {
		return 0, nil
	}
	valueOff := s.builder.CreateString(lit.Value)
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise string literal location: %w", err)
	}

	ast_schema_gen.StringLiteralFBStart(s.builder)
	ast_schema_gen.StringLiteralFBAddValue(s.builder, valueOff)
	ast_schema_gen.StringLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.StringLiteralFBEnd(s.builder), nil
}

// buildIntegerLiteral serialises an integer literal to FlatBuffers format.
//
// Takes lit (*ast_domain.IntegerLiteral) which is the literal to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when the location cannot be built.
func (s *encoder) buildIntegerLiteral(lit *ast_domain.IntegerLiteral) (flatbuffers.UOffsetT, error) {
	if lit == nil {
		return 0, nil
	}
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise integer literal location: %w", err)
	}

	ast_schema_gen.IntegerLiteralFBStart(s.builder)
	ast_schema_gen.IntegerLiteralFBAddValue(s.builder, lit.Value)
	ast_schema_gen.IntegerLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.IntegerLiteralFBEnd(s.builder), nil
}

// buildFloatLiteral serialises a float literal to the flatbuffer format.
//
// Takes lit (*ast_domain.FloatLiteral) which is the float literal to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when the location cannot be built.
func (s *encoder) buildFloatLiteral(lit *ast_domain.FloatLiteral) (flatbuffers.UOffsetT, error) {
	if lit == nil {
		return 0, nil
	}
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise float literal location: %w", err)
	}

	ast_schema_gen.FloatLiteralFBStart(s.builder)
	ast_schema_gen.FloatLiteralFBAddValue(s.builder, lit.Value)
	ast_schema_gen.FloatLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.FloatLiteralFBEnd(s.builder), nil
}

// buildBooleanLiteral converts a boolean literal AST node to FlatBuffer format.
//
// Takes lit (*ast_domain.BooleanLiteral) which is the boolean literal to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when location serialisation fails.
func (s *encoder) buildBooleanLiteral(lit *ast_domain.BooleanLiteral) (flatbuffers.UOffsetT, error) {
	if lit == nil {
		return 0, nil
	}
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise boolean literal location: %w", err)
	}

	ast_schema_gen.BooleanLiteralFBStart(s.builder)
	ast_schema_gen.BooleanLiteralFBAddValue(s.builder, lit.Value)
	ast_schema_gen.BooleanLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.BooleanLiteralFBEnd(s.builder), nil
}

// buildNilLiteral serialises a nil literal AST node to FlatBuffers format.
//
// Takes lit (*ast_domain.NilLiteral) which is the nil literal node to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised node.
// Returns error when building the location fails.
func (s *encoder) buildNilLiteral(lit *ast_domain.NilLiteral) (flatbuffers.UOffsetT, error) {
	if lit == nil {
		return 0, nil
	}
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise nil literal location: %w", err)
	}

	ast_schema_gen.NilLiteralFBStart(s.builder)
	ast_schema_gen.NilLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.NilLiteralFBEnd(s.builder), nil
}

// buildDecimalLiteral serialises a decimal literal to its flatbuffer form.
//
// Takes lit (*ast_domain.DecimalLiteral) which is the decimal literal to
// serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when location serialisation fails.
func (s *encoder) buildDecimalLiteral(lit *ast_domain.DecimalLiteral) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if lit == nil {
		return 0, nil
	}
	valueOff := s.builder.CreateString(lit.Value)
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise decimal literal location: %w", err)
	}

	ast_schema_gen.DecimalLiteralFBStart(s.builder)
	ast_schema_gen.DecimalLiteralFBAddValue(s.builder, valueOff)
	ast_schema_gen.DecimalLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.DecimalLiteralFBEnd(s.builder), nil
}

// buildBigIntLiteral serialises a big integer literal to FlatBuffers format.
//
// Takes lit (*ast_domain.BigIntLiteral) which is the literal to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when the location cannot be built.
func (s *encoder) buildBigIntLiteral(lit *ast_domain.BigIntLiteral) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if lit == nil {
		return 0, nil
	}
	valueOff := s.builder.CreateString(lit.Value)
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise big integer literal location: %w", err)
	}

	ast_schema_gen.BigIntLiteralFBStart(s.builder)
	ast_schema_gen.BigIntLiteralFBAddValue(s.builder, valueOff)
	ast_schema_gen.BigIntLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.BigIntLiteralFBEnd(s.builder), nil
}

// buildRuneLiteral serialises a rune literal AST node to flatbuffer format.
//
// Takes lit (*ast_domain.RuneLiteral) which is the rune literal to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised node.
// Returns error when the location cannot be built.
func (s *encoder) buildRuneLiteral(lit *ast_domain.RuneLiteral) (flatbuffers.UOffsetT, error) {
	if lit == nil {
		return 0, nil
	}
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise rune literal location: %w", err)
	}

	ast_schema_gen.RuneLiteralFBStart(s.builder)
	ast_schema_gen.RuneLiteralFBAddValue(s.builder, int32(lit.Value))
	ast_schema_gen.RuneLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.RuneLiteralFBEnd(s.builder), nil
}

// buildDateTimeLiteral serialises a date-time literal to flatbuffer format.
//
// Takes lit (*ast_domain.DateTimeLiteral) which is the literal to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when building the location fails.
func (s *encoder) buildDateTimeLiteral(lit *ast_domain.DateTimeLiteral) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if lit == nil {
		return 0, nil
	}
	valueOff := s.builder.CreateString(lit.Value)
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise date-time literal location: %w", err)
	}

	ast_schema_gen.DateTimeLiteralFBStart(s.builder)
	ast_schema_gen.DateTimeLiteralFBAddValue(s.builder, valueOff)
	ast_schema_gen.DateTimeLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.DateTimeLiteralFBEnd(s.builder), nil
}

// buildDateLiteral serialises a date literal to the flatbuffer format.
//
// Takes lit (*ast_domain.DateLiteral) which is the date literal to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised literal.
// Returns error when building the location fails.
func (s *encoder) buildDateLiteral(lit *ast_domain.DateLiteral) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if lit == nil {
		return 0, nil
	}
	valueOff := s.builder.CreateString(lit.Value)
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise date literal location: %w", err)
	}

	ast_schema_gen.DateLiteralFBStart(s.builder)
	ast_schema_gen.DateLiteralFBAddValue(s.builder, valueOff)
	ast_schema_gen.DateLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.DateLiteralFBEnd(s.builder), nil
}

// buildTimeLiteral converts a time literal AST node to FlatBuffers format.
//
// Takes lit (*ast_domain.TimeLiteral) which is the time literal to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the converted node.
// Returns error when the location cannot be built.
func (s *encoder) buildTimeLiteral(lit *ast_domain.TimeLiteral) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if lit == nil {
		return 0, nil
	}
	valueOff := s.builder.CreateString(lit.Value)
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise time literal location: %w", err)
	}

	ast_schema_gen.TimeLiteralFBStart(s.builder)
	ast_schema_gen.TimeLiteralFBAddValue(s.builder, valueOff)
	ast_schema_gen.TimeLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.TimeLiteralFBEnd(s.builder), nil
}

// buildDurationLiteral writes a duration literal to the flatbuffer.
//
// Takes lit (*ast_domain.DurationLiteral) which is the duration literal to
// write.
//
// Returns flatbuffers.UOffsetT which is the offset of the written literal.
// Returns error when the location cannot be built.
func (s *encoder) buildDurationLiteral(lit *ast_domain.DurationLiteral) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if lit == nil {
		return 0, nil
	}
	valueOff := s.builder.CreateString(lit.Value)
	locOff, err := s.buildLocation(&lit.RelativeLocation)
	if err != nil {
		return 0, fmt.Errorf("serialise duration literal location: %w", err)
	}

	ast_schema_gen.DurationLiteralFBStart(s.builder)
	ast_schema_gen.DurationLiteralFBAddValue(s.builder, valueOff)
	ast_schema_gen.DurationLiteralFBAddRelativeLocation(s.builder, locOff)
	return ast_schema_gen.DurationLiteralFBEnd(s.builder), nil
}

// unpackIdentifier converts a FlatBuffer identifier to a domain identifier.
//
// Takes fb (*ast_schema_gen.IdentifierFB) which is the serialised identifier.
// Takes sourceLength (int) which specifies the length in the source text.
//
// Returns *ast_domain.Identifier which is the converted domain object, or nil
// if fb is nil.
// Returns error when location unpacking fails.
func (d *decoder) unpackIdentifier(fb *ast_schema_gen.IdentifierFB, sourceLength int) (*ast_domain.Identifier, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack identifier location: %w", err)
	}
	return &ast_domain.Identifier{
		Name:             mem.String(fb.Name()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackStringLiteral converts a FlatBuffer string literal to a domain object.
//
// Takes fb (*ast_schema_gen.StringLiteralFB) which is the FlatBuffer to unpack.
// Takes sourceLength (int) which is the length of the source representation.
//
// Returns *ast_domain.StringLiteral which is the domain object, or nil if fb
// is nil.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackStringLiteral(fb *ast_schema_gen.StringLiteralFB, sourceLength int) (*ast_domain.StringLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack string literal location: %w", err)
	}
	return &ast_domain.StringLiteral{
		Value:            mem.String(fb.Value()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackIntegerLiteral converts a FlatBuffer integer literal to its domain
// representation.
//
// Takes fb (*ast_schema_gen.IntegerLiteralFB) which is the FlatBuffer to
// convert.
// Takes sourceLength (int) which specifies the length in the source code.
//
// Returns *ast_domain.IntegerLiteral which is the domain representation, or
// nil if fb is nil.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackIntegerLiteral(fb *ast_schema_gen.IntegerLiteralFB, sourceLength int) (*ast_domain.IntegerLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack integer literal location: %w", err)
	}
	return &ast_domain.IntegerLiteral{
		Value:            fb.Value(),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackFloatLiteral converts a FlatBuffer float literal to a domain model.
//
// Takes fb (*ast_schema_gen.FloatLiteralFB) which is the FlatBuffer to unpack.
// Takes sourceLength (int) which is the length of the source representation.
//
// Returns *ast_domain.FloatLiteral which is the domain model, or nil if fb is
// nil.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackFloatLiteral(fb *ast_schema_gen.FloatLiteralFB, sourceLength int) (*ast_domain.FloatLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack float literal location: %w", err)
	}
	return &ast_domain.FloatLiteral{
		Value:            fb.Value(),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackBooleanLiteral converts a FlatBuffer boolean literal to a domain
// boolean literal.
//
// Takes fb (*ast_schema_gen.BooleanLiteralFB) which is the FlatBuffer to
// convert.
// Takes sourceLength (int) which specifies the length in the source code.
//
// Returns *ast_domain.BooleanLiteral which is the domain representation.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackBooleanLiteral(fb *ast_schema_gen.BooleanLiteralFB, sourceLength int) (*ast_domain.BooleanLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack boolean literal location: %w", err)
	}
	return &ast_domain.BooleanLiteral{
		Value:            fb.Value(),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackNilLiteral converts a FlatBuffer nil literal to a domain nil literal.
//
// Takes fb (*ast_schema_gen.NilLiteralFB) which is the FlatBuffer to convert.
// Takes sourceLength (int) which is the length in the source code.
//
// Returns *ast_domain.NilLiteral which is the converted domain object.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackNilLiteral(fb *ast_schema_gen.NilLiteralFB, sourceLength int) (*ast_domain.NilLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack nil literal location: %w", err)
	}
	return &ast_domain.NilLiteral{
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackDecimalLiteral converts a FlatBuffer decimal literal to a domain
// object.
//
// Takes fb (*ast_schema_gen.DecimalLiteralFB) which is the FlatBuffer to
// convert.
// Takes sourceLength (int) which specifies the length in the source code.
//
// Returns *ast_domain.DecimalLiteral which is the converted domain object, or
// nil if fb is nil.
// Returns error when location unpacking fails.
func (d *decoder) unpackDecimalLiteral(fb *ast_schema_gen.DecimalLiteralFB, sourceLength int) (*ast_domain.DecimalLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack decimal literal location: %w", err)
	}
	return &ast_domain.DecimalLiteral{
		Value:            mem.String(fb.Value()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackBigIntLiteral converts a FlatBuffer big integer literal to its domain
// form.
//
// Takes fb (*ast_schema_gen.BigIntLiteralFB) which is the FlatBuffer to unpack.
// Takes sourceLength (int) which is the length of the source text.
//
// Returns *ast_domain.BigIntLiteral which is the domain object, or nil if fb
// is nil.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackBigIntLiteral(fb *ast_schema_gen.BigIntLiteralFB, sourceLength int) (*ast_domain.BigIntLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack big integer literal location: %w", err)
	}
	return &ast_domain.BigIntLiteral{
		Value:            mem.String(fb.Value()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackRuneLiteral converts a FlatBuffer rune literal to its domain model.
//
// Takes fb (*ast_schema_gen.RuneLiteralFB) which is the FlatBuffer to convert.
// Takes sourceLength (int) which specifies the length in the source code.
//
// Returns *ast_domain.RuneLiteral which is the domain model, or nil if fb is
// nil.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackRuneLiteral(fb *ast_schema_gen.RuneLiteralFB, sourceLength int) (*ast_domain.RuneLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack rune literal location: %w", err)
	}
	return &ast_domain.RuneLiteral{
		Value:            rune(fb.Value()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackDateTimeLiteral converts a FlatBuffer date-time literal to its domain
// form.
//
// Takes fb (*ast_schema_gen.DateTimeLiteralFB) which is the FlatBuffer to
// convert.
// Takes sourceLength (int) which specifies the length in the source text.
//
// Returns *ast_domain.DateTimeLiteral which is the domain form, or nil if fb
// is nil.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackDateTimeLiteral(fb *ast_schema_gen.DateTimeLiteralFB, sourceLength int) (*ast_domain.DateTimeLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack date-time literal location: %w", err)
	}
	return &ast_domain.DateTimeLiteral{
		Value:            mem.String(fb.Value()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackDateLiteral converts a FlatBuffer date literal to a domain date
// literal.
//
// Takes fb (*ast_schema_gen.DateLiteralFB) which is the FlatBuffer
// representation to convert.
// Takes sourceLength (int) which specifies the length in the original source.
//
// Returns *ast_domain.DateLiteral which is the domain representation.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackDateLiteral(fb *ast_schema_gen.DateLiteralFB, sourceLength int) (*ast_domain.DateLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack date literal location: %w", err)
	}
	return &ast_domain.DateLiteral{
		Value:            mem.String(fb.Value()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackTimeLiteral converts a FlatBuffer time literal to a domain object.
//
// Takes fb (*ast_schema_gen.TimeLiteralFB) which is the FlatBuffer to convert.
// Takes sourceLength (int) which specifies the length in the source code.
//
// Returns *ast_domain.TimeLiteral which is the converted domain object, or nil
// if fb is nil.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackTimeLiteral(fb *ast_schema_gen.TimeLiteralFB, sourceLength int) (*ast_domain.TimeLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack time literal location: %w", err)
	}
	return &ast_domain.TimeLiteral{
		Value:            mem.String(fb.Value()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}

// unpackDurationLiteral converts a FlatBuffer duration literal to domain form.
//
// Takes fb (*ast_schema_gen.DurationLiteralFB) which is the serialised duration
// literal to convert.
// Takes sourceLength (int) which specifies the length in the source code.
//
// Returns *ast_domain.DurationLiteral which is the converted domain object, or
// nil if fb is nil.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackDurationLiteral(fb *ast_schema_gen.DurationLiteralFB, sourceLength int) (*ast_domain.DurationLiteral, error) {
	if fb == nil {
		return nil, nil
	}
	location, err := d.unpackLocation(fb.RelativeLocation(&d.locFB))
	if err != nil {
		return nil, fmt.Errorf("unpack duration literal location: %w", err)
	}
	return &ast_domain.DurationLiteral{
		Value:            mem.String(fb.Value()),
		RelativeLocation: location,
		SourceLength:     sourceLength,
	}, nil
}
