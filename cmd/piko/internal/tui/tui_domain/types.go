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

package tui_domain

import (
	"time"
)

const (
	// SpanStatusUnset is the default value when no status has been set.
	SpanStatusUnset SpanStatus = iota

	// SpanStatusOK indicates the span completed successfully.
	SpanStatusOK

	// SpanStatusError indicates the span encountered an error.
	SpanStatusError
)

const (
	// ResourceStatusUnknown means the status is not known or cannot be found.
	ResourceStatusUnknown ResourceStatus = iota

	// ResourceStatusHealthy indicates the resource is working correctly.
	ResourceStatusHealthy

	// ResourceStatusDegraded indicates the resource is working but with issues.
	ResourceStatusDegraded

	// ResourceStatusUnhealthy indicates the resource is not working.
	ResourceStatusUnhealthy

	// ResourceStatusPending indicates the resource is waiting to be processed.
	ResourceStatusPending
)

const (
	// ProviderStatusDisconnected means the provider has no active connection.
	ProviderStatusDisconnected ProviderStatus = iota

	// ProviderStatusConnecting indicates the provider is establishing connection.
	ProviderStatusConnecting

	// ProviderStatusConnected indicates the provider is connected and healthy.
	ProviderStatusConnected

	// ProviderStatusError indicates the provider encountered an error.
	ProviderStatusError
)

const (
	// HealthStateUnknown indicates that the health state could not be found.
	HealthStateUnknown HealthState = iota

	// HealthStateHealthy means the component is working normally.
	HealthStateHealthy

	// HealthStateDegraded indicates the component is working but with reduced
	// performance or features.
	HealthStateDegraded

	// HealthStateUnhealthy indicates the component is not working.
	HealthStateUnhealthy
)

// MetricValue represents a single metric data point with its timestamp and labels.
type MetricValue struct {
	// Timestamp is when the metric value was recorded.
	Timestamp time.Time

	// Labels holds key-value pairs that identify this metric.
	Labels map[string]string

	// Value is the numeric measurement at this point in time.
	Value float64
}

// MetricSeries represents a time series of metric values.
type MetricSeries struct {
	// Name is the identifier for the metric.
	Name string

	// Unit is the unit of measurement (e.g., "ms", "bytes", "requests").
	Unit string

	// Description explains what this metric measures.
	Description string

	// Values holds the data points in time order, from oldest to newest.
	Values []MetricValue
}

// Latest returns the most recent value in the series, or nil if empty.
//
// Returns *MetricValue which is the newest entry, or nil when the series has
// no values.
func (s *MetricSeries) Latest() *MetricValue {
	if len(s.Values) == 0 {
		return nil
	}
	return &s.Values[len(s.Values)-1]
}

// SpanStatus represents the current state of a trace span.
type SpanStatus int

// String returns a human-readable representation of the span status.
//
// Returns string which is one of "unset", "ok", or "error".
func (s SpanStatus) String() string {
	return [...]string{"unset", "ok", "error"}[s]
}

// Span represents a single trace span for display in the TUI.
type Span struct {
	// StartTime is when the span began, in RFC3339 format when serialised.
	StartTime time.Time

	// Attributes holds key-value pairs of span metadata such as method and path.
	Attributes map[string]string

	// TraceID is the unique identifier for the distributed trace this span belongs to.
	TraceID string

	// SpanID is the unique identifier for this span.
	SpanID string

	// ParentID is the span ID of the parent span; empty if this is a root span.
	ParentID string

	// Name is the operation name for this span.
	Name string

	// Service is the name of the service that produced this span.
	Service string

	// StatusMessage is a short description of the status that is easy to read.
	StatusMessage string

	// Children contains the nested spans within this span.
	Children []Span

	// Duration is the total time the span took to complete.
	Duration time.Duration

	// Status indicates the result of this span; used for error filtering.
	Status SpanStatus
}

// IsRoot reports whether this span has no parent.
//
// Returns bool which is true if the span is a root span with no parent.
func (s *Span) IsRoot() bool {
	return s.ParentID == ""
}

// IsError returns true if this span has an error status.
//
// Returns bool which indicates whether the span status is an error.
func (s *Span) IsError() bool {
	return s.Status == SpanStatusError
}

// ResourceStatus represents the health state of a resource. It implements
// fmt.Stringer.
type ResourceStatus int

// String returns a human-readable representation of the resource status.
//
// Returns string which is one of: unknown, healthy, degraded, unhealthy, or
// pending.
func (s ResourceStatus) String() string {
	return [...]string{"unknown", "healthy", "degraded", "unhealthy", "pending"}[s]
}

// Resource represents an item that can be shown in the TUI panels.
// It holds shared data for registry artefacts, orchestrator tasks, workflows,
// and other domain objects displayed in the interface.
type Resource struct {
	// CreatedAt is when the resource was created.
	CreatedAt time.Time

	// UpdatedAt is when the resource was last modified.
	UpdatedAt time.Time

	// Metadata holds key-value pairs such as priority, attempt count, and
	// progress details for the resource.
	Metadata map[string]string

	// Kind specifies the type of resource.
	Kind string

	// ID is the unique identifier for this resource.
	ID string

	// Name is the display name of the resource.
	Name string

	// StatusText is the text shown to the user for the current status.
	StatusText string

	// Children holds the nested resources under this resource.
	Children []Resource

	// Status is the current state of the resource.
	Status ResourceStatus
}

// HasChildren returns true if this resource has child resources.
//
// Returns bool which is true when the resource has one or more children.
func (r *Resource) HasChildren() bool {
	return len(r.Children) > 0
}

// ProviderStatus represents the connection state of a provider.
// It implements fmt.Stringer for display purposes.
type ProviderStatus int

// String returns a human-readable representation of the provider status.
//
// Returns string which is one of "disconnected", "connecting", "connected",
// or "error".
func (s ProviderStatus) String() string {
	return [...]string{"disconnected", "connecting", "connected", "error"}[s]
}

// ProviderInfo contains runtime status and statistics for a provider.
type ProviderInfo struct {
	// LastRefresh is when the provider data was last updated.
	LastRefresh time.Time

	// LastError holds the most recent error from this provider; nil means no error.
	LastError error

	// Name is the display name for the provider.
	Name string

	// Status is the current connection state of the provider.
	Status ProviderStatus

	// RefreshCount is the number of times the provider data has been refreshed.
	RefreshCount int64

	// ErrorCount is the total number of errors from this provider.
	ErrorCount int64
}

// DataUpdatedMessage is a Bubble Tea message that tells panels when providers have
// new data. Panels should listen for this message to refresh their display.
type DataUpdatedMessage struct {
	// Time is when the data was last updated.
	Time time.Time
}

// TickMessage is a bubbletea message sent at regular intervals to all panels.
// Panels can use this for background updates such as sampling stats.
type TickMessage struct {
	// Time is the moment when the tick occurred.
	Time time.Time
}

// HealthState represents the health of a component. It implements
// fmt.Stringer for display purposes.
type HealthState int

// String returns a human-readable representation of the health state.
//
// Returns string which is the lowercase name of the state.
func (s HealthState) String() string {
	return [...]string{"unknown", "healthy", "degraded", "unhealthy"}[s]
}

// HealthStatus represents the health of a component and its dependencies.
type HealthStatus struct {
	// Timestamp is when the health check was done.
	Timestamp time.Time

	// Name is the identifier for this health dependency.
	Name string

	// Message provides extra details about the health state; empty when healthy.
	Message string

	// Dependencies holds the health status of each sub-component.
	Dependencies []*HealthStatus

	// State is the current health state of the component.
	State HealthState

	// Duration is the time taken to run the health check.
	Duration time.Duration
}

// IsHealthy reports whether the health state is healthy.
//
// Returns bool which is true when the state equals HealthStateHealthy.
func (s *HealthStatus) IsHealthy() bool {
	return s.State == HealthStateHealthy
}

// CountByState returns the count of dependencies in each state.
//
// Returns map[HealthState]int which maps each health state to the number of
// dependencies in that state.
func (s *HealthStatus) CountByState() map[HealthState]int {
	counts := make(map[HealthState]int)
	for _, dependency := range s.Dependencies {
		counts[dependency.State]++
	}
	return counts
}

// Providers holds all the data providers for the TUI.
// This is passed to NewService to inject dependencies.
type Providers struct {
	// Metrics holds the data providers for metrics collection.
	Metrics []MetricsProvider

	// Traces holds the providers that supply trace data.
	Traces []TracesProvider

	// Resources holds the providers that supply resource data.
	Resources []ResourceProvider

	// Health holds the providers for checking service health.
	Health []HealthProvider

	// System contains providers that supply system statistics.
	System []SystemProvider

	// FDs holds providers that track open file descriptors.
	FDs []FDsProvider

	// Watchdog holds providers that surface runtime anomaly-detector
	// state, profiles, history, and live events.
	Watchdog []WatchdogProvider

	// ProvidersInfo holds the inspector port for `piko get providers`.
	ProvidersInfo []ProvidersInspector

	// DLQ holds the inspector port for `piko get dlq`.
	DLQ []DLQInspector

	// RateLimiter holds the inspector port for `piko get ratelimiter`.
	RateLimiter []RateLimiterInspector

	// Profiling holds the inspector port for `piko profile`.
	Profiling []ProfilingInspector

	// Panels holds custom panels to display in the terminal interface.
	Panels []Panel
}
