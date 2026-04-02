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

package cache_domain

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"slices"

	"piko.sh/piko/internal/provider/provider_domain"
)

// valueUnknown is the fallback display value for unavailable counts.
const valueUnknown = "unknown"

var _ provider_domain.ResourceDescriptor = (*service)(nil)
var _ provider_domain.SubResourceDescriptor = (*service)(nil)
var _ provider_domain.ResourceTypeDescriptor = (*service)(nil)

// ResourceType returns the CLI resource name for the cache hexagon.
//
// Returns string which is "cache".
func (*service) ResourceType() string {
	return "cache"
}

// ResourceListColumns returns column definitions for the cache provider list
// table.
//
// Returns []provider_domain.ColumnDefinition which describes the NAME and TYPE
// columns.
func (*service) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "type"},
	}
}

// ResourceListProviders returns all registered cache providers as list rows.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per
// provider.
//
// Safe for concurrent use. Uses a read lock to access the provider map.
func (s *service) ResourceListProviders(_ context.Context) []provider_domain.ProviderListEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := slices.Sorted(maps.Keys(s.providers))

	entries := make([]provider_domain.ProviderListEntry, len(names))
	for i, name := range names {
		providerType := valueUnknown
		if p, ok := s.providers[name]; ok {
			if meta, ok := p.(provider_domain.ProviderMetadata); ok {
				providerType = meta.GetProviderType()
			}
		}

		entries[i] = provider_domain.ProviderListEntry{
			Name:      name,
			IsDefault: name == s.defaultProvider,
			Values: map[string]string{
				"name": name,
				"type": providerType,
			},
		}
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a single named
// cache provider.
//
// Takes name (string) which identifies the provider to describe.
//
// Returns *provider_domain.ProviderDetail which contains the structured
// sections.
// Returns error when the named provider is not found.
//
// Safe for concurrent use; reads are protected by a read lock.
func (s *service) ResourceDescribeProvider(_ context.Context, name string) (*provider_domain.ProviderDetail, error) {
	s.mu.RLock()
	providerAny, ok := s.providers[name]
	isDefault := name == s.defaultProvider
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	sections := []provider_domain.InfoSection{
		cacheOverviewSection(name, providerAny, isDefault),
	}
	sections = appendCacheConfigSection(sections, providerAny)
	sections = appendCacheNamespacesSection(sections, providerAny)

	return &provider_domain.ProviderDetail{
		Name:     name,
		Sections: sections,
	}, nil
}

// ResourceSubResourceName returns the display name for cache sub-resources.
//
// Returns string which is "namespaces".
func (*service) ResourceSubResourceName() string {
	return "namespaces"
}

// ResourceSubResourceColumns returns column definitions for the namespace
// sub-resource table.
//
// Returns []provider_domain.ColumnDefinition which describes the NAMESPACE and
// ENTRIES columns.
func (*service) ResourceSubResourceColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAMESPACE", Key: "namespace"},
		{Header: "ENTRIES", Key: "entries"},
	}
}

// ResourceListSubResources returns all namespaces for a named cache provider.
//
// Takes providerName (string) which identifies the provider.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per
// namespace.
// Returns error when the provider is not found or does not support namespace
// listing.
//
// Safe for concurrent use.
func (s *service) ResourceListSubResources(_ context.Context, providerName string) ([]provider_domain.ProviderListEntry, error) {
	s.mu.RLock()
	providerAny, ok := s.providers[providerName]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", providerName)
	}

	lister, ok := providerAny.(interface {
		ListNamespaces() map[string]any
	})
	if !ok {
		return nil, provider_domain.ErrNoSubResources
	}

	namespaces := lister.ListNamespaces()
	if len(namespaces) == 0 {
		return nil, nil
	}

	names := slices.Sorted(maps.Keys(namespaces))

	entries := make([]provider_domain.ProviderListEntry, len(names))
	for i, n := range names {
		entryCount := valueUnknown
		if sizer, ok := namespaces[n].(interface{ EstimatedSize() int }); ok {
			entryCount = fmt.Sprintf("%d", sizer.EstimatedSize())
		}

		entries[i] = provider_domain.ProviderListEntry{
			Name: n,
			Values: map[string]string{
				"namespace": n,
				"entries":   entryCount,
			},
		}
	}

	return entries, nil
}

// ResourceDescribeType returns a service-level overview of the cache system.
//
// Returns *provider_domain.ProviderDetail which contains provider count and
// default provider information.
//
// Safe for concurrent use. Uses a read lock to access provider state.
func (s *service) ResourceDescribeType(_ context.Context) *provider_domain.ProviderDetail {
	s.mu.RLock()
	providerCount := len(s.providers)
	defaultProvider := s.defaultProvider
	s.mu.RUnlock()

	return &provider_domain.ProviderDetail{
		Name: "cache",
		Sections: []provider_domain.InfoSection{
			{
				Title: "Overview",
				Entries: []provider_domain.InfoEntry{
					{Key: "Resource Type", Value: "cache"},
					{Key: "Provider Count", Value: fmt.Sprintf("%d", providerCount)},
					{Key: "Default Provider", Value: defaultProvider},
				},
			},
		},
	}
}

// cacheOverviewSection builds the overview info section for a cache provider.
//
// Takes name (string) which is the provider name.
// Takes providerAny (any) which is the provider instance to inspect.
// Takes isDefault (bool) which indicates whether this is the default provider.
//
// Returns provider_domain.InfoSection which contains the provider overview.
func cacheOverviewSection(name string, providerAny any, isDefault bool) provider_domain.InfoSection {
	providerType := valueUnknown
	if meta, ok := providerAny.(provider_domain.ProviderMetadata); ok {
		providerType = meta.GetProviderType()
	}

	defaultString := "false"
	if isDefault {
		defaultString = "true"
	}

	return provider_domain.InfoSection{
		Title: "Overview",
		Entries: []provider_domain.InfoEntry{
			{Key: "Name", Value: name},
			{Key: "Type", Value: providerType},
			{Key: "Default", Value: defaultString},
		},
	}
}

// appendCacheConfigSection appends a Configuration section when the provider
// exposes metadata.
//
// Takes sections ([]provider_domain.InfoSection) which is the current list.
// Takes providerAny (any) which is the provider instance to inspect.
//
// Returns []provider_domain.InfoSection which is the updated section list.
func appendCacheConfigSection(sections []provider_domain.InfoSection, providerAny any) []provider_domain.InfoSection {
	meta, ok := providerAny.(provider_domain.ProviderMetadata)
	if !ok {
		return sections
	}
	metadata := meta.GetProviderMetadata()
	if len(metadata) == 0 {
		return sections
	}

	entries := make([]provider_domain.InfoEntry, 0, len(metadata))
	for k, v := range metadata {
		entries = append(entries, provider_domain.InfoEntry{
			Key:   k,
			Value: fmt.Sprintf("%v", v),
		})
	}
	slices.SortFunc(entries, func(a, b provider_domain.InfoEntry) int {
		return cmp.Compare(a.Key, b.Key)
	})
	return append(sections, provider_domain.InfoSection{
		Title:   "Configuration",
		Entries: entries,
	})
}

// appendCacheNamespacesSection appends a Namespaces section when the provider
// supports namespace listing and has entries.
//
// Takes sections ([]provider_domain.InfoSection) which is the current list.
// Takes providerAny (any) which is the provider instance to inspect.
//
// Returns []provider_domain.InfoSection which is the updated section list.
func appendCacheNamespacesSection(sections []provider_domain.InfoSection, providerAny any) []provider_domain.InfoSection {
	lister, ok := providerAny.(interface {
		ListNamespaces() map[string]any
	})
	if !ok {
		return sections
	}

	namespaces := lister.ListNamespaces()
	if len(namespaces) == 0 {
		return sections
	}

	nsNames := slices.Sorted(maps.Keys(namespaces))
	nsEntries := make([]provider_domain.InfoEntry, len(nsNames))
	for i, n := range nsNames {
		entryCount := valueUnknown
		if sizer, ok := namespaces[n].(interface{ EstimatedSize() int }); ok {
			entryCount = fmt.Sprintf("%d entries", sizer.EstimatedSize())
		}
		nsEntries[i] = provider_domain.InfoEntry{
			Key:   n,
			Value: entryCount,
		}
	}

	return append(sections, provider_domain.InfoSection{
		Title:   fmt.Sprintf("Namespaces (%d)", len(nsNames)),
		Entries: nsEntries,
	})
}
