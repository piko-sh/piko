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
	"errors"
	"fmt"
	"sync"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/ast/ast_schema"
	"piko.sh/piko/internal/ast/ast_schema/ast_schema_gen"
	"piko.sh/piko/internal/fbs"
)

var (
	// builderPool provides a pool of FlatBuffers builders to reduce allocation
	// overhead during AST serialisation. Builders are reset before being returned
	// to the pool.
	builderPool = sync.Pool{
		New: func() any {
			return flatbuffers.NewBuilder(8192)
		},
	}

	// errASTSchemaVersionMismatch indicates the cached AST was encoded with a
	// different schema version. The caller should treat this as a cache miss and
	// regenerate the AST.
	errASTSchemaVersionMismatch = fbs.ErrSchemaVersionMismatch
)

// encoder holds the state for converting Go types to FlatBuffers format.
type encoder struct {
	// builder holds the FlatBuffers builder used to encode data.
	builder *flatbuffers.Builder
}

// decoder holds state for converting FlatBuffers data to Go types.
//
// The FlatBuffer wrapper fields (locFB, rangeFB, etc.) are reused across
// unpacking calls to avoid per-accessor heap allocations. Each FlatBuffer
// accessor's Init method overwrites two fields (Bytes, Pos) and the unpacker
// reads values immediately into stack-allocated value types, making reuse safe.
type decoder struct {
	// locFB is a reusable wrapper for decoding LocationFB accessors.
	locFB ast_schema_gen.LocationFB

	// rangeFB is a reusable wrapper for decoding RangeFB accessors.
	rangeFB ast_schema_gen.RangeFB

	// dirFB is a reusable wrapper for decoding DirectiveFB accessors.
	dirFB ast_schema_gen.DirectiveFB

	// expressionNodeFB is a reusable wrapper for decoding ExpressionNodeFB accessors.
	expressionNodeFB ast_schema_gen.ExpressionNodeFB

	// goAnnotFB is a reusable wrapper for decoding GoGeneratorAnnotationFB accessors.
	goAnnotFB ast_schema_gen.GoGeneratorAnnotationFB

	// rtAnnotFB is a reusable wrapper for decoding RuntimeAnnotationFB accessors.
	rtAnnotFB ast_schema_gen.RuntimeAnnotationFB

	// dwFB is a reusable wrapper for decoding DirectWriterFB accessors.
	dwFB ast_schema_gen.DirectWriterFB

	// identFB is a reusable wrapper for decoding IdentifierFB accessors.
	identFB ast_schema_gen.IdentifierFB

	// resolvedFB is a reusable wrapper for decoding ResolvedTypeInfoFB accessors.
	resolvedFB ast_schema_gen.ResolvedTypeInfoFB

	// symbolFB is a reusable wrapper for decoding ResolvedSymbolFB accessors.
	symbolFB ast_schema_gen.ResolvedSymbolFB

	// pdsFB is a reusable wrapper for decoding PropDataSourceFB accessors.
	pdsFB ast_schema_gen.PropDataSourceFB

	// partialFB is a reusable wrapper for decoding PartialInvocationInfoFB accessors.
	partialFB ast_schema_gen.PartialInvocationInfoFB

	// propValFB is a reusable wrapper for decoding PropValueFB accessors.
	propValFB ast_schema_gen.PropValueFB

	// tableFB is a reusable generic FlatBuffers table for union decoding.
	tableFB flatbuffers.Table

	// skipRanges when true omits location and range fields during decoding.
	skipRanges bool
}

// buildFunc is a function type that builds a FlatBuffers table from a Go
// struct pointer. T is the Go domain struct type.
type buildFunc[T any] func(s *encoder, item *T) (flatbuffers.UOffsetT, error)

// unpackerFunc is a function type that converts a FlatBuffers table to a Go
// struct. FBType is the FlatBuffers type and GoType is the Go domain type.
type unpackerFunc[FBType any, GoType any] func(d *decoder, fb *FBType) (GoType, error)

// unpackerPtrFunc is a function type that converts a FlatBuffers table to a
// Go pointer struct. FBType is the FlatBuffers type and GoType is the target
// Go type.
type unpackerPtrFunc[FBType any, GoType any] func(d *decoder, fb *FBType) (*GoType, error)

// EncodeAST converts an in-memory TemplateAST to its versioned FlatBuffers
// byte representation. The output includes a 32-byte schema hash prefix for
// automatic cache invalidation when the schema changes between Piko versions.
//
// Takes ast (*ast_domain.TemplateAST) which is the template AST to encode.
//
// Returns []byte which contains the encoded FlatBuffers data with schema
// hash prefix.
// Returns error when the builder pool returns an unexpected type or
// encoding fails.
func EncodeAST(ast *ast_domain.TemplateAST) ([]byte, error) {
	builder, ok := builderPool.Get().(*flatbuffers.Builder)
	if !ok {
		return nil, errors.New("failed to get builder from pool: unexpected type")
	}

	s := &encoder{builder: builder}
	rootOffset, err := s.buildTemplateAST(ast)
	if err != nil {
		builder.Reset()
		builderPool.Put(builder)
		return nil, fmt.Errorf("failed to encode AST: %w", err)
	}
	s.builder.Finish(rootOffset)

	payload := s.builder.FinishedBytes()
	result := make([]byte, fbs.PackedSize(len(payload)))
	ast_schema.PackInto(result, payload)

	builder.Reset()
	builderPool.Put(builder)

	return result, nil
}

// DecodeAST converts a versioned FlatBuffers byte slice back into an
// in-memory TemplateAST.
//
// Takes ctx (context.Context) which carries logging and cancellation context
// through the deserialisation path.
// Takes data ([]byte) which contains the encoded FlatBuffers data.
//
// Returns *ast_domain.TemplateAST which is the decoded AST structure.
// Returns error when the data is empty or cannot be unpacked.
//
// Returns errASTSchemaVersionMismatch if the data was encoded with a
// different schema version.
//
// SAFETY: The returned AST contains strings that reference 'data' directly via
// mem.String. Go's GC keeps 'data' alive through these string references. The
// caller must not modify 'data' while the AST is in use.
func DecodeAST(ctx context.Context, data []byte) (*ast_domain.TemplateAST, error) {
	if len(data) == 0 {
		return nil, errors.New("cannot decode empty byte slice")
	}

	payload, err := ast_schema.Unpack(data)
	if err != nil {
		if errors.Is(err, fbs.ErrSchemaVersionMismatch) {
			return nil, errASTSchemaVersionMismatch
		}
		return nil, fmt.Errorf("failed to unpack versioned AST data: %w", err)
	}

	root := ast_schema_gen.GetRootAsTemplateASTFB(payload, 0)
	d := &decoder{}
	ast, err := d.unpackTemplateAST(context.WithoutCancel(ctx), root)
	if err != nil {
		return nil, fmt.Errorf("decoding AST: %w", err)
	}
	return ast, nil
}

// DecodeASTForRender converts a versioned FlatBuffers byte slice back into an
// in-memory TemplateAST, skipping location and range fields that are only
// needed by the LSP, formatter, and error reporter.
//
// Takes ctx (context.Context) which carries logging and cancellation context
// through the deserialisation path.
// Takes data ([]byte) which contains the encoded FlatBuffers data.
//
// Returns *ast_domain.TemplateAST which is the decoded AST structure.
// Returns error when the data is empty or cannot be unpacked.
//
// Returns errASTSchemaVersionMismatch if the data was encoded with a
// different schema version.
//
// SAFETY: The returned AST contains strings that reference 'data' directly via
// mem.String. Go's GC keeps 'data' alive through these string references. The
// caller must not modify 'data' while the AST is in use.
func DecodeASTForRender(ctx context.Context, data []byte) (*ast_domain.TemplateAST, error) {
	if len(data) == 0 {
		return nil, errors.New("cannot decode empty byte slice")
	}

	payload, err := ast_schema.Unpack(data)
	if err != nil {
		if errors.Is(err, fbs.ErrSchemaVersionMismatch) {
			return nil, errASTSchemaVersionMismatch
		}
		return nil, fmt.Errorf("failed to unpack versioned AST data: %w", err)
	}

	root := ast_schema_gen.GetRootAsTemplateASTFB(payload, 0)
	d := &decoder{skipRanges: true}
	result, err := d.unpackTemplateAST(context.WithoutCancel(ctx), root)
	if err != nil {
		return nil, fmt.Errorf("decoding AST for render: %w", err)
	}
	return result, nil
}
