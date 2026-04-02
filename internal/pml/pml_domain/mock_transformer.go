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
	"sync/atomic"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/pml/pml_dto"
)

// MockTransformer is a test double for Transformer where nil function
// fields return zero values and call counts are tracked atomically.
type MockTransformer struct {
	// TransformFunc is the function called by
	// Transform.
	TransformFunc func(ast *ast_domain.TemplateAST, config *pml_dto.Config) (*ast_domain.TemplateAST, string, []*Error)

	// TransformForEmailFunc is the function called by
	// TransformForEmail.
	TransformForEmailFunc func(ast *ast_domain.TemplateAST, config *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*Error)

	// TransformCallCount tracks how many times
	// Transform was called.
	TransformCallCount int64

	// TransformForEmailCallCount tracks how many times
	// TransformForEmail was called.
	TransformForEmailCallCount int64
}

var _ Transformer = (*MockTransformer)(nil)

// Transform delegates to TransformFunc if set.
//
// Takes ast (*ast_domain.TemplateAST) which is the template AST to transform.
// Takes config (*pml_dto.Config) which provides the transformation settings.
//
// Returns the input AST, an empty string, and nil if TransformFunc is nil.
func (m *MockTransformer) Transform(_ context.Context, ast *ast_domain.TemplateAST, config *pml_dto.Config) (*ast_domain.TemplateAST, string, []*Error) {
	atomic.AddInt64(&m.TransformCallCount, 1)
	if m.TransformFunc != nil {
		return m.TransformFunc(ast, config)
	}
	return ast, "", nil
}

// TransformForEmail delegates to TransformForEmailFunc if set.
//
// Takes ast (*ast_domain.TemplateAST) which is the template AST to transform.
// Takes config (*pml_dto.Config) which provides the transformation settings.
//
// Returns the input AST, an empty string, nil asset requests, and nil errors
// if TransformForEmailFunc is nil.
func (m *MockTransformer) TransformForEmail(_ context.Context, ast *ast_domain.TemplateAST, config *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*Error) {
	atomic.AddInt64(&m.TransformForEmailCallCount, 1)
	if m.TransformForEmailFunc != nil {
		return m.TransformForEmailFunc(ast, config)
	}
	return ast, "", nil, nil
}
