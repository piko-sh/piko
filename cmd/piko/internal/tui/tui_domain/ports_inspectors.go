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
	"context"
	"time"
)

// ProvidersInspector exposes the read-only provider-info API surface
// the CLI uses for `piko get providers` / `piko describe provider`. It
// is the data port for ProvidersPanel.
type ProvidersInspector interface {
	Provider

	// ListProviders returns every registered provider grouped by
	// resource type.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns []ProviderEntry which is the (possibly empty) list.
	// Returns error when the call fails.
	ListProviders(ctx context.Context) ([]ProviderEntry, error)

	// DescribeProvider returns full detail for a single provider.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes resourceType (string) which is the provider's resource type.
	// Takes name (string) which is the provider name.
	//
	// Returns *ProviderDetail with sections and sub-resources.
	// Returns error when the call fails or the provider is unknown.
	DescribeProvider(ctx context.Context, resourceType, name string) (*ProviderDetail, error)
}

// ProviderEntry is a single registered provider as listed by the
// ProviderInfo service.
type ProviderEntry struct {
	// Values is the column-keyed metadata returned by the service.
	Values map[string]string

	// ResourceType is the kind of resource the provider serves
	// (artefacts, tasks, workflows, ...).
	ResourceType string

	// Name is the provider identifier within its type.
	Name string

	// IsDefault is true when this provider is the default for its type.
	IsDefault bool
}

// ProviderDetail is the full describe-output for a single provider.
type ProviderDetail struct {
	// ResourceType is the kind of resource the provider serves.
	ResourceType string

	// Name is the provider identifier within its type.
	Name string

	// Sections holds the labelled key/value groups returned by
	// DescribeProvider.
	Sections []ProviderSection

	// SubResources lists the addressable sub-resources owned by the
	// provider; may be empty when sub-resources are unsupported.
	SubResources []ProviderSubResource
}

// ProviderSection is a labelled set of key/value rows.
type ProviderSection struct {
	// Title is the section heading rendered above its entries.
	Title string

	// Entries are the key/value rows under the heading.
	Entries []ProviderField
}

// ProviderField is a key/value row inside a ProviderSection.
type ProviderField struct {
	// Key is the row label.
	Key string

	// Value is the rendered value text.
	Value string
}

// ProviderSubResource is a child resource owned by a provider, with
// arbitrary key/value metadata.
type ProviderSubResource struct {
	// Values holds arbitrary metadata.
	Values map[string]string

	// Type identifies the sub-resource kind.
	Type string

	// Name identifies the instance within Type.
	Name string
}

// DLQInspector exposes the dispatcher summaries + DLQ entries surface
// of the gRPC API. It is the data port for DLQPanel.
type DLQInspector interface {
	Provider

	// DispatcherSummaries returns one summary per dispatcher type.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns []DispatcherSummary.
	// Returns error when the call fails.
	DispatcherSummaries(ctx context.Context) ([]DispatcherSummary, error)

	// ListDLQEntries returns the dead-letter-queue entries for a
	// dispatcher type.
	//
	// Takes ctx (context.Context) for cancellation.
	// Takes dispatcherType (string) which selects the dispatcher.
	// Takes limit (int) which caps the returned slice.
	//
	// Returns []DLQEntry.
	// Returns error when the call fails.
	ListDLQEntries(ctx context.Context, dispatcherType string, limit int) ([]DLQEntry, error)
}

// DispatcherSummary is one row from GetDispatcherSummary.
type DispatcherSummary struct {
	// Type is the dispatcher kind (e.g. tasks, workflows).
	Type string

	// QueuedItems is the count currently queued.
	QueuedItems int32

	// DeadLetterCount is the count of entries in the dead-letter
	// queue awaiting human attention.
	DeadLetterCount int32

	// TotalProcessed is the lifetime count of processed items.
	TotalProcessed int64

	// TotalSuccessful is the lifetime count of items that completed
	// successfully.
	TotalSuccessful int64

	// TotalFailed is the lifetime count of items that ended in failure.
	TotalFailed int64

	// RetryQueueSize is the count of items pending retry.
	RetryQueueSize int32

	// TotalRetries is the lifetime retry count.
	TotalRetries int64

	// Uptime is how long the dispatcher has been running.
	Uptime time.Duration
}

// DLQEntry is one row from ListDLQEntries.
type DLQEntry struct {
	// AddedAt is when the entry was first enqueued in the DLQ.
	AddedAt time.Time

	// LastAttempt is when the most recent retry attempt occurred.
	LastAttempt time.Time

	// ID identifies the dead-lettered item.
	ID string

	// Type is the dispatcher kind that produced this entry.
	Type string

	// OriginalError is the error message captured at the failing
	// attempt.
	OriginalError string

	// TotalAttempts is the count of retries before the entry was
	// dead-lettered.
	TotalAttempts int32
}

// RateLimiterInspector exposes the GetRateLimiterStatus RPC. It is the
// data port for RateLimiterPanel.
type RateLimiterInspector interface {
	Provider

	// GetStatus returns the current rate-limiter status.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns *RateLimiterStatus.
	// Returns error when the call fails.
	GetStatus(ctx context.Context) (*RateLimiterStatus, error)
}

// RateLimiterStatus mirrors the GetRateLimiterStatusResponse fields the
// TUI surfaces.
type RateLimiterStatus struct {
	// TokenBucketStore identifies the token-bucket backing store.
	TokenBucketStore string

	// CounterStore identifies the counter backing store.
	CounterStore string

	// FailPolicy is the configured fail-open / fail-closed mode.
	FailPolicy string

	// KeyPrefix is the namespace under which rate-limit keys are
	// stored.
	KeyPrefix string

	// TotalChecks is the lifetime count of rate-limit checks.
	TotalChecks int64

	// TotalAllowed is the lifetime count of allowed checks.
	TotalAllowed int64

	// TotalDenied is the lifetime count of denied checks.
	TotalDenied int64

	// TotalErrors is the lifetime count of checks that errored.
	TotalErrors int64
}

// ProfilingInspector exposes the profiling control + capture surface.
// It is the data port for ProfilingPanel.
type ProfilingInspector interface {
	Provider

	// Status returns the current profiling enable/disable state.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns *ProfilingStatus.
	// Returns error when the call fails.
	Status(ctx context.Context) (*ProfilingStatus, error)

	// Enable turns on the server's on-demand profiling for a fixed
	// window. The set of profiles exposed is decided by the server
	// build, so callers cannot select individual profile kinds.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns error when the call fails.
	Enable(ctx context.Context) error

	// Disable turns off all profiles.
	//
	// Takes ctx (context.Context) for cancellation.
	//
	// Returns error when the call fails.
	Disable(ctx context.Context) error

	// Capture takes a one-shot profile and returns the raw bytes.
	//
	// Takes ctx (context.Context) for cancellation; the call also
	// observes duration as a deadline.
	// Takes profile (string) which is the profile kind ("cpu",
	// "heap", ...).
	// Takes duration (time.Duration) which is the sampling window.
	//
	// Returns []byte with the pprof-encoded profile data.
	// Returns error when the call fails.
	Capture(ctx context.Context, profile string, duration time.Duration) ([]byte, error)
}

// ProfilingStatus mirrors the GetProfilingStatusResponse surface.
type ProfilingStatus struct {
	// ExpiresAt is when on-demand profiling will turn itself off;
	// zero when no expiry is set.
	ExpiresAt time.Time

	// PprofBaseURL is the base URL exposed by the server for net/http
	// pprof endpoints, when on-demand profiling is enabled.
	PprofBaseURL string

	// AvailableProfiles is the set of profile kinds the server can
	// produce in this build.
	AvailableProfiles []string

	// Remaining is how long is left before automatic disable.
	Remaining time.Duration

	// BlockProfileRate is the runtime block-profile sampling rate.
	BlockProfileRate int32

	// MutexProfileFraction is the runtime mutex-profile sampling fraction.
	MutexProfileFraction int32

	// MemProfileRate is the runtime heap-allocation sampling rate.
	MemProfileRate int32

	// Port is the port on which pprof is exposed.
	Port int32

	// Enabled is true when on-demand profiling is active.
	Enabled bool
}
