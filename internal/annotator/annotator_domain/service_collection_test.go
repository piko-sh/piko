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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

func newTestAnnotatorService(t *testing.T, collectionService CollectionServicePort) *AnnotatorService {
	t.Helper()

	resolver := &resolver_domain.MockResolver{
		GetBaseDirFunc: func() string { return "/project" },
		ResolvePKPathFunc: func(_ context.Context, importPath string, _ string) (string, error) {
			return importPath, nil
		},
	}

	svc, err := NewAnnotatorService(context.Background(), &AnnotatorServiceConfig{
		Resolver:          resolver,
		CollectionService: collectionService,
		InMemoryMode:      true,
	})
	require.NoError(t, err)

	return svc
}

func TestExpandCollectionDirectives_ProviderNotFound_SkipsGracefully(t *testing.T) {
	t.Parallel()

	mockService := &MockCollectionService{
		ProcessCollectionDirectiveFunc: func(
			_ context.Context,
			_ *collection_dto.CollectionDirectiveInfo,
		) ([]*collection_dto.CollectionEntryPoint, error) {
			return nil, fmt.Errorf(
				"%w: unknown collection provider 'markdown'; available providers: []",
				collection_dto.ErrProviderNotFound,
			)
		},
	}

	svc := newTestAnnotatorService(t, mockService)

	graph := &annotator_dto.ComponentGraph{
		Components: map[string]*annotator_dto.ParsedComponent{
			"blog_slug": {
				SourcePath:         "/project/pages/blog/{slug}.pk",
				HasCollection:      true,
				CollectionProvider: "markdown",
				CollectionName:     "blog",
			},
			"about": {
				SourcePath:    "/project/pages/about.pk",
				HasCollection: false,
			},
		},
		PathToHashedName: map[string]string{},
		HashedNameToPath: map[string]string{},
	}

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "/project/pages/blog/{slug}.pk", IsPage: true},
		{Path: "/project/pages/about.pk", IsPage: true},
	}

	expandedEPs, diagnostics, err := svc.expandCollectionDirectives(
		context.Background(), graph, entryPoints, &annotationOptions{},
	)

	require.NoError(t, err)

	require.Len(t, diagnostics, 1)
	assert.Equal(t, ast_domain.Warning, diagnostics[0].Severity)
	assert.Equal(t, annotator_dto.CodeCollectionProviderNotFound, diagnostics[0].Code)
	assert.Contains(t, diagnostics[0].Message, "markdown")
	assert.Equal(t, "/project/pages/blog/{slug}.pk", diagnostics[0].SourcePath)

	require.Len(t, expandedEPs, 1)
	assert.Equal(t, "/project/pages/about.pk", expandedEPs[0].Path)
}

func TestExpandCollectionDirectives_OtherError_StillFails(t *testing.T) {
	t.Parallel()

	mockService := &MockCollectionService{
		ProcessCollectionDirectiveFunc: func(
			_ context.Context,
			_ *collection_dto.CollectionDirectiveInfo,
		) ([]*collection_dto.CollectionEntryPoint, error) {
			return nil, errors.New("network timeout fetching content")
		},
	}

	svc := newTestAnnotatorService(t, mockService)

	graph := &annotator_dto.ComponentGraph{
		Components: map[string]*annotator_dto.ParsedComponent{
			"blog_slug": {
				SourcePath:         "/project/pages/blog/{slug}.pk",
				HasCollection:      true,
				CollectionProvider: "markdown",
				CollectionName:     "blog",
			},
		},
		PathToHashedName: map[string]string{},
		HashedNameToPath: map[string]string{},
	}

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "/project/pages/blog/{slug}.pk", IsPage: true},
	}

	_, diagnostics, err := svc.expandCollectionDirectives(
		context.Background(), graph, entryPoints, &annotationOptions{},
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout fetching content")
	assert.Empty(t, diagnostics)
}

func TestExpandCollectionDirectives_NilCollectionService_Skips(t *testing.T) {
	t.Parallel()

	svc := newTestAnnotatorService(t, nil)

	graph := &annotator_dto.ComponentGraph{
		Components: map[string]*annotator_dto.ParsedComponent{
			"blog_slug": {
				SourcePath:         "/project/pages/blog/{slug}.pk",
				HasCollection:      true,
				CollectionProvider: "markdown",
				CollectionName:     "blog",
			},
		},
		PathToHashedName: map[string]string{},
		HashedNameToPath: map[string]string{},
	}

	entryPoints := []annotator_dto.EntryPoint{
		{Path: "/project/pages/blog/{slug}.pk", IsPage: true},
	}

	expandedEPs, diagnostics, err := svc.expandCollectionDirectives(
		context.Background(), graph, entryPoints, &annotationOptions{},
	)

	require.NoError(t, err)
	assert.Nil(t, diagnostics)
	assert.Equal(t, entryPoints, expandedEPs)
}
