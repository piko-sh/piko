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

package provider_domain

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// configurationSectionTitle is the section heading for provider metadata entries.
	configurationSectionTitle = "Configuration"

	// hoursPerDay is the number of hours in a day, used for duration formatting.
	hoursPerDay = 24
)

// BuildMetadataSection builds the "Configuration" InfoSection from a provider's
// GetProviderMetadata map, sorted by key (cmp.Compare). ok is false when the provider
// does not implement ProviderMetadata or exposes no metadata.
//
// Takes provider (any) which is checked for ProviderMetadata implementation.
//
// Returns InfoSection which contains the sorted metadata entries.
// Returns bool which is false when the provider has no metadata to display.
func BuildMetadataSection(provider any) (InfoSection, bool) {
	meta, ok := provider.(ProviderMetadata)
	if !ok {
		return InfoSection{}, false
	}

	metadata := meta.GetProviderMetadata()
	if len(metadata) == 0 {
		return InfoSection{}, false
	}

	entries := make([]InfoEntry, 0, len(metadata))
	for k, v := range metadata {
		entries = append(entries, InfoEntry{
			Key:   k,
			Value: fmt.Sprintf("%v", v),
		})
	}
	slices.SortFunc(entries, func(a, b InfoEntry) int {
		return cmp.Compare(a.Key, b.Key)
	})

	return InfoSection{
		Title:   configurationSectionTitle,
		Entries: entries,
	}, true
}

// FormatRegisteredAge returns a human-readable age since registeredAt (e.g. "5m ago"), or
// "unknown" for a zero time, using safeconv for the int conversions.
//
// Takes registeredAt (time.Time) which is the timestamp when registration occurred.
//
// Returns string which is the formatted age, or "unknown" if the timestamp is zero.
func FormatRegisteredAge(registeredAt time.Time) string {
	if registeredAt.IsZero() {
		return "unknown"
	}

	duration := time.Since(registeredAt)

	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%ds ago", safeconv.Int64ToInt(int64(duration.Seconds())))
	case duration < time.Hour:
		return fmt.Sprintf("%dm ago", safeconv.Int64ToInt(int64(duration.Minutes())))
	case duration < hoursPerDay*time.Hour:
		return fmt.Sprintf("%dh ago", safeconv.Int64ToInt(int64(duration.Hours())))
	default:
		return fmt.Sprintf("%dd ago", safeconv.Int64ToInt(int64(duration.Hours()/hoursPerDay)))
	}
}
