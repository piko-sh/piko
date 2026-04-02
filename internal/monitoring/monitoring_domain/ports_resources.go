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

package monitoring_domain

import "piko.sh/piko/internal/provider/provider_domain"

// HealthProbeStatus holds the result of a health check probe.
type HealthProbeStatus struct {
	// Name is the identifier of the health probe.
	Name string

	// State is the current health state of the probe.
	State string

	// Message string // Message provides details about the health probe status.
	Message string

	// Duration is the time taken to complete the health probe.
	Duration string

	// Dependencies holds the health status of dependent services.
	Dependencies []HealthProbeStatus

	// Timestamp is when the probe was executed, in milliseconds since Unix epoch.
	Timestamp int64
}

// ResourceData holds details about resources and their groups.
type ResourceData struct {
	// Categories contains the grouped resource information.
	Categories []ResourceCategory

	// Total is the count of all resources.
	Total int32

	// TimestampMs is the Unix timestamp in milliseconds when the data was
	// collected.
	TimestampMs int64
}

// ResourceCategory groups resources by their type or purpose.
type ResourceCategory struct {
	// Category is the name of this resource category.
	Category string

	// Resources contains the resource entries in this category.
	Resources []ResourceInfo

	// Count is the number of resources in this category.
	Count int32
}

// ResourceInfo holds details about a single resource.
type ResourceInfo struct {
	// Category classifies the resource type.
	Category string

	// Target is the destination path or address the resource refers to.
	Target string

	// FirstSeenMs is the timestamp when the resource was first observed,
	// in milliseconds since Unix epoch.
	FirstSeenMs int64

	// AgeMs is the age of the resource in milliseconds.
	AgeMs int64

	// FD is the resource number.
	FD int32
}

// ProviderListResult holds column definitions and provider rows for a resource
// type's provider list table.
type ProviderListResult struct {
	// SubResourceName is non-empty when this result represents sub-resources
	// rather than providers (e.g. "namespaces", "repositories").
	SubResourceName string

	// Columns defines the table structure for the provider list.
	Columns []provider_domain.ColumnDefinition

	// Rows contains one entry per registered provider.
	Rows []provider_domain.ProviderListEntry
}
