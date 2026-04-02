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

// ProviderMetadata is an optional interface for exposing provider capabilities.
// Used for discovery and monitoring.
//
// Implementation guidelines:
//   - GetProviderType should return a stable, lowercase identifier
//     (e.g., "smtp", "ses", "s3", "redis").
//   - GetProviderMetadata can include version, region, capabilities, limits,
//     and configuration details.
//   - Metadata is included in ListProviders() responses for monitoring
//     dashboards.
type ProviderMetadata interface {
	// GetProviderType returns the provider implementation type.
	//
	// Returns string which is a lowercase, stable identifier suitable for
	// monitoring.
	GetProviderType() string

	// GetProviderMetadata returns arbitrary metadata about the provider, such as
	// version, region, capabilities, and limits, for discovery and monitoring.
	//
	// Returns map[string]any which contains provider-specific metadata.
	GetProviderMetadata() map[string]any
}
