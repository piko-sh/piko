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

package daemon_domain

import (
	"context"
	"sync/atomic"

	"piko.sh/piko/internal/registry/registry_dto"
)

// MockOnDemandVariantGenerator is a test double for
// OnDemandVariantGenerator where nil function fields return zero values
// and call counts are tracked atomically.
type MockOnDemandVariantGenerator struct {
	// GenerateVariantFunc is the function called by
	// GenerateVariant.
	GenerateVariantFunc func(ctx context.Context, artefact *registry_dto.ArtefactMeta, profileName string) (*registry_dto.Variant, error)

	// ParseProfileNameFunc is the function called by
	// ParseProfileName.
	ParseProfileNameFunc func(profileName string) *ParsedImageProfile

	// GenerateVariantCallCount tracks how many times
	// GenerateVariant was called.
	GenerateVariantCallCount int64

	// ParseProfileNameCallCount tracks how many times
	// ParseProfileName was called.
	ParseProfileNameCallCount int64
}

var _ OnDemandVariantGenerator = (*MockOnDemandVariantGenerator)(nil)

// GenerateVariant generates a variant for an artefact based on the profile.
//
// Takes ctx (context.Context) which carries deadlines and
// cancellation signals.
// Takes artefact (*registry_dto.ArtefactMeta) which is the
// artefact to generate a variant for.
// Takes profileName (string) which identifies the image
// profile to apply.
//
// Returns (*Variant, error), or (nil, nil) if GenerateVariantFunc is nil.
func (m *MockOnDemandVariantGenerator) GenerateVariant(ctx context.Context, artefact *registry_dto.ArtefactMeta, profileName string) (*registry_dto.Variant, error) {
	atomic.AddInt64(&m.GenerateVariantCallCount, 1)
	if m.GenerateVariantFunc != nil {
		return m.GenerateVariantFunc(ctx, artefact, profileName)
	}
	return nil, nil
}

// ParseProfileName parses a profile name and returns the parsed settings.
//
// Takes profileName (string) which is the profile name to parse.
//
// Returns *ParsedImageProfile, or nil if ParseProfileNameFunc is nil.
func (m *MockOnDemandVariantGenerator) ParseProfileName(profileName string) *ParsedImageProfile {
	atomic.AddInt64(&m.ParseProfileNameCallCount, 1)
	if m.ParseProfileNameFunc != nil {
		return m.ParseProfileNameFunc(profileName)
	}
	return nil
}
