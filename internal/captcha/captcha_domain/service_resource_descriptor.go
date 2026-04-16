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

package captcha_domain

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

var _ provider_domain.ResourceDescriptor = (*captchaService)(nil)

// ResourceType returns the resource name for the captcha hexagon.
//
// Returns string which is the resource type identifier.
func (*captchaService) ResourceType() string {
	return "captcha"
}

// ResourceListColumns returns column definitions for the captcha provider list
// table.
//
// Returns []provider_domain.ColumnDefinition which defines the table columns.
func (*captchaService) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "TYPE", Key: "type"},
		{Header: "REGISTERED", Key: "registered"},
	}
}

// ResourceListProviders returns all registered captcha providers as list rows.
//
// Returns []provider_domain.ProviderListEntry which contains a row per
// provider.
func (s *captchaService) ResourceListProviders(ctx context.Context) []provider_domain.ProviderListEntry {
	providers := s.registry.ListProviders(ctx)
	entries := make([]provider_domain.ProviderListEntry, len(providers))

	for i, info := range providers {
		providerType := info.ProviderType
		if captchaProvider, err := s.registry.GetProvider(ctx, info.Name); err == nil {
			if providerType == "" || providerType == "unknown" {
				providerType = string(captchaProvider.Type())
			}
		}

		entries[i] = provider_domain.ProviderListEntry{
			Name:      info.Name,
			IsDefault: info.IsDefault,
			Values: map[string]string{
				"name":       info.Name,
				"type":       providerType,
				"registered": formatRegisteredAge(info.RegisteredAt),
			},
		}
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a single named
// captcha provider.
//
// Takes name (string) which identifies the provider to describe.
//
// Returns *provider_domain.ProviderDetail which contains sections of provider
// information.
// Returns error when the named provider does not exist.
func (s *captchaService) ResourceDescribeProvider(ctx context.Context, name string) (*provider_domain.ProviderDetail, error) {
	captchaProvider, err := s.registry.GetProvider(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("captcha provider %q not found: %w", name, err)
	}

	info := findProviderInfo(s.registry.ListProviders(ctx), name)

	isDefault := "false"
	if info.IsDefault {
		isDefault = "true"
	}

	sections := []provider_domain.InfoSection{
		{
			Title: "Overview",
			Entries: []provider_domain.InfoEntry{
				{Key: "Name", Value: info.Name},
				{Key: "Type", Value: string(captchaProvider.Type())},
				{Key: "Default", Value: isDefault},
				{Key: "Site Key", Value: captchaProvider.SiteKey()},
				{Key: "Script URL", Value: captchaProvider.ScriptURL()},
				{Key: "Registered", Value: formatRegisteredAge(info.RegisteredAt)},
			},
		},
	}

	if metaSection, ok := buildMetadataSection(captchaProvider); ok {
		sections = append(sections, metaSection)
	}

	return &provider_domain.ProviderDetail{
		Name:     name,
		Sections: sections,
	}, nil
}

// findProviderInfo locates a provider by name in the given slice, returning a
// zero-value entry with the name set if not found.
//
// Takes infos ([]provider_domain.ProviderInfo) which is the list to search.
// Takes name (string) which is the provider name to find.
//
// Returns provider_domain.ProviderInfo which is the matched entry, or a
// zero-value entry with the name set if not found.
func findProviderInfo(infos []provider_domain.ProviderInfo, name string) provider_domain.ProviderInfo {
	for _, info := range infos {
		if info.Name == name {
			return info
		}
	}
	return provider_domain.ProviderInfo{Name: name}
}

// buildMetadataSection extracts provider metadata into an InfoSection if the
// provider implements the ProviderMetadata interface.
//
// Takes captchaProvider (any) which is the provider to extract metadata from.
//
// Returns provider_domain.InfoSection which contains the sorted metadata
// entries.
// Returns bool which is true when the provider had metadata to display.
func buildMetadataSection(captchaProvider any) (provider_domain.InfoSection, bool) {
	meta, ok := captchaProvider.(provider_domain.ProviderMetadata)
	if !ok {
		return provider_domain.InfoSection{}, false
	}

	metadata := meta.GetProviderMetadata()
	if len(metadata) == 0 {
		return provider_domain.InfoSection{}, false
	}

	entries := make([]provider_domain.InfoEntry, 0, len(metadata))
	for key, value := range metadata {
		entries = append(entries, provider_domain.InfoEntry{
			Key:   key,
			Value: fmt.Sprintf("%v", value),
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

// formatRegisteredAge formats a registration timestamp as a human-readable
// relative age string.
//
// Takes registeredAt (time.Time) which is the timestamp to format.
//
// Returns string which is the human-readable relative age.
func formatRegisteredAge(registeredAt time.Time) string {
	if registeredAt.IsZero() {
		return "unknown"
	}

	duration := time.Since(registeredAt)

	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%ds ago", int(duration.Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", int(duration.Minutes()))
	case duration < hoursPerDay*time.Hour:
		return fmt.Sprintf("%dh ago", int(duration.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/hoursPerDay))
	}
}
