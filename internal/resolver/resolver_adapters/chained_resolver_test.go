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

package resolver_adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/resolver/resolver_domain"
)

func TestChainedResolver_ResolvePKPath_FirstResolverSucceeds(t *testing.T) {
	ctx := context.Background()

	firstResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			return "/path/to/local/component.pk", nil
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			t.Fatal("Second resolver should not be called when first succeeds")
			return "", nil
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	result, err := chainedResolver.ResolvePKPath(ctx, "myproject/components/card.pk", "")
	require.NoError(t, err)
	assert.Equal(t, "/path/to/local/component.pk", result)
}

func TestChainedResolver_ResolvePKPath_FirstFailsSecondSucceeds(t *testing.T) {
	ctx := context.Background()

	firstCalled := false
	secondCalled := false

	firstResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			firstCalled = true
			return "", errors.New("not in local module")
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			secondCalled = true
			return "/path/to/gomodcache/component.pk", nil
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	result, err := chainedResolver.ResolvePKPath(ctx, "github.com/ui/lib/button.pk", "")
	require.NoError(t, err)
	assert.Equal(t, "/path/to/gomodcache/component.pk", result)
	assert.True(t, firstCalled, "First resolver should have been called")
	assert.True(t, secondCalled, "Second resolver should have been called")
}

func TestChainedResolver_ResolvePKPath_AllResolversFail(t *testing.T) {
	ctx := context.Background()

	firstResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			return "", errors.New("not in local module 'myproject'")
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			return "", errors.New("module 'github.com/missing/lib' not found in cache")
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	_, err := chainedResolver.ResolvePKPath(ctx, "github.com/missing/lib/button.pk", "")
	require.Error(t, err)

	assert.Contains(t, err.Error(), "failed to resolve component")
	assert.Contains(t, err.Error(), "not in local module")
	assert.Contains(t, err.Error(), "not found in cache")
	assert.Contains(t, err.Error(), "2 resolvers")
}

func TestChainedResolver_ResolvePKPath_EmptyChain(t *testing.T) {
	ctx := context.Background()
	chainedResolver := NewChainedResolver()

	_, err := chainedResolver.ResolvePKPath(ctx, "some/path.pk", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no resolvers configured")
}

func TestChainedResolver_ResolveCSSPath_Delegation(t *testing.T) {
	ctx := context.Background()

	firstResolver := &resolver_domain.MockResolver{
		ResolveCSSPathFunc: func(ctx context.Context, importPath string, containingDir string) (string, error) {
			return "", errors.New("not local CSS")
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		ResolveCSSPathFunc: func(ctx context.Context, importPath string, containingDir string) (string, error) {
			return "/path/to/external.css", nil
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	result, err := chainedResolver.ResolveCSSPath(ctx, "github.com/ui/lib/styles.css", "/some/directory")
	require.NoError(t, err)
	assert.Equal(t, "/path/to/external.css", result)
}

func TestChainedResolver_DetectLocalModule_InitialisesAllResolvers(t *testing.T) {
	ctx := context.Background()

	firstCalled := false
	secondCalled := false

	firstResolver := &resolver_domain.MockResolver{
		DetectLocalModuleFunc: func(ctx context.Context) error {
			firstCalled = true
			return nil
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		DetectLocalModuleFunc: func(ctx context.Context) error {
			secondCalled = true
			return nil
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	err := chainedResolver.DetectLocalModule(ctx)
	require.NoError(t, err)
	assert.True(t, firstCalled, "First resolver should be called")
	assert.True(t, secondCalled, "Second resolver should also be called for initialisation")
}

func TestChainedResolver_DetectLocalModule_SecondaryFailureIsNonFatal(t *testing.T) {
	ctx := context.Background()

	firstResolver := &resolver_domain.MockResolver{
		DetectLocalModuleFunc: func(ctx context.Context) error {
			return nil
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		DetectLocalModuleFunc: func(ctx context.Context) error {
			return errors.New("secondary resolver failure")
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	err := chainedResolver.DetectLocalModule(ctx)
	require.NoError(t, err, "Secondary resolver failures should be non-fatal")
}

func TestChainedResolver_DetectLocalModule_PrimaryFailureIsFatal(t *testing.T) {
	ctx := context.Background()

	firstResolver := &resolver_domain.MockResolver{
		DetectLocalModuleFunc: func(ctx context.Context) error {
			return errors.New("primary resolver failure")
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		DetectLocalModuleFunc: func(ctx context.Context) error {
			t.Fatal("Secondary resolver should not be called if primary fails")
			return nil
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	err := chainedResolver.DetectLocalModule(ctx)
	require.Error(t, err, "Primary resolver failures should be fatal")
	assert.Contains(t, err.Error(), "primary resolver failure")
}

func TestChainedResolver_GetModuleName_DelegatesToFirst(t *testing.T) {
	firstResolver := &resolver_domain.MockResolver{
		GetModuleNameFunc: func() string {
			return "myproject"
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		GetModuleNameFunc: func() string {
			t.Fatal("Second resolver's GetModuleName should not be called")
			return ""
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	moduleName := chainedResolver.GetModuleName()
	assert.Equal(t, "myproject", moduleName)
}

func TestChainedResolver_GetBaseDir_DelegatesToFirst(t *testing.T) {
	firstResolver := &resolver_domain.MockResolver{
		GetBaseDirFunc: func() string {
			return "/path/to/project"
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		GetBaseDirFunc: func() string {
			t.Fatal("Second resolver's GetBaseDir should not be called")
			return ""
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	baseDir := chainedResolver.GetBaseDir()
	assert.Equal(t, "/path/to/project", baseDir)
}

func TestChainedResolver_ConvertEntryPointPathToManifestKey_DelegatesToFirst(t *testing.T) {
	firstResolver := &resolver_domain.MockResolver{
		ConvertEntryPointPathToManifestKeyFunc: func(entryPointPath string) string {
			return "pages/index.pk"
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		ConvertEntryPointPathToManifestKeyFunc: func(entryPointPath string) string {
			t.Fatal("Second resolver's ConvertEntryPointPathToManifestKey should not be called")
			return ""
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	key := chainedResolver.ConvertEntryPointPathToManifestKey("myproject/pages/index.pk")
	assert.Equal(t, "pages/index.pk", key)
}

func TestChainedResolver_ResolvePKPath_LocalPrecedence(t *testing.T) {
	ctx := context.Background()

	localResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			return "/local/project/components/button.pk", nil
		},
	}

	externalResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			t.Fatal("External resolver should not be called when local resolver succeeds")
			return "/gomodcache/components/button.pk", nil
		},
	}

	chainedResolver := NewChainedResolver(localResolver, externalResolver)

	result, err := chainedResolver.ResolvePKPath(ctx, "myproject/components/button.pk", "")
	require.NoError(t, err)
	assert.Equal(t, "/local/project/components/button.pk", result)
}

func TestChainedResolver_IntegrationWithRealResolvers(t *testing.T) {
	ctx := context.Background()

	localResolver := NewLocalModuleResolver(".")
	_ = localResolver.DetectLocalModule(ctx)

	cacheResolver := NewGoModuleCacheResolver()

	chainedResolver := NewChainedResolver(localResolver, cacheResolver)

	var _ resolver_domain.ResolverPort = chainedResolver

	_ = chainedResolver.GetModuleName()
	_ = chainedResolver.GetBaseDir()
	_ = chainedResolver.ConvertEntryPointPathToManifestKey("test/path.pk")
}

func BenchmarkChainedResolver_FirstResolverHit(b *testing.B) {
	ctx := context.Background()

	firstResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			return "/path/to/component.pk", nil
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			return "", errors.New("should not be called")
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	b.ResetTimer()
	for b.Loop() {
		_, _ = chainedResolver.ResolvePKPath(ctx, "test/component.pk", "")
	}
}

func BenchmarkChainedResolver_SecondResolverHit(b *testing.B) {
	ctx := context.Background()

	firstResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			return "", errors.New("not found")
		},
	}

	secondResolver := &resolver_domain.MockResolver{
		ResolvePKPathFunc: func(ctx context.Context, importPath string, containingFilePath string) (string, error) {
			return "/path/to/component.pk", nil
		},
	}

	chainedResolver := NewChainedResolver(firstResolver, secondResolver)

	b.ResetTimer()
	for b.Loop() {
		_, _ = chainedResolver.ResolvePKPath(ctx, "test/component.pk", "")
	}
}
