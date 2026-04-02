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
	"fmt"
	"slices"
	"sync"

	"piko.sh/piko/internal/provider/provider_domain"
)

// errFmtUnknownResourceType is the format string for unrecognised resource types.
const errFmtUnknownResourceType = "unknown resource type: %s"

// ProviderInfoAggregator collects ResourceDescriptors from hexagon services
// and implements ProviderInfoInspector for the monitoring gRPC service.
type ProviderInfoAggregator struct {
	// descriptors maps resource type names to their descriptors.
	descriptors map[string]provider_domain.ResourceDescriptor

	// mu guards concurrent access to descriptors.
	mu sync.RWMutex
}

var _ ProviderInfoInspector = (*ProviderInfoAggregator)(nil)

// NewProviderInfoAggregator creates a new empty aggregator.
//
// Returns *ProviderInfoAggregator which is ready for descriptor registration.
func NewProviderInfoAggregator() *ProviderInfoAggregator {
	return &ProviderInfoAggregator{
		descriptors: make(map[string]provider_domain.ResourceDescriptor),
	}
}

// Register adds a ResourceDescriptor to the aggregator. If a descriptor
// with the same ResourceType is already registered, it is replaced.
//
// Takes descriptor (provider_domain.ResourceDescriptor) which provides
// provider information for its resource type.
//
// Safe for concurrent use.
func (a *ProviderInfoAggregator) Register(descriptor provider_domain.ResourceDescriptor) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.descriptors[descriptor.ResourceType()] = descriptor
}

// HasDescriptors reports whether any ResourceDescriptors have been registered.
//
// Returns bool which is true if at least one descriptor is registered.
//
// Safe for concurrent use.
func (a *ProviderInfoAggregator) HasDescriptors() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return len(a.descriptors) > 0
}

// ListResourceTypes returns the names of all registered resource types,
// sorted alphabetically.
//
// Returns []string which contains the resource type names.
//
// Safe for concurrent use.
func (a *ProviderInfoAggregator) ListResourceTypes(_ context.Context) []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	types := make([]string, 0, len(a.descriptors))
	for t := range a.descriptors {
		types = append(types, t)
	}

	slices.Sort(types)

	return types
}

// ListProviders returns column definitions and provider rows for the given
// resource type.
//
// Takes resourceType (string) which identifies which resource to query.
//
// Returns *ProviderListResult which contains columns and rows.
// Returns error when the resource type is not registered.
//
// Safe for concurrent use. Uses a read lock to access the descriptors.
func (a *ProviderInfoAggregator) ListProviders(ctx context.Context, resourceType string) (*ProviderListResult, error) {
	a.mu.RLock()
	descriptor, ok := a.descriptors[resourceType]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf(errFmtUnknownResourceType, resourceType)
	}

	return &ProviderListResult{
		Columns: descriptor.ResourceListColumns(),
		Rows:    descriptor.ResourceListProviders(ctx),
	}, nil
}

// DescribeProvider returns detailed information for a single provider within
// the given resource type.
//
// Takes resourceType (string) which identifies the resource.
// Takes name (string) which identifies the provider.
//
// Returns *provider_domain.ProviderDetail which contains structured sections.
// Returns error when the resource type is not registered or the provider is
// not found.
//
// Safe for concurrent use.
func (a *ProviderInfoAggregator) DescribeProvider(ctx context.Context, resourceType, name string) (*provider_domain.ProviderDetail, error) {
	a.mu.RLock()
	descriptor, ok := a.descriptors[resourceType]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf(errFmtUnknownResourceType, resourceType)
	}

	return descriptor.ResourceDescribeProvider(ctx, name)
}

// ListSubResources returns sub-resources for a named provider. The service
// must implement SubResourceDescriptor; otherwise ErrNoSubResources is
// returned.
//
// Takes resourceType (string) which identifies the resource.
// Takes providerName (string) which identifies the provider.
//
// Returns *ProviderListResult which contains sub-resource columns and rows.
// Returns error when the resource type is not found or the service does not
// support sub-resources.
//
// Safe for concurrent use.
func (a *ProviderInfoAggregator) ListSubResources(ctx context.Context, resourceType, providerName string) (*ProviderListResult, error) {
	a.mu.RLock()
	descriptor, ok := a.descriptors[resourceType]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf(errFmtUnknownResourceType, resourceType)
	}

	sub, ok := descriptor.(provider_domain.SubResourceDescriptor)
	if !ok {
		return nil, provider_domain.ErrNoSubResources
	}

	rows, err := sub.ResourceListSubResources(ctx, providerName)
	if err != nil {
		return nil, fmt.Errorf("listing sub-resources for provider %q of type %q: %w", providerName, resourceType, err)
	}

	return &ProviderListResult{
		Columns:         sub.ResourceSubResourceColumns(),
		Rows:            rows,
		SubResourceName: sub.ResourceSubResourceName(),
	}, nil
}

// DescribeResourceType returns a service-level overview for the given resource
// type. The service must implement ResourceTypeDescriptor.
//
// Takes resourceType (string) which identifies the resource.
//
// Returns *provider_domain.ProviderDetail which contains the overview.
// Returns error when the resource type is not found or the service does not
// support type-level describe.
//
// Safe for concurrent use.
func (a *ProviderInfoAggregator) DescribeResourceType(ctx context.Context, resourceType string) (*provider_domain.ProviderDetail, error) {
	a.mu.RLock()
	descriptor, ok := a.descriptors[resourceType]
	a.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf(errFmtUnknownResourceType, resourceType)
	}

	typed, ok := descriptor.(provider_domain.ResourceTypeDescriptor)
	if !ok {
		return nil, fmt.Errorf("resource type %q does not support type-level describe", resourceType)
	}

	return typed.ResourceDescribeType(ctx), nil
}
