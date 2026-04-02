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

package monitoring_domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/provider/provider_domain"
)

type mockDescriptor struct {
	detailErr    error
	detail       *provider_domain.ProviderDetail
	resourceType string
	columns      []provider_domain.ColumnDefinition
	providers    []provider_domain.ProviderListEntry
}

func (m *mockDescriptor) ResourceType() string {
	return m.resourceType
}

func (m *mockDescriptor) ResourceListColumns() []provider_domain.ColumnDefinition {
	return m.columns
}

func (m *mockDescriptor) ResourceListProviders(_ context.Context) []provider_domain.ProviderListEntry {
	return m.providers
}

func (m *mockDescriptor) ResourceDescribeProvider(_ context.Context, _ string) (*provider_domain.ProviderDetail, error) {
	if m.detailErr != nil {
		return nil, m.detailErr
	}
	return m.detail, nil
}

type mockSubResourceDescriptor struct {
	mockDescriptor
	subResourceErr     error
	subResourceName    string
	subResourceColumns []provider_domain.ColumnDefinition
	subResources       []provider_domain.ProviderListEntry
}

func (m *mockSubResourceDescriptor) ResourceSubResourceName() string {
	return m.subResourceName
}

func (m *mockSubResourceDescriptor) ResourceSubResourceColumns() []provider_domain.ColumnDefinition {
	return m.subResourceColumns
}

func (m *mockSubResourceDescriptor) ResourceListSubResources(_ context.Context, _ string) ([]provider_domain.ProviderListEntry, error) {
	if m.subResourceErr != nil {
		return nil, m.subResourceErr
	}
	return m.subResources, nil
}

type mockTypeDescriptor struct {
	typeDetail *provider_domain.ProviderDetail
	mockDescriptor
}

func (m *mockTypeDescriptor) ResourceDescribeType(_ context.Context) *provider_domain.ProviderDetail {
	return m.typeDetail
}

func TestNewProviderInfoAggregator_IsEmpty(t *testing.T) {
	agg := NewProviderInfoAggregator()

	assert.False(t, agg.HasDescriptors())
	assert.Empty(t, agg.ListResourceTypes(context.Background()))
}

func TestProviderInfoAggregator_RegisterAndHasDescriptors(t *testing.T) {
	agg := NewProviderInfoAggregator()

	agg.Register(&mockDescriptor{resourceType: "cache"})

	assert.True(t, agg.HasDescriptors())
}

func TestProviderInfoAggregator_ListResourceTypes_Sorted(t *testing.T) {
	agg := NewProviderInfoAggregator()

	agg.Register(&mockDescriptor{resourceType: "storage"})
	agg.Register(&mockDescriptor{resourceType: "cache"})
	agg.Register(&mockDescriptor{resourceType: "email"})

	types := agg.ListResourceTypes(context.Background())
	assert.Equal(t, []string{"cache", "email", "storage"}, types)
}

func TestProviderInfoAggregator_ListProviders_UnknownType(t *testing.T) {
	agg := NewProviderInfoAggregator()

	_, err := agg.ListProviders(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestProviderInfoAggregator_ListProviders_Success(t *testing.T) {
	columns := []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "type"},
	}
	providers := []provider_domain.ProviderListEntry{
		{Name: "redis", Values: map[string]string{"name": "redis", "type": "memory"}},
	}

	agg := NewProviderInfoAggregator()
	agg.Register(&mockDescriptor{
		resourceType: "cache",
		columns:      columns,
		providers:    providers,
	})

	result, err := agg.ListProviders(context.Background(), "cache")
	require.NoError(t, err)
	assert.Equal(t, columns, result.Columns)
	assert.Equal(t, providers, result.Rows)
}

func TestProviderInfoAggregator_DescribeProvider_UnknownType(t *testing.T) {
	agg := NewProviderInfoAggregator()

	_, err := agg.DescribeProvider(context.Background(), "nonexistent", "name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestProviderInfoAggregator_DescribeProvider_Success(t *testing.T) {
	detail := &provider_domain.ProviderDetail{
		Name: "redis",
		Sections: []provider_domain.InfoSection{
			{Title: "Config", Entries: []provider_domain.InfoEntry{{Key: "host", Value: "localhost"}}},
		},
	}

	agg := NewProviderInfoAggregator()
	agg.Register(&mockDescriptor{
		resourceType: "cache",
		detail:       detail,
	})

	result, err := agg.DescribeProvider(context.Background(), "cache", "redis")
	require.NoError(t, err)
	assert.Equal(t, detail, result)
}

func TestProviderInfoAggregator_ListSubResources_NotSupported(t *testing.T) {
	agg := NewProviderInfoAggregator()
	agg.Register(&mockDescriptor{resourceType: "email"})

	_, err := agg.ListSubResources(context.Background(), "email", "smtp")
	assert.ErrorIs(t, err, provider_domain.ErrNoSubResources)
}

func TestProviderInfoAggregator_ListSubResources_UnknownType(t *testing.T) {
	agg := NewProviderInfoAggregator()

	_, err := agg.ListSubResources(context.Background(), "nonexistent", "provider")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestProviderInfoAggregator_ListSubResources_Success(t *testing.T) {
	subCols := []provider_domain.ColumnDefinition{
		{Header: "NAMESPACE", Key: "namespace"},
	}
	subRows := []provider_domain.ProviderListEntry{
		{Name: "default", Values: map[string]string{"namespace": "default"}},
	}

	agg := NewProviderInfoAggregator()
	agg.Register(&mockSubResourceDescriptor{
		mockDescriptor:     mockDescriptor{resourceType: "cache"},
		subResourceName:    "namespaces",
		subResourceColumns: subCols,
		subResources:       subRows,
	})

	result, err := agg.ListSubResources(context.Background(), "cache", "redis")
	require.NoError(t, err)
	assert.Equal(t, subCols, result.Columns)
	assert.Equal(t, subRows, result.Rows)
	assert.Equal(t, "namespaces", result.SubResourceName)
}

func TestProviderInfoAggregator_DescribeResourceType_NotSupported(t *testing.T) {
	agg := NewProviderInfoAggregator()
	agg.Register(&mockDescriptor{resourceType: "email"})

	_, err := agg.DescribeResourceType(context.Background(), "email")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support type-level describe")
}

func TestProviderInfoAggregator_DescribeResourceType_UnknownType(t *testing.T) {
	agg := NewProviderInfoAggregator()

	_, err := agg.DescribeResourceType(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestProviderInfoAggregator_DescribeResourceType_Success(t *testing.T) {
	typeDetail := &provider_domain.ProviderDetail{
		Name: "cache",
		Sections: []provider_domain.InfoSection{
			{Title: "Overview", Entries: []provider_domain.InfoEntry{{Key: "providers", Value: "3"}}},
		},
	}

	agg := NewProviderInfoAggregator()
	agg.Register(&mockTypeDescriptor{
		mockDescriptor: mockDescriptor{resourceType: "cache"},
		typeDetail:     typeDetail,
	})

	result, err := agg.DescribeResourceType(context.Background(), "cache")
	require.NoError(t, err)
	assert.Equal(t, typeDetail, result)
}

func TestProviderInfoAggregator_RegisterReplacesExisting(t *testing.T) {
	agg := NewProviderInfoAggregator()

	agg.Register(&mockDescriptor{
		resourceType: "cache",
		columns:      []provider_domain.ColumnDefinition{{Header: "OLD", Key: "old"}},
	})

	newColumns := []provider_domain.ColumnDefinition{{Header: "NEW", Key: "new"}}
	agg.Register(&mockDescriptor{
		resourceType: "cache",
		columns:      newColumns,
	})

	result, err := agg.ListProviders(context.Background(), "cache")
	require.NoError(t, err)
	assert.Equal(t, newColumns, result.Columns)
}
