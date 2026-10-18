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

package registry_domain

import (
	"piko.sh/piko/internal/registry/registry_dto"
)

// upsertConfig carries the optional behaviours of one UpsertArtefact call.
type upsertConfig struct {
	// mergeProfiles adds the given desired profiles to an existing artefact's profiles
	// instead of replacing them.
	mergeProfiles bool
}

// UpsertOption configures optional UpsertArtefact behaviour.
type UpsertOption func(*upsertConfig)

// WithProfileMerge adds desired profiles without replacing existing profiles of the same
// name.
//
// Returns UpsertOption which enables the merge.
func WithProfileMerge() UpsertOption {
	return func(config *upsertConfig) {
		config.mergeProfiles = true
	}
}

// UpsertMergesProfiles reports whether options request merging desired profiles in
// UpsertArtefact.
//
// Takes opts (...UpsertOption) which are the options an UpsertArtefact call received.
//
// Returns bool which is true when WithProfileMerge is among the options.
func UpsertMergesProfiles(opts ...UpsertOption) bool {
	return resolveUpsertOptions(opts).mergeProfiles
}

// resolveUpsertOptions applies a set of options to a zero upsertConfig.
//
// Takes opts ([]UpsertOption) which are the options to apply; nil entries are ignored.
//
// Returns upsertConfig which holds the requested behaviours.
func resolveUpsertOptions(opts []UpsertOption) upsertConfig {
	var config upsertConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return config
}

// mergeDesiredProfiles appends profiles with new names while preserving existing
// definitions and order.
//
// When no profiles are added, the existing slice is returned unchanged.
//
// Takes existing ([]registry_dto.NamedProfile) which are the artefact's current profiles.
// Takes incoming ([]registry_dto.NamedProfile) which are the profiles the caller wants.
//
// Returns []registry_dto.NamedProfile which is the merged profile set.
func mergeDesiredProfiles(existing, incoming []registry_dto.NamedProfile) []registry_dto.NamedProfile {
	added := 0
	for i := range incoming {
		if !hasProfileNamed(existing, incoming[i].Name) {
			added++
		}
	}
	if added == 0 {
		return existing
	}

	merged := make([]registry_dto.NamedProfile, 0, len(existing)+added)
	merged = append(merged, existing...)
	for i := range incoming {
		if !hasProfileNamed(merged, incoming[i].Name) {
			merged = append(merged, incoming[i])
		}
	}
	return merged
}

// hasProfileNamed reports whether the list contains a profile with the given name.
//
// Takes profiles ([]registry_dto.NamedProfile) which is the list to search.
// Takes name (string) which is the profile name to find.
//
// Returns bool which is true when a profile of that name is present.
func hasProfileNamed(profiles []registry_dto.NamedProfile, name string) bool {
	for i := range profiles {
		if profiles[i].Name == name {
			return true
		}
	}
	return false
}
