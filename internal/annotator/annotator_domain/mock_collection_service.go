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

package annotator_domain

import (
	"context"
	goast "go/ast"
	"sync/atomic"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

// MockCollectionService is a test double for CollectionServicePort
// where nil function fields return zero values and call counts are
// tracked atomically.
type MockCollectionService struct {
	// ProcessGetCollectionCallFunc is the function called
	// by ProcessGetCollectionCall.
	ProcessGetCollectionCallFunc func(
		ctx context.Context,
		collectionName string,
		targetTypeName string,
		targetTypeExpr goast.Expr,
		options any,
	) (*ast_domain.GoGeneratorAnnotation, error)

	// ProcessCollectionDirectiveFunc is the function
	// called by ProcessCollectionDirective.
	ProcessCollectionDirectiveFunc func(
		ctx context.Context,
		directive *collection_dto.CollectionDirectiveInfo,
	) ([]*collection_dto.CollectionEntryPoint, error)

	// ProcessGetCollectionCallCallCount tracks how many
	// times ProcessGetCollectionCall was called.
	ProcessGetCollectionCallCallCount int64

	// ProcessCollectionDirectiveCallCount tracks how many
	// times ProcessCollectionDirective was called.
	ProcessCollectionDirectiveCallCount int64
}

var _ CollectionServicePort = (*MockCollectionService)(nil)

// ProcessGetCollectionCall delegates to ProcessGetCollectionCallFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes collectionName (string) which identifies the collection by name.
// Takes targetTypeName (string) which is the target Go type name.
// Takes targetTypeExpr (goast.Expr) which is the AST expression for
// the target type.
// Takes options (any) which provides additional collection options.
//
// Returns (nil, nil) if ProcessGetCollectionCallFunc is nil.
func (m *MockCollectionService) ProcessGetCollectionCall(
	ctx context.Context,
	collectionName string,
	targetTypeName string,
	targetTypeExpr goast.Expr,
	options any,
) (*ast_domain.GoGeneratorAnnotation, error) {
	atomic.AddInt64(&m.ProcessGetCollectionCallCallCount, 1)
	if m.ProcessGetCollectionCallFunc != nil {
		return m.ProcessGetCollectionCallFunc(ctx, collectionName, targetTypeName, targetTypeExpr, options)
	}
	return nil, nil
}

// ProcessCollectionDirective delegates to ProcessCollectionDirectiveFunc if
// set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes directive (*collection_dto.CollectionDirectiveInfo) which
// contains the collection directive to process.
//
// Returns (nil, nil) if ProcessCollectionDirectiveFunc is nil.
func (m *MockCollectionService) ProcessCollectionDirective(
	ctx context.Context,
	directive *collection_dto.CollectionDirectiveInfo,
) ([]*collection_dto.CollectionEntryPoint, error) {
	atomic.AddInt64(&m.ProcessCollectionDirectiveCallCount, 1)
	if m.ProcessCollectionDirectiveFunc != nil {
		return m.ProcessCollectionDirectiveFunc(ctx, directive)
	}
	return nil, nil
}
