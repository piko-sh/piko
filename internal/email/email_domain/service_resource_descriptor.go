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

package email_domain

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	"piko.sh/piko/internal/provider/provider_domain"
)

// hoursPerDay is the number of hours in a day, used for duration formatting.
const hoursPerDay = 24

var _ provider_domain.ResourceDescriptor = (*service)(nil)

// ResourceType returns the CLI resource name for the email hexagon.
//
// Returns string which is "email".
func (*service) ResourceType() string {
	return "email"
}

// ResourceListColumns returns column definitions for the email provider list
// table.
//
// Returns []provider_domain.ColumnDefinition which describes the NAME, TYPE,
// and REGISTERED columns.
func (*service) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "type"},
		{Header: "REGISTERED", Key: "registered"},
	}
}

// ResourceListProviders returns all registered email providers as list rows.
//
// Returns []provider_domain.ProviderListEntry which contains one entry per
// provider.
func (s *service) ResourceListProviders(ctx context.Context) []provider_domain.ProviderListEntry {
	providers := s.registry.ListProviders(ctx)
	entries := make([]provider_domain.ProviderListEntry, len(providers))

	for i, info := range providers {
		entries[i] = provider_domain.ProviderListEntry{
			Name:      info.Name,
			IsDefault: info.IsDefault,
			Values: map[string]string{
				"name":       info.Name,
				"type":       info.ProviderType,
				"registered": formatRegisteredAge(info.RegisteredAt),
			},
		}
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a single named
// email provider.
//
// Takes name (string) which identifies the provider to describe.
//
// Returns *provider_domain.ProviderDetail which contains the structured
// sections.
// Returns error when the named provider is not found.
func (s *service) ResourceDescribeProvider(ctx context.Context, name string) (*provider_domain.ProviderDetail, error) {
	provider, err := s.registry.GetProvider(ctx, name)
	if err != nil {
		return nil, fmt.Errorf(errProviderNotFoundFmt, name)
	}

	info := findProviderInfo(s.registry.ListProviders(ctx), name)

	sections := []provider_domain.InfoSection{
		buildOverviewSection(info),
	}

	if metaSection, ok := buildMetadataSection(provider); ok {
		sections = append(sections, metaSection)
	}

	return &provider_domain.ProviderDetail{
		Name:     name,
		Sections: sections,
	}, nil
}

// findProviderInfo finds a ProviderInfo by name in a slice.
//
// Takes infos ([]provider_domain.ProviderInfo) which is the slice to search.
// Takes name (string) which is the provider name to find.
//
// Returns provider_domain.ProviderInfo which is the matching provider, or a
// new ProviderInfo with the given name if no match is found.
func findProviderInfo(infos []provider_domain.ProviderInfo, name string) provider_domain.ProviderInfo {
	for _, info := range infos {
		if info.Name == name {
			return info
		}
	}
	return provider_domain.ProviderInfo{Name: name}
}

// buildOverviewSection creates the overview section for a provider detail view.
//
// Takes info (provider_domain.ProviderInfo) which contains the provider data
// to display.
//
// Returns provider_domain.InfoSection which contains the formatted overview
// with name, type, default status, and registration age.
func buildOverviewSection(info provider_domain.ProviderInfo) provider_domain.InfoSection {
	isDefault := "false"
	if info.IsDefault {
		isDefault = "true"
	}

	return provider_domain.InfoSection{
		Title: "Overview",
		Entries: []provider_domain.InfoEntry{
			{Key: "Name", Value: info.Name},
			{Key: "Type", Value: info.ProviderType},
			{Key: "Default", Value: isDefault},
			{Key: "Registered", Value: formatRegisteredAge(info.RegisteredAt)},
		},
	}
}

// buildMetadataSection creates a configuration section from ProviderMetadata
// if the provider implements it.
//
// Takes provider (any) which is checked for ProviderMetadata implementation.
//
// Returns provider_domain.InfoSection which contains the metadata entries.
// Returns bool which indicates whether the provider implements ProviderMetadata.
func buildMetadataSection(provider any) (provider_domain.InfoSection, bool) {
	meta, ok := provider.(provider_domain.ProviderMetadata)
	if !ok {
		return provider_domain.InfoSection{}, false
	}

	metadata := meta.GetProviderMetadata()
	if len(metadata) == 0 {
		return provider_domain.InfoSection{}, false
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

	return provider_domain.InfoSection{
		Title:   "Configuration",
		Entries: entries,
	}, true
}

// formatRegisteredAge returns a human-readable duration since registration.
//
// Takes registeredAt (time.Time) which is the timestamp when registration
// occurred.
//
// Returns string which is the formatted duration (e.g. "5m ago", "3d ago") or
// "unknown" if the timestamp is zero.
func formatRegisteredAge(registeredAt time.Time) string {
	if registeredAt.IsZero() {
		return "unknown"
	}

	d := time.Since(registeredAt)

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < hoursPerDay*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/hoursPerDay))
	}
}
