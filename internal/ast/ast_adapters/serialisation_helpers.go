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
	"piko.sh/piko/wdk/safeconv"
)

// uoffsetTSize is the size in bytes of a FlatBuffers UOffsetT.
const uoffsetTSize = 4

// buildLocation creates a FlatBuffer representation of a source location.
//
// Takes location (*ast_domain.Location) which specifies the position to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the built location.
// Returns error when the conversion fails.
func (s *encoder) buildLocation(location *ast_domain.Location) (flatbuffers.UOffsetT, error) {
	if location == nil {
		return 0, nil
	}
	ast_schema_gen.LocationFBStart(s.builder)
	ast_schema_gen.LocationFBAddLine(s.builder, safeconv.IntToInt32(location.Line))
	ast_schema_gen.LocationFBAddColumn(s.builder, safeconv.IntToInt32(location.Column))
	ast_schema_gen.LocationFBAddOffset(s.builder, safeconv.IntToInt32(location.Offset))
	return ast_schema_gen.LocationFBEnd(s.builder), nil
}

// unpackLocation converts a FlatBuffer location to a domain location.
//
// Takes fb (*ast_schema_gen.LocationFB) which is the FlatBuffer location to
// convert.
//
// Returns ast_domain.Location which contains the line, column, and offset.
// Returns error which is always nil as conversion cannot fail.
func (*decoder) unpackLocation(fb *ast_schema_gen.LocationFB) (ast_domain.Location, error) {
	if fb == nil {
		return ast_domain.Location{}, nil
	}
	return ast_domain.Location{
		Line:   int(fb.Line()),
		Column: int(fb.Column()),
		Offset: int(fb.Offset()),
	}, nil
}

// buildRange serialises a source range to a flatbuffer offset.
//
// Takes r (*ast_domain.Range) which specifies the start and end locations.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised range.
// Returns error when building the start or end location fails.
func (s *encoder) buildRange(r *ast_domain.Range) (flatbuffers.UOffsetT, error) {
	if r == nil {
		return 0, nil
	}
	startOff, err := s.buildLocation(&r.Start)
	if err != nil {
		return 0, fmt.Errorf("building range start location: %w", err)
	}
	endOff, err := s.buildLocation(&r.End)
	if err != nil {
		return 0, fmt.Errorf("building range end location: %w", err)
	}

	ast_schema_gen.RangeFBStart(s.builder)
	ast_schema_gen.RangeFBAddStart(s.builder, startOff)
	ast_schema_gen.RangeFBAddEnd(s.builder, endOff)
	return ast_schema_gen.RangeFBEnd(s.builder), nil
}

// unpackRange converts a FlatBuffer range into a domain Range.
//
// Takes fb (*ast_schema_gen.RangeFB) which is the FlatBuffer range to convert.
//
// Returns ast_domain.Range which is the converted domain range.
// Returns error when unpacking the start or end location fails.
func (d *decoder) unpackRange(fb *ast_schema_gen.RangeFB) (ast_domain.Range, error) {
	if fb == nil {
		return ast_domain.Range{}, nil
	}
	r := ast_domain.Range{}
	var err error
	if startFB := fb.Start(&d.locFB); startFB != nil {
		r.Start, err = d.unpackLocation(startFB)
		if err != nil {
			return r, fmt.Errorf("unpacking range start location: %w", err)
		}
	}
	if endFB := fb.End(&d.locFB); endFB != nil {
		r.End, err = d.unpackLocation(endFB)
		if err != nil {
			return r, fmt.Errorf("unpacking range end location: %w", err)
		}
	}
	return r, nil
}

// buildDiagnosticRelatedInfo serialises diagnostic related info to FlatBuffers.
//
// Takes info (*ast_domain.DiagnosticRelatedInfo) which contains the related
// info to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the serialised data.
// Returns error when building the location fails.
func (s *encoder) buildDiagnosticRelatedInfo(info *ast_domain.DiagnosticRelatedInfo) (flatbuffers.UOffsetT, error) { //nolint:dupl // type-specific FlatBuffer serialisation
	if info == nil {
		return 0, nil
	}

	messageOff := s.builder.CreateString(info.Message)
	locOff, err := s.buildLocation(&info.Location)
	if err != nil {
		return 0, fmt.Errorf("building diagnostic related info location: %w", err)
	}

	ast_schema_gen.DiagnosticRelatedInfoFBStart(s.builder)
	ast_schema_gen.DiagnosticRelatedInfoFBAddLocation(s.builder, locOff)
	ast_schema_gen.DiagnosticRelatedInfoFBAddMessage(s.builder, messageOff)
	return ast_schema_gen.DiagnosticRelatedInfoFBEnd(s.builder), nil
}

// buildDiagnostic converts a Diagnostic to its FlatBuffer form.
//
// Takes diagnostic (*ast_domain.Diagnostic) which is the diagnostic to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the converted
// diagnostic.
// Returns error when building nested parts fails.
func (s *encoder) buildDiagnostic(diagnostic *ast_domain.Diagnostic) (flatbuffers.UOffsetT, error) {
	if diagnostic == nil {
		return 0, nil
	}

	messageOff := s.builder.CreateString(diagnostic.Message)
	expressionOff := s.builder.CreateString(diagnostic.Expression)
	sourcePathOff := s.builder.CreateString(diagnostic.SourcePath)
	codeOff := s.builder.CreateString(diagnostic.Code)

	locOff, err := s.buildLocation(&diagnostic.Location)
	if err != nil {
		return 0, fmt.Errorf("building diagnostic location: %w", err)
	}

	relatedInfoVec, err := buildVectorOfValues(s, diagnostic.RelatedInfo, (*encoder).buildDiagnosticRelatedInfo)
	if err != nil {
		return 0, fmt.Errorf("building diagnostic related info: %w", err)
	}

	dataVec, err := s.buildDiagnosticDataMap(diagnostic.Data)
	if err != nil {
		return 0, fmt.Errorf("building diagnostic data map: %w", err)
	}

	ast_schema_gen.DiagnosticFBStart(s.builder)
	ast_schema_gen.DiagnosticFBAddMessage(s.builder, messageOff)
	ast_schema_gen.DiagnosticFBAddSeverity(s.builder, ast_schema_gen.Severity(safeconv.IntToUint8(int(diagnostic.Severity))))
	ast_schema_gen.DiagnosticFBAddLocation(s.builder, locOff)
	ast_schema_gen.DiagnosticFBAddExpression(s.builder, expressionOff)
	ast_schema_gen.DiagnosticFBAddSourcePath(s.builder, sourcePathOff)
	ast_schema_gen.DiagnosticFBAddCode(s.builder, codeOff)
	ast_schema_gen.DiagnosticFBAddRelatedInfo(s.builder, relatedInfoVec)
	ast_schema_gen.DiagnosticFBAddData(s.builder, dataVec)
	ast_schema_gen.DiagnosticFBAddSourceLength(s.builder, safeconv.IntToInt32(diagnostic.SourceLength))
	return ast_schema_gen.DiagnosticFBEnd(s.builder), nil
}

// unpackDiagnosticRelatedInfo converts a FlatBuffer diagnostic related info
// into its domain representation.
//
// Takes fb (*ast_schema_gen.DiagnosticRelatedInfoFB) which is the FlatBuffer
// to unpack.
//
// Returns ast_domain.DiagnosticRelatedInfo which contains the unpacked data.
// Returns error when the location cannot be unpacked.
func (d *decoder) unpackDiagnosticRelatedInfo(fb *ast_schema_gen.DiagnosticRelatedInfoFB) (ast_domain.DiagnosticRelatedInfo, error) {
	if fb == nil {
		return ast_domain.DiagnosticRelatedInfo{}, nil
	}

	info := ast_domain.DiagnosticRelatedInfo{
		Message: mem.String(fb.Message()),
	}

	var err error
	if locFB := fb.Location(&d.locFB); locFB != nil {
		info.Location, err = d.unpackLocation(locFB)
		if err != nil {
			return info, fmt.Errorf("failed to unpack related info location: %w", err)
		}
	}

	return info, nil
}

// unpackDiagnostic converts a FlatBuffer diagnostic into a domain diagnostic.
//
// Takes fb (*ast_schema_gen.DiagnosticFB) which is the FlatBuffer to convert.
//
// Returns *ast_domain.Diagnostic which is the converted domain object, or nil
// if fb is nil.
// Returns error when unpacking the location, related info, or data fails.
func (d *decoder) unpackDiagnostic(fb *ast_schema_gen.DiagnosticFB) (*ast_domain.Diagnostic, error) {
	if fb == nil {
		return nil, nil
	}

	diagnostic := &ast_domain.Diagnostic{
		Message:      mem.String(fb.Message()),
		Severity:     ast_domain.Severity(fb.Severity()),
		Expression:   mem.String(fb.Expression()),
		SourcePath:   mem.String(fb.SourcePath()),
		Code:         mem.String(fb.Code()),
		SourceLength: int(fb.SourceLength()),
	}

	var err error
	if locFB := fb.Location(&d.locFB); locFB != nil {
		diagnostic.Location, err = d.unpackLocation(locFB)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack diagnostic location: %w", err)
		}
	}

	diagnostic.RelatedInfo, err = unpackVector(d, fb.RelatedInfoLength(), fb.RelatedInfo, (*decoder).unpackDiagnosticRelatedInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack related info: %w", err)
	}

	diagnostic.Data, err = d.unpackDiagnosticDataMap(fb)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack diagnostic data: %w", err)
	}

	return diagnostic, nil
}

// buildVectorOfValues builds a FlatBuffers vector from a slice of Go struct
// values using a generic helper pattern.
//
// When the items slice is empty, returns zero offset without error.
//
// Takes s (*encoder) which provides the FlatBuffers builder context.
// Takes items ([]T) which contains the Go struct values to serialise.
// Takes builder (buildFunc[T]) which converts each item to a FlatBuffers offset.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector.
// Returns error when any item fails to build.
func buildVectorOfValues[T any](s *encoder, items []T, builder buildFunc[T]) (flatbuffers.UOffsetT, error) {
	if len(items) == 0 {
		return 0, nil
	}
	offsets := make([]flatbuffers.UOffsetT, len(items))
	for i := range items {
		offset, err := builder(s, &items[i])
		if err != nil {
			return 0, fmt.Errorf("failed to build value item at index %d: %w", i, err)
		}
		offsets[i] = offset
	}
	return createVector(s, offsets), nil
}

// buildVectorOfPtrs builds a FlatBuffers vector from a slice of pointers to
// Go structs.
//
// Takes s (*encoder) which provides the FlatBuffers builder state.
// Takes items ([]*T) which contains the pointer items to convert.
// Takes builder (buildFunc[T]) which turns each item into a FlatBuffers offset.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector, or
// zero if items is empty.
// Returns error when building any item fails.
func buildVectorOfPtrs[T any](s *encoder, items []*T, builder buildFunc[T]) (flatbuffers.UOffsetT, error) {
	if len(items) == 0 {
		return 0, nil
	}
	offsets := make([]flatbuffers.UOffsetT, len(items))
	for i, item := range items {
		offset, err := builder(s, item)
		if err != nil {
			return 0, fmt.Errorf("failed to build pointer item at index %d: %w", i, err)
		}
		offsets[i] = offset
	}
	return createVector(s, offsets), nil
}

// createVector writes a slice of FlatBuffers offsets into the buffer as a
// vector.
//
// Takes s (*encoder) which provides the FlatBuffers builder instance.
// Takes offsets ([]flatbuffers.UOffsetT) which contains the offsets to write.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector, or
// zero when the offsets slice is empty.
func createVector(s *encoder, offsets []flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if len(offsets) == 0 {
		return 0
	}

	s.builder.StartVector(uoffsetTSize, len(offsets), uoffsetTSize)
	for i := len(offsets) - 1; i >= 0; i-- {
		s.builder.PrependUOffsetT(offsets[i])
	}
	return s.builder.EndVector(len(offsets))
}

// unpackVector converts a FlatBuffers vector into a slice of Go types.
//
// FBType is the FlatBuffers struct type (e.g., ast_schema_gen.HTMLAttributeFB).
// GoType is the domain struct type (e.g., ast_domain.HTMLAttribute).
//
// Takes d (*decoder) which provides the decoding context.
// Takes length (int) which specifies how many items are in the vector.
// Takes getter (func(...)) which fetches each FlatBuffers item by index.
// Takes unpacker (unpackerFunc) which converts a FlatBuffers item to a Go type.
//
// Returns []GoType which contains the converted Go values.
// Returns error when an item fails to unpack.
func unpackVector[FBType any, GoType any](d *decoder, length int, getter func(*FBType, int) bool, unpacker unpackerFunc[FBType, GoType]) ([]GoType, error) {
	if length == 0 {
		return nil, nil
	}
	result := make([]GoType, length)
	var fbItem FBType
	for i := range length {
		if getter(&fbItem, i) {
			goItem, err := unpacker(d, &fbItem)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack item at index %d: %w", i, err)
			}
			result[i] = goItem
		}
	}
	return result, nil
}

// unpackPtrVector deserialises a FlatBuffers vector into a slice of Go
// pointer types.
//
// FBType is the FlatBuffers struct type (e.g., ast_schema_gen.TemplateNodeFB).
// GoType is the domain struct type (e.g., ast_domain.TemplateNode).
//
// Takes d (*decoder) which provides the deserialisation context.
// Takes length (int) which specifies the number of items in the vector.
// Takes getter (func(...)) which retrieves each FlatBuffers item by index.
// Takes unpacker (unknown) which converts a FlatBuffers item to a Go type.
//
// Returns []*GoType which contains the deserialised Go pointer items.
// Returns error when an item cannot be unpacked.
func unpackPtrVector[FBType any, GoType any](d *decoder, length int, getter func(*FBType, int) bool, unpacker unpackerPtrFunc[FBType, GoType]) ([]*GoType, error) {
	if length == 0 {
		return nil, nil
	}
	result := make([]*GoType, length)
	var fbItem FBType
	for i := range length {
		if getter(&fbItem, i) {
			goItem, err := unpacker(d, &fbItem)
			if err != nil {
				return nil, fmt.Errorf("failed to unpack pointer item at index %d: %w", i, err)
			}
			result[i] = goItem
		}
	}
	return result, nil
}
