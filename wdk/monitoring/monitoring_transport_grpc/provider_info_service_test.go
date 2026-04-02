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

package monitoring_transport_grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/monitoring/monitoring_domain"
	"piko.sh/piko/internal/provider/provider_domain"
	pb "piko.sh/piko/wdk/monitoring/monitoring_api/gen"
)

type mockProviderInfoInspector struct {
	listProvidersErr error
	describeErr      error
	subResourcesErr  error
	describeTypeErr  error
	listProvidersRes *monitoring_domain.ProviderListResult
	describeRes      *provider_domain.ProviderDetail
	subResourcesRes  *monitoring_domain.ProviderListResult
	describeTypeRes  *provider_domain.ProviderDetail
	resourceTypes    []string
}

func (m *mockProviderInfoInspector) ListResourceTypes(_ context.Context) []string {
	return m.resourceTypes
}

func (m *mockProviderInfoInspector) ListProviders(_ context.Context, _ string) (*monitoring_domain.ProviderListResult, error) {
	return m.listProvidersRes, m.listProvidersErr
}

func (m *mockProviderInfoInspector) DescribeProvider(_ context.Context, _, _ string) (*provider_domain.ProviderDetail, error) {
	return m.describeRes, m.describeErr
}

func (m *mockProviderInfoInspector) ListSubResources(_ context.Context, _, _ string) (*monitoring_domain.ProviderListResult, error) {
	return m.subResourcesRes, m.subResourcesErr
}

func (m *mockProviderInfoInspector) DescribeResourceType(_ context.Context, _ string) (*provider_domain.ProviderDetail, error) {
	return m.describeTypeRes, m.describeTypeErr
}

var _ monitoring_domain.ProviderInfoInspector = (*mockProviderInfoInspector)(nil)

func TestNewProviderInfoService(t *testing.T) {
	t.Parallel()

	service := NewProviderInfoService(nil)
	require.NotNil(t, service)

	svc2 := NewProviderInfoService(&mockProviderInfoInspector{})
	require.NotNil(t, svc2)
}

func TestProviderInfoService_ListResourceTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector     monitoring_domain.ProviderInfoInspector
		name          string
		expectedTypes []string
	}{
		{
			name:          "nil inspector returns empty",
			inspector:     nil,
			expectedTypes: nil,
		},
		{
			name: "returns resource types",
			inspector: &mockProviderInfoInspector{
				resourceTypes: []string{"email", "storage", "cache"},
			},
			expectedTypes: []string{"email", "storage", "cache"},
		},
		{
			name: "returns empty list",
			inspector: &mockProviderInfoInspector{
				resourceTypes: []string{},
			},
			expectedTypes: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewProviderInfoService(tc.inspector)
			response, err := service.ListResourceTypes(context.Background(), &pb.ListResourceTypesRequest{})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedTypes, response.ResourceTypes)
		})
	}
}

func TestProviderInfoService_ListProviders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector   monitoring_domain.ProviderInfoInspector
		name        string
		colCount    int
		rowCount    int
		expectError bool
	}{
		{
			name:        "nil inspector returns error",
			inspector:   nil,
			expectError: true,
		},
		{
			name: "returns providers with columns and rows",
			inspector: &mockProviderInfoInspector{
				listProvidersRes: &monitoring_domain.ProviderListResult{
					Columns: []provider_domain.ColumnDefinition{
						{Header: "NAME", Key: "name", WideOnly: false},
						{Header: "HOST", Key: "host", WideOnly: true},
					},
					Rows: []provider_domain.ProviderListEntry{
						{
							Name:      "smtp-provider",
							IsDefault: true,
							Values:    map[string]string{"name": "smtp-provider", "host": "smtp.example.com"},
						},
						{
							Name:      "ses-provider",
							IsDefault: false,
							Values:    map[string]string{"name": "ses-provider", "host": "ses.amazonaws.com"},
						},
					},
				},
			},
			colCount:    2,
			rowCount:    2,
			expectError: false,
		},
		{
			name: "propagates error",
			inspector: &mockProviderInfoInspector{
				listProvidersErr: errors.New("unknown resource type"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewProviderInfoService(tc.inspector)
			response, err := service.ListProviders(context.Background(), &pb.ListProvidersRequest{
				ResourceType: "email",
			})

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, response.Columns, tc.colCount)
			require.Len(t, response.Rows, tc.rowCount)

			if tc.colCount > 0 {
				assert.Equal(t, "NAME", response.Columns[0].Header)
				assert.Equal(t, "name", response.Columns[0].Key)
				assert.False(t, response.Columns[0].WideOnly)

				assert.Equal(t, "HOST", response.Columns[1].Header)
				assert.True(t, response.Columns[1].WideOnly)
			}

			if tc.rowCount > 0 {
				assert.Equal(t, "smtp-provider", response.Rows[0].Name)
				assert.True(t, response.Rows[0].IsDefault)
				assert.Equal(t, "ses-provider", response.Rows[1].Name)
				assert.False(t, response.Rows[1].IsDefault)
			}
		})
	}
}

func TestProviderInfoService_DescribeProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector   monitoring_domain.ProviderInfoInspector
		name        string
		expectError bool
	}{
		{
			name:        "nil inspector returns error",
			inspector:   nil,
			expectError: true,
		},
		{
			name: "returns provider detail",
			inspector: &mockProviderInfoInspector{
				describeRes: &provider_domain.ProviderDetail{
					Name: "smtp-provider",
					Sections: []provider_domain.InfoSection{
						{
							Title: "Configuration",
							Entries: []provider_domain.InfoEntry{
								{Key: "Host", Value: "smtp.example.com"},
								{Key: "Port", Value: "587"},
							},
						},
						{
							Title: "Health",
							Entries: []provider_domain.InfoEntry{
								{Key: "Status", Value: "healthy"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "propagates error",
			inspector: &mockProviderInfoInspector{
				describeErr: errors.New("provider not found"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewProviderInfoService(tc.inspector)
			response, err := service.DescribeProvider(context.Background(), &pb.DescribeProviderRequest{
				ResourceType: "email",
				Name:         "smtp-provider",
			})

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "smtp-provider", response.Name)
			require.Len(t, response.Sections, 2)

			assert.Equal(t, "Configuration", response.Sections[0].Title)
			require.Len(t, response.Sections[0].Entries, 2)
			assert.Equal(t, "Host", response.Sections[0].Entries[0].Key)
			assert.Equal(t, "smtp.example.com", response.Sections[0].Entries[0].Value)
			assert.Equal(t, "Port", response.Sections[0].Entries[1].Key)
			assert.Equal(t, "587", response.Sections[0].Entries[1].Value)

			assert.Equal(t, "Health", response.Sections[1].Title)
			require.Len(t, response.Sections[1].Entries, 1)
			assert.Equal(t, "Status", response.Sections[1].Entries[0].Key)
			assert.Equal(t, "healthy", response.Sections[1].Entries[0].Value)
		})
	}
}

func TestProviderInfoService_ListSubResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector   monitoring_domain.ProviderInfoInspector
		name        string
		expectError bool
	}{
		{
			name:        "nil inspector returns error",
			inspector:   nil,
			expectError: true,
		},
		{
			name: "returns sub-resources",
			inspector: &mockProviderInfoInspector{
				subResourcesRes: &monitoring_domain.ProviderListResult{
					SubResourceName: "namespaces",
					Columns: []provider_domain.ColumnDefinition{
						{Header: "NAMESPACE", Key: "namespace"},
					},
					Rows: []provider_domain.ProviderListEntry{
						{Name: "default", Values: map[string]string{"namespace": "default"}},
						{Name: "sessions", Values: map[string]string{"namespace": "sessions"}},
					},
				},
			},
			expectError: false,
		},
		{
			name: "propagates error",
			inspector: &mockProviderInfoInspector{
				subResourcesErr: errors.New("sub-resources not supported"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewProviderInfoService(tc.inspector)
			response, err := service.ListSubResources(context.Background(), &pb.ListSubResourcesRequest{
				ResourceType: "cache",
				ProviderName: "redis",
			})

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "namespaces", response.SubResourceName)
			require.Len(t, response.Columns, 1)
			assert.Equal(t, "NAMESPACE", response.Columns[0].Header)
			require.Len(t, response.Rows, 2)
			assert.Equal(t, "default", response.Rows[0].Name)
			assert.Equal(t, "sessions", response.Rows[1].Name)
		})
	}
}

func TestProviderInfoService_DescribeResourceType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		inspector   monitoring_domain.ProviderInfoInspector
		name        string
		expectError bool
	}{
		{
			name:        "nil inspector returns error",
			inspector:   nil,
			expectError: true,
		},
		{
			name: "returns resource type detail",
			inspector: &mockProviderInfoInspector{
				describeTypeRes: &provider_domain.ProviderDetail{
					Name: "email",
					Sections: []provider_domain.InfoSection{
						{
							Title: "Overview",
							Entries: []provider_domain.InfoEntry{
								{Key: "Active Provider", Value: "smtp"},
								{Key: "Providers", Value: "2"},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "propagates error",
			inspector: &mockProviderInfoInspector{
				describeTypeErr: errors.New("type not found"),
			},
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := NewProviderInfoService(tc.inspector)
			response, err := service.DescribeResourceType(context.Background(), &pb.DescribeResourceTypeRequest{
				ResourceType: "email",
			})

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, "email", response.Name)
			require.Len(t, response.Sections, 1)
			assert.Equal(t, "Overview", response.Sections[0].Title)
			require.Len(t, response.Sections[0].Entries, 2)
		})
	}
}
