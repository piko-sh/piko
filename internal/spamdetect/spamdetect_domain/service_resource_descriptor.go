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

package spamdetect_domain

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"piko.sh/piko/internal/provider/provider_domain"
)

// hoursPerDay is the number of hours in a day for age formatting.
const hoursPerDay = 24

var _ provider_domain.ResourceDescriptor = (*spamDetectService)(nil)

// ResourceType returns the resource name for the spam detection hexagon.
//
// Returns string which is "spamdetect".
func (*spamDetectService) ResourceType() string {
	return "spamdetect"
}

// ResourceListColumns returns column definitions for the detector list
// table.
//
// Returns []provider_domain.ColumnDefinition which defines the table columns.
func (*spamDetectService) ResourceListColumns() []provider_domain.ColumnDefinition {
	return []provider_domain.ColumnDefinition{
		{Header: "NAME", Key: "name"},
		{Header: "SIGNALS", Key: "signals"},
		{Header: "REGISTERED", Key: "registered"},
	}
}

// ResourceListProviders returns all registered detectors as list rows.
//
// Returns []provider_domain.ProviderListEntry which contains the detector rows.
func (s *spamDetectService) ResourceListProviders(ctx context.Context) []provider_domain.ProviderListEntry {
	detectors := s.registry.ListProviders(ctx)
	entries := make([]provider_domain.ProviderListEntry, len(detectors))

	for index, info := range detectors {
		signalsDisplay := ""
		if detector, err := s.registry.GetProvider(ctx, info.Name); err == nil {
			signals := detector.Signals()
			signalStrings := make([]string, len(signals))
			for signalIndex, signal := range signals {
				signalStrings[signalIndex] = signal.String()
			}
			signalsDisplay = strings.Join(signalStrings, ", ")
		}

		entries[index] = provider_domain.ProviderListEntry{
			Name:      info.Name,
			IsDefault: info.IsDefault,
			Values: map[string]string{
				"name":       info.Name,
				"signals":    signalsDisplay,
				"registered": formatRegisteredAge(info.RegisteredAt),
			},
		}
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a named
// detector.
//
// Takes name (string) which identifies the detector.
//
// Returns *provider_domain.ProviderDetail which contains the detector details.
// Returns error when the detector is not found.
func (s *spamDetectService) ResourceDescribeProvider(ctx context.Context, name string) (*provider_domain.ProviderDetail, error) {
	detector, err := s.registry.GetProvider(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("spam detection detector %q not found: %w", name, err)
	}

	info := findDetectorInfo(s.registry.ListProviders(ctx), name)

	signals := detector.Signals()
	signalStrings := make([]string, len(signals))
	for index, signal := range signals {
		signalStrings[index] = signal.String()
	}

	sections := []provider_domain.InfoSection{
		{
			Title: "Overview",
			Entries: []provider_domain.InfoEntry{
				{Key: "Name", Value: info.Name},
				{Key: "Signals", Value: strings.Join(signalStrings, ", ")},
				{Key: "Registered", Value: formatRegisteredAge(info.RegisteredAt)},
			},
		},
	}

	if metaSection, ok := buildMetadataSection(detector); ok {
		sections = append(sections, metaSection)
	}

	return &provider_domain.ProviderDetail{
		Name:     name,
		Sections: sections,
	}, nil
}

// findDetectorInfo looks up a ProviderInfo by name from a list.
//
// Takes infos ([]provider_domain.ProviderInfo) which is the list.
// Takes name (string) which identifies the detector.
//
// Returns provider_domain.ProviderInfo which is the matching entry,
// or a zero-value entry with the given name if not found.
func findDetectorInfo(infos []provider_domain.ProviderInfo, name string) provider_domain.ProviderInfo {
	for _, info := range infos {
		if info.Name == name {
			return info
		}
	}
	return provider_domain.ProviderInfo{Name: name}
}

// buildMetadataSection creates an InfoSection from a detector's
// metadata.
//
// Takes detector (any) which may implement ProviderMetadata.
//
// Returns provider_domain.InfoSection which contains the metadata.
// Returns bool which is true when metadata was found.
func buildMetadataSection(detector any) (provider_domain.InfoSection, bool) {
	meta, ok := detector.(provider_domain.ProviderMetadata)
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

// formatRegisteredAge formats a registration timestamp as a
// human-readable age.
//
// Takes registeredAt (time.Time) which is the registration timestamp.
//
// Returns string which is the formatted age string.
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
