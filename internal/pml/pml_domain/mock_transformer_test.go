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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/pml/pml_dto"
)

func TestMockTransformer_Transform(t *testing.T) {
	t.Parallel()

	t.Run("nil TransformFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		inputAST := &ast_domain.TemplateAST{}

		mock := &MockTransformer{
			TransformFunc:              nil,
			TransformForEmailFunc:      nil,
			TransformCallCount:         0,
			TransformForEmailCallCount: 0,
		}

		resultAST, css, errs := mock.Transform(context.Background(), inputAST, &pml_dto.Config{})

		assert.Same(t, inputAST, resultAST)
		assert.Equal(t, "", css)
		assert.Nil(t, errs)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TransformCallCount))
	})

	t.Run("delegates to TransformFunc", func(t *testing.T) {
		t.Parallel()

		inputAST := &ast_domain.TemplateAST{}
		inputConfig := &pml_dto.Config{}
		outputAST := &ast_domain.TemplateAST{}

		var capturedAST *ast_domain.TemplateAST
		var capturedConfig *pml_dto.Config

		mock := &MockTransformer{
			TransformFunc: func(ast *ast_domain.TemplateAST, config *pml_dto.Config) (*ast_domain.TemplateAST, string, []*Error) {
				capturedAST = ast
				capturedConfig = config
				return outputAST, "body { margin: 0; }", nil
			},
			TransformForEmailFunc:      nil,
			TransformCallCount:         0,
			TransformForEmailCallCount: 0,
		}

		resultAST, css, errs := mock.Transform(context.Background(), inputAST, inputConfig)

		assert.Same(t, outputAST, resultAST)
		assert.Equal(t, "body { margin: 0; }", css)
		assert.Nil(t, errs)
		assert.Same(t, inputAST, capturedAST)
		assert.Same(t, inputConfig, capturedConfig)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TransformCallCount))
	})

	t.Run("propagates errors from TransformFunc", func(t *testing.T) {
		t.Parallel()

		inputAST := &ast_domain.TemplateAST{}
		expectedErrors := []*Error{
			{Message: "invalid tag", TagName: "pml-unknown", Severity: SeverityError},
		}

		mock := &MockTransformer{
			TransformFunc: func(_ *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*Error) {
				return nil, "", expectedErrors
			},
			TransformForEmailFunc:      nil,
			TransformCallCount:         0,
			TransformForEmailCallCount: 0,
		}

		resultAST, css, errs := mock.Transform(context.Background(), inputAST, &pml_dto.Config{})

		assert.Nil(t, resultAST)
		assert.Equal(t, "", css)
		require.Len(t, errs, 1)
		assert.Equal(t, "invalid tag", errs[0].Message)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TransformCallCount))
	})
}

func TestMockTransformer_TransformForEmail(t *testing.T) {
	t.Parallel()

	t.Run("nil TransformForEmailFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		inputAST := &ast_domain.TemplateAST{}

		mock := &MockTransformer{
			TransformFunc:              nil,
			TransformForEmailFunc:      nil,
			TransformCallCount:         0,
			TransformForEmailCallCount: 0,
		}

		resultAST, css, assets, errs := mock.TransformForEmail(context.Background(), inputAST, &pml_dto.Config{})

		assert.Same(t, inputAST, resultAST)
		assert.Equal(t, "", css)
		assert.Nil(t, assets)
		assert.Nil(t, errs)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TransformForEmailCallCount))
	})

	t.Run("delegates to TransformForEmailFunc", func(t *testing.T) {
		t.Parallel()

		inputAST := &ast_domain.TemplateAST{}
		inputConfig := &pml_dto.Config{}
		outputAST := &ast_domain.TemplateAST{}
		expectedAssets := []*email_dto.EmailAssetRequest{
			{SourcePath: "assets/logo.png"},
		}

		var capturedAST *ast_domain.TemplateAST
		var capturedConfig *pml_dto.Config

		mock := &MockTransformer{
			TransformFunc: nil,
			TransformForEmailFunc: func(ast *ast_domain.TemplateAST, config *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*Error) {
				capturedAST = ast
				capturedConfig = config
				return outputAST, "img { max-width: 100%; }", expectedAssets, nil
			},
			TransformCallCount:         0,
			TransformForEmailCallCount: 0,
		}

		resultAST, css, assets, errs := mock.TransformForEmail(context.Background(), inputAST, inputConfig)

		assert.Same(t, outputAST, resultAST)
		assert.Equal(t, "img { max-width: 100%; }", css)
		require.Len(t, assets, 1)
		assert.Equal(t, "assets/logo.png", assets[0].SourcePath)
		assert.Nil(t, errs)
		assert.Same(t, inputAST, capturedAST)
		assert.Same(t, inputConfig, capturedConfig)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TransformForEmailCallCount))
	})

	t.Run("propagates errors from TransformForEmailFunc", func(t *testing.T) {
		t.Parallel()

		inputAST := &ast_domain.TemplateAST{}
		expectedErrors := []*Error{
			{Message: "missing src attribute", TagName: "pml-img", Severity: SeverityWarning},
		}

		mock := &MockTransformer{
			TransformFunc: nil,
			TransformForEmailFunc: func(_ *ast_domain.TemplateAST, _ *pml_dto.Config) (*ast_domain.TemplateAST, string, []*email_dto.EmailAssetRequest, []*Error) {
				return nil, "", nil, expectedErrors
			},
			TransformCallCount:         0,
			TransformForEmailCallCount: 0,
		}

		resultAST, css, assets, errs := mock.TransformForEmail(context.Background(), inputAST, &pml_dto.Config{})

		assert.Nil(t, resultAST)
		assert.Equal(t, "", css)
		assert.Nil(t, assets)
		require.Len(t, errs, 1)
		assert.Equal(t, "missing src attribute", errs[0].Message)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TransformForEmailCallCount))
	})
}

func TestMockTransformer_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockTransformer

	inputAST := &ast_domain.TemplateAST{}
	config := &pml_dto.Config{}

	resultAST, css, errs := mock.Transform(context.Background(), inputAST, config)
	assert.Same(t, inputAST, resultAST)
	assert.Equal(t, "", css)
	assert.Nil(t, errs)

	resultAST2, css2, assets, errs2 := mock.TransformForEmail(context.Background(), inputAST, config)
	assert.Same(t, inputAST, resultAST2)
	assert.Equal(t, "", css2)
	assert.Nil(t, assets)
	assert.Nil(t, errs2)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TransformCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.TransformForEmailCallCount))
}

func TestMockTransformer_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockTransformer{
		TransformFunc:              nil,
		TransformForEmailFunc:      nil,
		TransformCallCount:         0,
		TransformForEmailCallCount: 0,
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			inputAST := &ast_domain.TemplateAST{}
			_, _, _ = mock.Transform(context.Background(), inputAST, &pml_dto.Config{})
		}()
		go func() {
			defer wg.Done()
			inputAST := &ast_domain.TemplateAST{}
			_, _, _, _ = mock.TransformForEmail(context.Background(), inputAST, &pml_dto.Config{})
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.TransformCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.TransformForEmailCallCount))
}
