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
	"context"
	"fmt"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/provider/provider_domain"
)

var (
	_ provider_domain.ResourceDescriptor = (*spamDetectService)(nil)

	_ provider_domain.ReadinessProbeNamed = (*spamDetectService)(nil)
)

// ResourceType returns the resource name for the spam detection hexagon.
//
// Returns string which is "spamdetect".
func (*spamDetectService) ResourceType() string {
	return "spamdetect"
}

// ProbeName returns the readiness health-probe name of the spam detection service,
// bridging the "spamdetect" resource type to the "SpamDetectService" readiness dependency
// so a readiness collector can attach this descriptor's provider info to the matching
// dependency. It returns the same constant the service's health-probe Name method
// returns.
//
// Returns string which is healthProbeName ("SpamDetectService").
func (*spamDetectService) ProbeName() string {
	return healthProbeName
}

// ResourceListColumns returns column definitions for the detector list table.
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
	ctx, l := logger_domain.From(ctx, log)
	detectors := s.registry.ListProviders(ctx)
	entries := make([]provider_domain.ProviderListEntry, len(detectors))

	for index, info := range detectors {
		signalsDisplay := "unknown"
		if detector, err := s.registry.GetProvider(ctx, info.Name); err == nil {
			signals := detector.Signals()
			signalStrings := make([]string, len(signals))
			for signalIndex, signal := range signals {
				signalStrings[signalIndex] = signal.String()
			}
			signalsDisplay = strings.Join(signalStrings, ", ")
		} else {
			l.Warn("Failed to resolve detector for resource listing",
				logger_domain.String(attributeKeyDetector, info.Name),
				logger_domain.Error(err),
			)
		}

		entries[index] = provider_domain.ProviderListEntry{
			Name:      info.Name,
			IsDefault: info.IsDefault,
			Values: map[string]string{
				"name":       info.Name,
				"signals":    signalsDisplay,
				"registered": provider_domain.FormatRegisteredAge(info.RegisteredAt),
			},
		}
	}

	return entries
}

// ResourceDescribeProvider returns detailed information for a named detector.
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
				{Key: "Registered", Value: provider_domain.FormatRegisteredAge(info.RegisteredAt)},
			},
		},
	}

	if metaSection, ok := provider_domain.BuildMetadataSection(detector); ok {
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
// Returns provider_domain.ProviderInfo which is the matching entry, or a zero-value entry
// with the given name if not found.
func findDetectorInfo(infos []provider_domain.ProviderInfo, name string) provider_domain.ProviderInfo {
	for _, info := range infos {
		if info.Name == name {
			return info
		}
	}
	return provider_domain.ProviderInfo{Name: name}
}
