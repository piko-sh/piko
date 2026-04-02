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
	"errors"
	goast "go/ast"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

func TestMockCollectionService_ProcessGetCollectionCall(t *testing.T) {
	t.Parallel()

	t.Run("nil ProcessGetCollectionCallFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCollectionService{}

		got, err := mock.ProcessGetCollectionCall(
			context.Background(),
			"blog",
			"Post",
			goast.NewIdent("Post"),
			nil,
		)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessGetCollectionCallCallCount))
	})

	t.Run("delegates to ProcessGetCollectionCallFunc", func(t *testing.T) {
		t.Parallel()

		wantAnnotation := &ast_domain.GoGeneratorAnnotation{
			IsCollectionCall: true,
		}
		sentExpr := goast.NewIdent("Product")

		mock := &MockCollectionService{
			ProcessGetCollectionCallFunc: func(
				ctx context.Context,
				collectionName string,
				targetTypeName string,
				targetTypeExpr goast.Expr,
				options any,
			) (*ast_domain.GoGeneratorAnnotation, error) {
				assert.Equal(t, "products", collectionName)
				assert.Equal(t, "Product", targetTypeName)
				assert.Equal(t, sentExpr, targetTypeExpr)
				assert.Equal(t, "some-option", options)
				return wantAnnotation, nil
			},
		}

		got, err := mock.ProcessGetCollectionCall(
			context.Background(),
			"products",
			"Product",
			sentExpr,
			"some-option",
		)

		require.NoError(t, err)
		assert.Same(t, wantAnnotation, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessGetCollectionCallCallCount))
	})

	t.Run("propagates error from ProcessGetCollectionCallFunc", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("collection not found")

		mock := &MockCollectionService{
			ProcessGetCollectionCallFunc: func(
				_ context.Context,
				_ string,
				_ string,
				_ goast.Expr,
				_ any,
			) (*ast_domain.GoGeneratorAnnotation, error) {
				return nil, wantErr
			},
		}

		got, err := mock.ProcessGetCollectionCall(
			context.Background(), "", "", nil, nil,
		)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, wantErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessGetCollectionCallCallCount))
	})
}

func TestMockCollectionService_ProcessCollectionDirective(t *testing.T) {
	t.Parallel()

	t.Run("nil ProcessCollectionDirectiveFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCollectionService{}

		got, err := mock.ProcessCollectionDirective(
			context.Background(),
			&collection_dto.CollectionDirectiveInfo{
				CollectionName: "blog",
				ProviderName:   "markdown",
			},
		)

		assert.Nil(t, got)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessCollectionDirectiveCallCount))
	})

	t.Run("delegates to ProcessCollectionDirectiveFunc", func(t *testing.T) {
		t.Parallel()

		wantEntryPoints := []*collection_dto.CollectionEntryPoint{
			{Path: "/blog/hello.pk", IsVirtual: true},
			{Path: "/blog/world.pk", IsVirtual: true},
		}

		sentDirective := &collection_dto.CollectionDirectiveInfo{
			CollectionName: "blog",
			ProviderName:   "markdown",
			LayoutPath:     "pages/blog/{slug}.pk",
		}

		mock := &MockCollectionService{
			ProcessCollectionDirectiveFunc: func(
				ctx context.Context,
				directive *collection_dto.CollectionDirectiveInfo,
			) ([]*collection_dto.CollectionEntryPoint, error) {
				assert.Same(t, sentDirective, directive)
				return wantEntryPoints, nil
			},
		}

		got, err := mock.ProcessCollectionDirective(
			context.Background(),
			sentDirective,
		)

		require.NoError(t, err)
		assert.Equal(t, wantEntryPoints, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessCollectionDirectiveCallCount))
	})

	t.Run("propagates error from ProcessCollectionDirectiveFunc", func(t *testing.T) {
		t.Parallel()

		wantErr := errors.New("provider unavailable")

		mock := &MockCollectionService{
			ProcessCollectionDirectiveFunc: func(
				_ context.Context,
				_ *collection_dto.CollectionDirectiveInfo,
			) ([]*collection_dto.CollectionEntryPoint, error) {
				return nil, wantErr
			},
		}

		got, err := mock.ProcessCollectionDirective(
			context.Background(),
			&collection_dto.CollectionDirectiveInfo{},
		)

		assert.Nil(t, got)
		assert.ErrorIs(t, err, wantErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessCollectionDirectiveCallCount))
	})
}

func TestMockCollectionService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockCollectionService

	ann, err := mock.ProcessGetCollectionCall(
		context.Background(), "", "", nil, nil,
	)
	assert.Nil(t, ann)
	assert.NoError(t, err)

	entries, err := mock.ProcessCollectionDirective(
		context.Background(), &collection_dto.CollectionDirectiveInfo{},
	)
	assert.Nil(t, entries)
	assert.NoError(t, err)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessGetCollectionCallCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessCollectionDirectiveCallCount))
}

func TestMockCollectionService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	mock := &MockCollectionService{
		ProcessGetCollectionCallFunc: func(
			_ context.Context,
			_ string,
			_ string,
			_ goast.Expr,
			_ any,
		) (*ast_domain.GoGeneratorAnnotation, error) {
			return &ast_domain.GoGeneratorAnnotation{}, nil
		},
		ProcessCollectionDirectiveFunc: func(
			_ context.Context,
			_ *collection_dto.CollectionDirectiveInfo,
		) ([]*collection_dto.CollectionEntryPoint, error) {
			return []*collection_dto.CollectionEntryPoint{}, nil
		},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = mock.ProcessGetCollectionCall(
				context.Background(), "c", "T", goast.NewIdent("T"), nil,
			)
		}()
		go func() {
			defer wg.Done()
			_, _ = mock.ProcessCollectionDirective(
				context.Background(), &collection_dto.CollectionDirectiveInfo{},
			)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ProcessGetCollectionCallCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ProcessCollectionDirectiveCallCount))
}
