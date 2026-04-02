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

package collection_domain

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"piko.sh/piko/internal/provider/provider_domain"
)

var _ provider_domain.ResourceDescriptor = (*collectionService)(nil)

// ResourceType returns the CLI resource name for the collection hexagon.
//
// Returns string which is "collection".
func (*collectionService) ResourceType() string {
	return "collection"
}

// ResourceListColumns returns column definitions for the collection provider
// list table.
//
// Returns []provider_domain.ColumnDefinition which describes the NAME and TYPE
// columns.
func (*collectionService) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "type"},
	}
}

// ResourceListProviders returns all registered collection providers as list
// rows.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per
// provider.
func (s *collectionService) ResourceListProviders(_ context.Context) []provider_domain.ProviderListEntry {
	names := s.registry.List()
	slices.Sort(names)

	entries := make([]provider_domain.ProviderListEntry, 0, len(names))
	for _, name := range names {
		provider, ok := s.registry.Get(name)
		if !ok {
			continue
		}

		entries = append(entries, provider_domain.ProviderListEntry{
			Name:      name,
			IsDefault: false,
			Values: map[string]string{
				"name": name,
				"type": string(provider.Type()),
			},
		})
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a single named
// collection provider.
//
// Takes name (string) which identifies the provider to describe.
//
// Returns *provider_domain.ProviderDetail which contains the structured
// sections.
// Returns error when the named provider is not found.
func (s *collectionService) ResourceDescribeProvider(_ context.Context, name string) (*provider_domain.ProviderDetail, error) {
	provider, ok := s.registry.Get(name)
	if !ok {
		return nil, fmt.Errorf("provider '%s' not found", name)
	}

	sections := []provider_domain.InfoSection{
		{
			Title: "Overview",
			Entries: []provider_domain.InfoEntry{
				{Key: "Name", Value: provider.Name()},
				{Key: "Type", Value: string(provider.Type())},
			},
		},
	}

	if meta, ok := provider.(provider_domain.ProviderMetadata); ok {
		metadata := meta.GetProviderMetadata()
		if len(metadata) > 0 {
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
			sections = append(sections, provider_domain.InfoSection{
				Title:   "Configuration",
				Entries: entries,
			})
		}
	}

	return &provider_domain.ProviderDetail{
		Name:     name,
		Sections: sections,
	}, nil
}
