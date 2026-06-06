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

package querier_domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/querier/querier_dto"
)

type stubCatalogueProvider struct {
	catalogue   *querier_dto.Catalogue
	diagnostics []querier_dto.SourceError
	buildError  error
	calls       int
}

func (s *stubCatalogueProvider) BuildCatalogue(_ context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
	s.calls++
	return s.catalogue, s.diagnostics, s.buildError
}

func TestCompositeCatalogueProvider_RepropagatesDataAccessAcrossProviders(t *testing.T) {
	t.Parallel()

	caller := &querier_dto.FunctionSignature{
		Name:            "caller",
		DataAccess:      querier_dto.DataAccessReadOnly,
		CalledFunctions: []string{"writer"},
	}
	upstream := &stubCatalogueProvider{catalogue: &querier_dto.Catalogue{
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name:      "public",
				Functions: map[string][]*querier_dto.FunctionSignature{"caller": {caller}},
			},
		},
	}}

	writer := &querier_dto.FunctionSignature{
		Name:       "writer",
		DataAccess: querier_dto.DataAccessModifiesData,
	}
	downstream := &stubCatalogueProvider{catalogue: &querier_dto.Catalogue{
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name:      "public",
				Functions: map[string][]*querier_dto.FunctionSignature{"writer": {writer}},
			},
		},
	}}

	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{upstream, downstream})
	catalogue, _, err := provider.BuildCatalogue(context.Background())
	require.NoError(t, err)

	mergedCaller := catalogue.Schemas["public"].Functions["caller"]
	require.Len(t, mergedCaller, 1)
	assert.Equal(t, querier_dto.DataAccessModifiesData, mergedCaller[0].DataAccess,
		"caller must be re-promoted to modifies-data after the cross-provider merge")
}

func TestCompositeCatalogueProvider_EmptyChainReturnsEmptyCatalogue(t *testing.T) {
	t.Parallel()

	provider := NewCompositeCatalogueProvider(nil)
	catalogue, diagnostics, err := provider.BuildCatalogue(context.Background())

	require.NoError(t, err)
	require.NotNil(t, catalogue)
	assert.Empty(t, catalogue.Schemas)
	assert.Empty(t, catalogue.Extensions)
	assert.Empty(t, diagnostics)
}

func TestCompositeCatalogueProvider_SingleProviderPassesThrough(t *testing.T) {
	t.Parallel()

	source := &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name:           "public",
				Tables:         map[string]*querier_dto.Table{"users": {Name: "users"}},
				Views:          map[string]*querier_dto.View{},
				Enums:          map[string]*querier_dto.Enum{},
				Functions:      map[string][]*querier_dto.FunctionSignature{},
				CompositeTypes: map[string]*querier_dto.CompositeType{},
				Sequences:      map[string]*querier_dto.Sequence{},
			},
		},
		Extensions: map[string]struct{}{"pgcrypto": {}},
	}
	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{
		&stubCatalogueProvider{catalogue: source},
	})

	catalogue, diagnostics, err := provider.BuildCatalogue(context.Background())

	require.NoError(t, err)
	require.NotNil(t, catalogue)
	assert.Equal(t, "public", catalogue.DefaultSchema)
	require.Contains(t, catalogue.Schemas, "public")
	require.Contains(t, catalogue.Schemas["public"].Tables, "users")
	require.Contains(t, catalogue.Extensions, "pgcrypto")
	assert.Empty(t, diagnostics)
}

func TestCompositeCatalogueProvider_MergesTablesAcrossProviders(t *testing.T) {
	t.Parallel()

	first := &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name:           "public",
				Tables:         map[string]*querier_dto.Table{"users": {Name: "users"}},
				Views:          map[string]*querier_dto.View{},
				Enums:          map[string]*querier_dto.Enum{},
				Functions:      map[string][]*querier_dto.FunctionSignature{},
				CompositeTypes: map[string]*querier_dto.CompositeType{},
				Sequences:      map[string]*querier_dto.Sequence{},
			},
		},
		Extensions: map[string]struct{}{},
	}
	second := &querier_dto.Catalogue{
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name:           "public",
				Tables:         map[string]*querier_dto.Table{"orders": {Name: "orders"}},
				Views:          map[string]*querier_dto.View{},
				Enums:          map[string]*querier_dto.Enum{},
				Functions:      map[string][]*querier_dto.FunctionSignature{},
				CompositeTypes: map[string]*querier_dto.CompositeType{},
				Sequences:      map[string]*querier_dto.Sequence{},
			},
		},
		Extensions: map[string]struct{}{},
	}

	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{
		&stubCatalogueProvider{catalogue: first},
		&stubCatalogueProvider{catalogue: second},
	})

	catalogue, _, err := provider.BuildCatalogue(context.Background())

	require.NoError(t, err)
	public := catalogue.Schemas["public"]
	require.Contains(t, public.Tables, "users")
	require.Contains(t, public.Tables, "orders")
}

func TestCompositeCatalogueProvider_LaterProviderWinsOnTableCollision(t *testing.T) {
	t.Parallel()

	winner := &querier_dto.Table{Name: "users", Schema: "later"}
	first := schemaWithTable("users", &querier_dto.Table{Name: "users", Schema: "earlier"})
	second := schemaWithTable("users", winner)

	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{
		&stubCatalogueProvider{catalogue: first},
		&stubCatalogueProvider{catalogue: second},
	})

	catalogue, _, err := provider.BuildCatalogue(context.Background())

	require.NoError(t, err)
	assert.Same(t, winner, catalogue.Schemas["public"].Tables["users"])
}

func TestCompositeCatalogueProvider_FunctionOverloadsDedupeBySignature(t *testing.T) {
	t.Parallel()

	intArg := querier_dto.FunctionArgument{Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger}}
	textArg := querier_dto.FunctionArgument{Type: querier_dto.SQLType{Category: querier_dto.TypeCategoryText}}

	earlyInt := &querier_dto.FunctionSignature{Name: "format", Arguments: []querier_dto.FunctionArgument{intArg}}
	earlyText := &querier_dto.FunctionSignature{Name: "format", Arguments: []querier_dto.FunctionArgument{textArg}}
	lateInt := &querier_dto.FunctionSignature{Name: "format", Arguments: []querier_dto.FunctionArgument{intArg}}

	first := schemaWithFunctions("format", earlyInt, earlyText)
	second := schemaWithFunctions("format", lateInt)

	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{
		&stubCatalogueProvider{catalogue: first},
		&stubCatalogueProvider{catalogue: second},
	})

	catalogue, _, err := provider.BuildCatalogue(context.Background())
	require.NoError(t, err)

	overloads := catalogue.Schemas["public"].Functions["format"]
	require.Len(t, overloads, 2, "duplicate integer overload must be deduped")
	assert.Same(t, lateInt, overloads[0], "later integer overload must replace earlier one")
	assert.Same(t, earlyText, overloads[1], "text overload from first provider must survive")
}

func TestCompositeCatalogueProvider_UnionsExtensions(t *testing.T) {
	t.Parallel()

	first := &querier_dto.Catalogue{
		Schemas:    map[string]*querier_dto.Schema{},
		Extensions: map[string]struct{}{"pgcrypto": {}},
	}
	second := &querier_dto.Catalogue{
		Schemas:    map[string]*querier_dto.Schema{},
		Extensions: map[string]struct{}{"citext": {}},
	}

	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{
		&stubCatalogueProvider{catalogue: first},
		&stubCatalogueProvider{catalogue: second},
	})

	catalogue, _, err := provider.BuildCatalogue(context.Background())

	require.NoError(t, err)
	assert.Contains(t, catalogue.Extensions, "pgcrypto")
	assert.Contains(t, catalogue.Extensions, "citext")
}

func TestCompositeCatalogueProvider_AccumulatesDiagnostics(t *testing.T) {
	t.Parallel()

	first := &stubCatalogueProvider{
		catalogue: &querier_dto.Catalogue{Schemas: map[string]*querier_dto.Schema{}, Extensions: map[string]struct{}{}},
		diagnostics: []querier_dto.SourceError{
			{Message: "from provider 0", Severity: querier_dto.SeverityWarning},
		},
	}
	second := &stubCatalogueProvider{
		catalogue: &querier_dto.Catalogue{Schemas: map[string]*querier_dto.Schema{}, Extensions: map[string]struct{}{}},
		diagnostics: []querier_dto.SourceError{
			{Message: "from provider 1", Severity: querier_dto.SeverityWarning},
		},
	}

	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{first, second})

	_, diagnostics, err := provider.BuildCatalogue(context.Background())

	require.NoError(t, err)
	require.Len(t, diagnostics, 2)
	assert.Equal(t, "from provider 0", diagnostics[0].Message)
	assert.Equal(t, "from provider 1", diagnostics[1].Message)
}

func TestCompositeCatalogueProvider_StopsOnFatalError(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	first := &stubCatalogueProvider{
		catalogue: &querier_dto.Catalogue{Schemas: map[string]*querier_dto.Schema{}, Extensions: map[string]struct{}{}},
		diagnostics: []querier_dto.SourceError{
			{Message: "warning before failure"},
		},
	}
	failing := &stubCatalogueProvider{buildError: sentinel}
	last := &stubCatalogueProvider{
		catalogue: &querier_dto.Catalogue{Schemas: map[string]*querier_dto.Schema{}, Extensions: map[string]struct{}{}},
	}

	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{first, failing, last})

	catalogue, diagnostics, err := provider.BuildCatalogue(context.Background())

	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
	assert.Nil(t, catalogue)
	require.Len(t, diagnostics, 1, "diagnostics produced before the failure must survive")
	assert.Equal(t, 0, last.calls, "providers past the failing one must not be invoked")
}

func TestCompositeCatalogueProvider_HonoursCancellationBetweenProviders(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	first := &stubCatalogueProvider{
		catalogue: &querier_dto.Catalogue{Schemas: map[string]*querier_dto.Schema{}, Extensions: map[string]struct{}{}},
	}
	second := &stubCatalogueProvider{
		catalogue: &querier_dto.Catalogue{Schemas: map[string]*querier_dto.Schema{}, Extensions: map[string]struct{}{}},
	}
	cancelling := &cancellingProvider{cancel: cancel, inner: first}
	provider := NewCompositeCatalogueProvider([]CatalogueProviderPort{cancelling, second})

	catalogue, _, err := provider.BuildCatalogue(ctx)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, catalogue)
	assert.Equal(t, 0, second.calls, "downstream provider must not be invoked after cancellation")
}

type cancellingProvider struct {
	cancel context.CancelFunc
	inner  CatalogueProviderPort
}

func (c *cancellingProvider) BuildCatalogue(ctx context.Context) (*querier_dto.Catalogue, []querier_dto.SourceError, error) {
	catalogue, diagnostics, err := c.inner.BuildCatalogue(ctx)
	c.cancel()
	return catalogue, diagnostics, err
}

func schemaWithTable(name string, table *querier_dto.Table) *querier_dto.Catalogue {
	return &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name:           "public",
				Tables:         map[string]*querier_dto.Table{name: table},
				Views:          map[string]*querier_dto.View{},
				Enums:          map[string]*querier_dto.Enum{},
				Functions:      map[string][]*querier_dto.FunctionSignature{},
				CompositeTypes: map[string]*querier_dto.CompositeType{},
				Sequences:      map[string]*querier_dto.Sequence{},
			},
		},
		Extensions: map[string]struct{}{},
	}
}

func schemaWithFunctions(name string, signatures ...*querier_dto.FunctionSignature) *querier_dto.Catalogue {
	return &querier_dto.Catalogue{
		DefaultSchema: "public",
		Schemas: map[string]*querier_dto.Schema{
			"public": {
				Name:           "public",
				Tables:         map[string]*querier_dto.Table{},
				Views:          map[string]*querier_dto.View{},
				Enums:          map[string]*querier_dto.Enum{},
				Functions:      map[string][]*querier_dto.FunctionSignature{name: signatures},
				CompositeTypes: map[string]*querier_dto.CompositeType{},
				Sequences:      map[string]*querier_dto.Sequence{},
			},
		},
		Extensions: map[string]struct{}{},
	}
}
